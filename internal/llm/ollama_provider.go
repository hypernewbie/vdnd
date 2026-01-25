package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ollama/ollama/api"
)

type OllamaFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type OllamaTool struct {
	Type     string         `json:"type"`
	Function OllamaFunction `json:"function"`
}

type OllamaToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

type OllamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []OllamaToolCall `json:"tool_calls,omitempty"`
}

type OllamaResponseWithTools struct {
	Model   string        `json:"model"`
	Message OllamaMessage `json:"message"`
	Done    bool          `json:"done"`
}

type OllamaProvider struct {
	client *api.Client
	model  string
}

func NewOllamaProvider(model string) (*OllamaProvider, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("ollama not found. start with: ollama serve. error: %w", err)
	}
	if model == "" {
		model = "deepseek-r1:7b" // Default for this task
	}
	return &OllamaProvider{
		client: client,
		model:  model,
	}, nil
}

func (p *OllamaProvider) Name() string      { return "ollama" }
func (p *OllamaProvider) ModelName() string { return p.model }

func (p *OllamaProvider) Generate(ctx context.Context, messages []Message) (string, error) {
	var ollamaMessages []api.Message
	for _, msg := range messages {
		role := msg.Role
		if role == "model" {
			role = "assistant"
		}
		ollamaMessages = append(ollamaMessages, api.Message{
			Role:    role,
			Content: msg.Content,
		})
	}

	req := &api.ChatRequest{
		Model:    p.model,
		Messages: ollamaMessages,
		Stream:   new(bool), // false
	}

	var responseText string
	fn := func(resp api.ChatResponse) error {
		responseText += resp.Message.Content
		return nil
	}

	if err := p.client.Chat(ctx, req, fn); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return "", fmt.Errorf("model %s not found. run: ollama pull %s", p.model, p.model)
		}
		return "", fmt.Errorf("ollama generation failed: %w", err)
	}

	return responseText, nil
}

func (p *OllamaProvider) GenerateWithTools(ctx context.Context, messages []Message, tools []Tool) (GenerationResponse, error) {
	ollamaMessages := p.convertToOllamaMessages(messages)
	ollamaTools := p.convertToOllamaTools(tools)

	reqBody := map[string]interface{}{
		"model":    p.model,
		"messages": ollamaMessages,
		"tools":    ollamaTools,
		"stream":   false,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return GenerationResponse{}, err
	}

	// Standard Ollama endpoint
	url := "http://127.0.0.1:11434/api/chat"

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return GenerationResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	httpResp, err := client.Do(req)
	if err != nil {
		return GenerationResponse{}, fmt.Errorf("ollama request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return GenerationResponse{}, fmt.Errorf("ollama error (status %d): %s", httpResp.StatusCode, string(body))
	}

	var resp OllamaResponseWithTools
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return GenerationResponse{}, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	if len(resp.Message.ToolCalls) > 0 {
		var toolCalls []ToolCall
		for _, c := range resp.Message.ToolCalls {
			toolCalls = append(toolCalls, ToolCall{
				ID:        c.ID,
				Name:      c.Function.Name,
				Arguments: string(c.Function.Arguments),
			})
		}
		return GenerationResponse{
			ToolCalls:    toolCalls,
			FinishReason: "tool_calls",
		}, nil
	}

	return GenerationResponse{
		Content:      resp.Message.Content,
		FinishReason: "stop",
	}, nil
}

func (p *OllamaProvider) SupportsToolCalling() bool {
	// deepseek-r1 models do not support native tool-calling via Ollama API reliably.
	if strings.Contains(strings.ToLower(p.model), "deepseek-r1") {
		return false
	}
	return true
}

func (p *OllamaProvider) convertToOllamaMessages(messages []Message) []api.Message {
	var result []api.Message
	for _, msg := range messages {
		role := msg.Role
		if role == "model" {
			role = "assistant"
		}
		// If last message was tool result, we use VAI fallback: user role with prefix
		if role == "tool" {
			result = append(result, api.Message{
				Role:    "user",
				Content: fmt.Sprintf("Tool %s returned: %s", msg.Name, msg.Content),
			})
			continue
		}
		result = append(result, api.Message{
			Role:    role,
			Content: msg.Content,
		})
	}
	return result
}

func (p *OllamaProvider) convertToOllamaTools(tools []Tool) []OllamaTool {
	var result []OllamaTool
	for _, t := range tools {
		result = append(result, OllamaTool{
			Type: "function",
			Function: OllamaFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}
	return result
}
