package subagents

import (
	"context"
	"fmt"

	"uaa/vdnd/internal/llm/llmtypes"
)

// PythonReadOnlySubagent handles direct Python execution in read-only mode.
type PythonReadOnlySubagent struct {
	repl *REPLExecutor
}

func NewPythonReadOnlySubagent(pythonPath, scriptPath string) (*PythonReadOnlySubagent, error) {
	repl, err := NewREPLExecutorWithEnv(pythonPath, scriptPath, []string{"VDM_PYTHON_READONLY=1"})
	if err != nil {
		return nil, err
	}
	return &PythonReadOnlySubagent{repl: repl}, nil
}

func (a *PythonReadOnlySubagent) Name() string {
	return "execute_python_readonly"
}

func (a *PythonReadOnlySubagent) Description() string {
	return "Execute Python code in strictly read-only sandbox for file inspection."
}

func (a *PythonReadOnlySubagent) ToolDefinition() llmtypes.Tool {
	return llmtypes.Tool{
		Name:        a.Name(),
		Description: a.Description(),
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"code": map[string]any{"type": "string", "description": "The Python code to execute."},
			},
			"required": []string{"code"},
		},
	}
}

func (a *PythonReadOnlySubagent) Run(ctx context.Context, code string, history []llmtypes.Message) (string, error) {
	result, err := a.repl.Execute(code)
	if err != nil {
		return "", fmt.Errorf("python execution error: %w", err)
	}

	observation := result.Stdout
	if result.Error != "" {
		observation += "\nError:\n" + result.Error
	}
	if observation == "" {
		observation = "(Success: no output)"
	}

	return observation, nil
}
