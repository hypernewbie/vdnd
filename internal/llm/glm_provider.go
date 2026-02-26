package llm

import (
	"context"
	"uaa/vdnd/internal/llm/llmtypes"
)

type GLMProvider struct {
	*OpenAIProvider
}

// NewGLMProvider creates a new GLM provider.
// Base URL: https://api.z.ai/api/coding/paas/v4/chat/completions
func NewGLMProvider(apiKey string, model string, enableThinking bool) (*GLMProvider, error) {
	if model == "" {
		model = "glm-4.7"
	}
	config := OpenAIProviderConfig{
		Name:           "glm",
		BaseURL:        "https://api.z.ai/api/coding/paas/v4/chat/completions",
		APIKey:         apiKey,
		Model:          model,
		SupportsTools:  true,
		EnableThinking: enableThinking,
	}
	return &GLMProvider{
		OpenAIProvider: NewOpenAIProvider(config),
	}, nil
}

func (p *GLMProvider) Generate(ctx context.Context, messages []llmtypes.Message) (string, error) {
	return p.OpenAIProvider.Generate(ctx, messages)
}

func (p *GLMProvider) GenerateWithTools(ctx context.Context, messages []llmtypes.Message, tools []llmtypes.Tool) (llmtypes.GenerationResponse, error) {
	return p.OpenAIProvider.GenerateWithTools(ctx, messages, tools)
}
