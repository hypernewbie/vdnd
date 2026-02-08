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
	"uaa/vdnd/internal/llm/vdengine"
	"uaa/vdnd/internal/llm/vdhelpers"
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
	engine       *vdengine.VDEngine
	tools        []llmtypes.Tool
	history      []llmtypes.Message
	promptMode   bool
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
Output immersive narration that incorporates the results of your tool calls, and ONLY the narration should be returned in the final response
(tools are for your internal use to determine the narration, but the user should not see raw tool output or your thought process, ONLY the Dungeon Master narration).

CRITICAL RULES:
1. **NEVER write Python code to simulate combat, damage, healing, or condition changes.** All mechanical changes must be performed via VD tools.
2. **Always check the current status** using 'vd_status' if you are unsure about the state.
3. **Use 'vd_scene_new'** to start a new game if one hasn't started.
4. **Use 'vd_manual'** to look up the full CLI reference when you need syntax.
5. **Use 'ripgrep'** to quickly search rule files for specific terms (faster than Python regex).
6. **For state changes**, call the appropriate structured tool (e.g., 'vd_action_strike', 'vd_damage', 'vd_heal', 'vd_condition_add').
7. **If a structured tool doesn't exist**, use the generic 'vd' tool with the exact CLI command string.

AVAILABLE TOOLS (call them directly):
- vd_scene_new, vd_scene_save, vd_scene_load – scene management
- vd_status – get current game state
- vd_action_strike – perform an attack (actor, target, weapon?, map?)
- vd_damage – apply damage (id, amount, type?)
- vd_heal – restore HP (id, amount)
- vd_condition_add – apply a condition (id, condition, value?, duration?, source?)
- vd – execute any VD CLI command as a raw string (use for commands not covered above)
- vd_manual – retrieve the full VD CLI manual
- ripgrep – fast text search in rule files (pattern, path?)

EXAMPLES:
1. Player: "The hero attacks the goblin."
   → Call vd_status to see current entities.
   → Call vd_action_strike with {"actor": "hero", "target": "goblin"}.
   → Narrate the result.

2. Player: "The wizard casts fireball on the room."
   → Call vd to execute: vd action cast wizard fireball --zone room_a --dc 22 --damage 6d6 --type fire --basic_save
   → Narrate the outcome.

3. Player: "How does grappling work?"
   → Call ripgrep with {"pattern": "grapple"} to search rule files.
   → Read the results, then provide an explanation.

LEGACY MARKER (still works but prefer tools):
If you want to suggest a command for the user to consider, you can use:
>VD_SUGGEST_CMD command here
But tool calling is preferred because it executes the command immediately.

NARRATION:
After each tool call, incorporate the tool's output into your immersive, storytelling narration.
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

	if o.researchRLM != nil && o.vdRLM != nil {
		// 1. Call Research RLM
		researchNotes, _, err := o.researchRLM.Complete(runCtx, input, stdout, o.history)
		if err != nil {
			slog.Error("Research RLM failed", "error", err)
			researchNotes = fmt.Sprintf("(Research failed: %v)", err)
		}
		slog.Info("RESEARCH_END", "notes", researchNotes)

		// 2. Combine research notes with original query for VDLM
		vdQuery := fmt.Sprintf("Research notes:\n%s\n\nOriginal request: %s", researchNotes, input)

		// 3. Call VDLM (this will execute VD tools via its own tool-calling loop)
		finalResp, thinking, err := o.vdRLM.Complete(runCtx, vdQuery, stdout, o.history)
		if err != nil {
			return "", err
		}
		slog.Info("VDLM_END", "response", finalResp)

		// Process markers (for legacy support)
		finalResp = o.handleTextMarkers(finalResp)

		// Append to history
		o.history = append(o.history, llmtypes.Message{Role: "user", Content: input})
		o.history = append(o.history, llmtypes.Message{Role: "model", Content: finalResp, Thinking: thinking})

		return finalResp, nil
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
				result := o.executeTool(call)

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
	sb.WriteString("You are the Virtual Dungeon Master (VDM) for a Pathfinder 2nd Edition game.\n")
	sb.WriteString("Your goal is to narrate the game and use tools to manage rules.\n\n")
	sb.WriteString("REASONING:\n")
	sb.WriteString("Before responding, you should \"think\" inside <thought> tags.\n\n")
	sb.WriteString("CRITICAL: Do NOT write Python code to simulate combat, damage, healing, or condition changes.\n")
	sb.WriteString("All mechanical changes MUST be performed via VD tools listed below.\n")
	sb.WriteString("Python is only for rule lookup when ripgrep is unavailable.\n\n")
	sb.WriteString("You MUST respond in valid JSON format only when you need to call a tool.\n")
	sb.WriteString("JSON Schema for tool calls:\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"tool\": \"tool_name\",\n")
	sb.WriteString("  \"arguments\": { \"param1\": \"value1\" },\n")
	sb.WriteString("  \"narration\": \"Your immersive narration of what is happening\"\n")
	sb.WriteString("}\n\n")
	sb.WriteString("Available Tools:\n")
	for _, t := range o.tools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", t.Name, t.Description))
		if t.Parameters != nil {
			if props, ok := t.Parameters["properties"].(map[string]any); ok {
				sb.WriteString("  Parameters:\n")
				for k, v := range props {
					prop := v.(map[string]any)
					sb.WriteString(fmt.Sprintf("    - %s (%s): %s\n", k, prop["type"], prop["description"]))
				}
			}
		}
	}
	sb.WriteString("\nIf no tool is needed, respond with JSON where 'tool' is empty and 'narration' contains your response.\n")
	sb.WriteString("Do not include any text outside the JSON object.\n")
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