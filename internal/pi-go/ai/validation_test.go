package ai

import (
	"testing"
)

func TestValidateToolCall_Found(t *testing.T) {
	tools := []Tool{
		{
			Name:        "read_file",
			Description: "Read a file",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string"},
				},
				"required": []any{"path"},
			},
		},
	}

	tc := ToolCall{
		Type: ContentTypeToolCall,
		ID:   "call_1",
		Name: "read_file",
		Arguments: map[string]any{
			"path": "/tmp/test.txt",
		},
	}

	args, err := ValidateToolCall(tools, tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args["path"] != "/tmp/test.txt" {
		t.Fatalf("expected path '/tmp/test.txt', got %v", args["path"])
	}
}

func TestValidateToolCall_NotFound(t *testing.T) {
	tools := []Tool{
		{Name: "other_tool", Parameters: map[string]any{}},
	}
	tc := ToolCall{Name: "missing_tool"}

	_, err := ValidateToolCall(tools, tc)
	if err == nil {
		t.Fatal("expected error for missing tool")
	}
}

func TestValidateToolArguments_MissingRequired(t *testing.T) {
	tool := Tool{
		Name: "write_file",
		Parameters: map[string]any{
			"type":     "object",
			"required": []any{"path", "content"},
		},
	}
	tc := ToolCall{
		Name:      "write_file",
		Arguments: map[string]any{"path": "/tmp/out.txt"},
	}

	_, err := ValidateToolArguments(tool, tc)
	if err == nil {
		t.Fatal("expected validation error for missing 'content'")
	}
}

func TestValidateToolArguments_NilArgs(t *testing.T) {
	tool := Tool{
		Name:       "no_args_tool",
		Parameters: map[string]any{"type": "object"},
	}
	tc := ToolCall{Name: "no_args_tool"}

	args, err := ValidateToolArguments(tool, tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args == nil {
		t.Fatal("expected non-nil args map")
	}
}
