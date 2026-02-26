package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm/llmtypes"
	"uaa/vdnd/internal/llm/subagents"
	)

	// ErrOrchestratorBusy is returned when the orchestrator is already processing a request.
	var ErrOrchestratorBusy = fmt.Errorf("the DM is currently busy thinking... please wait")

	// StreamReporter receives status and token chunks for progressive UX updates.
	type StreamReporter interface {
	Status(msg string)
	Chunk(delta string)
	}

	// Orchestrator coordinates communication between the user, the LLM, and subagents.
	type Orchestrator struct {
	mu           sync.Mutex
	activeCancel context.CancelFunc
	provider     llmtypes.Provider
	deps         cli.Deps
	prompt       string
	subagents    map[string]llmtypes.Subagent
	tools        []llmtypes.Tool
	history      []llmtypes.Message
	pythonRepl   *subagents.REPLExecutor
	pythonPath   string
	scriptPath   string
	}

func (o *Orchestrator) getVDManual() (string, error) {
	content, err := o.deps.Store.GetManual()
	if err != nil {
		return "", fmt.Errorf("failed to read VD manual: %w", err)
	}
	return content, nil
}

// SetPrompt sets the system prompt for the orchestrator.
func (o *Orchestrator) SetPrompt(prompt string) {
	o.prompt = prompt
}

// NewOrchestrator creates a new LLM orchestrator.
func NewOrchestrator(context context.Context, provider llmtypes.Provider, deps cli.Deps, pythonPath, scriptPath string) (*Orchestrator, error) {
	var repl *subagents.REPLExecutor
	var err error
	if pythonPath != "" && scriptPath != "" {
		repl, err = subagents.NewREPLExecutor(pythonPath, scriptPath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize orchestrator python: %w", err)
		}
	}

	o := &Orchestrator{
		provider:   provider,
		deps:       deps,
		subagents:  make(map[string]llmtypes.Subagent),
		pythonRepl: repl,
		pythonPath: pythonPath,
		scriptPath: scriptPath,
	}

	o.history = []llmtypes.Message{
		{Role: "model", Content: "I am ready to be your Dungeon Master. What happens next?"},
	}

	return o, nil
}

func (o *Orchestrator) injectSandboxState() string {
	historyJSON, _ := json.Marshal(o.history)
	return fmt.Sprintf("message_history = json.loads(%q)\ncontext = \"\"", string(historyJSON))
}

// RegisterSubagents sets the available subagents as callable tools.
func (o *Orchestrator) RegisterSubagents(agents ...llmtypes.Subagent) {
	o.subagents = make(map[string]llmtypes.Subagent)
	o.tools = o.tools[:0]
	for _, agent := range agents {
		if agent == nil {
			continue
		}
		o.subagents[agent.Name()] = agent
		o.tools = append(o.tools, agent.ToolDefinition())
	}

	// Add direct tools
	o.tools = append(o.tools, llmtypes.Tool{
		Name:        "get_vd_manual",
		Description: "Get the full VD CLI manual for command reference.",
	})
	o.tools = append(o.tools, llmtypes.Tool{
		Name:        "execute_python",
		Description: "Execute Python code in the sandbox. CRITICAL: DO NOT import os/json. Use list_dir('sandbox') to see your files. message_history is available.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"code": map[string]any{"type": "string", "description": "The Python code to execute."},
			},
			"required": []string{"code"},
		},
	})
}

// ProviderInfo returns the provider name and model name.
func (o *Orchestrator) ProviderInfo() (string, string) {
	if o.provider == nil {
		return "none", "none"
	}
	return o.provider.Name(), o.provider.ModelName()
}

