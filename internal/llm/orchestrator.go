package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm/ripgrep"
	"uaa/vdnd/internal/llm/vdhelpers"
)

// ErrOrchestratorBusy is returned when the orchestrator is already processing a request.
var ErrOrchestratorBusy = fmt.Errorf("the DM is currently busy thinking... please wait")

const defaultModel = "gemini-2.0-flash-exp"

type RLMCompleter interface {
	Complete(ctx context.Context, query string, contextData string, history []Message) (string, string, error)
}

// Orchestrator coordinates communication between the user, the LLM, and the rules engine.
type Orchestrator struct {
	mu           sync.Mutex
	activeCancel context.CancelFunc
	provider     Provider
	rlm          RLMCompleter
	deps         cli.Deps
	tools        []Tool
	history      []Message
	promptMode   bool
}

// NewOrchestrator creates a new LLM orchestrator.
func NewOrchestrator(context context.Context, provider Provider, deps cli.Deps) *Orchestrator {
	promptMode := !provider.SupportsToolCalling()

	o := &Orchestrator{
		provider:   provider,
		deps:       deps,
		tools:      defineTools(),
		promptMode: promptMode,
	}

	o.history = []Message{
		{Role: "model", Content: "I am ready to be your Dungeon Master. What happens next?"},
	}

	return o
}

