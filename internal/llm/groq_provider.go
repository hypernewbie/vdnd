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

// OpenAI-compatible structs for Groq
type GroqMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	Reasoning  string         `json:"reasoning,omitempty"`
	ToolCalls  []GroqToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	Name       string         `json:"name,omitempty"`
}

type GroqToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type GroqTool struct {
	Type     string       `json:"type"`
	Function GroqFunction `json:"function"`
}

type GroqFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type GroqChatRequest struct {
	Model    string        `json:"model"`
	Messages []GroqMessage `json:"messages"`
	Tools    []GroqTool    `json:"tools,omitempty"`
	Stream   bool          `json:"stream"`
}

type GroqChatResponse struct {
	Choices []struct {
		Message GroqMessage `json:"message"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

type GroqProvider struct {
	apiKey string
	model  string
}

func NewGroqProvider(apiKey string, model string) (*GroqProvider, error) {
	if model == "" {
		model = "qwen/qwen3-32b" // Default as requested
	}
	return &GroqProvider{
		apiKey: apiKey,
		model:  model,
	}, nil
}

func (p *GroqProvider) Generate(ctx context.Context, messages []Message) (string, error) {
	resp, err := p.GenerateWithTools(ctx, messages, nil)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (p *GroqProvider) GenerateWithTools(ctx context.Context, messages []Message, tools []Tool) (GenerationResponse, error) {
	groqMessages := p.convertToGroqMessages(messages)
	groqTools := p.convertToGroqTools(tools)

	reqBody := GroqChatRequest{
		Model:    p.model,
		Messages: groqMessages,
		Tools:    groqTools,
		Stream:   false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return GenerationResponse{}, err
	}

	url := "https://api.groq.com/openai/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return GenerationResponse{}, err
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return GenerationResponse{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var errResp GroqChatResponse
		json.Unmarshal(body, &errResp)
		if errResp.Error.Message != "" {
			return GenerationResponse{}, fmt.Errorf("groq api error: %s", errResp.Error.Message)
		}
		return GenerationResponse{}, fmt.Errorf("groq api error (status %d): %s", resp.StatusCode, string(body))
	}

	var groqResp GroqChatResponse
	if err := json.Unmarshal(body, &groqResp); err != nil {
		return GenerationResponse{}, err
	}

	if len(groqResp.Choices) == 0 {
		return GenerationResponse{}, fmt.Errorf("empty response from groq")
	}

	msg := groqResp.Choices[0].Message

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
			Thinking:     msg.Reasoning,
			FinishReason: "tool_calls",
		}, nil
	}

	return GenerationResponse{
		Content:      msg.Content,
		Thinking:     msg.Reasoning,
		FinishReason: "stop",
	}, nil
}

func (p *GroqProvider) Name() string { return "groq" }

func (p *GroqProvider) ModelName() string { return p.model }

func (p *GroqProvider) SupportsToolCalling() bool { return true }

func (p *GroqProvider) convertToGroqMessages(messages []Message) []GroqMessage {
	var groqMsgs []GroqMessage
	for _, m := range messages {
		role := m.Role
		if role == "model" {
			role = "assistant"
		}

		gm := GroqMessage{
			Role:    role,
			Content: m.Content,
			Name:    m.Name,
		}

		if m.ToolCallID != "" {
			gm.ToolCallID = m.ToolCallID
		}

		if len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				gtc := GroqToolCall{
					ID:   tc.ID,
					Type: "function",
				}
				gtc.Function.Name = tc.Name
				gtc.Function.Arguments = tc.Arguments
				gm.ToolCalls = append(gm.ToolCalls, gtc)
			}
		}

		groqMsgs = append(groqMsgs, gm)
	}
	return groqMsgs
}

func (p *GroqProvider) convertToGroqTools(tools []Tool) []GroqTool {
	if len(tools) == 0 {
		return nil
	}
	var groqTools []GroqTool
	for _, t := range tools {
		gt := GroqTool{
			Type: "function",
			Function: GroqFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		}
		if gt.Function.Parameters == nil {
			gt.Function.Parameters = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}
		groqTools = append(groqTools, gt)
	}
	return groqTools
}

func (p *GroqProvider) Close() error { return nil }