// ProcessInput takes natural language input and returns a narrated response.
func (o *Orchestrator) ProcessInput(ctx context.Context, input string, reporter StreamReporter) (string, error) {
	if !o.mu.TryLock() {
		return "", ErrOrchestratorBusy
	}
	defer o.mu.Unlock()

	runCtx, cancel := context.WithCancel(ctx)
	o.activeCancel = cancel
	defer func() {
		cancel()
		o.activeCancel = nil
	}()
	defer o.truncateHistory()

	stdout, exitCode := cli.Run([]string{"status"}, o.deps)
	if exitCode != 0 {
		stdout = "No active game session found. A new session must be created."
	}
	cli.PrintInfo("Orchestrator processing player input")

	systemPrompt := fmt.Sprintf("%s\n\nCurrent game state:\n%s", o.prompt, stdout)
	messages := append([]llmtypes.Message{}, o.history...)
	messages = append(messages, llmtypes.Message{Role: "system", Content: systemPrompt})
	messages = append(messages, llmtypes.Message{Role: "user", Content: input})

	for i := 0; i < 300; i++ {
		response, err := o.provider.GenerateWithTools(runCtx, messages, o.tools)
		if err != nil {
			return "", err
		}

		if response.FinishReason == "tool_calls" {
			if err := o.executeSubagentToolCalls(runCtx, &messages, response.ToolCalls, response.Thinking, reporter); err != nil {
				return "", err
			}
			continue
		}

		if response.FinishReason == "stop" {
			if response.Content == "" {
				// Handle edge case where model stops without content (Anthropic sometimes does this)
				return "The DM remains silent.", nil
			}
			finalResp := response.Content
			if reporter != nil {
				streamedResp, streamedContent, err := o.streamFinalResponse(runCtx, messages, reporter)
				if err != nil {
					return "", err
				}
				if streamedResp.FinishReason == "tool_calls" {
					if err := o.executeSubagentToolCalls(runCtx, &messages, streamedResp.ToolCalls, streamedResp.Thinking, reporter); err != nil {
						return "", err
					}
					continue
				}
				if strings.TrimSpace(streamedContent) != "" {
					finalResp = streamedContent
				} else {
					reporter.Chunk(finalResp)
				}
			}

			finalResp = o.handleTextMarkers(finalResp)
			finalResp = o.filterThinkingTags(finalResp)
			o.history = append(o.history, llmtypes.Message{Role: "user", Content: input})
			o.history = append(o.history, llmtypes.Message{Role: "model", Content: finalResp, Thinking: response.Thinking})
			slog.Info("ORCHESTRATOR_END", "response", finalResp)

			// Sync to Python sandbox
			o.syncToSandbox()

			return finalResp, nil
		}

		// Safety break: if we get here without a known finish reason, stop to avoid spin loops
		if response.FinishReason == "" && response.Content == "" && len(response.ToolCalls) == 0 {
			return "The DM is lost in thought.", nil
		}
	}

	return "", fmt.Errorf("max orchestrator iterations exceeded")
}

func (o *Orchestrator) syncToSandbox() {
	if o.pythonRepl == nil {
		return
	}
	historyJSON, _ := json.Marshal(o.history)
	code := fmt.Sprintf("message_history = json.loads(%q)", string(historyJSON))
	o.pythonRepl.Execute(code)
}

