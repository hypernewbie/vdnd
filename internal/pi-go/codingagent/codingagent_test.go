package codingagent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Path Utils Tests ---

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // partial match (since ~ depends on env)
	}{
		{"plain path", "foo/bar", "foo/bar"},
		{"at prefix", "@foo/bar", "foo/bar"},
		{"unicode spaces", "foo\u00A0bar", "foo bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandPath(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("ExpandPath(%q) = %q, want contains %q", tt.input, result, tt.contains)
			}
		})
	}
}

func TestResolveToCwd(t *testing.T) {
	result := ResolveToCwd("foo.txt", "/home/user/project")
	if result != "/home/user/project/foo.txt" {
		t.Errorf("ResolveToCwd = %q, want /home/user/project/foo.txt", result)
	}

	result = ResolveToCwd("/absolute/path.txt", "/home/user/project")
	if result != "/absolute/path.txt" {
		t.Errorf("ResolveToCwd absolute = %q, want /absolute/path.txt", result)
	}
}

// --- Truncation Tests ---

func TestTruncateHead(t *testing.T) {
	content := strings.Repeat("line\n", 100)
	result := TruncateHead(content, TruncationOptions{MaxLines: 10, MaxBytes: 1024 * 1024})

	if !result.Truncated {
		t.Error("expected truncation")
	}
	if result.OutputLines != 10 {
		t.Errorf("expected 10 output lines, got %d", result.OutputLines)
	}
	if result.TruncatedBy != "lines" {
		t.Errorf("expected truncated by lines, got %q", result.TruncatedBy)
	}
}

func TestTruncateHead_NoTruncation(t *testing.T) {
	content := "short content"
	result := TruncateHead(content, TruncationOptions{})
	if result.Truncated {
		t.Error("expected no truncation")
	}
	if result.Content != content {
		t.Error("expected content unchanged")
	}
}

func TestTruncateTail(t *testing.T) {
	content := strings.Repeat("line\n", 100)
	result := TruncateTail(content, TruncationOptions{MaxLines: 10, MaxBytes: 1024 * 1024})

	if !result.Truncated {
		t.Error("expected truncation")
	}
	if result.OutputLines != 10 {
		t.Errorf("expected 10 output lines, got %d", result.OutputLines)
	}
}

func TestTruncateLine(t *testing.T) {
	short := "hello"
	text, truncated := TruncateLine(short, 10)
	if truncated {
		t.Error("short line should not be truncated")
	}
	if text != short {
		t.Errorf("expected %q, got %q", short, text)
	}

	long := strings.Repeat("x", 20)
	text, truncated = TruncateLine(long, 10)
	if !truncated {
		t.Error("long line should be truncated")
	}
	if !strings.Contains(text, "[truncated]") {
		t.Error("expected [truncated] suffix")
	}
}

func TestFormatSize(t *testing.T) {
	if FormatSize(500) != "500B" {
		t.Errorf("FormatSize(500) = %q", FormatSize(500))
	}
	result := FormatSize(2048)
	if !strings.Contains(result, "KB") {
		t.Errorf("FormatSize(2048) = %q, want KB", result)
	}
}

// --- Tool Tests ---

func TestCreateReadTool(t *testing.T) {
	// Create a temp file to read
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	os.WriteFile(testFile, []byte("hello world\nline 2\nline 3"), 0644)

	tool := CreateReadTool(dir, nil)

	if tool.Tool.Name != "read" {
		t.Errorf("expected tool name 'read', got %q", tool.Tool.Name)
	}

	result, err := tool.Execute(context.Background(), "id1", map[string]any{"path": "test.txt"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Content) == 0 {
		t.Fatal("expected content")
	}
	if !strings.Contains(result.Content[0].Text, "hello world") {
		t.Error("expected file content in result")
	}
}

func TestCreateWriteTool(t *testing.T) {
	dir := t.TempDir()
	tool := CreateWriteTool(dir, nil)

	_, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":    "new_file.txt",
		"content": "hello from write tool",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was written
	data, err := os.ReadFile(filepath.Join(dir, "new_file.txt"))
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if string(data) != "hello from write tool" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestCreateWriteTool_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	tool := CreateWriteTool(dir, nil)

	_, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":    "sub/dir/file.txt",
		"content": "nested",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "sub/dir/file.txt"))
	if string(data) != "nested" {
		t.Errorf("unexpected content: %q", string(data))
	}
}

