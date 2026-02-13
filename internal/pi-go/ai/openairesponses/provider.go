// Package openairesponses implements the OpenAI Responses API provider.
// This is OpenAI's newer API format used by OpenAI native and GitHub Copilot.
package openairesponses

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"uaa/vdnd/internal/pi-go/ai"
)

func init() {
	ai.RegisterApiProvider(ai.ApiProviderImpl{
		Api:          ai.ApiOpenAIResponses,
		StreamFn:     streamFn,
		StreamSimple: streamSimpleFn,
	}, "openai-responses")
}

// --- Request types ---

type request struct {
	Model           string     `json:"model"`
	Input           []any      `json:"input"`
	Stream          bool       `json:"stream"`
	MaxOutputTokens *int       `json:"max_output_tokens,omitempty"`
	Temperature     *float64   `json:"temperature,omitempty"`
	Tools           []apiTool  `json:"tools,omitempty"`
	Store           bool       `json:"store"`
	Reasoning       *reasoning `json:"reasoning,omitempty"`
	PromptCacheKey  string     `json:"prompt_cache_key,omitempty"`
}

type reasoning struct {
	Effort  string `json:"effort,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type apiTool struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	Strict      bool           `json:"strict"`
}

// --- SSE event types ---

type sseEvent struct {
	Type string          `json:"type"`
	Item json.RawMessage `json:"item,omitempty"`
	// For delta events
	Delta    string          `json:"delta,omitempty"`
	Part     json.RawMessage `json:"part,omitempty"`
	Response json.RawMessage `json:"response,omitempty"`
	// Error events
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type outputItem struct {
	Type      string `json:"type"`
	ID        string `json:"id,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	Status    string `json:"status,omitempty"`
	Content   []struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	} `json:"content,omitempty"`
	Summary []struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	} `json:"summary,omitempty"`
}

