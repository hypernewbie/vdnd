package rlm

import (
	"context"
	"path/filepath"
	"testing"

	"uaa/vdnd/internal/llm"
)

type MockProvider struct {
	responses []string
	calls     int
}

func (m *MockProvider) Name() string      { return "mock" }
func (m *MockProvider) ModelName() string { return "mock-model" }
func (m *MockProvider) Generate(ctx context.Context, messages []llm.Message) (string, error) {
	if m.calls < len(m.responses) {
		res := m.responses[m.calls]
		m.calls++
		return res, nil
	}
	return "FINAL_ANSWER: I don't know", nil
}

func (m *MockProvider) GenerateWithTools(ctx context.Context, messages []llm.Message, tools []llm.Tool) (llm.GenerationResponse, error) {
	return llm.GenerationResponse{}, nil
}

func (m *MockProvider) SupportsToolCalling() bool { return false }

func TestRLMStepByStep(t *testing.T) {
	absRoot, _ := filepath.Abs("../../../")
	pythonPath := FindPythonPath(absRoot)
	scriptPath := filepath.Join(absRoot, "py", "restricted_python.py")

	mock := &MockProvider{
		responses: []string{
			"```python\nprint(context[:5])\n```",
			"```python\nFINAL(\"Hello\")\n```",
		},
	}

	rlm := NewRLM(mock, Config{
		MaxIterations: 5,
		PythonPath:    pythonPath,
		ScriptPath:    scriptPath,
	})

	answer, err := rlm.Complete(context.Background(), "What is the start?", "Hello World", nil)
	if err != nil {
		t.Fatalf("RLM Complete failed: %v", err)
	}

	if answer != "Hello" {
		t.Errorf("Expected 'Hello', got %q", answer)
	}

	if mock.calls != 2 {
		t.Errorf("Expected 2 LLM calls, got %d", mock.calls)
	}
}

func TestRLMRecursion(t *testing.T) {
	absRoot, _ := filepath.Abs("../../../")
	pythonPath := FindPythonPath(absRoot)
	scriptPath := filepath.Join(absRoot, "py", "restricted_python.py")

	// This mock will handle both the top-level call and the recursive call
	mock := &MockProvider{
		responses: []string{
			"```python\nsub_res = recursive_llm(\"What is X?\", \"X is 42\")\nFINAL(f\"The answer is {sub_res}\")\n```",
			"```python\nFINAL(\"42\")\n```",
		},
	}

	rlm := NewRLM(mock, Config{
		MaxIterations: 5,
		MaxDepth:      2,
		PythonPath:    pythonPath,
		ScriptPath:    scriptPath,
	})

	answer, err := rlm.Complete(context.Background(), "Get sub answer", "Ignored", nil)
	if err != nil {
		t.Fatalf("RLM Complete failed: %v", err)
	}

	if answer != "The answer is 42" {
		t.Errorf("Expected 'The answer is 42', got %q", answer)
	}

	// 1 for top level, 1 for sub-call
	if mock.calls != 2 {
		t.Errorf("Expected 2 LLM calls total, got %d", mock.calls)
	}
}
