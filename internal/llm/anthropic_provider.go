package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AnthropicCacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

// Anthropic API structures
type AnthropicContentBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	ID    string `json:"id,omitempty"`    // for tool_use
	Name  string `json:"name,omitempty"`  // for tool_use
	Input any    `json:"input,omitempty"` // for tool_use

	ToolUseID    string                 `json:"tool_use_id,omitempty"` // for tool_result
	Content      string                 `json:"content,omitempty"`     // for tool_result
	IsError      bool                   `json:"is_error,omitempty"`    // for tool_result
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

type AnthropicMessage struct {
	Role    string                  `json:"role"`
	Content []AnthropicContentBlock `json:"content"`
}

type AnthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type AnthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []AnthropicMessage `json:"messages"`
	System    any                `json:"system,omitempty"` // string or []AnthropicContentBlock
	MaxTokens int                `json:"max_tokens"`
	Tools     []AnthropicTool    `json:"tools,omitempty"`
	Stream    bool               `json:"stream"`
}

type AnthropicResponse struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Role       string                  `json:"role"`
	Content    []AnthropicContentBlock `json:"content"`
	Model      string                  `json:"model"`
	StopReason string                  `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

type AnthropicProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewAnthropicProvider(apiKey string, model string) (*AnthropicProvider, error) {
	if model == "" {
		model = "claude-3-5-sonnet-20240620"
	}
	return &AnthropicProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.anthropic.com/v1/messages",
		client:  &http.Client{Timeout: 90 * time.Second},
	}, nil
}

func (p *AnthropicProvider) Name() string      { return "anthropic" }
func (p *AnthropicProvider) ModelName() string { return p.model }

func (p *AnthropicProvider) Generate(ctx context.Context, messages []Message) (string, error) {
	resp, err := p.GenerateWithTools(ctx, messages, nil)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (p *AnthropicProvider) GenerateWithTools(ctx context.Context, messages []Message, tools []Tool) (GenerationResponse, error) {
	anthropicMessages, systemPrompt := p.convertMessages(messages)
	anthropicTools := p.convertTools(tools)

	// If system prompt is set, convert it to a block and enable caching
	var system any
	if systemPrompt != "" {
		system = []AnthropicContentBlock{
			{
				Type:         "text",
				Text:         systemPrompt,
				CacheControl: &AnthropicCacheControl{Type: "ephemeral"},
			},
		}
	}

	reqBody := AnthropicRequest{
		Model:     p.model,
		Messages:  anthropicMessages,
		System:    system,
		MaxTokens: 1600,
		Tools:     anthropicTools,
		Stream:    false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return GenerationResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return GenerationResponse{}, err
	}

	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "prompt-caching-2024-07-31")
	req.Header.Set("content-type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return GenerationResponse{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var errResp AnthropicResponse
		json.Unmarshal(body, &errResp)
		if errResp.Error.Message != "" {
			return GenerationResponse{}, fmt.Errorf("anthropic api error: %s", errResp.Error.Message)
		}
		return GenerationResponse{}, fmt.Errorf("anthropic api error (status %d): %s", resp.StatusCode, string(body))
	}

	var anthropicResp AnthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return GenerationResponse{}, err
	}

	result := GenerationResponse{
		FinishReason: "stop",
	}

	for _, block := range anthropicResp.Content {
		if block.Type == "text" {
			result.Content += block.Text
		} else if block.Type == "tool_use" {
			args, _ := json.Marshal(block.Input)
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: string(args),
			})
			result.FinishReason = "tool_calls"
		}
	}

	// Basic thinking extraction from content if tags are used
	result.Thinking = ExtractThinking(result.Content)
	result.Content = StripThinking(result.Content)

	return result, nil
}

func (p *AnthropicProvider) SupportsToolCalling() bool {
	return true
}

func (p *AnthropicProvider) convertMessages(messages []Message) ([]AnthropicMessage, string) {
	var systemPrompt string
	var anthropicMsgs []AnthropicMessage

	for _, m := range messages {
		if m.Role == "system" {
			systemPrompt = m.Content
			continue
		}

		role := m.Role
		if role == "model" {
			role = "assistant"
		}

		var content []AnthropicContentBlock
		if m.Content != "" && m.Role != "tool" {
			content = append(content, AnthropicContentBlock{
				Type: "text",
				Text: m.Content,
			})
		}

		if len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				var input any
				json.Unmarshal([]byte(tc.Arguments), &input)
				content = append(content, AnthropicContentBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: input,
				})
			}
		}

		if m.Role == "tool" {
			role = "user" // Tool results must be sent as a user message in Anthropic
			content = append(content, AnthropicContentBlock{
				Type:      "tool_result",
				ToolUseID: m.ToolCallID,
				Content:   m.Content,
			})
		}

		// Anthropic requires alternating user/assistant messages.
		// If we have consecutive messages of the same role, we might need to merge them.
		if len(anthropicMsgs) > 0 && anthropicMsgs[len(anthropicMsgs)-1].Role == role {
			anthropicMsgs[len(anthropicMsgs)-1].Content = append(anthropicMsgs[len(anthropicMsgs)-1].Content, content...)
		} else {
			anthropicMsgs = append(anthropicMsgs, AnthropicMessage{
				Role:    role,
				Content: content,
			})
		}
	}

	return anthropicMsgs, systemPrompt
}

func (p *AnthropicProvider) convertTools(tools []Tool) []AnthropicTool {
	if len(tools) == 0 {
		return nil
	}
	var anthropicTools []AnthropicTool
	for _, t := range tools {
		anthropicTools = append(anthropicTools, AnthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}
	return anthropicTools
}
