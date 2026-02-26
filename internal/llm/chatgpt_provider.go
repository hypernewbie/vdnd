package llm

import (
	"context"

	"uaa/vdnd/internal/llm/llmtypes"
)

type ChatGPTProvider struct {
	*OpenAIProvider
}

func NewChatGPTProvider(apiKey string, model string, enableThinking bool) (*ChatGPTProvider, error) {
	if model == "" {
		model = "gpt-4o"
	}
	config := OpenAIProviderConfig{
		Name:           "chatgpt",
		BaseURL:        "https://api.openai.com/v1/chat/completions",
		APIKey:         apiKey,
		Model:          model,
		SupportsTools:  true,
		EnableThinking: enableThinking,
	}
	return &ChatGPTProvider{
		OpenAIProvider: NewOpenAIProvider(config),
	}, nil
}

func (p *ChatGPTProvider) Generate(ctx context.Context, messages []llmtypes.Message) (string, error) {
	return p.OpenAIProvider.Generate(ctx, messages)
}

func (p *ChatGPTProvider) GenerateWithTools(ctx context.Context, messages []llmtypes.Message, tools []llmtypes.Tool) (llmtypes.GenerationResponse, error) {
	return p.OpenAIProvider.GenerateWithTools(ctx, messages, tools)
}

func (p *ChatGPTProvider) Close() error { return nil }
