package rlm

import (
	"context"
	"path/filepath"
	"testing"

	"uaa/vdnd/internal/llm"
)

type MockProvider struct {
	responses []llm.GenerationResponse
	calls     int
}

func (m *MockProvider) Name() string      { return "mock" }
func (m *MockProvider) ModelName() string { return "mock-model" }
func (m *MockProvider) Generate(ctx context.Context, messages []llm.Message) (string, error) {
	return "", nil
}

func (m *MockProvider) GenerateWithTools(ctx context.Context, messages []llm.Message, tools []llm.Tool) (llm.GenerationResponse, error) {
	if m.calls < len(m.responses) {
		res := m.responses[m.calls]
		m.calls++
		return res, nil
	}
	return llm.GenerationResponse{Content: "FINAL_ANSWER: I don't know", FinishReason: "stop"}, nil
}

func (m *MockProvider) SupportsToolCalling() bool { return true }

func TestRLMStepByStep(t *testing.T) {
	absRoot, _ := filepath.Abs("../../../")
	pythonPath := FindPythonPath(absRoot)
	scriptPath := filepath.Join(absRoot, "py", "restricted_python.py")

	mock := &MockProvider{
		responses: []llm.GenerationResponse{
			{
				FinishReason: "tool_calls",
				ToolCalls: []llm.ToolCall{
					{
						ID:        "call_1",
						Name:      "execute_python",
						Arguments: `{"code": "print(context[:5])"}`,
					},
				},
			},
			{
				FinishReason: "tool_calls",
				ToolCalls: []llm.ToolCall{
					{
						ID:        "call_2",
						Name:      "execute_python",
						Arguments: `{"code": "print('done')"}`,
					},
				},
			},
			{
				FinishReason: "stop",
				Content:      "Hello",
			},
		},
	}

	mockPromptBuilder := func(size, depth int) string {
		return "Mock Prompt"
	}

	rlm := NewRLM(mock, Config{
		MaxIterations:       5,
		PythonPath:          pythonPath,
		ScriptPath:          scriptPath,
		SystemPromptBuilder: mockPromptBuilder,
	})

	answer, _, err := rlm.Complete(context.Background(), "What is the start?", "Hello World", nil)
	if err != nil {
		t.Fatalf("RLM Complete failed: %v", err)
	}

	if answer != "Hello" {
		t.Errorf("Expected 'Hello', got %q", answer)
	}

	if mock.calls != 3 {
		t.Errorf("Expected 3 LLM calls, got %d", mock.calls)
	}
}

func TestRLMRecursion(t *testing.T) {
	absRoot, _ := filepath.Abs("../../../")
	pythonPath := FindPythonPath(absRoot)
	scriptPath := filepath.Join(absRoot, "py", "restricted_python.py")

	// This mock will handle both the top-level call and the recursive call
	mock := &MockProvider{
		responses: []llm.GenerationResponse{
			{
				FinishReason: "tool_calls",
				ToolCalls: []llm.ToolCall{
					{
						ID:        "call_1",
						Name:      "execute_python",
						Arguments: `{"code": "sub_res = recursive_llm(\"What is X?\", \"X is 42\")"}`,
					},
				},
			},
			{
				// This response is for the sub-call
				FinishReason: "stop",
				Content:      "42",
			},
			{
				// Final response for top-level call
				FinishReason: "stop",
				Content:      "The answer is 42",
			},
		},
	}

	mockPromptBuilder := func(size, depth int) string {
		return "Mock Prompt"
	}

	rlm := NewRLM(mock, Config{
		MaxIterations:       5,
		MaxDepth:            2,
		PythonPath:          pythonPath,
		ScriptPath:          scriptPath,
		SystemPromptBuilder: mockPromptBuilder,
	})

	answer, _, err := rlm.Complete(context.Background(), "Get sub answer", "Ignored", nil)
	if err != nil {
		t.Fatalf("RLM Complete failed: %v", err)
	}

	if answer != "The answer is 42" {
		t.Errorf("Expected 'The answer is 42', got %q", answer)
	}

	if mock.calls != 3 {
		t.Errorf("Expected 3 LLM calls total (2 for main, 1 for sub), got %d", mock.calls)
	}
}
