package llm

import (
	"context"
	"strings"
	"testing"

	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm/llmtypes"
)

// mockRLM is a simple mock that returns predefined responses
type mockRLM struct {
	Response string
	Called   bool
	LastQuery string
}

func (m *mockRLM) Complete(ctx context.Context, query string, contextData string, history []llmtypes.Message) (string, string, error) {
	m.Called = true
	m.LastQuery = query
	return m.Response, "", nil
}

// scriptProvider mocks a conversation flow for the Orchestrator
type scriptProvider struct {
	t             *testing.T
	responses     []llmtypes.GenerationResponse
	callIndex     int
	supportsTools bool
}

func (p *scriptProvider) Name() string              { return "script-mock" }
func (p *scriptProvider) ModelName() string         { return "script-model" }
func (p *scriptProvider) SupportsToolCalling() bool { return p.supportsTools }
func (p *scriptProvider) Generate(ctx context.Context, messages []llmtypes.Message) (string, error) {
	return "", nil
}
func (p *scriptProvider) GenerateWithTools(ctx context.Context, messages []llmtypes.Message, tools []llmtypes.Tool) (llmtypes.GenerationResponse, error) {
	if p.callIndex >= len(p.responses) {
		return llmtypes.GenerationResponse{FinishReason: "stop", Content: "End of script"}, nil
	}
	resp := p.responses[p.callIndex]
	p.callIndex++
	return resp, nil
}

func TestOrchestrator_SupervisorFlow(t *testing.T) {
	// Setup Mocks
	researchMock := &mockRLM{Response: "Longsword deals 1d8 slashing damage."}
	vdlmMock := &mockRLM{Response: "Action: Strike. Result: Hit! Damage: 5 slashing."}

	// Setup Scripted Provider for Orchestrator
	// Step 1: Orchestrator decides to call Research
	step1 := llmtypes.GenerationResponse{
		FinishReason: "tool_calls",
		ToolCalls: []llmtypes.ToolCall{
			{
				ID:   "call_1",
				Name: "call_research_assistant",
				Arguments: `{"query": "longsword damage"}`,
			},
		},
	}
	// Step 2: Orchestrator receives research, decides to call VDLM
	step2 := llmtypes.GenerationResponse{
		FinishReason: "tool_calls",
		ToolCalls: []llmtypes.ToolCall{
			{
				ID:   "call_2",
				Name: "call_vdm_execution",
				Arguments: `{"instruction": "Attack goblin with longsword", "research_notes": "Longsword deals 1d8 slashing damage."}`,
			},
		},
	}
	// Step 3: Orchestrator receives execution result, narrates final response
	step3 := llmtypes.GenerationResponse{
		FinishReason: "stop",
		Content:      "You swing your longsword and hit the goblin for 5 damage!",
	}

	provider := &scriptProvider{
		t:             t,
		responses:     []llmtypes.GenerationResponse{step1, step2, step3},
		supportsTools: true,
	}

	// Initialize Orchestrator
	deps := cli.DefaultDeps() // Use default deps (or mock if needed, but status is just text)
	o := NewOrchestrator(context.Background(), provider, deps)
	o.SetRLMs(researchMock, vdlmMock)

	// Execute
	input := "How much damage does a longsword do, and can I attack the goblin?"
	result, err := o.ProcessInput(context.Background(), input)

	// Verify Orchestrator Output
	if err != nil {
		t.Fatalf("ProcessInput failed: %v", err)
	}
	expectedNarration := "You swing your longsword and hit the goblin for 5 damage!"
	if result != expectedNarration {
		t.Errorf("Expected final narration %q, got %q", expectedNarration, result)
	}

	// Verify Research RLM was called
	if !researchMock.Called {
		t.Error("Research RLM was not called")
	}
	if !strings.Contains(researchMock.LastQuery, "longsword damage") {
		t.Errorf("Research query mismatch. Got: %s", researchMock.LastQuery)
	}

	// Verify VDLM was called
	if !vdlmMock.Called {
		t.Error("VDLM was not called")
	}
	// VDLM query is constructed inside executeTool
	if !strings.Contains(vdlmMock.LastQuery, "Attack goblin") {
		t.Errorf("VDLM instruction missing in query. Got: %s", vdlmMock.LastQuery)
	}
	if !strings.Contains(vdlmMock.LastQuery, "Longsword deals 1d8") {
		t.Errorf("Research notes missing in VDLM query. Got: %s", vdlmMock.LastQuery)
	}
}
