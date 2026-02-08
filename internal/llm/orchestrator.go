package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"uaa/vdnd/internal/cli"
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

RULES:
1. Always check the current status using 'vd_status' if you are unsure about the state.
2. Use 'vd_scene_new' to start a new game if one hasn't started.
3. Narrate results in an immersive, storytelling way.

COMMAND SUGGESTION FORMAT:
If you want to suggest a command for the user to consider, use this format:
>VD_SUGGEST_CMD command here

Example:
"You can try to hit the orc again."
>VD_SUGGEST_CMD action strike hero orc

The system will automatically recognize and execute these markers if they appear in your response.
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
	cmdArgs := o.mapToolToArgs(call)
	if cmdArgs == nil {
		return fmt.Sprintf("Error: Unknown tool %s", call.Name)
	}

	stdout, _ := cli.Run(cmdArgs, o.deps)

	// Wrap in a result map for Gemini
	res := map[string]any{"result": stdout}
	b, _ := json.Marshal(res)
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
