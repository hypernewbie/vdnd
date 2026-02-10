package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm/llmtypes"
)

// ErrOrchestratorBusy is returned when the orchestrator is already processing a request.
var ErrOrchestratorBusy = fmt.Errorf("the DM is currently busy thinking... please wait")

const defaultModel = "gemini-2.0-flash-exp"

type RLMCompleter interface {
	Complete(ctx context.Context, query string, contextData string, history []llmtypes.Message) (string, string, error)
}

// Orchestrator coordinates communication between the user, the LLM, and the rules engine.
type Orchestrator struct {
	mu           sync.Mutex
	activeCancel context.CancelFunc
	provider     llmtypes.Provider
	researchRLM  RLMCompleter
	vdRLM        RLMCompleter
	deps         cli.Deps
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

	o := &Orchestrator{
		provider:   provider,
		deps:       deps,
		tools:      supervisorTools,
		promptMode: promptMode,
	}

	o.history = []llmtypes.Message{
		{Role: "model", Content: "I am ready to be your Dungeon Master. What happens next?"},
	}

	return o
}

func (o *Orchestrator) SetRLMs(research, vd RLMCompleter) {
	o.researchRLM = research
	o.vdRLM = vd
}

// ProviderInfo returns the provider name and model name.
func (o *Orchestrator) ProviderInfo() (string, string) {
	if o.provider == nil {
		return "none", "none"
	}
	return o.provider.Name(), o.provider.ModelName()
}

