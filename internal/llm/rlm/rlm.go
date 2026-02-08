package rlm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"uaa/vdnd/internal/llm"
	"uaa/vdnd/internal/llm/ripgrep"
)

// SystemPromptBuilder is a function that constructs the system prompt.
type SystemPromptBuilder func(contextSize int, depth int) string

var RLMTools = []llm.Tool{
	{
		Name:        "execute_python",
		Description: "Execute Python code in the persistent sandbox environment to explore context, search rules, or perform calculations.",
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
	// NEW: ripgrep for fast rule lookup
	{
		Name:        "ripgrep",
		Description: "Search for text in rule files using ripgrep (fast). If rg is not installed, a loud warning will be printed.",
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

// RLM implementation.
type RLM struct {
	provider      llm.Provider
	maxIterations int
	maxDepth      int
	currentDepth  int
	pythonPath    string
	scriptPath    string
	promptBuilder SystemPromptBuilder
}

// Config for RLM.
type Config struct {
	MaxIterations       int
	MaxDepth            int
	PythonPath          string
	ScriptPath          string
	SystemPromptBuilder SystemPromptBuilder
}

// NewRLM creates a new RLM instance.
func NewRLM(provider llm.Provider, cfg Config) *RLM {
	if cfg.MaxIterations == 0 {
		cfg.MaxIterations = 100
	}
	if cfg.MaxDepth == 0 {
		cfg.MaxDepth = 1
	}
	if cfg.SystemPromptBuilder == nil {
		cfg.SystemPromptBuilder = BuildSystemPrompt
	}
	return &RLM{
		provider:      provider,
		maxIterations: cfg.MaxIterations,
		maxDepth:      cfg.MaxDepth,
		pythonPath:    cfg.PythonPath,
		scriptPath:    cfg.ScriptPath,
		promptBuilder: cfg.SystemPromptBuilder,
	}
}

// Complete processes the query against the context using iterative REPL exploration.
func (r *RLM) Complete(ctx context.Context, query string, contextData string, history []llm.Message) (string, string, error) {
	if r.currentDepth >= r.maxDepth {
		return "", "", fmt.Errorf("max recursion depth (%d) exceeded", r.maxDepth)
	}

	repl, err := NewREPLExecutor(r.pythonPath, r.scriptPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to start REPL: %w", err)
	}
	defer repl.Close()

	// Handle recursive calls
	repl.RecursiveHandler = func(q, c string) (string, error) {
		subRLM := &RLM{
			provider:      r.provider,
			maxIterations: r.maxIterations,
			maxDepth:      r.maxDepth,
			currentDepth:  r.currentDepth + 1,
			pythonPath:    r.pythonPath,
			scriptPath:    r.scriptPath,
			promptBuilder: r.promptBuilder,
		}
		// Pass same history to sub-calls for now
		resp, _, err := subRLM.Complete(ctx, q, c, history)
		return resp, err
	}

	// Serialize history
	historyJSON, _ := json.Marshal(history)
	if string(historyJSON) == "null" {
		historyJSON = []byte("[]")
	}

	// Inject context, query, and message_history into REPL
	setupCode := fmt.Sprintf("context = %q\nquery = %q\nmessage_history = json.loads(%q)\ndef FINAL(ans): print(f'FINAL_ANSWER: {ans}')", contextData, query, string(historyJSON))
	slog.Debug("REPL_SETUP", "code", setupCode)
	if result, err := repl.Execute(setupCode); err != nil {
		return "", "", fmt.Errorf("failed to setup REPL environment: %w", err)
	} else if result.Error != "" {
		return "", "", fmt.Errorf("Python error in setup: %s", result.Error)
	}

	systemPrompt := r.promptBuilder(len(contextData), r.currentDepth)
	messages := append([]llm.Message{}, history...)
	messages = append(messages, llm.Message{Role: "system", Content: systemPrompt})
	messages = append(messages, llm.Message{Role: "user", Content: query})

	for i := 0; i < r.maxIterations; i++ {
		response, err := r.provider.GenerateWithTools(ctx, messages, RLMTools)
		if err != nil {
			return "", "", fmt.Errorf("LLM generation failed: %w", err)
		}

		if response.Thinking != "" {
			slog.Debug("RLM_THINKING", "thinking", response.Thinking)
		}

		if response.FinishReason == "stop" {
			// When the model stops without calling a tool, we treat its content as the final answer.
			// This matches Orchestrator's behavior.
			if response.Content != "" {
				return response.Content, response.Thinking, nil
			}
			continue
		}

		if response.FinishReason == "tool_calls" {
			messages = append(messages, llm.Message{Role: "model", ToolCalls: response.ToolCalls, Thinking: response.Thinking})

			for _, call := range response.ToolCalls {
				if call.Name == "execute_python" {
					var args struct {
						Code string `json:"code"`
					}
					if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
						messages = append(messages, llm.Message{
							Role:       "tool",
							Name:       call.Name,
							ToolCallID: call.ID,
							Content:    fmt.Sprintf("Error parsing arguments: %v", err),
						})
						continue
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

					messages = append(messages, llm.Message{
						Role:       "tool",
						Name:       call.Name,
						ToolCallID: call.ID,
						Content:    observation,
					})
				} else if call.Name == "ripgrep" {
					var args struct {
						Pattern string `json:"pattern"`
						Path    string `json:"path"`
					}
					if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
						messages = append(messages, llm.Message{
							Role:       "tool",
							Name:       call.Name,
							ToolCallID: call.ID,
							Content:    fmt.Sprintf("Error parsing arguments: %v", err),
						})
						continue
					}
					
					slog.Info("TOOL_CALL",
						"tool", "ripgrep",
						"arguments", call.Arguments,
						"provider", r.provider.Name(),
						"model", r.provider.ModelName(),
					)
					result, err := ripgrep.Search(args.Pattern, args.Path)
					observation := ""
					if err != nil {
						observation = fmt.Sprintf("Ripgrep error: %v", err)
					} else {
						observation = result.ToJSON()
					}
					messages = append(messages, llm.Message{
						Role:       "tool",
						Name:       call.Name,
						ToolCallID: call.ID,
						Content:    observation,
					})
				} else {
					messages = append(messages, llm.Message{
						Role:       "tool",
						Name:       call.Name,
						ToolCallID: call.ID,
						Content:    fmt.Sprintf("Error: Unknown tool %s", call.Name),
					})
				}
			}
			continue
		}
	}

	return "", "", fmt.Errorf("max iterations (%d) exceeded without final answer", r.maxIterations)
}
