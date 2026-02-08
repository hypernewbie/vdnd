package llm

import (
	"context"
	"strings"
	"uaa/vdnd/internal/llm/llmtypes"
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

func (p *DeepSeekProvider) Generate(ctx context.Context, messages []llmtypes.Message) (string, error) {
	return p.OpenAIProvider.Generate(ctx, messages)
}

func (p *DeepSeekProvider) GenerateWithTools(ctx context.Context, messages []llmtypes.Message, tools []llmtypes.Tool) (llmtypes.GenerationResponse, error) {
	return p.OpenAIProvider.GenerateWithTools(ctx, messages, tools)
}
