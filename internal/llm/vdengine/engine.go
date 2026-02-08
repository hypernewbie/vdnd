package vdengine

import (
	"encoding/json"
	"fmt"
	"strconv"
	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm/llmtypes"
	"uaa/vdnd/internal/llm/ripgrep"
	"uaa/vdnd/internal/llm/vdhelpers"
)

// VDEngine handles VD tool execution.
type VDEngine struct {
	deps cli.Deps
}

// New creates a new VDEngine.
func New(deps cli.Deps) *VDEngine {
	return &VDEngine{deps: deps}
}

// Tools returns the list of VD tools.
func (e *VDEngine) Tools() []llmtypes.Tool {
	return []llmtypes.Tool{
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
			Name:        "vd_manual",
			Description: "Retrieve the full VD CLI manual (vd_manual.md).",
			Parameters:  nil,
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
}

// ExecuteTool handles a VD tool call.
func (e *VDEngine) ExecuteTool(call llmtypes.ToolCall) (stdout string, exitCode int, cmdArgs []string, err error) {
	switch call.Name {
	case "vd":
		var args map[string]any
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "", 1, nil, fmt.Errorf("error parsing arguments: %w", err)
		}
		cmd, _ := args["cmd"].(string)
		if cmd == "" {
			return "", 1, nil, fmt.Errorf("missing 'cmd' field")
		}
		res := vdhelpers.ExecuteGenericVD(cmd, e.deps)
		var vdr vdhelpers.VDResult
		json.Unmarshal([]byte(res), &vdr)
		return vdr.Stdout, vdr.ExitCode, nil, nil

	case "vd_manual":
		content, err := e.deps.Store.GetManual()
		if err != nil {
			// Fallback if GetManual not implemented on store
			return "", 1, nil, fmt.Errorf("could not read vd_manual.md: %w", err)
		}
		return content, 0, nil, nil

	case "ripgrep":
		var args map[string]any
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "", 1, nil, fmt.Errorf("error parsing arguments: %w", err)
		}
		pattern, _ := args["pattern"].(string)
		path, _ := args["path"].(string)
		if pattern == "" {
			return "", 1, nil, fmt.Errorf("missing 'pattern' field")
		}
		result, err := ripgrep.Search(pattern, path)
		if err != nil {
			return "", 1, nil, fmt.Errorf("ripgrep search failed: %w", err)
		}
		return result.ToJSON(), 0, nil, nil
	}

	cmdArgs = e.mapToolToArgs(call)
	if cmdArgs == nil {
		return "", 1, nil, fmt.Errorf("unknown tool or invalid arguments for %s", call.Name)
	}

	stdout, exitCode = cli.Run(cmdArgs, e.deps)
	return stdout, exitCode, cmdArgs, nil
}

func (e *VDEngine) mapToolToArgs(call llmtypes.ToolCall) []string {
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
