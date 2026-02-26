package subagents

import (
	"context"
	"fmt"

	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm/llmtypes"
)

// PythonSubagent handles direct Python execution for the Orchestrator.
type PythonSubagent struct {
	repl *REPLExecutor
}

func NewPythonSubagent(pythonPath, scriptPath string) (*PythonSubagent, error) {
	repl, err := NewREPLExecutor(pythonPath, scriptPath)
	if err != nil {
		return nil, err
	}
	return &PythonSubagent{repl: repl}, nil
}

func (a *PythonSubagent) Name() string {
	return "execute_python"
}

func (a *PythonSubagent) Description() string {
	return "Execute Python code in the persistent sandbox for reading/writing files and performing logic."
}

func (a *PythonSubagent) ToolDefinition() llmtypes.Tool {
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

func (a *PythonSubagent) Run(ctx context.Context, code string, history []llmtypes.Message) (string, error) {
	result, err := a.repl.Execute(code)
	if err != nil {
		return "", fmt.Errorf("python execution error: %w", err)
	}
	
	observation := result.Stdout
	if result.Error != "" {
		cli.PrintError(fmt.Sprintf("Python Sandbox Error: %s", result.Error))
		observation += "\nError:\n" + result.Error
	}
	if observation == "" {
		observation = "(Success: no output)"
	}
	
	return observation, nil
}
