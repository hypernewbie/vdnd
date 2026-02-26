package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"uaa/vdnd/internal/llm/llmtypes"
)

// Gemini API structures
type GeminiFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations"`
}

type GeminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type GeminiFunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type GeminiPart struct {
	Text                  string                  `json:"text,omitempty"`
	Thought               bool                    `json:"thought,omitempty"`
	ThoughtSignature      string                  `json:"thought_signature,omitempty"`
	ThoughtSignatureCamel string                  `json:"thoughtSignature,omitempty"`
	FunctionCall          *GeminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse      *GeminiFunctionResponse `json:"functionResponse,omitempty"`
}

type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiRequest struct {
	Contents         []GeminiContent   `json:"contents"`
	Tools            []GeminiTool      `json:"tools,omitempty"`
	GenerationConfig *GeminiGenConfig `json:"generationConfig,omitempty"`
}

type GeminiGenConfig struct {
	ThinkingConfig *GeminiThinkingConfig `json:"thinking_config,omitempty"`
}

type GeminiThinkingConfig struct {
	IncludeThoughts bool `json:"include_thoughts"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Role  string       `json:"role"`
			Parts []GeminiPart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
}

type GeminiProvider struct {
	apiKey         string
	model          string
	enableThinking bool
}

func NewGeminiProvider(ctx context.Context, apiKey string, modelName string, enableThinking bool) (*GeminiProvider, error) {
	if modelName == "" {
		modelName = "gemini-2.0-flash-exp"
	}
	return &GeminiProvider{
		apiKey:         apiKey,
		model:          modelName,
		enableThinking: enableThinking,
	}, nil
}

func (p *GeminiProvider) Name() string      { return "gemini" }
func (p *GeminiProvider) ModelName() string { return p.model }

func (p *GeminiProvider) Generate(ctx context.Context, messages []llmtypes.Message) (string, error) {
	contents := p.convertMessagesToGeminiContents(messages)
	request := GeminiRequest{Contents: contents}
	if p.enableThinking {
		request.GenerationConfig = &GeminiGenConfig{
			ThinkingConfig: &GeminiThinkingConfig{IncludeThoughts: true},
		}
	}

	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	respBody, err := p.callAPI(ctx, jsonBytes)
	if err != nil {
		return "", err
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}

	var content string
	var thinking string
	for _, part := range geminiResp.Candidates[0].Content.Parts {
		if part.Thought {
			thinking += part.Text
		} else if part.Text != "" {
			content += part.Text
		}
	}

	if thinking != "" {
		return fmt.Sprintf("<thought>\n%s\n</thought>\n%s", thinking, content), nil
	}
	return content, nil
}

func (p *GeminiProvider) GenerateWithTools(ctx context.Context, messages []llmtypes.Message, tools []llmtypes.Tool) (llmtypes.GenerationResponse, error) {
	contents := p.convertMessagesToGeminiContents(messages)
	geminiTools := p.convertToGeminiTools(tools)

	request := GeminiRequest{
		Contents: contents,
		Tools:    geminiTools,
	}
	if p.enableThinking {
		request.GenerationConfig = &GeminiGenConfig{
			ThinkingConfig: &GeminiThinkingConfig{IncludeThoughts: true},
		}
	}

	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return llmtypes.GenerationResponse{}, err
	}

	respBody, err := p.callAPI(ctx, jsonBytes)
	if err != nil {
		return llmtypes.GenerationResponse{}, err
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return llmtypes.GenerationResponse{}, err
	}

	if len(geminiResp.Candidates) == 0 {
		return llmtypes.GenerationResponse{}, fmt.Errorf("empty response from Gemini")
	}

	candidate := geminiResp.Candidates[0]
	result := llmtypes.GenerationResponse{
		FinishReason: "stop",
	}

	for _, part := range candidate.Content.Parts {
		if part.Thought {
			result.Thinking += part.Text
		} else if part.Text != "" {
			result.Content += part.Text
		}

		if part.FunctionCall != nil {
			args, _ := json.Marshal(part.FunctionCall.Args)
			sig := part.ThoughtSignature
			if sig == "" {
				sig = part.ThoughtSignatureCamel
			}

			result.ToolCalls = append(result.ToolCalls, llmtypes.ToolCall{
				ID:               "", // Gemini doesn't use IDs in the same way as OpenAI
				Name:             part.FunctionCall.Name,
				Arguments:        string(args),
				ThoughtSignature: sig,
			})
			result.FinishReason = "tool_calls"
		}
	}

	// Internal logic uses tags for thinking if it's mixed in content,
	// but here we have explicit thinking parts.
	if result.Thinking == "" {
		result.Thinking = ExtractThinking(result.Content)
		result.Content = StripThinking(result.Content)
	}

	return result, nil
}

func (p *GeminiProvider) GenerateStream(ctx context.Context, messages []llmtypes.Message, tools []llmtypes.Tool, callback func(chunk string) error) (llmtypes.GenerationResponse, error) {
	resp, err := p.GenerateWithTools(ctx, messages, tools)
	if err != nil {
		return llmtypes.GenerationResponse{}, err
	}
	if resp.FinishReason == "stop" && resp.Content != "" && callback != nil {
		if err := callback(resp.Content); err != nil {
			return llmtypes.GenerationResponse{}, err
		}
	}
	return resp, nil
}

func (p *GeminiProvider) SupportsToolCalling() bool { return true }

func (p *GeminiProvider) callAPI(ctx context.Context, jsonData []byte) ([]byte, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", p.model, p.apiKey)

	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (p *GeminiProvider) convertMessagesToGeminiContents(messages []llmtypes.Message) []GeminiContent {
	var contents []GeminiContent
	for _, m := range messages {
		role := m.Role
		if role == "system" {
			role = "user"
		}
		if role == "assistant" {
			role = "model"
		}

		var parts []GeminiPart
		if m.Thinking != "" {
			parts = append(parts, GeminiPart{
				Text:    m.Thinking,
				Thought: true,
			})
		}

		if m.Content != "" {
			parts = append(parts, GeminiPart{Text: m.Content})
		}

		if len(m.ToolCalls) > 0 {
			role = "model"
			for _, tc := range m.ToolCalls {
				var args map[string]interface{}
				json.Unmarshal([]byte(tc.Arguments), &args)
				parts = append(parts, GeminiPart{
					ThoughtSignature:      tc.ThoughtSignature,
					ThoughtSignatureCamel: tc.ThoughtSignature,
					FunctionCall: &GeminiFunctionCall{
						Name: tc.Name,
						Args: args,
					},
				})
			}
		}

		if m.Role == "tool" {
			role = "user"
			parts = []GeminiPart{
				{
					FunctionResponse: &GeminiFunctionResponse{
						Name: m.Name,
						Response: map[string]interface{}{
							"result": m.Content,
						},
					},
				},
			}
		}

		contents = append(contents, GeminiContent{
			Role:  role,
			Parts: parts,
		})
	}
	return contents
}

func (p *GeminiProvider) convertToGeminiTools(tools []llmtypes.Tool) []GeminiTool {
	if len(tools) == 0 {
		return nil
	}
	var declarations []GeminiFunctionDeclaration
	for _, t := range tools {
		declarations = append(declarations, GeminiFunctionDeclaration{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
		})
	}
	return []GeminiTool{{FunctionDeclarations: declarations}}
}

func (p *GeminiProvider) Close() error { return nil }
