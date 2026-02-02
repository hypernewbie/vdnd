package llm

import (
	"context"
	"strings"
)

type DeepSeekProvider struct {
	*OpenAIProvider
}

func NewDeepSeekProvider(apiKey string, model string) (*DeepSeekProvider, error) {
	if model == "" {
		model = "deepseek-chat"
	}
	config := OpenAIProviderConfig{
		Name:          "deepseek",
		BaseURL:       "https://api.deepseek.com/chat/completions",
		APIKey:        apiKey,
		Model:         model,
		SupportsTools: !strings.Contains(model, "reasoner"),
	}
	return &DeepSeekProvider{
		OpenAIProvider: NewOpenAIProvider(config),
	}, nil
}

func (p *DeepSeekProvider) Generate(ctx context.Context, messages []Message) (string, error) {
	return p.OpenAIProvider.Generate(ctx, messages)
}

func (p *DeepSeekProvider) GenerateWithTools(ctx context.Context, messages []Message, tools []Tool) (GenerationResponse, error) {
	return p.OpenAIProvider.GenerateWithTools(ctx, messages, tools)
}
