package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidateToolCall finds a tool by name from the given tools and validates the tool call arguments.
func ValidateToolCall(tools []Tool, toolCall ToolCall) (map[string]any, error) {
	for _, t := range tools {
		if t.Name == toolCall.Name {
			return ValidateToolArguments(t, toolCall)
		}
	}
	return nil, fmt.Errorf("tool %q not found", toolCall.Name)
}

// ValidateToolArguments validates tool call arguments against the tool's JSON Schema parameters.
// This is a lightweight validation that checks required fields and basic types.
// For full JSON Schema validation, consider using a dedicated library.
func ValidateToolArguments(tool Tool, toolCall ToolCall) (map[string]any, error) {
	args := toolCall.Arguments
	if args == nil {
		args = make(map[string]any)
	}

	// Check required fields if specified in the schema
	required, _ := tool.Parameters["required"].([]any)
	var missing []string
	for _, req := range required {
		name, ok := req.(string)
		if !ok {
			continue
		}
		if _, exists := args[name]; !exists {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		argsJSON, _ := json.MarshalIndent(args, "", "  ")
		return nil, fmt.Errorf(
			"validation failed for tool %q:\n  - missing required fields: %s\n\nReceived arguments:\n%s",
			toolCall.Name, strings.Join(missing, ", "), string(argsJSON),
		)
	}

	return args, nil
}
