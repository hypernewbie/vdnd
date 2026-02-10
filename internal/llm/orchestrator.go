package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm/llmtypes"
	"uaa/vdnd/internal/llm/rlm"
	"uaa/vdnd/internal/llm/vdengine"
	"uaa/vdnd/internal/llm/vdhelpers"
)

// ErrOrchestratorBusy is returned when the orchestrator is already processing a request.
var ErrOrchestratorBusy = fmt.Errorf("the DM is currently busy thinking... please wait")

type RLMCompleter interface {
	Complete(ctx context.Context, query string, contextData string, history []llmtypes.Message) (string, string, error)
}

// Orchestrator coordinates communication between the user, the LLM, and the rules engine.
type Orchestrator struct {
	mu           sync.Mutex
	activeCancel context.CancelFunc
	provider     llmtypes.Provider
	sandboxRLM   RLMCompleter
	vdRLM        RLMCompleter
	deps         cli.Deps
	engine       *vdengine.VDEngine
	tools        []llmtypes.Tool
	history      []llmtypes.Message
	promptMode   bool
}

var supervisorTools = []llmtypes.Tool{
	{
		Name:        "call_research_assistant",
		Description: "Call a specialized research agent to look up rules, calculate stats, or inspect the game state using Python. Use this when you are unsure about rules or need to check something before deciding.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "The question or research topic for the assistant."},
			},
			"required": []string{"query"},
		},
	},
	{
		Name:        "call_vdm_execution",
		Description: "Call the VDM Execution Engine to perform game state changes (attacks, damage, healing, scene management, etc.). Provide the research notes (if any) and the specific instruction.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"instruction":    map[string]any{"type": "string", "description": "The specific instruction for state modification (e.g. 'Hero attacks goblin')."},
				"research_notes": map[string]any{"type": "string", "description": "Relevant findings from the research assistant to guide execution."},
			},
			"required": []string{"instruction"},
		},
	},
	{
		Name:        "vd_status",
		Description: "Get the current game state (entities, HP, positions, etc.).",
	},
}

// NewOrchestrator creates a new LLM orchestrator.
func NewOrchestrator(context context.Context, provider llmtypes.Provider, deps cli.Deps) *Orchestrator {
	promptMode := !provider.SupportsToolCalling()

	engine := vdengine.New(deps)
	o := &Orchestrator{
		provider:   provider,
		deps:       deps,
		engine:     engine,
		tools:      engine.Tools(),
		promptMode: promptMode,
	}

	o.history = []llmtypes.Message{
		{Role: "model", Content: "I am ready to be your Dungeon Master. What happens next?"},
	}

	return o
}

func (o *Orchestrator) SetRLMs(sandbox, vd RLMCompleter) {
	o.sandboxRLM = sandbox
	o.vdRLM = vd
}

// ProviderInfo returns the provider name and model name.
func (o *Orchestrator) ProviderInfo() (string, string) {
	if o.provider == nil {
		return "none", "none"
	}
	return o.provider.Name(), o.provider.ModelName()
}

func isRLMNil(i RLMCompleter) bool {
	if i == nil {
		return true
	}
	// Detect typed nil (e.g., *rlm.RLM(nil) wrapped in interface)
	if r, ok := i.(*rlm.RLM); ok {
		return r == nil
	}
	return false
}

// ProcessInput takes natural language input and returns a narrated response.
func (o *Orchestrator) ProcessInput(ctx context.Context, input string) (string, error) {
	if !o.mu.TryLock() {
		return "", ErrOrchestratorBusy
	}
	defer o.mu.Unlock()

	// Create a cancelable context for this specific run
	runCtx, cancel := context.WithCancel(ctx)
	o.activeCancel = cancel
	defer func() {
		cancel()
		o.activeCancel = nil
	}()
	defer o.truncateHistory()

	// Include status context in the message content
	stdout, exitCode := cli.Run([]string{"status"}, o.deps)
	if exitCode != 0 {
		stdout = "No active game session found. A new session must be created."
	}

	if isRLMNil(o.sandboxRLM) {
		return "", fmt.Errorf("Sandbox RLM not initialized")
	}

	// 1. Call Sandbox RLM
	sandboxNotes, sandboxThinking, err := o.sandboxRLM.Complete(runCtx, input, stdout, o.history)
	if err != nil {
		slog.Error("Sandbox RLM failed", "error", err)
		sandboxNotes = fmt.Sprintf("(Sandbox failed: %v)", err)
	}
	slog.Info("SANDBOX_END", "notes", sandboxNotes)

	var finalResp string
	var thinking string

	if isRLMNil(o.vdRLM) {
		finalResp = sandboxNotes
		thinking = sandboxThinking
	} else {
		// 2. Combine sandbox notes with original query for VDLM
		vdQuery := fmt.Sprintf("Sandbox notes:\n%s\n\nOriginal request: %s", sandboxNotes, input)

		// 3. Call VDLM (this will execute VD tools via its own tool-calling loop)
		finalResp, thinking, err = o.vdRLM.Complete(runCtx, vdQuery, stdout, o.history)
		if err != nil {
			return "", err
		}
	}

	slog.Info("ORCHESTRATOR_END", "response", finalResp)

	// Process markers (for legacy support)
	finalResp = o.handleTextMarkers(finalResp)

	// Append to history
	o.history = append(o.history, llmtypes.Message{Role: "user", Content: input})
	o.history = append(o.history, llmtypes.Message{Role: "model", Content: finalResp, Thinking: thinking})

	return finalResp, nil
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

func (o *Orchestrator) executeTool(call llmtypes.ToolCall) string {
	start := time.Now()
	stdout, exitCode, cmdArgs, err := o.engine.ExecuteTool(call)
	if err != nil {
		return errorJSON(err.Error())
	}

	duration := time.Since(start)

	// Create a summary of the result (first 100 chars)
	resultSummary := stdout
	if len(resultSummary) > 100 {
		resultSummary = resultSummary[:100] + "..."
	}

	pName, pModel := o.ProviderInfo()

	slog.Info("TOOL_CALL",
		"tool", call.Name,
		"arguments", call.Arguments,
		"mapped_args", cmdArgs,
		"stdout_len", len(stdout),
		"result_summary", resultSummary,
		"exit_code", exitCode,
		"duration_ms", duration.Milliseconds(),
		"provider", pName,
		"model", pModel,
	)

	result := vdhelpers.VDResult{
		Stdout:   stdout,
		ExitCode: exitCode,
	}
	if exitCode != 0 && result.Error == "" {
		result.Error = "Command failed (non-zero exit)"
	}
	b, _ := json.Marshal(result)
	return string(b)
}

// Helper for error responses
func errorJSON(msg string) string {
	result := vdhelpers.VDResult{
		Stdout:   "",
		ExitCode: 1,
		Error:    msg,
	}
	b, _ := json.Marshal(result)
	return string(b)
}

// Close cleans up the LLM client.
func (o *Orchestrator) Close() {
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

	return nil
}

// LoadHistory loads the conversation history from a JSON file.
func (o *Orchestrator) LoadHistory(path string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No history to load
		}
		return fmt.Errorf("failed to read history file: %w", err)
	}

	var history []llmtypes.Message
	if err := json.Unmarshal(data, &history); err != nil {
		return fmt.Errorf("failed to unmarshal history: %w", err)
	}

	o.history = history
	return nil
}

func (o *Orchestrator) truncateHistory() {
	const maxSizeBytes = 10 * 1024
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
