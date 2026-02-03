package rlm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"uaa/vdnd/internal/llm"
)

// SystemPromptBuilder is a function that constructs the system prompt.
type SystemPromptBuilder func(contextSize int, depth int) string

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
		cfg.MaxIterations = 30
	}
	if cfg.MaxDepth == 0 {
		cfg.MaxDepth = 5
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
func (r *RLM) Complete(ctx context.Context, query string, contextData string, history []llm.Message) (string, error) {
	if r.currentDepth >= r.maxDepth {
		return "", fmt.Errorf("max recursion depth (%d) exceeded", r.maxDepth)
	}

	repl, err := NewREPLExecutor(r.pythonPath, r.scriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to start REPL: %w", err)
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
		return subRLM.Complete(ctx, q, c, history)
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
		return "", fmt.Errorf("failed to setup REPL environment: %w", err)
	} else if result.Error != "" {
		return "", fmt.Errorf("Python error in setup: %s", result.Error)
	}

	systemPrompt := r.promptBuilder(len(contextData), r.currentDepth)
	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: query},
	}

	finalAnswerPattern := regexp.MustCompile(`(?s)FINAL_ANSWER:\s*(.*)`)
	codeBlockPattern := regexp.MustCompile("(?s)```python\n(.*?)\n```")

	for i := 0; i < r.maxIterations; i++ {
		response, err := r.provider.Generate(ctx, messages)
		if err != nil {
			return "", fmt.Errorf("LLM generation failed: %w", err)
		}

		// Handle thinking if present in response
		cleanResponse := llm.StripThinking(response)
		messages = append(messages, llm.Message{Role: "model", Content: response})

		// Try to find code blocks
		matches := codeBlockPattern.FindStringSubmatch(cleanResponse)
		var code string
		if len(matches) > 1 {
			code = matches[1]
		} else {
			// Fallback: maybe it just wrote code without blocks? Or it's the final answer.
			// If it contains FINAL(...), we'll catch it in the REPL execution result or by parsing.
			code = cleanResponse
		}

		// Execute in REPL
		slog.Debug("REPL_EXECUTE", "code", code)
		replResult, err := repl.Execute(code)
		if err != nil {
			observation := fmt.Sprintf("Execution error: %v", err)
			messages = append(messages, llm.Message{Role: "user", Content: observation})
			continue
		}

		observation := ""
		if replResult.Error != "" {
			observation = fmt.Sprintf("Python Error:\n%s", replResult.Error)
		} else {
			observation = replResult.Stdout
		}

		// Check for final answer in either LLM response or REPL output
		if finalMatches := finalAnswerPattern.FindStringSubmatch(cleanResponse); len(finalMatches) > 1 {
			return strings.TrimSpace(finalMatches[1]), nil
		}
		if finalMatches := finalAnswerPattern.FindStringSubmatch(observation); len(finalMatches) > 1 {
			return strings.TrimSpace(finalMatches[1]), nil
		}

		if observation == "" {
			observation = "(Success: no output)"
		}

		messages = append(messages, llm.Message{Role: "user", Content: observation})
	}

	return "", fmt.Errorf("max iterations (%d) exceeded without final answer", r.maxIterations)
}
