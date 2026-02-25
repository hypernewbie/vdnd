package llm

import (
	"context"
	"io"
	"testing"

	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm/llmtypes"
	"uaa/vdnd/internal/state"
)

type scriptedProvider struct {
	responses []llmtypes.GenerationResponse
	calls     int
}

func (p *scriptedProvider) Name() string              { return "scripted" }
func (p *scriptedProvider) ModelName() string         { return "test-model" }
func (p *scriptedProvider) SupportsToolCalling() bool { return true }
func (p *scriptedProvider) Generate(ctx context.Context, messages []llmtypes.Message) (string, error) {
	return "", nil
}
func (p *scriptedProvider) GenerateWithTools(ctx context.Context, messages []llmtypes.Message, tools []llmtypes.Tool) (llmtypes.GenerationResponse, error) {
	if p.calls >= len(p.responses) {
		return llmtypes.GenerationResponse{Content: "fallback", FinishReason: "stop"}, nil
	}
	resp := p.responses[p.calls]
	p.calls++
	return resp, nil
}
func (p *scriptedProvider) GenerateStream(ctx context.Context, messages []llmtypes.Message, tools []llmtypes.Tool, callback func(chunk string) error) (llmtypes.GenerationResponse, error) {
	resp, err := p.GenerateWithTools(ctx, messages, tools)
	if err != nil {
		return llmtypes.GenerationResponse{}, err
	}
	if resp.Content != "" && callback != nil {
		if err := callback(resp.Content); err != nil {
			return llmtypes.GenerationResponse{}, err
		}
	}
	return resp, nil
}

type fakeSubagent struct {
	name     string
	result   string
	lastRun  string
	runCount int
}

func (a *fakeSubagent) Name() string { return a.name }
func (a *fakeSubagent) Description() string {
	return "fake"
}
func (a *fakeSubagent) ToolDefinition() llmtypes.Tool {
	return llmtypes.Tool{
		Name:        a.name,
		Description: "fake",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string"},
			},
			"required": []string{"query"},
		},
	}
}
func (a *fakeSubagent) Run(ctx context.Context, query string, history []llmtypes.Message) (string, error) {
	a.runCount++
	a.lastRun = query
	return a.result, nil
}

func TestOrchestrator_DelegatesToSubagent(t *testing.T) {
	deps := cli.Deps{
		Store: &state.MemoryStore{State: &state.GameState{
			Entities:      make(map[string]*state.EntityState),
			ReactionsUsed: make(map[string]bool),
		}},
		Stderr: io.Discard,
	}

	provider := &scriptedProvider{responses: []llmtypes.GenerationResponse{
		{
			FinishReason: "tool_calls",
			ToolCalls: []llmtypes.ToolCall{{
				ID:        "call_1",
				Name:      "call_research_assistant",
				Arguments: `{"query":"check flanking rule"}`,
			}},
		},
		{
			FinishReason: "stop",
			Content:      "You gain +2 circumstance bonus from flanking.",
		},
	}}

	agent := &fakeSubagent{name: "call_research_assistant", result: "Flanking gives +2."}
	orch := NewOrchestrator(context.Background(), provider, deps)
	orch.RegisterSubagents(agent)

	got, err := orch.ProcessInput(context.Background(), "Can I flank the enemy?", nil)
	if err != nil {
		t.Fatalf("ProcessInput error: %v", err)
	}
	if got != "You gain +2 circumstance bonus from flanking." {
		t.Fatalf("unexpected response: %q", got)
	}
	if agent.runCount != 1 {
		t.Fatalf("expected subagent to run once, got %d", agent.runCount)
	}
	if agent.lastRun != "check flanking rule" {
		t.Fatalf("unexpected subagent input: %q", agent.lastRun)
	}
}
