// Package anthropic implements the Anthropic Messages API provider for the ai package.
package anthropic

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

const (
	defaultBaseURL = "https://api.anthropic.com"
	apiVersion     = "2023-06-01"
)

func init() {
	ai.RegisterApiProvider(ai.ApiProviderImpl{
		Api:          ai.ApiAnthropicMessages,
		StreamFn:     streamFn,
		StreamSimple: streamSimpleFn,
	}, "anthropic")
}

// --- Request types ---

type request struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	System      string          `json:"system,omitempty"`
	Messages    []apiMessage    `json:"messages"`
	Tools       []apiTool       `json:"tools,omitempty"`
	Stream      bool            `json:"stream"`
	Temperature *float64        `json:"temperature,omitempty"`
	Thinking    *thinkingConfig `json:"thinking,omitempty"`
}

type thinkingConfig struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

type apiMessage struct {
	Role    string       `json:"role"`
	Content []apiContent `json:"content"`
}

type apiContent struct {
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	ToolUseID string         `json:"tool_use_id,omitempty"`
	Content   string         `json:"content,omitempty"` // for tool_result
	IsError   bool           `json:"is_error,omitempty"`
	Thinking  string         `json:"thinking,omitempty"`
	Signature string         `json:"signature,omitempty"`
}

type apiTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema"`
}

// --- SSE event types ---

type sseMessageStart struct {
	Type    string            `json:"type"`
	Message sseMessagePayload `json:"message"`
}

type sseMessagePayload struct {
	ID           string   `json:"id"`
	Role         string   `json:"role"`
	Content      []any    `json:"content"`
	Model        string   `json:"model"`
	StopReason   *string  `json:"stop_reason"`
	StopSequence *string  `json:"stop_sequence"`
	Usage        sseUsage `json:"usage"`
}

type sseUsage struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	CacheReadTokens     int `json:"cache_read_input_tokens"`
	CacheCreationTokens int `json:"cache_creation_input_tokens"`
}

type sseContentBlockStart struct {
	Type         string     `json:"type"`
	Index        int        `json:"index"`
	ContentBlock sseContent `json:"content_block"`
}

type sseContent struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`
}

type sseContentBlockDelta struct {
	Type  string   `json:"type"`
	Index int      `json:"index"`
	Delta sseDelta `json:"delta"`
}

type sseDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	Thinking    string `json:"thinking,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	Signature   string `json:"signature,omitempty"`
}

type sseMessageDelta struct {
	Type  string                 `json:"type"`
	Delta sseMessageDeltaPayload `json:"delta"`
	Usage sseUsage               `json:"usage"`
}

type sseMessageDeltaPayload struct {
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
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
				Provider:     model.Provider,
				Model:        model.ID,
				StopReason:   ai.StopReasonError,
				ErrorMessage: err.Error(),
				Timestamp:    time.Now().UnixMilli(),
			}
			stream.Push(ai.AssistantMessageEvent{
				Type:  ai.EventError,
				Error: &errMsg,
			})
			stream.End(errMsg)
		} else {
			stream.Push(ai.AssistantMessageEvent{
				Type:    ai.EventDone,
				Reason:  msg.StopReason,
				Message: &msg,
			})
			stream.End(msg)
		}
	}()

	return stream
}