func (o *Orchestrator) SetRLM(rlm RLMCompleter) {
	o.rlm = rlm
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

func defineTools() []Tool {
	return []Tool{
		{
			Name:        "vd_scene_new",
			Description: "Create a new combat scene",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string", "description": "Name of the scene"},
				},
				"required": []any{"name"},
			},
		},
		{
			Name:        "vd_scene_save",
			Description: "Save the current scene state",
		},
		{
			Name:        "vd_scene_load",
			Description: "Load an existing scene",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string", "description": "Name of the scene to load"},
				},
				"required": []any{"name"},
			},
		},
		{
			Name:        "vd_status",
			Description: "Get the current scene status and entity list",
		},
		// ----- NEW TOOLS -----
		// A) Generic vd tool
		{
			Name:        "vd",
			Description: "Execute any VD CLI command as a raw string.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cmd": map[string]any{
						"type":        "string",
						"description": "Full command string (e.g., 'action strike hero goblin --weapon sword')",
					},
				},
				"required": []any{"cmd"},
			},
		},

		// B) Top-5 structured tools
		{
			Name:        "vd_action_strike",
			Description: "Perform a melee or ranged attack.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"actor":  map[string]any{"type": "string", "description": "Attacker entity ID"},
					"target": map[string]any{"type": "string", "description": "Target entity ID"},
					"weapon": map[string]any{"type": "string", "description": "Weapon ID (optional)"},
					"map":    map[string]any{"type": "integer", "description": "Multi-Attack Penalty (0=None,1=-5,2=-10)", "enum": []int{0, 1, 2}},
				},
				"required": []any{"actor", "target"},
			},
		},
		{
			Name:        "vd_damage",
			Description: "Apply damage to an entity, respecting immunities/weaknesses/resistances.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "string", "description": "Target entity ID"},
					"amount": map[string]any{"type": "integer", "description": "Damage amount"},
					"type":   map[string]any{"type": "string", "description": "Damage type (optional)"},
				},
				"required": []any{"id", "amount"},
			},
		},
		{
			Name:        "vd_heal",
			Description: "Restore HP to an entity.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "string", "description": "Target entity ID"},
					"amount": map[string]any{"type": "integer", "description": "Healing amount"},
				},
				"required": []any{"id", "amount"},
			},
		},
		{
			Name:        "vd_condition_add",
			Description: "Apply a condition to an entity.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":        map[string]any{"type": "string", "description": "Target entity ID"},
					"condition": map[string]any{"type": "string", "description": "Condition name (e.g., 'frightened', 'prone')"},
					"value":     map[string]any{"type": "integer", "description": "Condition value (optional)"},
					"duration":  map[string]any{"type": "integer", "description": "Duration in rounds (optional)"},
					"source":    map[string]any{"type": "string", "description": "Source of condition (optional)"},
				},
				"required": []any{"id", "condition"},
			},
		},

		{
			Name:        "vd_action_stride",
			Description: "Move an entity to a new zone, potentially triggering reactions.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"actor": map[string]any{"type": "string", "description": "Entity ID to move"},
					"to":    map[string]any{"type": "string", "description": "Target zone ID"},
				},
				"required": []any{"actor", "to"},
			},
		},
		// C) Manual tool
		{
			Name:        "vd_manual",
			Description: "Retrieve the full VD CLI manual (vd_manual.md).",
			Parameters:  nil, // no arguments
		},

		// D) Ripgrep tool
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

	if o.rlm != nil {
		resp, thinking, err := o.rlm.Complete(runCtx, input, stdout, o.history)
		if err != nil {
			return "", err
		}
		slog.Info("RLM_END", "response", resp)

		// Process markers and tools
		processedResp := o.handleTextMarkers(resp)

		// Include user input as a quote if desired (though orchestrator handles this in Loop,
		// for RLM we should check if we want it here too. For now let's just use finalResp)
		finalResp := processedResp

		// We still append to history for the simple state tracking
		o.history = append(o.history, Message{Role: "user", Content: input})
		o.history = append(o.history, Message{Role: "model", Content: finalResp, Thinking: thinking})

		return finalResp, nil
	}

	contextInput := fmt.Sprintf("Current Game State:\n%s\n\nUser Input: %s", stdout, input)

	o.history = append(o.history, Message{Role: "user", Content: contextInput})

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
		var resp GenerationResponse
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

			o.history = append(o.history, Message{Role: "model", Content: resp.Content, Thinking: resp.Thinking})
			return resp.Content, nil
		}

		if resp.FinishReason == "tool_calls" {
			slog.Info("LLM_TOOL_CHOICE",
				"tools", resp.ToolCalls,
				"thinking", resp.Thinking,
			)
			// Add assistant tool calls to history
			o.history = append(o.history, Message{Role: "model", ToolCalls: resp.ToolCalls, Thinking: resp.Thinking})

			for _, call := range resp.ToolCalls {
				// Execute tool
				result := o.executeTool(call)

				// Add tool result to history
				o.history = append(o.history, Message{
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

func (o *Orchestrator) parseJSONResponse(content string) GenerationResponse {
	// Simple JSON extraction (finding first { and last })
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 || end <= start {
		return GenerationResponse{Content: content, FinishReason: "stop"}
	}

	jsonStr := content[start : end+1]
	var data struct {
		Tool      string         `json:"tool"`
		Arguments map[string]any `json:"arguments"`
		Narration string         `json:"narration"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return GenerationResponse{Content: content, FinishReason: "stop"}
	}

	if data.Tool == "" {
		return GenerationResponse{Content: data.Narration, FinishReason: "stop"}
	}

	args, _ := json.Marshal(data.Arguments)
	return GenerationResponse{
		Content: data.Narration,
		ToolCalls: []ToolCall{
			{
				Name:      data.Tool,
				Arguments: string(args),
			},
		},
		FinishReason: "tool_calls",
	}
}

func (o *Orchestrator) executeTool(call ToolCall) string {
	start := time.Now()
	var stdout string
	var exitCode int
	var cmdArgs []string

	// Special-case tools that don't go through mapToolToArgs
	switch call.Name {
	case "vd":
		var args map[string]any
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return errorJSON(fmt.Sprintf("Error parsing arguments: %v", err))
		}
		cmd, _ := args["cmd"].(string)
		if cmd == "" {
			return errorJSON("Missing 'cmd' field")
		}
		// Use the Phase-0 helper
		res := vdhelpers.ExecuteGenericVD(cmd, o.deps)
		var vdr vdhelpers.VDResult
		json.Unmarshal([]byte(res), &vdr)
		stdout = vdr.Stdout
		exitCode = vdr.ExitCode
		goto LOG

	case "vd_manual":
		// Read vd_manual.md from the project root
		content, err := os.ReadFile("vd_manual.md")
		if err != nil {
			return errorJSON(fmt.Sprintf("Could not read vd_manual.md: %v", err))
		}
		stdout = string(content)
		exitCode = 0
		goto LOG

	case "ripgrep":
		var args map[string]any
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return errorJSON(fmt.Sprintf("Error parsing arguments: %v", err))
		}
		pattern, _ := args["pattern"].(string)
		path, _ := args["path"].(string)
		if pattern == "" {
			return errorJSON("Missing 'pattern' field")
		}
		result, err := ripgrep.Search(pattern, path)
		if err != nil {
			return errorJSON(fmt.Sprintf("Ripgrep search failed: %v", err))
		}
		// ripgrep.Search already returns JSON-ready string via ToJSON()
		res := result.ToJSON()
		var resMap map[string]any
		json.Unmarshal([]byte(res), &resMap)
		slog.Info("RIPGREP_RESULTS", "count", len(res))
		stdout = res // For logging
		exitCode = 0
		goto LOG
	}

	// Structured tools (including the original 4 scene tools)
	cmdArgs = o.mapToolToArgs(call)
	if cmdArgs == nil {
		return errorJSON(fmt.Sprintf("Unknown tool or invalid arguments for %s", call.Name))
	}

	stdout, exitCode = cli.Run(cmdArgs, o.deps)

LOG:
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

func (o *Orchestrator) mapToolToArgs(call ToolCall) []string {
	var args map[string]any
	_ = json.Unmarshal([]byte(call.Arguments), &args)

	getString := func(key string) string {
		v, ok := args[key]
		if !ok || v == nil {
			return ""
		}
		s, _ := v.(string)
		return s
	}
	getInt := func(key string) int {
		v, ok := args[key]
		if !ok || v == nil {
			return 0
		}
		// JSON numbers decode as float64
		if f, ok := v.(float64); ok {
			return int(f)
		}
		return 0
	}

	switch call.Name {
	case "vd_scene_new":
		name := getString("name")
		if name == "" {
			name = "New Scene"
		}
		return []string{"scene", "new", name}
	case "vd_scene_save":
		return []string{"scene", "save"}
	case "vd_scene_load":
		name := getString("name")
		if name == "" {
			return nil
		}
		return []string{"scene", "load", name}
	case "vd_status":
		return []string{"status"}

	// ----- NEW STRUCTURED TOOLS -----
	case "vd_action_strike":
		actor := getString("actor")
		target := getString("target")
		weapon := getString("weapon")
		mapVal := getInt("map")
		if actor == "" || target == "" {
			return nil
		}
		argv := []string{"action", "strike", actor, target}
		if weapon != "" {
			argv = append(argv, "--weapon", weapon)
		}
		if mapVal > 0 && mapVal <= 2 {
			argv = append(argv, "--map", strconv.Itoa(mapVal))
		}
		return argv

	case "vd_action_stride":
		actor := getString("actor")
		to := getString("to")
		if actor == "" || to == "" {
			return nil
		}
		return []string{"action", "stride", actor, "--to", to}

	case "vd_damage":
		id := getString("id")
		amount := getInt("amount")
		dmgType := getString("type")
		if id == "" || amount == 0 {
			return nil
		}
		argv := []string{"damage", id, strconv.Itoa(amount)}
		if dmgType != "" {
			argv = append(argv, dmgType)
		}
		return argv

	case "vd_heal":
		id := getString("id")
		amount := getInt("amount")
		if id == "" || amount == 0 {
			return nil
		}
		return []string{"heal", id, strconv.Itoa(amount)}

	case "vd_condition_add":
		id := getString("id")
		condition := getString("condition")
		value := getInt("value")
		duration := getInt("duration")
		source := getString("source")
		if id == "" || condition == "" {
			return nil
		}
		argv := []string{"condition", "add", id, condition}
		if value != 0 {
			argv = append(argv, strconv.Itoa(value))
		}
		if duration > 0 {
			argv = append(argv, "--duration", strconv.Itoa(duration))
		}
		if source != "" {
			argv = append(argv, "--source", source)
		}
		return argv

	default:
		return nil
	}
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
