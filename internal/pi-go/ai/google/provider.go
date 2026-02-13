// Package google implements the Google Generative AI (Gemini) API provider.
// Uses the REST API with SSE streaming via ?alt=sse.
package google

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"uaa/vdnd/internal/pi-go/ai"
)

func init() {
	ai.RegisterApiProvider(ai.ApiProviderImpl{
		Api:          ai.ApiGoogleGenerativeAI,
		StreamFn:     streamFn,
		StreamSimple: streamSimpleFn,
	}, "google-generative-ai")
}

// --- Request types ---

type generateRequest struct {
	Contents          []content      `json:"contents"`
	Tools             []tool         `json:"tools,omitempty"`
	GenerationConfig  *generationCfg `json:"generationConfig,omitempty"`
	SystemInstruction *content       `json:"systemInstruction,omitempty"`
	ThinkingConfig    *thinkingCfg   `json:"thinkingConfig,omitempty"`
}

type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

type part struct {
	Text             string            `json:"text,omitempty"`
	InlineData       *inlineData       `json:"inlineData,omitempty"`
	FunctionCall     *functionCall     `json:"functionCall,omitempty"`
	FunctionResponse *functionResponse `json:"functionResponse,omitempty"`
	Thought          *bool             `json:"thought,omitempty"`
}

type inlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type functionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

type functionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type tool struct {
	FunctionDeclarations []functionDecl `json:"functionDeclarations,omitempty"`
}

type functionDecl struct {
	Name                 string         `json:"name"`
	Description          string         `json:"description,omitempty"`
	ParametersJsonSchema map[string]any `json:"parametersJsonSchema,omitempty"`
}

type generationCfg struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
}

type thinkingCfg struct {
	IncludeThoughts bool `json:"includeThoughts"`
	ThinkingBudget  *int `json:"thinkingBudget,omitempty"`
}

// --- SSE response types ---

type streamChunk struct {
	Candidates    []candidate    `json:"candidates"`
	UsageMetadata *usageMetadata `json:"usageMetadata,omitempty"`
}

type candidate struct {
	Content      *contentResp `json:"content,omitempty"`
	FinishReason string       `json:"finishReason,omitempty"`
}

type contentResp struct {
	Role  string     `json:"role"`
	Parts []partResp `json:"parts"`
}

type partResp struct {
	Text         string        `json:"text,omitempty"`
	Thought      *bool         `json:"thought,omitempty"`
	FunctionCall *functionCall `json:"functionCall,omitempty"`
}

type usageMetadata struct {
	PromptTokenCount        int `json:"promptTokenCount"`
	CandidatesTokenCount    int `json:"candidatesTokenCount"`
	TotalTokenCount         int `json:"totalTokenCount"`
	ThoughtsTokenCount      int `json:"thoughtsTokenCount"`
	CachedContentTokenCount int `json:"cachedContentTokenCount"`
}

// Unique ID counter for generated tool call IDs
var toolCallCounter int64

// --- Provider implementation ---

func streamFn(ctx context.Context, model ai.Model, llmCtx ai.Context, options *ai.StreamOptions) *ai.AssistantMessageEventStream {
	opts := &ai.SimpleStreamOptions{}
	if options != nil {
		opts.StreamOptions = *options
	}
	return streamSimpleFn(ctx, model, llmCtx, opts)
}

func streamSimpleFn(ctx context.Context, model ai.Model, llmCtx ai.Context, options *ai.SimpleStreamOptions) *ai.AssistantMessageEventStream {
	stream := ai.NewAssistantMessageEventStream()

	go func() {
		msg, err := doStream(ctx, model, llmCtx, options, stream)
		if err != nil {
			errMsg := ai.AssistantMessage{
				Role:         ai.RoleAssistant,
				Content:      []ai.ContentBlock{{Type: ai.ContentTypeText, Text: ""}},
				Api:          ai.ApiGoogleGenerativeAI,
				Provider:     model.Provider,
				Model:        model.ID,
				StopReason:   ai.StopReasonError,
				ErrorMessage: err.Error(),
				Timestamp:    time.Now().UnixMilli(),
			}
			stream.Push(ai.AssistantMessageEvent{Type: ai.EventError, Error: &errMsg})
			stream.End(errMsg)
		} else {
			stream.Push(ai.AssistantMessageEvent{Type: ai.EventDone, Reason: msg.StopReason, Message: &msg})
			stream.End(msg)
		}
	}()

	return stream
}

