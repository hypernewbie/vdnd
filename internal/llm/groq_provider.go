package llm

import (
	"context"
)

type GroqProvider struct {
	*OpenAIProvider
}

func NewGroqProvider(apiKey string, model string) (*GroqProvider, error) {
	if model == "" {
		model = "qwen/qwen3-32b"
	}
	config := OpenAIProviderConfig{
		Name:          "groq",
		BaseURL:       "https://api.groq.com/openai/v1/chat/completions",
		APIKey:        apiKey,
		Model:         model,
		SupportsTools: true,
	}
	return &GroqProvider{
		OpenAIProvider: NewOpenAIProvider(config),
	}, nil
}

func (p *GroqProvider) Generate(ctx context.Context, messages []Message) (string, error) {
	return p.OpenAIProvider.Generate(ctx, messages)
}

func (p *GroqProvider) GenerateWithTools(ctx context.Context, messages []Message, tools []Tool) (GenerationResponse, error) {
	return p.OpenAIProvider.GenerateWithTools(ctx, messages, tools)
}

func (p *GroqProvider) Close() error { return nil }
