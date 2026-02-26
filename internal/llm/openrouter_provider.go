package llm

import (
	"context"
	"uaa/vdnd/internal/llm/llmtypes"
)

type OpenRouterProvider struct {
	*OpenAIProvider
}

func NewOpenRouterProvider(apiKey string, model string) (*OpenRouterProvider, error) {
	if model == "" {
		model = "openrouter/auto"
	}
	config := OpenAIProviderConfig{
		Name:          "openrouter",
		BaseURL:       "https://openrouter.ai/api/v1/chat/completions",
		APIKey:        apiKey,
		Model:         model,
		SupportsTools: true,
		ExtraHeaders: map[string]string{
			"HTTP-Referer": "https://github.com/google/gemini-cli", // Required for some OpenRouter rankings/models
			"X-Title":      "VDND Virtual Dungeon Master",
		},
	}
	return &OpenRouterProvider{
		OpenAIProvider: NewOpenAIProvider(config),
	}, nil
}

func (p *OpenRouterProvider) Generate(ctx context.Context, messages []llmtypes.Message) (string, error) {
	return p.OpenAIProvider.Generate(ctx, messages)
}

func (p *OpenRouterProvider) GenerateWithTools(ctx context.Context, messages []llmtypes.Message, tools []llmtypes.Tool) (llmtypes.GenerationResponse, error) {
	return p.OpenAIProvider.GenerateWithTools(ctx, messages, tools)
}

func (p *OpenRouterProvider) GenerateStream(ctx context.Context, messages []llmtypes.Message, tools []llmtypes.Tool, callback func(chunk string) error) (llmtypes.GenerationResponse, error) {
	return p.OpenAIProvider.GenerateStream(ctx, messages, tools, callback)
}