func doStream(ctx context.Context, model ai.Model, llmCtx ai.Context, options *ai.SimpleStreamOptions, stream *ai.AssistantMessageEventStream) (ai.AssistantMessage, error) {
	baseURL := model.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	apiKey := ""
	if options != nil {
		apiKey = options.APIKey
	}
	if apiKey == "" {
		apiKey = ai.GetEnvApiKey(ai.ProviderAnthropic)
	}
	if apiKey == "" {
		return ai.AssistantMessage{}, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	// Convert messages
	msgs, err := convertMessages(llmCtx.Messages)
	if err != nil {
		return ai.AssistantMessage{}, err
	}

	// Convert tools
	var tools []apiTool
	for _, t := range llmCtx.Tools {
		schema := t.Parameters
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		tools = append(tools, apiTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
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
		Model:     model.ID,
		MaxTokens: maxTokens,
		System:    llmCtx.SystemPrompt,
		Messages:  msgs,
		Tools:     tools,
		Stream:    true,
	}

	if options != nil && options.Temperature != nil {
		req.Temperature = options.Temperature
	}

	body, err := json.Marshal(req)
	if err != nil {
		return ai.AssistantMessage{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return ai.AssistantMessage{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", apiKey)
	httpReq.Header.Set("Anthropic-Version", apiVersion)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return ai.AssistantMessage{}, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return ai.AssistantMessage{}, fmt.Errorf("Anthropic API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return parseSSEStream(resp.Body, model, stream)
}

func parseSSEStream(body io.Reader, model ai.Model, stream *ai.AssistantMessageEventStream) (ai.AssistantMessage, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for large responses

	result := ai.AssistantMessage{
		Role:      ai.RoleAssistant,
		Api:       ai.ApiAnthropicMessages,
		Provider:  model.Provider,
		Model:     model.ID,
		Timestamp: time.Now().UnixMilli(),
	}

	// Track content blocks being built
	type blockState struct {
		blockType string
		text      strings.Builder
		toolID    string
		toolName  string
		toolJSON  strings.Builder
		thinking  strings.Builder
		signature string
	}
	blocks := map[int]*blockState{}

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		// Parse event type from the JSON
		var raw struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(data), &raw); err != nil {
			continue
		}

		switch raw.Type {
		case "message_start":
			var evt sseMessageStart
			json.Unmarshal([]byte(data), &evt)
			result.Usage.Input = evt.Message.Usage.InputTokens
			result.Usage.CacheRead = evt.Message.Usage.CacheReadTokens
			result.Usage.CacheWrite = evt.Message.Usage.CacheCreationTokens

			stream.Push(ai.AssistantMessageEvent{
				Type:    ai.EventStart,
				Partial: &result,
			})

		case "content_block_start":
			var evt sseContentBlockStart
			json.Unmarshal([]byte(data), &evt)

			bs := &blockState{blockType: evt.ContentBlock.Type}
			if evt.ContentBlock.Type == "tool_use" {
				bs.toolID = evt.ContentBlock.ID
				bs.toolName = evt.ContentBlock.Name
				stream.Push(ai.AssistantMessageEvent{
					Type:         ai.EventToolCallStart,
					ContentIndex: evt.Index,
					Partial:      &result,
				})
			} else if evt.ContentBlock.Type == "thinking" {
				stream.Push(ai.AssistantMessageEvent{
					Type:         ai.EventThinkingStart,
					ContentIndex: evt.Index,
					Partial:      &result,
				})
			} else {
				stream.Push(ai.AssistantMessageEvent{
					Type:         ai.EventTextStart,
					ContentIndex: evt.Index,
					Partial:      &result,
				})
			}
			blocks[evt.Index] = bs

		case "content_block_delta":
			var evt sseContentBlockDelta
			json.Unmarshal([]byte(data), &evt)

			bs := blocks[evt.Index]
			if bs == nil {
				continue
			}

			switch evt.Delta.Type {
			case "text_delta":
				bs.text.WriteString(evt.Delta.Text)
				stream.Push(ai.AssistantMessageEvent{
					Type:         ai.EventTextDelta,
					ContentIndex: evt.Index,
					Delta:        evt.Delta.Text,
					Partial:      &result,
				})
			case "thinking_delta":
				bs.thinking.WriteString(evt.Delta.Thinking)
				stream.Push(ai.AssistantMessageEvent{
					Type:         ai.EventThinkingDelta,
					ContentIndex: evt.Index,
					Delta:        evt.Delta.Thinking,
					Partial:      &result,
				})
			case "input_json_delta":
				bs.toolJSON.WriteString(evt.Delta.PartialJSON)
				stream.Push(ai.AssistantMessageEvent{
					Type:         ai.EventToolCallDelta,
					ContentIndex: evt.Index,
					Delta:        evt.Delta.PartialJSON,
					Partial:      &result,
				})
			case "signature_delta":
				bs.signature = evt.Delta.Signature
			}

		case "content_block_stop":
			var evt struct {
				Index int `json:"index"`
			}
			json.Unmarshal([]byte(data), &evt)

			bs := blocks[evt.Index]
			if bs == nil {
				continue
			}

			switch bs.blockType {
			case "text":
				result.Content = append(result.Content, ai.ContentBlock{
					Type:          ai.ContentTypeText,
					Text:          bs.text.String(),
					TextSignature: bs.signature,
				})
				stream.Push(ai.AssistantMessageEvent{
					Type:         ai.EventTextEnd,
					ContentIndex: evt.Index,
					Content:      bs.text.String(),
					Partial:      &result,
				})
			case "thinking":
				result.Content = append(result.Content, ai.ContentBlock{
					Type:              ai.ContentTypeThinking,
					Thinking:          bs.thinking.String(),
					ThinkingSignature: bs.signature,
				})
				stream.Push(ai.AssistantMessageEvent{
					Type:         ai.EventThinkingEnd,
					ContentIndex: evt.Index,
					Content:      bs.thinking.String(),
					Partial:      &result,
				})
			case "tool_use":
				var args map[string]any
				jsonStr := bs.toolJSON.String()
				if jsonStr != "" {
					json.Unmarshal([]byte(jsonStr), &args)
				}
				tc := ai.ToolCall{
					Type:      ai.ContentTypeToolCall,
					ID:        bs.toolID,
					Name:      bs.toolName,
					Arguments: args,
				}
				result.Content = append(result.Content, ai.ContentBlock{
					Type:      ai.ContentTypeToolCall,
					ID:        bs.toolID,
					Name:      bs.toolName,
					Arguments: args,
				})
				stream.Push(ai.AssistantMessageEvent{
					Type:         ai.EventToolCallEnd,
					ContentIndex: evt.Index,
					ToolCall:     &tc,
					Partial:      &result,
				})
			}
			delete(blocks, evt.Index)

		case "message_delta":
			var evt sseMessageDelta
			json.Unmarshal([]byte(data), &evt)
			result.Usage.Output = evt.Usage.OutputTokens

			switch evt.Delta.StopReason {
			case "end_turn", "stop":
				result.StopReason = ai.StopReasonStop
			case "tool_use":
				result.StopReason = ai.StopReasonToolUse
			case "max_tokens":
				result.StopReason = ai.StopReasonLength
			default:
				result.StopReason = ai.StopReasonStop
			}

		case "message_stop":
			// Final message
			result.Usage.TotalTokens = result.Usage.Input + result.Usage.Output + result.Usage.CacheRead + result.Usage.CacheWrite
			return result, nil

		case "error":
			var errEvt struct {
				Error struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				} `json:"error"`
			}
			json.Unmarshal([]byte(data), &errEvt)
			return ai.AssistantMessage{}, fmt.Errorf("streaming error: %s: %s", errEvt.Error.Type, errEvt.Error.Message)
		}
	}

	if err := scanner.Err(); err != nil {
		return ai.AssistantMessage{}, fmt.Errorf("stream read error: %w", err)
	}

	// If we get here without message_stop, return what we have
	result.Usage.TotalTokens = result.Usage.Input + result.Usage.Output
	if result.StopReason == "" {
		result.StopReason = ai.StopReasonStop
	}
	return result, nil
}

// convertMessages converts ai.Message to the Anthropic API format.
func convertMessages(messages []ai.Message) ([]apiMessage, error) {
	var result []apiMessage

	for _, msg := range messages {
		switch msg.Role {
		case ai.RoleUser:
			if msg.User == nil {
				continue
			}
			var content []apiContent
			for _, block := range msg.User.Content {
				switch block.Type {
				case ai.ContentTypeText:
					content = append(content, apiContent{Type: "text", Text: block.Text})
				case ai.ContentTypeImage:
					content = append(content, apiContent{
						Type: "image",
						// Anthropic uses source format, simplified here
						Text: "[image]",
					})
				}
			}
			if len(content) > 0 {
				result = append(result, apiMessage{Role: "user", Content: content})
			}

		case ai.RoleAssistant:
			if msg.Assistant == nil {
				continue
			}
			var content []apiContent
			for _, block := range msg.Assistant.Content {
				switch block.Type {
				case ai.ContentTypeText:
					content = append(content, apiContent{Type: "text", Text: block.Text})
				case ai.ContentTypeThinking:
					content = append(content, apiContent{
						Type:      "thinking",
						Thinking:  block.Thinking,
						Signature: block.ThinkingSignature,
					})
				case ai.ContentTypeToolCall:
					content = append(content, apiContent{
						Type:  "tool_use",
						ID:    block.ID,
						Name:  block.Name,
						Input: block.Arguments,
					})
				}
			}
			if len(content) > 0 {
				result = append(result, apiMessage{Role: "assistant", Content: content})
			}

		case ai.RoleToolResult:
			if msg.ToolResult == nil {
				continue
			}
			// Anthropic expects tool results as user messages with tool_result content
			text := ""
			for _, block := range msg.ToolResult.Content {
				if block.Type == ai.ContentTypeText {
					text += block.Text
				}
			}
			result = append(result, apiMessage{
				Role: "user",
				Content: []apiContent{{
					Type:      "tool_result",
					ToolUseID: msg.ToolResult.ToolCallID,
					Content:   text,
					IsError:   msg.ToolResult.IsError,
				}},
			})
		}
	}

	return result, nil
}
