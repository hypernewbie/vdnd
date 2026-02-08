package llm

import (
	"context"
	"fmt"
	"strings"
	"uaa/vdnd/internal/llm/llmtypes"
)

// DummyProvider is a provider that echoes back the prompt instead of calling an LLM.
type DummyProvider struct {
	model string
}

func NewDummyProvider(model string) *DummyProvider {
	if model == "" {
		model = "dummy-model"
	}
	return &DummyProvider{model: model}
}

func (p *DummyProvider) Name() string {
	return "dummy"
}

func (p *DummyProvider) ModelName() string {
	return p.model
}

func (p *DummyProvider) Generate(ctx context.Context, messages []llmtypes.Message) (string, error) {
	var sb strings.Builder
	sb.WriteString("=== DUMMY PROVIDER ECHO ===\n")
	for _, m := range messages {
		// Escape common JSON delimiters to avoid orchestrator parsing collisions
		content := m.Content
		content = strings.ReplaceAll(content, "{", "«")
		content = strings.ReplaceAll(content, "}", "»")
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", m.Role, content))
	}
	sb.WriteString("===========================\n")
	sb.WriteString("This is a dry run. No real LLM was called.")
	return sb.String(), nil
}

func (p *DummyProvider) GenerateWithTools(ctx context.Context, messages []llmtypes.Message, tools []llmtypes.Tool) (llmtypes.GenerationResponse, error) {
	content, _ := p.Generate(ctx, messages)
	return llmtypes.GenerationResponse{
		Content:      content,
		FinishReason: "stop",
	}, nil
}

func (p *DummyProvider) SupportsToolCalling() bool {
	return false
}
