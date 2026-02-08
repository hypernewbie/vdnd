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

// OpenAIInternalMessage represents the message format used by OpenAI-compatible APIs.
type OpenAIInternalMessage struct {
	Role             string           `json:"role"`
	Content          string           `json:"content,omitempty"`
	Reasoning        string           `json:"reasoning,omitempty"`         // Used by Groq
	ReasoningContent string           `json:"reasoning_content,omitempty"` // Used by DeepSeek
	ToolCalls        []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string           `json:"tool_call_id,omitempty"`
	Name             string           `json:"name,omitempty"`
}

type OpenAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type OpenAITool struct {
	Type     string         `json:"type"`
	Function OpenAIFunction `json:"function"`
}

type OpenAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type OpenAIChatRequest struct {
	Model    string                  `json:"model"`
	Messages []OpenAIInternalMessage `json:"messages"`
	Tools    []OpenAITool            `json:"tools,omitempty"`
	Stream   bool                    `json:"stream"`
}

type OpenAIChatResponse struct {
	Choices []struct {
		Message OpenAIInternalMessage `json:"message"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// OpenAIProviderConfig holds configuration for an OpenAI-compatible provider.
type OpenAIProviderConfig struct {
	Name          string
	BaseURL       string
	APIKey        string
	Model         string
	SupportsTools bool
}

// OpenAIProvider is a generic provider for OpenAI-compatible APIs.
type OpenAIProvider struct {
	config OpenAIProviderConfig
	client *http.Client
}

func NewOpenAIProvider(config OpenAIProviderConfig) *OpenAIProvider {
	return &OpenAIProvider{
		config: config,
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

func (p *OpenAIProvider) Name() string      { return p.config.Name }
func (p *OpenAIProvider) ModelName() string { return p.config.Model }

func (p *OpenAIProvider) Generate(ctx context.Context, messages []Message) (string, error) {
	resp, err := p.GenerateWithTools(ctx, messages, nil)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (p *OpenAIProvider) GenerateWithTools(ctx context.Context, messages []Message, tools []Tool) (GenerationResponse, error) {
	oaMessages := p.convertToOpenAIMessages(messages)
	oaTools := p.convertToOpenAITools(tools)

	reqBody := OpenAIChatRequest{
		Model:    p.config.Model,
		Messages: oaMessages,
		Tools:    oaTools,
		Stream:   false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return GenerationResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return GenerationResponse{}, err
	}

	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return GenerationResponse{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var errResp OpenAIChatResponse
		json.Unmarshal(body, &errResp)
		if errResp.Error.Message != "" {
			return GenerationResponse{}, fmt.Errorf("%s api error: %s", p.config.Name, errResp.Error.Message)
		}
		return GenerationResponse{}, fmt.Errorf("%s api error (status %d): %s", p.config.Name, resp.StatusCode, string(body))
	}

	var oaResp OpenAIChatResponse
	if err := json.Unmarshal(body, &oaResp); err != nil {
		return GenerationResponse{}, err
	}

	if len(oaResp.Choices) == 0 {
		return GenerationResponse{}, fmt.Errorf("empty response from %s", p.config.Name)
	}

	msg := oaResp.Choices[0].Message

	if len(msg.ToolCalls) > 0 {
		toolCalls := []ToolCall{}
		for _, tc := range msg.ToolCalls {
			toolCalls = append(toolCalls, ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
		return GenerationResponse{
			ToolCalls:    toolCalls,
			Thinking:     p.extractReasoning(msg),
			FinishReason: "tool_calls",
		}, nil
	}

	thinking := p.extractReasoning(msg)
	if thinking == "" {
		thinking = ExtractThinking(msg.Content)
	}
	content := StripThinking(msg.Content)

	return GenerationResponse{
		Content:      content,
		Thinking:     thinking,
		FinishReason: "stop",
	}, nil
}

func (p *OpenAIProvider) SupportsToolCalling() bool {
	return p.config.SupportsTools
}

func (p *OpenAIProvider) extractReasoning(msg OpenAIInternalMessage) string {
	if msg.ReasoningContent != "" {
		return msg.ReasoningContent
	}
	return msg.Reasoning
}

func (p *OpenAIProvider) convertToOpenAIMessages(messages []Message) []OpenAIInternalMessage {
	var oaMsgs []OpenAIInternalMessage
	for _, m := range messages {
		role := m.Role
		if role == "model" {
			role = "assistant"
		}

		om := OpenAIInternalMessage{
			Role:             role,
			Content:          m.Content,
			Name:             m.Name,
			ReasoningContent: m.Thinking, // For DeepSeek
			Reasoning:        m.Thinking, // For Groq
		}

		if m.ToolCallID != "" {
			om.ToolCallID = m.ToolCallID
		}

		if len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				otc := OpenAIToolCall{
					ID:   tc.ID,
					Type: "function",
				}
				otc.Function.Name = tc.Name
				otc.Function.Arguments = tc.Arguments
				om.ToolCalls = append(om.ToolCalls, otc)
			}
		}

		oaMsgs = append(oaMsgs, om)
	}
	return oaMsgs
}

func (p *OpenAIProvider) convertToOpenAITools(tools []Tool) []OpenAITool {
	if len(tools) == 0 {
		return nil
	}
	var oaTools []OpenAITool
	for _, t := range tools {
		ot := OpenAITool{
			Type: "function",
			Function: OpenAIFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		}
		if ot.Function.Parameters == nil {
			ot.Function.Parameters = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}
		oaTools = append(oaTools, ot)
	}
	return oaTools
}
