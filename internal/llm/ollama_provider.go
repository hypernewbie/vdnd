package llm

import (
	"context"
	"strings"
)

type OllamaProvider struct {
	*OpenAIProvider
}

func NewOllamaProvider(model string) (*OllamaProvider, error) {
	if model == "" {
		model = "deepseek-r1:7b"
	}
	config := OpenAIProviderConfig{
		Name:          "ollama",
		BaseURL:       "http://127.0.0.1:11434/v1/chat/completions",
		APIKey:        "ollama", // Required but ignored by Ollama
		Model:         model,
		SupportsTools: !strings.Contains(strings.ToLower(model), "deepseek-r1"),
	}
	return &OllamaProvider{
		OpenAIProvider: NewOpenAIProvider(config),
	}, nil
}

func (p *OllamaProvider) Generate(ctx context.Context, messages []Message) (string, error) {
	return p.OpenAIProvider.Generate(ctx, messages)
}

func (p *OllamaProvider) GenerateWithTools(ctx context.Context, messages []Message, tools []Tool) (GenerationResponse, error) {
	return p.OpenAIProvider.GenerateWithTools(ctx, messages, tools)
}