func TestCreateEditTool(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	os.WriteFile(testFile, []byte("hello world\nfoo bar\nbaz"), 0644)

	tool := CreateEditTool(dir, nil)

	_, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":    "test.txt",
		"oldText": "foo bar",
		"newText": "FOO BAR",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(testFile)
	if !strings.Contains(string(data), "FOO BAR") {
		t.Error("expected replacement text in file")
	}
	if strings.Contains(string(data), "foo bar") {
		t.Error("old text should be replaced")
	}
}

func TestCreateEditTool_NotFound(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	os.WriteFile(testFile, []byte("hello world"), 0644)

	tool := CreateEditTool(dir, nil)

	_, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":    "test.txt",
		"oldText": "does not exist",
		"newText": "replacement",
	}, nil)
	if err == nil {
		t.Error("expected error for text not found")
	}
}

func TestCreateLsTool(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644)
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)

	tool := CreateLsTool(dir, nil)

	result, err := tool.Execute(context.Background(), "id1", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := result.Content[0].Text
	if !strings.Contains(text, "a.txt") {
		t.Error("expected a.txt in listing")
	}
	if !strings.Contains(text, "subdir/") {
		t.Error("expected subdir/ with trailing slash")
	}
}

// --- System Prompt Tests ---

func TestBuildSystemPrompt(t *testing.T) {
	prompt := BuildSystemPrompt(BuildSystemPromptOptions{
		Cwd: "/home/user/project",
	})

	if !strings.Contains(prompt, "expert coding assistant") {
		t.Error("expected default prompt text")
	}
	if !strings.Contains(prompt, "/home/user/project") {
		t.Error("expected cwd in prompt")
	}
	if !strings.Contains(prompt, "read") {
		t.Error("expected read tool in prompt")
	}
}

func TestBuildSystemPrompt_Custom(t *testing.T) {
	prompt := BuildSystemPrompt(BuildSystemPromptOptions{
		CustomPrompt: "You are a custom assistant.",
		Cwd:          "/tmp",
	})

	if !strings.Contains(prompt, "custom assistant") {
		t.Error("expected custom prompt text")
	}
	if !strings.Contains(prompt, "/tmp") {
		t.Error("expected cwd in prompt")
	}
}

func TestBuildSystemPrompt_WithContext(t *testing.T) {
	prompt := BuildSystemPrompt(BuildSystemPromptOptions{
		Cwd: "/tmp",
		ContextFiles: []ContextFile{
			{Path: "CONVENTIONS.md", Content: "Use tabs."},
		},
	})

	if !strings.Contains(prompt, "CONVENTIONS.md") {
		t.Error("expected context file path")
	}
	if !strings.Contains(prompt, "Use tabs.") {
		t.Error("expected context file content")
	}
}

func TestCreateCodingTools(t *testing.T) {
	tools := CreateCodingTools("/tmp")
	if len(tools) != 4 {
		t.Errorf("expected 4 coding tools, got %d", len(tools))
	}
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Tool.Name] = true
	}
	for _, name := range []string{"read", "bash", "edit", "write"} {
		if !names[name] {
			t.Errorf("expected tool %q in coding tools", name)
		}
	}
}

func TestCreateAllTools(t *testing.T) {
	tools := CreateAllTools("/tmp")
	if len(tools) != 7 {
		t.Errorf("expected 7 tools, got %d", len(tools))
	}
}
