package subagents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm/llmtypes"
)

// ResearchSubagent handles rules lookup and game-state research through the Python sandbox.
type ResearchSubagent struct {
	provider   llmtypes.Provider
	pythonPath string
	scriptPath string
	prompt     string
}

func NewResearchSubagent(p llmtypes.Provider, python, script string) *ResearchSubagent {
	return &ResearchSubagent{
		provider:   p,
		pythonPath: python,
		scriptPath: script,
	}
}

func (a *ResearchSubagent) SetPrompt(prompt string) {
	if strings.TrimSpace(prompt) != "" {
		a.prompt = prompt
	}
}

func (a *ResearchSubagent) Name() string {
	return "call_research_assistant"
}

func (a *ResearchSubagent) Description() string {
	return "Delegate rules lookup and context research to a Python-powered research assistant."
}

func (a *ResearchSubagent) ToolDefinition() llmtypes.Tool {
	return llmtypes.Tool{
		Name:        a.Name(),
		Description: a.Description(),
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "The research question for the assistant."},
			},
			"required": []string{"query"},
		},
	}
}

func (a *ResearchSubagent) Run(ctx context.Context, query string, history []llmtypes.Message) (string, error) {
	repl, err := NewREPLExecutorWithEnv(a.pythonPath, a.scriptPath, []string{"VDM_PYTHON_READONLY=1"})
	if err != nil {
		return "", err
	}
	defer repl.Close()

	historyJSON, _ := json.Marshal(history)
	if string(historyJSON) == "null" {
		historyJSON = []byte("[]")
	}

	setupCode := fmt.Sprintf("context = \"\"\nquery = %q\nmessage_history = json.loads(%q)", query, string(historyJSON))
	if result, err := repl.Execute(setupCode); err != nil {
		return "", fmt.Errorf("failed to setup research sandbox: %w", err)
	} else if result.Error != "" {
		return "", fmt.Errorf("python setup error: %s", result.Error)
	}

	messages := append([]llmtypes.Message{}, history...)
	messages = append(messages, llmtypes.Message{Role: "system", Content: a.prompt})
	messages = append(messages, llmtypes.Message{Role: "user", Content: query})

	for i := 0; i < 10; i++ {
		response, err := a.provider.GenerateWithTools(ctx, messages, []llmtypes.Tool{researchPythonTool()})
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
			observation, err := a.executeToolCall(repl, call)
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

	return "", fmt.Errorf("research subagent max iterations exceeded")
}

func (a *ResearchSubagent) executeToolCall(repl *REPLExecutor, call llmtypes.ToolCall) (string, error) {
	if call.Name != "execute_python" {
		cli.PrintWarning(fmt.Sprintf("Research tool not supported: %s", call.Name))
		return "", fmt.Errorf("unknown research tool: %s", call.Name)
	}

	var args struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		return "", err
	}

	slog.Debug("RESEARCH_EXECUTE_PYTHON", "code", args.Code)
	cli.PrintSandboxExecution()
	replResult, err := repl.Execute(args.Code)
	if err != nil {
		return "", err
	}

	observation := replResult.Stdout
	if replResult.Error != "" {
		if observation != "" {
			observation += "\n"
		}
		observation += "Traceback:\n" + replResult.Error
	}
	if observation == "" {
		observation = "(Success: no output)"
	}

	return observation, nil
}

func researchPythonTool() llmtypes.Tool {
	return llmtypes.Tool{
		Name:        "execute_python",
		Description: "Execute Python code in the persistent sandbox for research and rules lookup.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"code": map[string]any{"type": "string", "description": "The Python code to execute."},
			},
			"required": []string{"code"},
		},
	}
}