func (o *Orchestrator) executeSubagentToolCalls(ctx context.Context, messages *[]llmtypes.Message, calls []llmtypes.ToolCall, thinking string, reporter StreamReporter) error {
	*messages = append(*messages, llmtypes.Message{Role: "model", ToolCalls: calls, Thinking: thinking})
	for _, call := range calls {
		if call.Name == "get_vd_manual" {
			cli.PrintInfo("[Orchestrator] Reading VD manual...")
			content, err := o.getVDManual()
			if err != nil {
				content = fmt.Sprintf("Error: %v", err)
			}
			*messages = append(*messages, llmtypes.Message{
				Role:       "tool",
				Name:       call.Name,
				ToolCallID: call.ID,
				Content:    content,
			})
			continue
		}

		if call.Name == "execute_python" {
			cli.PrintSubagentInvocation(call.Name, call.Arguments)
			var args map[string]any
			if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
				return fmt.Errorf("error parsing arguments: %w", err)
			}
			code, _ := args["code"].(string)
			
			// INTERCEPT: Catch common mistakes early
			if misuse := o.detectPythonMisuse(code); misuse != "" {
				cli.PrintWarning(fmt.Sprintf("Intercepted Python Misuse: %s", code))
				*messages = append(*messages, llmtypes.Message{
					Role:       "tool",
					Name:       call.Name,
					ToolCallID: call.ID,
					Content:    misuse,
				})
				continue
			}

			fullCode := o.injectSandboxState() + "\n" + code
			result, err := o.pythonRepl.Execute(fullCode)
			content := ""
			if err != nil {
				content = fmt.Sprintf("System Error: %v", err)
			} else if result.Error != "" {
				// Log raw error for debugging to vdm.log
				slog.Error("PYTHON_EXEC_ERROR", "error", result.Error, "code", code)

				// Sanitize the error for both the CLI and the LLM
				sanitizedError := o.pruneOutput(result.Error)
				cli.PrintError(fmt.Sprintf("Orchestrator Python Error: %s", sanitizedError))
				
				// Add helpful hints to the LLM to prevent loops
				hint := ""
				if strings.Contains(result.Error, "import ") {
					hint = "\n\nHINT: Imports are forbidden. Use pre-injected globals (list_dir, ripgrep, open, etc.) directly."
				} else if strings.Contains(result.Error, "PermissionError") {
					hint = "\n\nHINT: You only have Write access to the 'sandbox/' directory. 'rules/' is Read-Only."
				} else if strings.Contains(result.Error, "NameError") && strings.Contains(result.Error, "os") {
					hint = "\n\nHINT: Use 'list_dir()' instead of 'os.listdir()'."
				}
				
				content = "Python Error: " + sanitizedError + hint
			} else {
				content = o.pruneOutput(result.Stdout)
			}
			if content == "" {
				content = "(Success: no output)"
			}
			*messages = append(*messages, llmtypes.Message{
				Role:       "tool",
				Name:       call.Name,
				ToolCallID: call.ID,
				Content:    content,
			})
			continue
		}

		agent, ok := o.subagents[call.Name]
		if !ok {
			cli.PrintWarning(fmt.Sprintf("Unknown subagent requested: %s", call.Name))
			*messages = append(*messages, llmtypes.Message{
				Role:       "tool",
				Name:       call.Name,
				ToolCallID: call.ID,
				Content:    fmt.Sprintf("Error: Unknown subagent %s", call.Name),
			})
			continue
		}

		agentInput, err := toolInputForSubagent(call)
		if err != nil {
			cli.PrintWarning(fmt.Sprintf("Invalid subagent arguments for %s: %v", call.Name, err))
			*messages = append(*messages, llmtypes.Message{
				Role:       "tool",
				Name:       call.Name,
				ToolCallID: call.ID,
				Content:    fmt.Sprintf("Error: %v", err),
			})
			continue
		}

		cli.PrintSubagentInvocation(call.Name, agentInput)
		if reporter != nil {
			reporter.Status(statusMessageForTool(call.Name))
		}

		slog.Info("SUBAGENT_CALL", "name", call.Name, "input", agentInput)
		result, err := agent.Run(ctx, agentInput, nil)
		if err != nil {
			slog.Error("SUBAGENT_FAILED", "name", call.Name, "error", err)
			cli.PrintWarning(fmt.Sprintf("Subagent %s failed: %v", call.Name, err))
			result = fmt.Sprintf("Error: %v", err)
		} else {
			slog.Debug("SUBAGENT_RESULT", "name", call.Name, "output", result)
		}

		*messages = append(*messages, llmtypes.Message{
			Role:       "tool",
			Name:       call.Name,
			ToolCallID: call.ID,
			Content:    result,
		})
	}
	return nil
}

func (o *Orchestrator) streamFinalResponse(ctx context.Context, messages []llmtypes.Message, reporter StreamReporter) (llmtypes.GenerationResponse, string, error) {
	var streamed strings.Builder
	resp, err := o.provider.GenerateStream(ctx, messages, o.tools, func(chunk string) error {
		streamed.WriteString(chunk)
		reporter.Chunk(chunk)
		return nil
	})
	if err != nil {
		return llmtypes.GenerationResponse{}, "", err
	}

	if streamed.Len() == 0 && strings.TrimSpace(resp.Content) != "" {
		reporter.Chunk(resp.Content)
		streamed.WriteString(resp.Content)
	}

	return resp, streamed.String(), nil
}

