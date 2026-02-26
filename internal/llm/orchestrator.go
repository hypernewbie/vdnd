package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm/llmtypes"
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
	subagents    map[string]Subagent
	tools        []llmtypes.Tool
	history      []llmtypes.Message
}

// SetPrompt sets the system prompt for the orchestrator.
func (o *Orchestrator) SetPrompt(prompt string) {
	o.prompt = prompt
}

// NewOrchestrator creates a new LLM orchestrator.
func NewOrchestrator(context context.Context, provider llmtypes.Provider, deps cli.Deps) *Orchestrator {
	o := &Orchestrator{
		provider:  provider,
		deps:      deps,
		subagents: make(map[string]Subagent),
	}

	o.history = []llmtypes.Message{
		{Role: "model", Content: "I am ready to be your Dungeon Master. What happens next?"},
	}

	return o
}

// RegisterSubagents sets the available subagents as callable tools.
func (o *Orchestrator) RegisterSubagents(agents ...Subagent) {
	o.subagents = make(map[string]Subagent)
	o.tools = o.tools[:0]
	for _, agent := range agents {
		if agent == nil {
			continue
		}
		o.subagents[agent.Name()] = agent
		o.tools = append(o.tools, agent.ToolDefinition())
	}
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

	for i := 0; i < 12; i++ {
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

		if response.FinishReason == "stop" && response.Content != "" {
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
			o.history = append(o.history, llmtypes.Message{Role: "user", Content: input})
			o.history = append(o.history, llmtypes.Message{Role: "model", Content: finalResp, Thinking: response.Thinking})
			slog.Info("ORCHESTRATOR_END", "response", finalResp)
			return finalResp, nil
		}
	}

	return "", fmt.Errorf("max orchestrator iterations exceeded")
}

func (o *Orchestrator) executeSubagentToolCalls(ctx context.Context, messages *[]llmtypes.Message, calls []llmtypes.ToolCall, thinking string, reporter StreamReporter) error {
	*messages = append(*messages, llmtypes.Message{Role: "model", ToolCalls: calls, Thinking: thinking})
	for _, call := range calls {
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
			cli.PrintWarning(fmt.Sprintf("Subagent %s failed: %v", call.Name, err))
			result = fmt.Sprintf("Error: %v", err)
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
			return nil
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
