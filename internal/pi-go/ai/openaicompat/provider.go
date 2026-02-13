// Package openaicompat implements the OpenAI Chat Completions API provider.
// This is the most widely used API format, compatible with: OpenAI, Groq,
// DeepSeek, xAI, Cerebras, Mistral, OpenRouter, and many other providers.
package openaicompat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"uaa/vdnd/internal/pi-go/ai"
)

func init() {
	ai.RegisterApiProvider(ai.ApiProviderImpl{
		Api:          ai.ApiOpenAICompletions,
		StreamFn:     streamFn,
		StreamSimple: streamSimpleFn,
	}, "openai-completions")
}

// --- Request types ---

type request struct {
	Model           string       `json:"model"`
	Messages        []apiMessage `json:"messages"`
	Stream          bool         `json:"stream"`
	StreamOptions   *streamOpts  `json:"stream_options,omitempty"`
	Tools           []apiTool    `json:"tools,omitempty"`
	ToolChoice      any          `json:"tool_choice,omitempty"`
	Temperature     *float64     `json:"temperature,omitempty"`
	MaxTokens       *int         `json:"max_completion_tokens,omitempty"`
	ReasoningEffort string       `json:"reasoning_effort,omitempty"`
	Store           *bool        `json:"store,omitempty"`
	EnableThinking  *bool        `json:"enable_thinking,omitempty"` // Qwen
}

type streamOpts struct {
	IncludeUsage bool `json:"include_usage"`
}

type apiMessage struct {
	Role       string      `json:"role"`
	Content    any         `json:"content"` // string or []apiContent
	ToolCalls  []apiToolFn `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"` // for "tool" role
	Name       string      `json:"name,omitempty"`         // for "tool" role (Mistral)
}

type apiContent struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type apiToolFn struct {
	ID       string    `json:"id"`
	Type     string    `json:"type"`
	Function apiFnCall `json:"function"`
}

type apiFnCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type apiTool struct {
	Type     string   `json:"type"`
	Function apiFnDef `json:"function"`
}

type apiFnDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// --- SSE response types ---

type sseChunk struct {
	ID      string      `json:"id"`
	Choices []sseChoice `json:"choices"`
	Usage   *sseUsage   `json:"usage,omitempty"`
}

type sseChoice struct {
	Delta        sseDelta `json:"delta"`
	FinishReason *string  `json:"finish_reason"`
}

type sseDelta struct {
	Content          *string       `json:"content"`
	ToolCalls        []sseToolCall `json:"tool_calls,omitempty"`
	ReasoningContent *string       `json:"reasoning_content"`
	Reasoning        *string       `json:"reasoning"`
	ReasoningText    *string       `json:"reasoning_text"`
}

type sseToolCall struct {
	Index    int        `json:"index"`
	ID       string     `json:"id,omitempty"`
	Function *sseFnCall `json:"function,omitempty"`
}

type sseFnCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type sseUsage struct {
	PromptTokens            int                `json:"prompt_tokens"`
	CompletionTokens        int                `json:"completion_tokens"`
	TotalTokens             int                `json:"total_tokens"`
	PromptTokensDetails     *tokenDetails      `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *completionDetails `json:"completion_tokens_details,omitempty"`
}

type tokenDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type completionDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

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
				Api:          ai.ApiOpenAICompletions,
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
	baseURL := model.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	// Strip trailing /v1 if present (we add the endpoint path)
	baseURL = strings.TrimSuffix(baseURL, "/")
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL += "/v1"
	}

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

	// Convert messages
	msgs := convertMessages(llmCtx)

	// Convert tools
	var tools []apiTool
	for _, t := range llmCtx.Tools {
		params := t.Parameters
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		tools = append(tools, apiTool{
			Type: "function",
			Function: apiFnDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}

	maxTokens := 8192
	if model.MaxTokens > 0 {
		maxTokens = model.MaxTokens
	}
	if options != nil && options.MaxTokens != nil {
		maxTokens = *options.MaxTokens
	}

	storeFalse := false
	req := request{
		Model:         model.ID,
		Messages:      msgs,
		Stream:        true,
		StreamOptions: &streamOpts{IncludeUsage: true},
		Tools:         tools,
		MaxTokens:     &maxTokens,
		Store:         &storeFalse,
	}

	if options != nil && options.Temperature != nil {
		req.Temperature = options.Temperature
	}

	// Reasoning support
	if options != nil && options.Reasoning != "" {
		req.ReasoningEffort = string(options.Reasoning)
	}

	if len(tools) == 0 {
		req.Tools = nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return ai.AssistantMessage{}, fmt.Errorf("marshal request: %w", err)
	}

	slog.Debug("sending request", "url", baseURL+"/chat/completions", "payload", string(body))

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return ai.AssistantMessage{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return ai.AssistantMessage{}, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	slog.Info("received response", "status", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return ai.AssistantMessage{}, fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return parseSSEStream(resp.Body, model, stream)
}

// --- SSE parsing ---

type blockType int

const (
	btNone blockType = iota
	btText
	btThinking
	btToolCall
)

type toolCallState struct {
	id          string
	name        string
	partialArgs strings.Builder
}

func parseSSEStream(body io.Reader, model ai.Model, stream *ai.AssistantMessageEventStream) (ai.AssistantMessage, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	result := ai.AssistantMessage{
		Role:       ai.RoleAssistant,
		Api:        ai.ApiOpenAICompletions,
		Provider:   model.Provider,
		Model:      model.ID,
		StopReason: ai.StopReasonStop,
		Timestamp:  time.Now().UnixMilli(),
	}

	currentType := btNone
	var textBuf strings.Builder
	var thinkingBuf strings.Builder
	contentIndex := -1
	toolCalls := map[int]*toolCallState{} // indexed by tool_call index

	emitStarted := false

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 {
			slog.Info("SSE line", "length", len(line), "content", line)
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk sseChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		// Usage
		if chunk.Usage != nil {
			cachedTokens := 0
			if chunk.Usage.PromptTokensDetails != nil {
				cachedTokens = chunk.Usage.PromptTokensDetails.CachedTokens
			}
			reasoningTokens := 0
			if chunk.Usage.CompletionTokensDetails != nil {
				reasoningTokens = chunk.Usage.CompletionTokensDetails.ReasoningTokens
			}
			input := chunk.Usage.PromptTokens - cachedTokens
			output := chunk.Usage.CompletionTokens + reasoningTokens
			result.Usage = ai.Usage{
				Input:       input,
				Output:      output,
				CacheRead:   cachedTokens,
				TotalTokens: input + output + cachedTokens,
			}
		}

		if len(chunk.Choices) == 0 {
			continue
		}
		choice := chunk.Choices[0]

		if !emitStarted {
			stream.Push(ai.AssistantMessageEvent{Type: ai.EventStart, Partial: &result})
			emitStarted = true
		}

		// Stop reason
		if choice.FinishReason != nil {
			result.StopReason = mapStopReason(*choice.FinishReason)
		}

		delta := choice.Delta

		// Text content
		if delta.Content != nil && len(*delta.Content) > 0 {
			if currentType != btText {
				finishBlock(currentType, contentIndex, &textBuf, &thinkingBuf, &result, stream)
				currentType = btText
				textBuf.Reset()
				result.Content = append(result.Content, ai.ContentBlock{Type: ai.ContentTypeText, Text: ""})
				contentIndex = len(result.Content) - 1
				stream.Push(ai.AssistantMessageEvent{Type: ai.EventTextStart, ContentIndex: contentIndex, Partial: &result})
			}
			textBuf.WriteString(*delta.Content)
			result.Content[contentIndex].Text = textBuf.String()
			stream.Push(ai.AssistantMessageEvent{Type: ai.EventTextDelta, ContentIndex: contentIndex, Delta: *delta.Content, Partial: &result})
		}

		// Reasoning content (DeepSeek, llama.cpp, etc.)
		reasoning := firstNonEmpty(delta.ReasoningContent, delta.Reasoning, delta.ReasoningText)
		if reasoning != "" {
			if currentType != btThinking {
				finishBlock(currentType, contentIndex, &textBuf, &thinkingBuf, &result, stream)
				currentType = btThinking
				thinkingBuf.Reset()
				result.Content = append(result.Content, ai.ContentBlock{Type: ai.ContentTypeThinking, Thinking: ""})
				contentIndex = len(result.Content) - 1
				stream.Push(ai.AssistantMessageEvent{Type: ai.EventThinkingStart, ContentIndex: contentIndex, Partial: &result})
			}
			thinkingBuf.WriteString(reasoning)
			result.Content[contentIndex].Thinking = thinkingBuf.String()
			stream.Push(ai.AssistantMessageEvent{Type: ai.EventThinkingDelta, ContentIndex: contentIndex, Delta: reasoning, Partial: &result})
		}

		// Tool calls
		for _, tc := range delta.ToolCalls {
			state, exists := toolCalls[tc.Index]
			if !exists {
				// New tool call — finish previous block
				finishBlock(currentType, contentIndex, &textBuf, &thinkingBuf, &result, stream)
				currentType = btToolCall

				state = &toolCallState{id: tc.ID}
				if tc.Function != nil {
					state.name = tc.Function.Name
				}
				toolCalls[tc.Index] = state

				result.Content = append(result.Content, ai.ContentBlock{
					Type: ai.ContentTypeToolCall,
					ID:   state.id,
					Name: state.name,
				})
				contentIndex = len(result.Content) - 1
				stream.Push(ai.AssistantMessageEvent{Type: ai.EventToolCallStart, ContentIndex: contentIndex, Partial: &result})
			}

			if tc.ID != "" {
				state.id = tc.ID
				result.Content[contentIndex].ID = tc.ID
			}
			if tc.Function != nil {
				if tc.Function.Name != "" {
					state.name = tc.Function.Name
					result.Content[contentIndex].Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					state.partialArgs.WriteString(tc.Function.Arguments)
					// Try to parse partial JSON
					var args map[string]any
					if json.Unmarshal([]byte(state.partialArgs.String()), &args) == nil {
						result.Content[contentIndex].Arguments = args
					}
					stream.Push(ai.AssistantMessageEvent{
						Type:         ai.EventToolCallDelta,
						ContentIndex: contentIndex,
						Delta:        tc.Function.Arguments,
						Partial:      &result,
					})
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return ai.AssistantMessage{}, fmt.Errorf("stream read error: %w", err)
	}

	// Finish any pending block
	finishBlock(currentType, contentIndex, &textBuf, &thinkingBuf, &result, stream)

	// Finish pending tool calls
	for idx, state := range toolCalls {
		_ = idx
		var args map[string]any
		if json.Unmarshal([]byte(state.partialArgs.String()), &args) == nil {
			// Find matching content block
			for i := range result.Content {
				if result.Content[i].Type == ai.ContentTypeToolCall && result.Content[i].ID == state.id {
					result.Content[i].Arguments = args
					tc := ai.ToolCall{
						Type:      ai.ContentTypeToolCall,
						ID:        state.id,
						Name:      state.name,
						Arguments: args,
					}
					stream.Push(ai.AssistantMessageEvent{
						Type:         ai.EventToolCallEnd,
						ContentIndex: i,
						ToolCall:     &tc,
						Partial:      &result,
					})
					break
				}
			}
		}
	}

	result.Usage.TotalTokens = result.Usage.Input + result.Usage.Output + result.Usage.CacheRead
	return result, nil
}

func finishBlock(bt blockType, idx int, textBuf, thinkingBuf *strings.Builder, result *ai.AssistantMessage, stream *ai.AssistantMessageEventStream) {
	if idx < 0 {
		return
	}
	switch bt {
	case btText:
		stream.Push(ai.AssistantMessageEvent{Type: ai.EventTextEnd, ContentIndex: idx, Content: textBuf.String(), Partial: result})
	case btThinking:
		stream.Push(ai.AssistantMessageEvent{Type: ai.EventThinkingEnd, ContentIndex: idx, Content: thinkingBuf.String(), Partial: result})
	}
}

func firstNonEmpty(ptrs ...*string) string {
	for _, p := range ptrs {
		if p != nil && len(*p) > 0 {
			return *p
		}
	}
	return ""
}

func mapStopReason(reason string) ai.StopReason {
	switch reason {
	case "stop":
		return ai.StopReasonStop
	case "length":
		return ai.StopReasonLength
	case "tool_calls", "function_call":
		return ai.StopReasonToolUse
	case "content_filter":
		return ai.StopReasonError
	default:
		return ai.StopReasonStop
	}
}

// --- Message conversion ---

func convertMessages(llmCtx ai.Context) []apiMessage {
	var msgs []apiMessage

	if llmCtx.SystemPrompt != "" {
		msgs = append(msgs, apiMessage{Role: "system", Content: llmCtx.SystemPrompt})
	}

	for _, msg := range llmCtx.Messages {
		switch msg.Role {
		case ai.RoleUser:
			if msg.User == nil {
				continue
			}
			// Check if all content is text — use simple string format
			allText := true
			for _, b := range msg.User.Content {
				if b.Type != ai.ContentTypeText {
					allText = false
					break
				}
			}
			if allText && len(msg.User.Content) == 1 {
				msgs = append(msgs, apiMessage{Role: "user", Content: msg.User.Content[0].Text})
			} else {
				var parts []apiContent
				for _, b := range msg.User.Content {
					switch b.Type {
					case ai.ContentTypeText:
						parts = append(parts, apiContent{Type: "text", Text: b.Text})
					case ai.ContentTypeImage:
						parts = append(parts, apiContent{
							Type:     "image_url",
							ImageURL: &imageURL{URL: fmt.Sprintf("data:%s;base64,%s", b.MimeType, b.Data)},
						})
					}
				}
				msgs = append(msgs, apiMessage{Role: "user", Content: parts})
			}

		case ai.RoleAssistant:
			if msg.Assistant == nil {
				continue
			}
			// Skip errored/aborted messages
			if msg.Assistant.StopReason == ai.StopReasonError || msg.Assistant.StopReason == ai.StopReasonAborted {
				continue
			}

			am := apiMessage{Role: "assistant"}

			// Text content
			var textParts []apiContent
			for _, b := range msg.Assistant.Content {
				if b.Type == ai.ContentTypeText && strings.TrimSpace(b.Text) != "" {
					textParts = append(textParts, apiContent{Type: "text", Text: b.Text})
				}
			}
			if len(textParts) == 1 {
				am.Content = textParts[0].Text
			} else if len(textParts) > 1 {
				am.Content = textParts
			}

			// Tool calls
			var tcs []apiToolFn
			for _, b := range msg.Assistant.Content {
				if b.Type == ai.ContentTypeToolCall {
					argsJSON, _ := json.Marshal(b.Arguments)
					tcs = append(tcs, apiToolFn{
						ID:   b.ID,
						Type: "function",
						Function: apiFnCall{
							Name:      b.Name,
							Arguments: string(argsJSON),
						},
					})
				}
			}
			if len(tcs) > 0 {
				am.ToolCalls = tcs
			}

			// Skip empty assistant messages
			if am.Content == nil && len(am.ToolCalls) == 0 {
				continue
			}
			msgs = append(msgs, am)

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
			msgs = append(msgs, apiMessage{
				Role:       "tool",
				Content:    text,
				ToolCallID: msg.ToolResult.ToolCallID,
			})
		}
	}

	return msgs
}