func (o *Orchestrator) getSystemPrompt() string {
	if o.promptMode {
		return o.getSchemaPrompt()
	}
	return `You are the Virtual Dungeon Master (VDM) for a Pathfinder 2nd Edition game.
Your goal is to narrate the game and use the provided deterministic tools to manage the game rules.

AVAILABLE TOOLS:
- call_research_assistant: Your primary tool. Use this for general research, rule lookups, and checking context.
- call_vdm_execution: Use this for precise mathematics, complex combat rules, and changing the game state. VD SCENE MAY BE OUDATED, call_research_assistant IS ALWAYS GROUND TRUTH.
- vd_status: Check the current mechanical state of the game. NOTE: MAY BE OUTDATED. call_research_assistant IS GROUND TRUTH.

RULES:
1. **Narrate results** in an immersive, storytelling way.
2. **Research First:** Use the research assistant for most questions.
3. **Execute for Mechanics:** Use the execution engine for combat, math, and tricky rules.
   - **CRITICAL:** Ensure the VD scene is synced before executing mechanics. If the scene might be outdated (e.g. entities missing), verify with 'vd_status' or instruct the execution engine to "setup" the scene first.
   - Example: "Setup a scene with a Hero and Goblin, then calculate the attack roll."

You should only need call_research_assistant, not call_vdm_execution, most of the time, especially out of combat.
In combat, you will likely need to use call_vdm_execution for accurate mechanics, but always check if the scene is up to date first.
You are the storyteller. Use these tools to ensure your story respects the Pathfinder 2e rules.
Act as the narrator. do not output any other reply text besides the Virtual Dungeon Master narration.
`
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

	// Include status context in the message content as seen in previous implementation
	stdout, exitCode := cli.Run([]string{"status"}, o.deps)
	if exitCode != 0 {
		stdout = "No active game session found. A new session must be created."
	}

	if o.researchRLM == nil || o.vdRLM == nil {
		return "", fmt.Errorf("RLMs not initialized: ResearchRLM=%v, VDLM=%v", o.researchRLM, o.vdRLM)
	}

	contextInput := fmt.Sprintf("Current Game State:\n%s\n\nUser Input: %s", stdout, input)

	o.history = append(o.history, llmtypes.Message{Role: "user", Content: contextInput})

	return o.generationLoop(runCtx)
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

func (o *Orchestrator) generationLoop(ctx context.Context) (string, error) {
	for i := 0; i < 50; i++ { // Guard against infinite loops
		var resp llmtypes.GenerationResponse
		var err error

		// Log LLM Input (History)
		historyJSON, _ := json.MarshalIndent(o.history, "", "  ")
		slog.Debug("LLM_INPUT", "history", string(historyJSON))

		if o.promptMode {
			content, err := o.provider.Generate(ctx, o.history)
			if err != nil {
				return "", err
			}
			slog.Info("LLM_OUTPUT", "content", content)

			resp = o.parseJSONResponse(content)
		} else {
			resp, err = o.provider.GenerateWithTools(ctx, o.history, o.tools)
			if err != nil {
				return "", err
			}

			// For tool calling provider, we might want to log the structured response too
			respJSON, _ := json.MarshalIndent(resp, "", "  ")
			slog.Info("LLM_OUTPUT", "content", string(respJSON))
		}

		if resp.Thinking != "" {
			slog.Debug("LLM_THINKING", "thinking", resp.Thinking)
		}

		if resp.FinishReason == "stop" {
			resp.Content = o.handleTextMarkers(resp.Content)
			slog.Info("GENERATION_COMPLETE", "iterations", i+1)

			o.history = append(o.history, llmtypes.Message{Role: "model", Content: resp.Content, Thinking: resp.Thinking})
			return resp.Content, nil
		}

		if resp.FinishReason == "tool_calls" {
			slog.Info("LLM_TOOL_CHOICE",
				"tools", resp.ToolCalls,
				"thinking", resp.Thinking,
			)
			// Add assistant tool calls to history
			o.history = append(o.history, llmtypes.Message{Role: "model", ToolCalls: resp.ToolCalls, Thinking: resp.Thinking})

			for _, call := range resp.ToolCalls {
				// Execute tool
				result := o.executeTool(ctx, call)

				// Add tool result to history
				o.history = append(o.history, llmtypes.Message{
					Role:       "tool",
					Name:       call.Name,
					ToolCallID: call.ID,
					Content:    result,
				})
			}
			// Continue loop to get next response from model
			continue
		}
	}

	slog.Warn("GENERATION_FAILED", "iterations", 50)
	return "", fmt.Errorf("exceeded maximum tool calling iterations")
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

func (o *Orchestrator) getSchemaPrompt() string {
	var sb strings.Builder
	sb.WriteString("You are the Virtual Dungeon Master (VDM) Supervisor.\n")
	sb.WriteString("Manage the game by calling 'call_research_assistant' for rules and 'call_vdm_execution' for actions.\n")
	sb.WriteString("Respond in JSON.\n\n")
	sb.WriteString("Available Tools:\n")
	for _, t := range o.tools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", t.Name, t.Description))
	}
	return sb.String()
}

func (o *Orchestrator) parseJSONResponse(content string) llmtypes.GenerationResponse {
	// Simple JSON extraction (finding first { and last })
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 || end <= start {
		return llmtypes.GenerationResponse{Content: content, FinishReason: "stop"}
	}

	jsonStr := content[start : end+1]
	var data struct {
		Tool      string         `json:"tool"`
		Arguments map[string]any `json:"arguments"`
		Narration string         `json:"narration"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return llmtypes.GenerationResponse{Content: content, FinishReason: "stop"}
	}

	if data.Tool == "" {
		return llmtypes.GenerationResponse{Content: data.Narration, FinishReason: "stop"}
	}

	args, _ := json.Marshal(data.Arguments)
	return llmtypes.GenerationResponse{
		Content: data.Narration,
		ToolCalls: []llmtypes.ToolCall{
			{
				Name:      data.Tool,
				Arguments: string(args),
			},
		},
		FinishReason: "tool_calls",
	}
}

func (o *Orchestrator) executeTool(ctx context.Context, call llmtypes.ToolCall) string {
	start := time.Now()
	var stdout string

	// Default status context (the Orchestrator's view)
	// We might want to pass this to sub-agents or let them fetch it themselves.
	// Currently RLM.Complete takes `contextData` string.
	statusOut, _ := cli.Run([]string{"status"}, o.deps)

	switch call.Name {
	case "call_research_assistant":
		var args struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			stdout = fmt.Sprintf("Error parsing arguments: %v", err)
		} else {
			// Call Research RLM
			// We pass the current history? Or just empty history?
			// The sub-agent has its own prompt. Passing o.history might be too much noise if it contains raw tool calls.
			// But RLM.Complete signature expects history.
			// Let's pass o.history so it sees the conversation context.
			resp, _, rErr := o.researchRLM.Complete(ctx, args.Query, statusOut, o.history)
			if rErr != nil {
				stdout = fmt.Sprintf("Research failed: %v", rErr)
			} else {
				stdout = resp
			}
		}

	case "call_vdm_execution":
		var args struct {
			Instruction   string `json:"instruction"`
			ResearchNotes string `json:"research_notes"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			stdout = fmt.Sprintf("Error parsing arguments: %v", err)
		} else {
			// Construct query for VDLM
			vdQuery := fmt.Sprintf("Research notes:\n%s\n\nInstruction: %s", args.ResearchNotes, args.Instruction)
			resp, _, vErr := o.vdRLM.Complete(ctx, vdQuery, statusOut, o.history)
			if vErr != nil {
				stdout = fmt.Sprintf("Execution failed: %v", vErr)
			} else {
				stdout = resp
			}
		}

	case "vd_status":
		stdout = statusOut

	default:
		stdout = fmt.Sprintf("Unknown tool: %s", call.Name)
	}

	duration := time.Since(start)

	pName, pModel := o.ProviderInfo()

	slog.Info("SUPERVISOR_TOOL_CALL",
		"tool", call.Name,
		"arguments", call.Arguments,
		"stdout_len", len(stdout),
		"duration_ms", duration.Milliseconds(),
		"provider", pName,
		"model", pModel,
	)

	return stdout
}

// Close cleans up the LLM client.
func (o *Orchestrator) Close() {
	if gp, ok := o.provider.(*GeminiProvider); ok {
		gp.Close()
	}
}

func (o *Orchestrator) EnablePromptMode(enabled bool) {
	o.promptMode = enabled
	// System prompt is no longer stored in history at index 0.
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