func doStream(ctx context.Context, model ai.Model, llmCtx ai.Context, options *ai.SimpleStreamOptions, stream *ai.AssistantMessageEventStream) (ai.AssistantMessage, error) {
	apiKey := ""
	if options != nil {
		apiKey = options.APIKey
	}
	if apiKey == "" {
		apiKey = ai.GetEnvApiKey(model.Provider)
	}
	if apiKey == "" {
		return ai.AssistantMessage{}, fmt.Errorf("no API key for provider %q", model.Provider)
	}

	baseURL := model.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Build contents (conversation messages)
	contents := convertMessages(llmCtx)

	// Build tools
	var tools []tool
	if len(llmCtx.Tools) > 0 {
		var decls []functionDecl
		for _, t := range llmCtx.Tools {
			params := t.Parameters
			if params == nil {
				params = map[string]any{"type": "object", "properties": map[string]any{}}
			}
			decls = append(decls, functionDecl{
				Name:                 t.Name,
				Description:          t.Description,
				ParametersJsonSchema: params,
			})
		}
		tools = append(tools, tool{FunctionDeclarations: decls})
	}

	req := generateRequest{
		Contents: contents,
		Tools:    tools,
	}

	// System instruction
	if llmCtx.SystemPrompt != "" {
		req.SystemInstruction = &content{
			Parts: []part{{Text: llmCtx.SystemPrompt}},
		}
	}

	// Generation config
	genCfg := &generationCfg{}
	hasGenCfg := false
	if options != nil && options.Temperature != nil {
		genCfg.Temperature = options.Temperature
		hasGenCfg = true
	}
	maxTokens := 8192
	if model.MaxTokens > 0 {
		maxTokens = model.MaxTokens
	}
	if options != nil && options.MaxTokens != nil {
		maxTokens = *options.MaxTokens
	}
	genCfg.MaxOutputTokens = &maxTokens
	hasGenCfg = true
	if hasGenCfg {
		req.GenerationConfig = genCfg
	}

	// Thinking config
	if options != nil && options.Reasoning != "" {
		budget := getThinkingBudget(model, options.Reasoning)
		req.ThinkingConfig = &thinkingCfg{
			IncludeThoughts: true,
			ThinkingBudget:  &budget,
		}
	}

	if len(tools) == 0 {
		req.Tools = nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return ai.AssistantMessage{}, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", baseURL, model.ID, apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return ai.AssistantMessage{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return ai.AssistantMessage{}, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return ai.AssistantMessage{}, fmt.Errorf("Google API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return parseSSEStream(resp.Body, model, stream)
}

// --- SSE parsing ---

type blockKind int

const (
	bkNone blockKind = iota
	bkText
	bkThinking
)

func parseSSEStream(body io.Reader, model ai.Model, stream *ai.AssistantMessageEventStream) (ai.AssistantMessage, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	result := ai.AssistantMessage{
		Role:       ai.RoleAssistant,
		Api:        ai.ApiGoogleGenerativeAI,
		Provider:   model.Provider,
		Model:      model.ID,
		StopReason: ai.StopReasonStop,
		Timestamp:  time.Now().UnixMilli(),
	}

	currentKind := bkNone
	var textBuf strings.Builder
	var thinkingBuf strings.Builder
	contentIndex := -1

	emitStarted := false

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if !emitStarted {
			stream.Push(ai.AssistantMessageEvent{Type: ai.EventStart, Partial: &result})
			emitStarted = true
		}

		// Process candidates
		if len(chunk.Candidates) > 0 {
			cand := chunk.Candidates[0]
			if cand.Content != nil {
				for _, p := range cand.Content.Parts {
					isThinking := p.Thought != nil && *p.Thought

					// Handle text/thinking content
					if p.Text != "" {
						if isThinking {
							if currentKind != bkThinking {
								finishBlock(currentKind, contentIndex, &textBuf, &thinkingBuf, &result, stream)
								currentKind = bkThinking
								thinkingBuf.Reset()
								result.Content = append(result.Content, ai.ContentBlock{Type: ai.ContentTypeThinking, Thinking: ""})
								contentIndex = len(result.Content) - 1
								stream.Push(ai.AssistantMessageEvent{Type: ai.EventThinkingStart, ContentIndex: contentIndex, Partial: &result})
							}
							thinkingBuf.WriteString(p.Text)
							result.Content[contentIndex].Thinking = thinkingBuf.String()
							stream.Push(ai.AssistantMessageEvent{Type: ai.EventThinkingDelta, ContentIndex: contentIndex, Delta: p.Text, Partial: &result})
						} else {
							if currentKind != bkText {
								finishBlock(currentKind, contentIndex, &textBuf, &thinkingBuf, &result, stream)
								currentKind = bkText
								textBuf.Reset()
								result.Content = append(result.Content, ai.ContentBlock{Type: ai.ContentTypeText, Text: ""})
								contentIndex = len(result.Content) - 1
								stream.Push(ai.AssistantMessageEvent{Type: ai.EventTextStart, ContentIndex: contentIndex, Partial: &result})
							}
							textBuf.WriteString(p.Text)
							result.Content[contentIndex].Text = textBuf.String()
							stream.Push(ai.AssistantMessageEvent{Type: ai.EventTextDelta, ContentIndex: contentIndex, Delta: p.Text, Partial: &result})
						}
					}

					// Handle function calls
					if p.FunctionCall != nil {
						finishBlock(currentKind, contentIndex, &textBuf, &thinkingBuf, &result, stream)
						currentKind = bkNone

						// Generate unique ID
						id := fmt.Sprintf("%s_%d_%d", p.FunctionCall.Name, time.Now().UnixMilli(), atomic.AddInt64(&toolCallCounter, 1))

						tc := ai.ContentBlock{
							Type:      ai.ContentTypeToolCall,
							ID:        id,
							Name:      p.FunctionCall.Name,
							Arguments: p.FunctionCall.Args,
						}
						result.Content = append(result.Content, tc)
						contentIndex = len(result.Content) - 1
						stream.Push(ai.AssistantMessageEvent{Type: ai.EventToolCallStart, ContentIndex: contentIndex, Partial: &result})
						argsJSON, _ := json.Marshal(p.FunctionCall.Args)
						stream.Push(ai.AssistantMessageEvent{Type: ai.EventToolCallDelta, ContentIndex: contentIndex, Delta: string(argsJSON), Partial: &result})
						toolCall := ai.ToolCall{Type: ai.ContentTypeToolCall, ID: id, Name: p.FunctionCall.Name, Arguments: p.FunctionCall.Args}
						stream.Push(ai.AssistantMessageEvent{Type: ai.EventToolCallEnd, ContentIndex: contentIndex, ToolCall: &toolCall, Partial: &result})
					}
				}
			}

			// Finish reason
			if cand.FinishReason != "" {
				result.StopReason = mapFinishReason(cand.FinishReason)
				// Override to toolUse if we have tool calls
				for _, b := range result.Content {
					if b.Type == ai.ContentTypeToolCall {
						result.StopReason = ai.StopReasonToolUse
						break
					}
				}
			}
		}

		// Usage
		if chunk.UsageMetadata != nil {
			m := chunk.UsageMetadata
			result.Usage = ai.Usage{
				Input:       m.PromptTokenCount,
				Output:      m.CandidatesTokenCount + m.ThoughtsTokenCount,
				CacheRead:   m.CachedContentTokenCount,
				TotalTokens: m.TotalTokenCount,
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return ai.AssistantMessage{}, fmt.Errorf("stream read error: %w", err)
	}

	// Finish final block
	finishBlock(currentKind, contentIndex, &textBuf, &thinkingBuf, &result, stream)

	return result, nil
}

func finishBlock(kind blockKind, idx int, textBuf, thinkingBuf *strings.Builder, result *ai.AssistantMessage, stream *ai.AssistantMessageEventStream) {
	if idx < 0 {
		return
	}
	switch kind {
	case bkText:
		stream.Push(ai.AssistantMessageEvent{Type: ai.EventTextEnd, ContentIndex: idx, Content: textBuf.String(), Partial: result})
	case bkThinking:
		stream.Push(ai.AssistantMessageEvent{Type: ai.EventThinkingEnd, ContentIndex: idx, Content: thinkingBuf.String(), Partial: result})
	}
}

func mapFinishReason(reason string) ai.StopReason {
	switch reason {
	case "STOP":
		return ai.StopReasonStop
	case "MAX_TOKENS":
		return ai.StopReasonLength
	default:
		return ai.StopReasonError
	}
}

func getThinkingBudget(model ai.Model, effort ai.ThinkingLevel) int {
	switch effort {
	case "minimal":
		return 128
	case "low":
		return 2048
	case "medium":
		return 8192
	case "high":
		if strings.Contains(model.ID, "2.5-pro") {
			return 32768
		}
		return 24576
	default:
		return 8192
	}
}

// --- Message conversion ---

func convertMessages(llmCtx ai.Context) []content {
	var contents []content

	for _, msg := range llmCtx.Messages {
		switch msg.Role {
		case ai.RoleUser:
			if msg.User == nil {
				continue
			}
			var parts []part
			for _, b := range msg.User.Content {
				switch b.Type {
				case ai.ContentTypeText:
					parts = append(parts, part{Text: b.Text})
				case ai.ContentTypeImage:
					parts = append(parts, part{InlineData: &inlineData{MimeType: b.MimeType, Data: b.Data}})
				}
			}
			if len(parts) > 0 {
				contents = append(contents, content{Role: "user", Parts: parts})
			}

		case ai.RoleAssistant:
			if msg.Assistant == nil {
				continue
			}
			if msg.Assistant.StopReason == ai.StopReasonError || msg.Assistant.StopReason == ai.StopReasonAborted {
				continue
			}
			var parts []part
			for _, b := range msg.Assistant.Content {
				switch b.Type {
				case ai.ContentTypeText:
					if strings.TrimSpace(b.Text) != "" {
						parts = append(parts, part{Text: b.Text})
					}
				case ai.ContentTypeThinking:
					if strings.TrimSpace(b.Thinking) != "" {
						t := true
						parts = append(parts, part{Thought: &t, Text: b.Thinking})
					}
				case ai.ContentTypeToolCall:
					parts = append(parts, part{
						FunctionCall: &functionCall{
							Name: b.Name,
							Args: b.Arguments,
						},
					})
				}
			}
			if len(parts) > 0 {
				contents = append(contents, content{Role: "model", Parts: parts})
			}

		case ai.RoleToolResult:
			if msg.ToolResult == nil {
				continue
			}
			text := ""
			for _, b := range msg.ToolResult.Content {
				if b.Type == ai.ContentTypeText {
					text += b.Text
				}
			}
			if text == "" {
				text = "(no output)"
			}
			respKey := "output"
			if msg.ToolResult.IsError {
				respKey = "error"
			}
			frPart := part{
				FunctionResponse: &functionResponse{
					Name:     msg.ToolResult.ToolName,
					Response: map[string]any{respKey: text},
				},
			}
			// Merge with previous user turn that has function responses
			if len(contents) > 0 {
				last := &contents[len(contents)-1]
				if last.Role == "user" && len(last.Parts) > 0 && last.Parts[0].FunctionResponse != nil {
					last.Parts = append(last.Parts, frPart)
					continue
				}
			}
			contents = append(contents, content{Role: "user", Parts: []part{frPart}})
		}
	}

	return contents
}
