package rlm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"uaa/vdnd/internal/llm/llmtypes"
	"uaa/vdnd/internal/llm/ripgrep"
)

// ToolHandler processes a tool call, returning the observation string.
type ToolHandler func(ctx context.Context, call llmtypes.ToolCall, session any) (string, error)

// SessionFactory creates per-call session state (e.g., REPL, cli.Deps).
// cleanup() is called after the call completes.
type SessionFactory func() (session any, cleanup func(), err error)

// SystemPromptBuilder is a function that constructs the system prompt.
type SystemPromptBuilder func(contextSize int, depth int) string

// Config for a parameterized RLM.
type Config struct {
	MaxIterations       int
	MaxDepth            int
	Tools               []llmtypes.Tool
	ToolHandlers        map[string]ToolHandler
	SessionFactory      SessionFactory
	SystemPromptBuilder SystemPromptBuilder
}

// RLMTools is deprecated, use Config.Tools instead.
var RLMTools = []llmtypes.Tool{
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
	provider       llmtypes.Provider
	maxIterations  int
	maxDepth       int
	currentDepth   int
	tools          []llmtypes.Tool
	handlers       map[string]ToolHandler
	sessionFactory SessionFactory
	promptBuilder  SystemPromptBuilder
}

// NewRLM creates a new RLM instance with default research configuration.
func NewRLM(provider llmtypes.Provider, pythonPath, scriptPath string) *RLM {
	return NewResearchRLM(provider, pythonPath, scriptPath)
}

// NewRLMWithConfig creates a new RLM instance with the given config.
func NewRLMWithConfig(provider llmtypes.Provider, cfg Config) *RLM {
	if cfg.MaxIterations == 0 {
		cfg.MaxIterations = 100
	}
	if cfg.MaxDepth == 0 {
		cfg.MaxDepth = 1
	}
	if cfg.SystemPromptBuilder == nil {
		panic("SystemPromptBuilder must be provided")
	}
	return &RLM{
		provider:       provider,
		maxIterations:  cfg.MaxIterations,
		maxDepth:       cfg.MaxDepth,
		tools:          cfg.Tools,
		handlers:       cfg.ToolHandlers,
		sessionFactory: cfg.SessionFactory,
		promptBuilder:  cfg.SystemPromptBuilder,
	}
}

// Complete processes the query against the context using iterative exploration.
func (r *RLM) Complete(ctx context.Context, query string, contextData string, history []llmtypes.Message) (string, string, error) {
	if r.currentDepth >= r.maxDepth {
		return "", "", fmt.Errorf("max recursion depth (%d) exceeded", r.maxDepth)
	}

	var session any
	var cleanup func()
	var err error

	if r.sessionFactory != nil {
		session, cleanup, err = r.sessionFactory()
		if err != nil {
			return "", "", fmt.Errorf("failed to create session: %w", err)
		}
		if cleanup != nil {
			defer cleanup()
		}
	}

	// Session initialization for Research RLM (Python REPL)
	if repl, ok := session.(*REPLExecutor); ok {
		// Handle recursive calls
		repl.RecursiveHandler = func(q, c string) (string, error) {
			subRLM := &RLM{
				provider:       r.provider,
				maxIterations:  r.maxIterations,
				maxDepth:       r.maxDepth,
				currentDepth:   r.currentDepth + 1,
				tools:          r.tools,
				handlers:       r.handlers,
				sessionFactory: r.sessionFactory,
				promptBuilder:  r.promptBuilder,
			}
			resp, _, err := subRLM.Complete(ctx, q, c, history)
			return resp, err
		}

		historyJSON, _ := json.Marshal(history)
		if string(historyJSON) == "null" {
			historyJSON = []byte("[]")
		}
		setupCode := fmt.Sprintf("context = %q\nquery = %q\nmessage_history = json.loads(%q)\ndef FINAL(ans): print(f'FINAL_ANSWER: {ans}')", contextData, query, string(historyJSON))
		if result, err := repl.Execute(setupCode); err != nil {
			return "", "", fmt.Errorf("failed to setup REPL environment: %w", err)
		} else if result.Error != "" {
			return "", "", fmt.Errorf("Python error in setup: %s", result.Error)
		}
	}

	systemPrompt := r.promptBuilder(len(contextData), r.currentDepth)
	messages := append([]llmtypes.Message{}, history...)
	messages = append(messages, llmtypes.Message{Role: "system", Content: systemPrompt})
	messages = append(messages, llmtypes.Message{Role: "user", Content: query})

	for i := 0; i < r.maxIterations; i++ {
		response, err := r.provider.GenerateWithTools(ctx, messages, r.tools)
		if err != nil {
			return "", "", fmt.Errorf("LLM generation failed: %w", err)
		}

		if response.Thinking != "" {
			slog.Debug("RLM_THINKING", "thinking", response.Thinking)
		}

		if response.FinishReason == "stop" {
			if response.Content != "" {
				return response.Content, response.Thinking, nil
			}
			continue
		}

		if response.FinishReason == "tool_calls" {
			messages = append(messages, llmtypes.Message{Role: "model", ToolCalls: response.ToolCalls, Thinking: response.Thinking})

			for _, call := range response.ToolCalls {
				handler, ok := r.handlers[call.Name]
				if !ok {
					messages = append(messages, llmtypes.Message{
						Role:       "tool",
						Name:       call.Name,
						ToolCallID: call.ID,
						Content:    fmt.Sprintf("Error: Unknown tool %s", call.Name),
					})
					continue
				}

				observation, err := handler(ctx, call, session)
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
			continue
		}
	}

	return "", "", fmt.Errorf("max iterations (%d) exceeded without final answer", r.maxIterations)
}

// Common handlers

func RipgrepHandler(ctx context.Context, call llmtypes.ToolCall, session any) (string, error) {
	var args struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		return "", err
	}
	
	slog.Info("TOOL_CALL",
		"tool", "ripgrep",
		"arguments", call.Arguments,
	)
	
	result, err := ripgrep.Search(args.Pattern, args.Path)
	if err != nil {
		return fmt.Sprintf("Ripgrep error: %v", err), nil
	}
	return result.ToJSON(), nil
}