type completedResponse struct {
	Status string `json:"status"`
	Usage  *struct {
		InputTokens        int `json:"input_tokens"`
		OutputTokens       int `json:"output_tokens"`
		TotalTokens        int `json:"total_tokens"`
		InputTokensDetails *struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"input_tokens_details,omitempty"`
	} `json:"usage,omitempty"`
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
				Api:          ai.ApiOpenAIResponses,
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

	// Build input messages
	input := convertMessages(llmCtx)

	// Build tools
	var tools []apiTool
	for _, t := range llmCtx.Tools {
		params := t.Parameters
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		tools = append(tools, apiTool{
			Type:        "function",
			Name:        t.Name,
			Description: t.Description,
			Parameters:  params,
			Strict:      false,
		})
	}

	maxTokens := 8192
	if model.MaxTokens > 0 {
		maxTokens = model.MaxTokens
	}
	if options != nil && options.MaxTokens != nil {
		maxTokens = *options.MaxTokens
	}

	req := request{
		Model:           model.ID,
		Input:           input,
		Stream:          true,
		MaxOutputTokens: &maxTokens,
		Tools:           tools,
		Store:           false,
	}

	if options != nil && options.Temperature != nil {
		req.Temperature = options.Temperature
	}

	if options != nil && options.Reasoning != "" {
		req.Reasoning = &reasoning{
			Effort:  string(options.Reasoning),
			Summary: "auto",
		}
	}

	if options != nil && options.SessionID != "" {
		req.PromptCacheKey = options.SessionID
	}

	if len(tools) == 0 {
		req.Tools = nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return ai.AssistantMessage{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/responses", bytes.NewReader(body))
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

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return ai.AssistantMessage{}, fmt.Errorf("OpenAI Responses API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return parseSSEStream(resp.Body, model, stream)
}

// --- SSE parsing ---

func parseSSEStream(body io.Reader, model ai.Model, stream *ai.AssistantMessageEventStream) (ai.AssistantMessage, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	result := ai.AssistantMessage{
		Role:       ai.RoleAssistant,
		Api:        ai.ApiOpenAIResponses,
		Provider:   model.Provider,
		Model:      model.ID,
		StopReason: ai.StopReasonStop,
		Timestamp:  time.Now().UnixMilli(),
	}

	type itemState struct {
		itemType     string // "reasoning", "message", "function_call"
		contentIndex int
		callID       string
		name         string
		partialArgs  strings.Builder
		textBuf      strings.Builder
		thinkingBuf  strings.Builder
	}
	var current *itemState

	emitStarted := false
	// Track SSE event type from "event: " lines
	var eventType string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		if !emitStarted {
			stream.Push(ai.AssistantMessageEvent{Type: ai.EventStart, Partial: &result})
			emitStarted = true
		}

		switch eventType {
		case "response.output_item.added":
			var raw struct {
				Item outputItem `json:"item"`
			}
			if json.Unmarshal([]byte(data), &raw) != nil {
				continue
			}
			item := raw.Item

			switch item.Type {
			case "reasoning":
				result.Content = append(result.Content, ai.ContentBlock{Type: ai.ContentTypeThinking, Thinking: ""})
				idx := len(result.Content) - 1
				current = &itemState{itemType: "reasoning", contentIndex: idx}
				stream.Push(ai.AssistantMessageEvent{Type: ai.EventThinkingStart, ContentIndex: idx, Partial: &result})

			case "message":
				result.Content = append(result.Content, ai.ContentBlock{Type: ai.ContentTypeText, Text: ""})
				idx := len(result.Content) - 1
				current = &itemState{itemType: "message", contentIndex: idx}
				stream.Push(ai.AssistantMessageEvent{Type: ai.EventTextStart, ContentIndex: idx, Partial: &result})

			case "function_call":
				result.Content = append(result.Content, ai.ContentBlock{
					Type: ai.ContentTypeToolCall,
					ID:   item.CallID,
					Name: item.Name,
				})
				idx := len(result.Content) - 1
				current = &itemState{
					itemType:     "function_call",
					contentIndex: idx,
					callID:       item.CallID,
					name:         item.Name,
				}
				stream.Push(ai.AssistantMessageEvent{Type: ai.EventToolCallStart, ContentIndex: idx, Partial: &result})
			}

		case "response.reasoning_summary_text.delta":
			if current != nil && current.itemType == "reasoning" {
				var raw struct {
					Delta string `json:"delta"`
				}
				json.Unmarshal([]byte(data), &raw)
				current.thinkingBuf.WriteString(raw.Delta)
				result.Content[current.contentIndex].Thinking = current.thinkingBuf.String()
				stream.Push(ai.AssistantMessageEvent{
					Type:         ai.EventThinkingDelta,
					ContentIndex: current.contentIndex,
					Delta:        raw.Delta,
					Partial:      &result,
				})
			}

		case "response.output_text.delta":
			if current != nil && current.itemType == "message" {
				var raw struct {
					Delta string `json:"delta"`
				}
				json.Unmarshal([]byte(data), &raw)
				current.textBuf.WriteString(raw.Delta)
				result.Content[current.contentIndex].Text = current.textBuf.String()
				stream.Push(ai.AssistantMessageEvent{
					Type:         ai.EventTextDelta,
					ContentIndex: current.contentIndex,
					Delta:        raw.Delta,
					Partial:      &result,
				})
			}

		case "response.refusal.delta":
			if current != nil && current.itemType == "message" {
				var raw struct {
					Delta string `json:"delta"`
				}
				json.Unmarshal([]byte(data), &raw)
				current.textBuf.WriteString(raw.Delta)
				result.Content[current.contentIndex].Text = current.textBuf.String()
				stream.Push(ai.AssistantMessageEvent{
					Type:         ai.EventTextDelta,
					ContentIndex: current.contentIndex,
					Delta:        raw.Delta,
					Partial:      &result,
				})
			}

		case "response.function_call_arguments.delta":
			if current != nil && current.itemType == "function_call" {
				var raw struct {
					Delta string `json:"delta"`
				}
				json.Unmarshal([]byte(data), &raw)
				current.partialArgs.WriteString(raw.Delta)
				// Try parse partial JSON
				var args map[string]any
				if json.Unmarshal([]byte(current.partialArgs.String()), &args) == nil {
					result.Content[current.contentIndex].Arguments = args
				}
				stream.Push(ai.AssistantMessageEvent{
					Type:         ai.EventToolCallDelta,
					ContentIndex: current.contentIndex,
					Delta:        raw.Delta,
					Partial:      &result,
				})
			}

		case "response.output_item.done":
			if current == nil {
				continue
			}
			idx := current.contentIndex

			switch current.itemType {
			case "reasoning":
				stream.Push(ai.AssistantMessageEvent{
					Type:         ai.EventThinkingEnd,
					ContentIndex: idx,
					Content:      current.thinkingBuf.String(),
					Partial:      &result,
				})
			case "message":
				stream.Push(ai.AssistantMessageEvent{
					Type:         ai.EventTextEnd,
					ContentIndex: idx,
					Content:      current.textBuf.String(),
					Partial:      &result,
				})
			case "function_call":
				var args map[string]any
				if json.Unmarshal([]byte(current.partialArgs.String()), &args) == nil {
					result.Content[idx].Arguments = args
				}
				tc := ai.ToolCall{
					Type:      ai.ContentTypeToolCall,
					ID:        current.callID,
					Name:      current.name,
					Arguments: args,
				}
				stream.Push(ai.AssistantMessageEvent{
					Type:         ai.EventToolCallEnd,
					ContentIndex: idx,
					ToolCall:     &tc,
					Partial:      &result,
				})
			}
			current = nil

		case "response.completed":
			var raw struct {
				Response completedResponse `json:"response"`
			}
			if json.Unmarshal([]byte(data), &raw) == nil && raw.Response.Usage != nil {
				u := raw.Response.Usage
				cachedTokens := 0
				if u.InputTokensDetails != nil {
					cachedTokens = u.InputTokensDetails.CachedTokens
				}
				result.Usage = ai.Usage{
					Input:       u.InputTokens - cachedTokens,
					Output:      u.OutputTokens,
					CacheRead:   cachedTokens,
					TotalTokens: u.TotalTokens,
				}
				result.StopReason = mapResponseStatus(raw.Response.Status)
			}
			// If has tool calls, override stop reason
			for _, b := range result.Content {
				if b.Type == ai.ContentTypeToolCall {
					result.StopReason = ai.StopReasonToolUse
					break
				}
			}

		case "error":
			var raw struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			json.Unmarshal([]byte(data), &raw)
			return ai.AssistantMessage{}, fmt.Errorf("error %s: %s", raw.Code, raw.Message)

		case "response.failed":
			return ai.AssistantMessage{}, fmt.Errorf("response failed")
		}

		eventType = "" // reset
	}

	if err := scanner.Err(); err != nil {
		return ai.AssistantMessage{}, fmt.Errorf("stream read error: %w", err)
	}

	return result, nil
}

func mapResponseStatus(status string) ai.StopReason {
	switch status {
	case "completed":
		return ai.StopReasonStop
	case "incomplete":
		return ai.StopReasonLength
	case "failed", "cancelled":
		return ai.StopReasonError
	default:
		return ai.StopReasonStop
	}
}

// --- Message conversion ---

func convertMessages(llmCtx ai.Context) []any {
	var input []any

	if llmCtx.SystemPrompt != "" {
		input = append(input, map[string]any{
			"role":    "developer",
			"content": llmCtx.SystemPrompt,
		})
	}

	for _, msg := range llmCtx.Messages {
		switch msg.Role {
		case ai.RoleUser:
			if msg.User == nil {
				continue
			}
			var content []map[string]any
			for _, b := range msg.User.Content {
				switch b.Type {
				case ai.ContentTypeText:
					content = append(content, map[string]any{
						"type": "input_text",
						"text": b.Text,
					})
				case ai.ContentTypeImage:
					content = append(content, map[string]any{
						"type":      "input_image",
						"detail":    "auto",
						"image_url": fmt.Sprintf("data:%s;base64,%s", b.MimeType, b.Data),
					})
				}
			}
			if len(content) > 0 {
				input = append(input, map[string]any{
					"role":    "user",
					"content": content,
				})
			}

		case ai.RoleAssistant:
			if msg.Assistant == nil {
				continue
			}
			if msg.Assistant.StopReason == ai.StopReasonError || msg.Assistant.StopReason == ai.StopReasonAborted {
				continue
			}
			for _, b := range msg.Assistant.Content {
				switch b.Type {
				case ai.ContentTypeText:
					if strings.TrimSpace(b.Text) == "" {
						continue
					}
					input = append(input, map[string]any{
						"type":   "message",
						"role":   "assistant",
						"status": "completed",
						"content": []map[string]any{
							{"type": "output_text", "text": b.Text, "annotations": []any{}},
						},
					})
				case ai.ContentTypeToolCall:
					argsJSON, _ := json.Marshal(b.Arguments)
					input = append(input, map[string]any{
						"type":      "function_call",
						"call_id":   b.ID,
						"name":      b.Name,
						"arguments": string(argsJSON),
					})
				}
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
			callID := msg.ToolResult.ToolCallID
			// Strip pipe-separated item ID if present
			if idx := strings.Index(callID, "|"); idx >= 0 {
				callID = callID[:idx]
			}
			input = append(input, map[string]any{
				"type":    "function_call_output",
				"call_id": callID,
				"output":  text,
			})
		}
	}

	return input
}
