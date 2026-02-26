package llm

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
	"uaa/vdnd/internal/llm/llmtypes"
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
	Index    int    `json:"index,omitempty"`
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

type OpenAIStreamResponse struct {
	Choices []struct {
		Delta OpenAIInternalMessage `json:"delta"`
		// Some providers still emit full messages in streamed chunks.
		Message      OpenAIInternalMessage `json:"message"`
		FinishReason string                `json:"finish_reason"`
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
	ExtraHeaders  map[string]string
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

func (p *OpenAIProvider) Generate(ctx context.Context, messages []llmtypes.Message) (string, error) {
	resp, err := p.GenerateWithTools(ctx, messages, nil)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (p *OpenAIProvider) GenerateWithTools(ctx context.Context, messages []llmtypes.Message, tools []llmtypes.Tool) (llmtypes.GenerationResponse, error) {
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
		return llmtypes.GenerationResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return llmtypes.GenerationResponse{}, err
	}

	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range p.config.ExtraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return llmtypes.GenerationResponse{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var errResp OpenAIChatResponse
		json.Unmarshal(body, &errResp)
		if errResp.Error.Message != "" {
			return llmtypes.GenerationResponse{}, fmt.Errorf("%s api error: %s", p.config.Name, errResp.Error.Message)
		}
		return llmtypes.GenerationResponse{}, fmt.Errorf("%s api error (status %d): %s", p.config.Name, resp.StatusCode, string(body))
	}

	var oaResp OpenAIChatResponse
	if err := json.Unmarshal(body, &oaResp); err != nil {
		return llmtypes.GenerationResponse{}, err
	}

	if len(oaResp.Choices) == 0 {
		return llmtypes.GenerationResponse{}, fmt.Errorf("empty response from %s", p.config.Name)
	}

	msg := oaResp.Choices[0].Message

	if len(msg.ToolCalls) > 0 {
		toolCalls := []llmtypes.ToolCall{}
		for _, tc := range msg.ToolCalls {
			toolCalls = append(toolCalls, llmtypes.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
		return llmtypes.GenerationResponse{
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

	return llmtypes.GenerationResponse{
		Content:      content,
		Thinking:     thinking,
		FinishReason: "stop",
	}, nil
}

func (p *OpenAIProvider) GenerateStream(ctx context.Context, messages []llmtypes.Message, tools []llmtypes.Tool, callback func(chunk string) error) (llmtypes.GenerationResponse, error) {
	oaMessages := p.convertToOpenAIMessages(messages)
	oaTools := p.convertToOpenAITools(tools)

	reqBody := OpenAIChatRequest{
		Model:    p.config.Model,
		Messages: oaMessages,
		Tools:    oaTools,
		Stream:   true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return llmtypes.GenerationResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return llmtypes.GenerationResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range p.config.ExtraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return llmtypes.GenerationResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errResp OpenAIChatResponse
		_ = json.Unmarshal(body, &errResp)
		if errResp.Error.Message != "" {
			return llmtypes.GenerationResponse{}, fmt.Errorf("%s api error: %s", p.config.Name, errResp.Error.Message)
		}
		return llmtypes.GenerationResponse{}, fmt.Errorf("%s api error (status %d): %s", p.config.Name, resp.StatusCode, string(body))
	}

	var contentBuilder strings.Builder
	var thinkingBuilder strings.Builder
	toolCallOrder := []int{}
	toolCalls := map[int]*OpenAIToolCall{}
	finishReason := ""

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		payload := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		if payload == "[DONE]" {
			break
		}

		var chunk OpenAIStreamResponse
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return llmtypes.GenerationResponse{}, fmt.Errorf("failed to decode stream chunk: %w", err)
		}
		if chunk.Error.Message != "" {
			return llmtypes.GenerationResponse{}, fmt.Errorf("%s api error: %s", p.config.Name, chunk.Error.Message)
		}

		for _, choice := range chunk.Choices {
			msg := choice.Delta
			if msg.Content == "" && choice.Message.Content != "" {
				msg = choice.Message
			}

			if msg.Content != "" {
				contentBuilder.WriteString(msg.Content)
				if callback != nil {
					if err := callback(msg.Content); err != nil {
						return llmtypes.GenerationResponse{}, err
					}
				}
			}
			if msg.ReasoningContent != "" {
				thinkingBuilder.WriteString(msg.ReasoningContent)
			}
			if msg.Reasoning != "" {
				thinkingBuilder.WriteString(msg.Reasoning)
			}

			for _, tc := range msg.ToolCalls {
				call, exists := toolCalls[tc.Index]
				if !exists {
					tcCopy := OpenAIToolCall{Index: tc.Index}
					call = &tcCopy
					toolCalls[tc.Index] = call
					toolCallOrder = append(toolCallOrder, tc.Index)
				}
				if tc.ID != "" {
					call.ID = tc.ID
				}
				if tc.Type != "" {
					call.Type = tc.Type
				}
				if tc.Function.Name != "" {
					call.Function.Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					call.Function.Arguments += tc.Function.Arguments
				}
			}

			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return llmtypes.GenerationResponse{}, fmt.Errorf("stream read error: %w", err)
	}

	if len(toolCalls) > 0 || finishReason == "tool_calls" {
		resp := llmtypes.GenerationResponse{
			FinishReason: "tool_calls",
			Thinking:     thinkingBuilder.String(),
		}
		for _, idx := range toolCallOrder {
			call := toolCalls[idx]
			resp.ToolCalls = append(resp.ToolCalls, llmtypes.ToolCall{
				ID:        call.ID,
				Name:      call.Function.Name,
				Arguments: call.Function.Arguments,
			})
		}
		return resp, nil
	}

	return llmtypes.GenerationResponse{
		Content:      contentBuilder.String(),
		Thinking:     thinkingBuilder.String(),
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

func (p *OpenAIProvider) convertToOpenAIMessages(messages []llmtypes.Message) []OpenAIInternalMessage {
	var oaMsgs []OpenAIInternalMessage
	for _, m := range messages {
		role := m.Role
		if role == "model" {
			role = "assistant"
		}

		// Handle Assistant messages with both thinking and tool calls by splitting them.
		// Many providers (Minimax, DeepSeek) forbid combined reasoning + tools in history.
		if role == "assistant" && m.Thinking != "" && len(m.ToolCalls) > 0 {
			// First message: The reasoning/thinking turn
			oaMsgs = append(oaMsgs, OpenAIInternalMessage{
				Role:      role,
				Reasoning: m.Thinking, // OpenRouter normalized key
			})

			// Second message: The action/tool_calls turn
			om := OpenAIInternalMessage{
				Role: role,
			}
			for _, tc := range m.ToolCalls {
				otc := OpenAIToolCall{
					ID:   tc.ID,
					Type: "function",
				}
				if otc.ID == "" {
					otc.ID = fmt.Sprintf("call_gen_%d", time.Now().UnixNano())
				}
				otc.Function.Name = tc.Name
				otc.Function.Arguments = tc.Arguments
				om.ToolCalls = append(om.ToolCalls, otc)
			}
			oaMsgs = append(oaMsgs, om)
			continue
		}

		om := OpenAIInternalMessage{
			Role:             role,
			Content:          m.Content,
			Name:             m.Name,
			Reasoning:        m.Thinking, // Use 'reasoning' for history turns
			ReasoningContent: m.Thinking, // Keep 'reasoning_content' for dual-support
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
				if otc.ID == "" {
					otc.ID = fmt.Sprintf("call_gen_%d", time.Now().UnixNano())
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

func (p *OpenAIProvider) convertToOpenAITools(tools []llmtypes.Tool) []OpenAITool {
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