func statusMessageForTool(name string) string {
	switch name {
	case "execute_python":
		return "*(The DM is interacting with the sandbox...)*\n"
	case "call_research_assistant":
		return "*(The DM is checking the rules...)*\n"
	case "call_vdm_execution":
		return "*(The DM is resolving the action...)*\n"
	default:
		return "*(The DM is delegating work...)*\n"
	}
}

func toolInputForSubagent(call llmtypes.ToolCall) (string, error) {
	if strings.TrimSpace(call.Arguments) == "" {
		return "", fmt.Errorf("missing tool arguments")
	}

	var args map[string]any
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		return "", fmt.Errorf("invalid tool arguments: %w", err)
	}

	for _, key := range []string{"query", "instruction", "code"} {
		if v, ok := args[key].(string); ok && strings.TrimSpace(v) != "" {
			return v, nil
		}
	}
	for _, raw := range args {
		if v, ok := raw.(string); ok && strings.TrimSpace(v) != "" {
			return v, nil
		}
	}

	return "", fmt.Errorf("no string input argument found")
}

// Interrupt cancels any active processing in the orchestrator.
// Returns true if an active process was cancelled.
func (o *Orchestrator) Interrupt() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.activeCancel != nil {
		o.activeCancel()
		o.activeCancel = nil
		return true
	}
	return false
}

func (o *Orchestrator) handleTextMarkers(content string) string {
	lines := strings.Split(content, "\n")
	var newLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, ">VD_SUGGEST_CMD ") {
			cmdLine := strings.TrimPrefix(line, ">VD_SUGGEST_CMD ")
			stdout, _ := cli.Run(strings.Fields(cmdLine), o.deps)
			newLines = append(newLines, "\n**Command Executed:** `"+cmdLine+"`")
			newLines = append(newLines, stdout)
		} else {
			newLines = append(newLines, line)
		}
	}
	return strings.Join(newLines, "\n")
}

// Close cleans up the LLM client.
func (o *Orchestrator) Close() {
	if o.pythonRepl != nil {
		o.pythonRepl.Close()
	}
	if gp, ok := o.provider.(*GeminiProvider); ok {
		gp.Close()
	}
}

// SaveHistory saves the conversation history to a JSON file.
func (o *Orchestrator) SaveHistory(path string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	data, err := json.MarshalIndent(o.history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	// Save full history to sandbox
	o.saveSandboxHistory()

	return nil
}

func (o *Orchestrator) saveSandboxHistory() {
	if o.pythonRepl == nil {
		return
	}
	cli.PrintInfo("  [VDM] Flushing Deep History to sandbox disk...")
	// We tell Python to save its internal message_history directly to the file.
	// This is much faster than pulling the JSON through a pipe.
	saveCode := `
import json
with open('sandbox/message_history.json', 'w') as f:
    json.dump(message_history, f)
`
	result, err := o.pythonRepl.Execute(saveCode)
	if err != nil || result == nil || result.Error != "" {
		slog.Warn("failed to save message_history via python", "error", err)
		return
	}
	cli.PrintInfo("  [VDM] Deep History sync complete.")
}

// LoadHistory loads the conversation history from a JSON file.
func (o *Orchestrator) LoadHistory(path string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read history file: %w", err)
	}

	var history []llmtypes.Message
	if err := json.Unmarshal(data, &history); err != nil {
		return fmt.Errorf("failed to unmarshal history: %w", err)
	}

	o.history = history

	// Also load into Python sandbox
	o.loadSandboxHistory()

	return nil
}

