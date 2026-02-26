package subagents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm/llmtypes"
	"uaa/vdnd/internal/llm/vdengine"
	"uaa/vdnd/internal/llm/vdhelpers"
)

// ExecutionSubagent handles state mutation via VDEngine tools.
type ExecutionSubagent struct {
	provider llmtypes.Provider
	engine   *vdengine.VDEngine
	prompt   string
}

func NewExecutionSubagent(p llmtypes.Provider, deps cli.Deps) *ExecutionSubagent {
	return &ExecutionSubagent{
		provider: p,
		engine:   vdengine.New(deps),
	}
}

func (a *ExecutionSubagent) SetPrompt(prompt string) {
	if strings.TrimSpace(prompt) != "" {
		a.prompt = prompt
	}
}

func (a *ExecutionSubagent) Name() string {
	return "call_vdm_execution"
}

func (a *ExecutionSubagent) Description() string {
	return "Delegate state-changing actions to the dedicated VD execution agent."
}

func (a *ExecutionSubagent) ToolDefinition() llmtypes.Tool {
	return llmtypes.Tool{
		Name:        a.Name(),
		Description: a.Description(),
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"instruction": map[string]any{"type": "string", "description": "The state-changing instruction to execute."},
			},
			"required": []string{"instruction"},
		},
	}
}

func (a *ExecutionSubagent) Run(ctx context.Context, instruction string, history []llmtypes.Message) (string, error) {
	messages := append([]llmtypes.Message{}, history...)
	messages = append(messages, llmtypes.Message{Role: "system", Content: a.prompt})
	messages = append(messages, llmtypes.Message{Role: "user", Content: instruction})

	for i := 0; i < 5; i++ {
		response, err := a.provider.GenerateWithTools(ctx, messages, a.engine.Tools())
		if err != nil {
			return "", err
		}

		if response.FinishReason == "stop" && strings.TrimSpace(response.Content) != "" {
			return response.Content, nil
		}

		if response.FinishReason != "tool_calls" {
			continue
		}

		messages = append(messages, llmtypes.Message{Role: "model", ToolCalls: response.ToolCalls, Thinking: response.Thinking})
		for _, call := range response.ToolCalls {
			observation, err := a.executeToolCall(call)
			if err != nil {
				observation = fmt.Sprintf("Error: %v", err)
			}
			messages = append(messages, llmtypes.Message{
				Role:       "tool",
				Name:       call.Name,
				ToolCallID: call.ID,
				Content:    observation,
			})
		}
	}

	return "", fmt.Errorf("execution subagent max iterations exceeded")
}

func (a *ExecutionSubagent) executeToolCall(call llmtypes.ToolCall) (string, error) {
	cli.PrintVDEngineExecution(call.Name, call.Arguments)
	stdout, exitCode, _, err := a.engine.ExecuteTool(call)
	if err != nil {
		cli.PrintWarning(fmt.Sprintf("VD tool failed: %v", err))
		return errorJSON(err.Error()), nil
	}

	result := vdhelpers.VDResult{Stdout: stdout, ExitCode: exitCode}
	if exitCode != 0 && result.Error == "" {
		result.Error = "Command failed (non-zero exit)"
	}
	b, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func errorJSON(msg string) string {
	result := vdhelpers.VDResult{ExitCode: 1, Error: msg}
	b, _ := json.Marshal(result)
	return string(b)
}
