package rlm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"uaa/vdnd/internal/llm/llmtypes"
)

func NewResearchRLM(provider llmtypes.Provider, pythonPath, scriptPath string) *RLM {
	return NewRLMWithConfig(provider, Config{
		MaxIterations: 100,
		MaxDepth:      1,
		Tools:         ResearchTools(),
		ToolHandlers:  ResearchHandlers(),
		SessionFactory: NewREPLSessionFactory(pythonPath, scriptPath),
		SystemPromptBuilder: BuildDMSystemPrompt,
	})
}

func ResearchTools() []llmtypes.Tool {
	return []llmtypes.Tool{
		{
			Name:        "execute_python",
			Description: "Execute Python code in the persistent sandbox environment to explore context, search rules, or perform calculations. Use this for research and rule lookups ONLY. Do NOT attempt to simulate game state changes in Python.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]map[string]any{
					"code": {
						"type":        "string",
						"description": "The Python code to execute.",
					},
				},
				"required": []string{"code"},
			},
		},
		{
			Name:        "ripgrep",
			Description: "Search for text in rule files using ripgrep (fast).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{"type": "string", "description": "Search pattern (regex)"},
					"path":    map[string]any{"type": "string", "description": "Directory to search (default: 'rules/')"},
				},
				"required": []any{"pattern"},
			},
		},
	}
}

func ResearchHandlers() map[string]ToolHandler {
	return map[string]ToolHandler{
		"execute_python": func(ctx context.Context, call llmtypes.ToolCall, session any) (string, error) {
			repl, ok := session.(*REPLExecutor)
			if !ok {
				return "", fmt.Errorf("session is not a REPLExecutor")
			}
			var args struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
				return "", err
			}

			slog.Debug("REPL_EXECUTE", "code", args.Code)
			replResult, err := repl.Execute(args.Code)
			observation := ""
			if err != nil {
				observation = fmt.Sprintf("Error executing code: %v", err)
			} else {
				observation = replResult.Stdout
				if replResult.Error != "" {
					if observation != "" {
						observation += "\n"
					}
					observation += "Traceback:\n" + replResult.Error
				}
			}

			if observation == "" {
				observation = "(Success: no output)"
			}

			slog.Info("REPL_EXECUTE", "output", observation)
			return observation, nil
		},
		"ripgrep": RipgrepHandler,
	}
}

func NewREPLSessionFactory(pythonPath, scriptPath string) SessionFactory {
	return func() (any, func(), error) {
		repl, err := NewREPLExecutor(pythonPath, scriptPath)
		if err != nil {
			return nil, nil, err
		}
		return repl, func() { repl.Close() }, nil
	}
}