func (o *Orchestrator) loadSandboxHistory() {
	if o.pythonRepl == nil {
		return
	}
	sandboxHistoryPath := filepath.Join(filepath.Dir(o.scriptPath), "..", "sandbox", "message_history.json")
	if _, err := os.Stat(sandboxHistoryPath); os.IsNotExist(err) {
		return // No history file yet
	}
	data, err := os.ReadFile(sandboxHistoryPath)
	if err != nil {
		slog.Warn("failed to read sandbox history", "error", err)
		return
	}
	cli.PrintInfo("  [VDM] Injecting Deep History into sandbox...")
	code := fmt.Sprintf("message_history = json.loads(%q)", string(data))
	result, err := o.pythonRepl.Execute(code)
	if err != nil || result == nil || result.Error != "" {
		slog.Warn("failed to inject history into python", "error", err)
		return
	}
	cli.PrintInfo("  [VDM] Deep History loaded.")
}

func (o *Orchestrator) truncateHistory() {
	const maxSizeBytes = 20 * 1024
	if len(o.history) == 0 {
		return
	}

	totalSize := 0
	keepIdx := 0
	for i := len(o.history) - 1; i >= 0; i-- {
		b, _ := json.Marshal(o.history[i])
		if totalSize+len(b) > maxSizeBytes {
			keepIdx = i + 1
			break
		}
		totalSize += len(b)
	}

	if keepIdx > 0 {
		o.history = o.history[keepIdx:]
	}
}

func (o *Orchestrator) filterThinkingTags(content string) string {
	re := regexp.MustCompile(`(?s)<think>.*?</think>`)
	content = re.ReplaceAllString(content, "")
	
	// Minimax often puts its plan before a horizontal rule or double newline.
	if strings.Contains(content, "---") {
		parts := strings.SplitN(content, "---", 2)
		// If the first part looks like a plan (short, lists, or meta-talk), take the second part.
		first := strings.ToLower(parts[0])
		isPlan := len(parts[0]) < 800 && (strings.Contains(first, "1.") || 
			strings.Contains(first, "* ") || 
			strings.Contains(first, "- ") || 
			strings.Contains(first, "let me") || 
			strings.Contains(first, "i will") ||
			strings.Contains(first, "okay,"))
		
		if isPlan {
			content = parts[1]
		}
	}

	return strings.TrimSpace(content)
}

func (o *Orchestrator) detectPythonMisuse(code string) string {
	if strings.Contains(code, "import os") {
		return "Python Error: 'import os' is forbidden. Use the pre-injected global 'list_dir(\"sandbox\")' to see files."
	}
	if strings.Contains(code, "list_dir(\"/\")") || strings.Contains(code, "list_dir('/')") {
		return "Python Error: Access to root '/' is forbidden. You only have access to 'sandbox/'. Use 'list_dir(\"sandbox\")'."
	}
	if strings.Contains(code, "list_dir(\".\")") || strings.Contains(code, "list_dir('.')") {
		return "Python Error: Current directory listing is restricted. Use 'list_dir(\"sandbox\")' to see your campaign files."
	}
	return ""
}

func (o *Orchestrator) pruneOutput(output string) string {
	// 1. Mask absolute paths
	wd, _ := os.Getwd()
	if wd != "" {
		output = strings.ReplaceAll(output, wd, "[PROJECT_ROOT]")
	}

	// 2. Strip sandbox internal trace noise
	// RestrictedPython tracebacks often include lines like 'File "<sandbox>", line 1, in <module>'
	// We want to keep the error type and message but minimize the file stack noise.
	output = strings.ReplaceAll(output, "File \"<sandbox>\", ", "")

	// 3. Truncate by lines
	lines := strings.Split(output, "\n")
	const maxLines = 50
	if len(lines) > maxLines {
		half := maxLines / 2
		output = strings.Join(lines[:half], "\n") +
			"\n\n[... Output truncated for brevity/safety ...]\n\n" +
			strings.Join(lines[len(lines)-half:], "\n")
	}

	// 4. Final safety length check (roughly 4KB max)
	const maxChars = 4096
	if len(output) > maxChars {
		output = output[:maxChars] + " (truncated...)"
	}

	return output
}
