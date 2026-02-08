package llm

import (
	"context"
)

type ChatGPTProvider struct {
	*OpenAIProvider
}

func NewChatGPTProvider(apiKey string, model string) (*ChatGPTProvider, error) {
	if model == "" {
		model = "gpt-4o"
	}
	config := OpenAIProviderConfig{
		Name:          "chatgpt",
		BaseURL:       "https://api.openai.com/v1/chat/completions",
		APIKey:        apiKey,
		Model:         model,
		SupportsTools: true,
	}
	return &ChatGPTProvider{
		OpenAIProvider: NewOpenAIProvider(config),
	}, nil
}

func (p *ChatGPTProvider) Generate(ctx context.Context, messages []Message) (string, error) {
	return p.OpenAIProvider.Generate(ctx, messages)
}

func (p *ChatGPTProvider) GenerateWithTools(ctx context.Context, messages []Message, tools []Tool) (GenerationResponse, error) {
	return p.OpenAIProvider.GenerateWithTools(ctx, messages, tools)
}

func (p *ChatGPTProvider) Close() error { return nil }
