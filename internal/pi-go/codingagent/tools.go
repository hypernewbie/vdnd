package codingagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"uaa/vdnd/internal/pi-go/agent"
	"uaa/vdnd/internal/pi-go/ai"
)

// --- Operations interfaces (pluggable backends for remote execution) ---

// FileOps provides file system operations. Override for remote execution (SSH, pods).
type FileOps struct {
	ReadFile  func(absolutePath string) ([]byte, error)
	WriteFile func(absolutePath string, content []byte) error
	Access    func(absolutePath string) error // Check file exists and is accessible
	Stat      func(absolutePath string) (os.FileInfo, error)
	ReadDir   func(absolutePath string) ([]os.DirEntry, error)
	MkdirAll  func(dir string) error
}

// DefaultFileOps returns file operations using the local filesystem.
func DefaultFileOps() FileOps {
	return FileOps{
		ReadFile:  os.ReadFile,
		WriteFile: func(path string, content []byte) error { return os.WriteFile(path, content, 0644) },
		Access: func(path string) error {
			_, err := os.Stat(path)
			return err
		},
		Stat:     os.Stat,
		ReadDir:  os.ReadDir,
		MkdirAll: func(dir string) error { return os.MkdirAll(dir, 0755) },
	}
}

// BashOps provides bash command execution. Override for remote execution.
type BashOps struct {
	Exec func(ctx context.Context, command string, cwd string) (stdout string, exitCode int, err error)
}

// DefaultBashOps returns bash operations using local shell.
func DefaultBashOps() BashOps {
	return BashOps{
		Exec: func(ctx context.Context, command string, cwd string) (string, int, error) {
			cmd := exec.CommandContext(ctx, "bash", "-c", command)
			cmd.Dir = cwd
			cmd.Env = os.Environ()

			var buf bytes.Buffer
			cmd.Stdout = &buf
			cmd.Stderr = &buf

			err := cmd.Run()
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					return "", -1, err
				}
			}
			return buf.String(), exitCode, nil
		},
	}
}

// --- Tool parameter schemas (JSON Schema for LLM) ---

func readSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":   map[string]any{"type": "string", "description": "Path to the file to read (relative or absolute)"},
			"offset": map[string]any{"type": "number", "description": "Line number to start reading from (1-indexed)"},
			"limit":  map[string]any{"type": "number", "description": "Maximum number of lines to read"},
		},
		"required": []any{"path"},
	}
}

func bashSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{"type": "string", "description": "Bash command to execute"},
			"timeout": map[string]any{"type": "number", "description": "Timeout in seconds (optional)"},
		},
		"required": []any{"command"},
	}
}

func editSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string", "description": "Path to the file to edit (relative or absolute)"},
			"oldText": map[string]any{"type": "string", "description": "Exact text to find and replace (must match exactly)"},
			"newText": map[string]any{"type": "string", "description": "New text to replace the old text with"},
		},
		"required": []any{"path", "oldText", "newText"},
	}
}

func writeSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string", "description": "Path to the file to write (relative or absolute)"},
			"content": map[string]any{"type": "string", "description": "Content to write to the file"},
		},
		"required": []any{"path", "content"},
	}
}

func grepSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern":    map[string]any{"type": "string", "description": "Search pattern (regex by default, literal with literal=true)"},
			"path":       map[string]any{"type": "string", "description": "Directory or file to search (default: current directory)"},
			"glob":       map[string]any{"type": "string", "description": "File glob pattern to filter (e.g., '*.go')"},
			"ignoreCase": map[string]any{"type": "boolean", "description": "Case-insensitive search"},
			"literal":    map[string]any{"type": "boolean", "description": "Treat pattern as literal string"},
			"context":    map[string]any{"type": "number", "description": "Lines of context around matches (default: 0)"},
			"limit":      map[string]any{"type": "number", "description": "Maximum matches (default: 100)"},
		},
		"required": []any{"pattern"},
	}
}

func findSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{"type": "string", "description": "Glob pattern to search for (e.g., '*.go', 'test_*')"},
			"path":    map[string]any{"type": "string", "description": "Directory to search in (default: current directory)"},
			"limit":   map[string]any{"type": "number", "description": "Maximum results (default: 1000)"},
		},
		"required": []any{"pattern"},
	}
}

func lsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":  map[string]any{"type": "string", "description": "Directory to list (default: current directory)"},
			"limit": map[string]any{"type": "number", "description": "Maximum entries (default: 500)"},
		},
	}
}

// --- Tool factory functions ---

// CreateReadTool creates a file reading tool for the given working directory.
func CreateReadTool(cwd string, ops *FileOps) agent.Tool {
	fileOps := DefaultFileOps()
	if ops != nil {
		fileOps = *ops
	}

	return agent.Tool{
		Tool: ai.Tool{
			Name:        "read",
			Description: "Read file contents. Returns text for text files, base64 for images. Use offset/limit for large files.",
			Parameters:  readSchema(),
		},
		Label: "read",
		Execute: func(ctx context.Context, _ string, args map[string]any, _ agent.ToolUpdateFunc) (agent.ToolResult, error) {
			path, _ := args["path"].(string)
			if path == "" {
				return agent.ToolResult{}, fmt.Errorf("path is required")
			}

			absolutePath := ResolveReadPath(path, cwd)

			if err := fileOps.Access(absolutePath); err != nil {
				return agent.ToolResult{}, fmt.Errorf("file not found: %s", path)
			}

			data, err := fileOps.ReadFile(absolutePath)
			if err != nil {
				return agent.ToolResult{}, fmt.Errorf("failed to read file: %w", err)
			}

			content := string(data)
			lines := strings.Split(content, "\n")

			// Apply offset
			offset := 0
			if v, ok := args["offset"].(float64); ok && int(v) > 0 {
				offset = int(v) - 1 // Convert to 0-indexed
				if offset >= len(lines) {
					return textResult(fmt.Sprintf("Offset %d exceeds file length (%d lines)", offset+1, len(lines))), nil
				}
				lines = lines[offset:]
			}

			// Apply limit
			if v, ok := args["limit"].(float64); ok && int(v) > 0 {
				limit := int(v)
				if limit < len(lines) {
					lines = lines[:limit]
				}
			}

			output := strings.Join(lines, "\n")

			// Truncate if needed
			result := TruncateHead(output, TruncationOptions{})
			if result.Truncated {
				output = result.Content
				output += fmt.Sprintf("\n\n[Truncated: showing %d/%d lines, %s/%s. Use offset=%d to see more]",
					result.OutputLines, result.TotalLines,
					FormatSize(result.OutputBytes), FormatSize(result.TotalBytes),
					offset+result.OutputLines+1)
			}

			return textResult(output), nil
		},
	}
}

// CreateBashTool creates a bash command execution tool.
func CreateBashTool(cwd string, ops *BashOps) agent.Tool {
	bashOps := DefaultBashOps()
	if ops != nil {
		bashOps = *ops
	}

	return agent.Tool{
		Tool: ai.Tool{
			Name:        "bash",
			Description: "Execute a bash command. Output is captured and returned. Long output is truncated (tail kept).",
			Parameters:  bashSchema(),
		},
		Label: "bash",
		Execute: func(ctx context.Context, _ string, args map[string]any, onUpdate agent.ToolUpdateFunc) (agent.ToolResult, error) {
			command, _ := args["command"].(string)
			if command == "" {
				return agent.ToolResult{}, fmt.Errorf("command is required")
			}

			stdout, exitCode, err := bashOps.Exec(ctx, command, cwd)
			if err != nil {
				return agent.ToolResult{}, fmt.Errorf("failed to execute command: %w", err)
			}

			// Truncate from tail (keep end of output for errors)
			result := TruncateTail(stdout, TruncationOptions{})

			var output string
			if result.Truncated {
				output = fmt.Sprintf("[Output truncated: showing last %d/%d lines, %s/%s]\n\n%s",
					result.OutputLines, result.TotalLines,
					FormatSize(result.OutputBytes), FormatSize(result.TotalBytes),
					result.Content)
			} else {
				output = result.Content
			}

			if exitCode != 0 {
				output += fmt.Sprintf("\n\n[Exit code: %d]", exitCode)
			}

			return textResult(output), nil
		},
	}
}

// CreateEditTool creates a surgical text editing tool.
func CreateEditTool(cwd string, ops *FileOps) agent.Tool {
	fileOps := DefaultFileOps()
	if ops != nil {
		fileOps = *ops
	}

	return agent.Tool{
		Tool: ai.Tool{
			Name:        "edit",
			Description: "Edit a file by replacing exact text. The oldText must match exactly (including whitespace). Use for precise, surgical edits.",
			Parameters:  editSchema(),
		},
		Label: "edit",
		Execute: func(ctx context.Context, _ string, args map[string]any, _ agent.ToolUpdateFunc) (agent.ToolResult, error) {
			path, _ := args["path"].(string)
			oldText, _ := args["oldText"].(string)
			newText, _ := args["newText"].(string)

			if path == "" {
				return agent.ToolResult{}, fmt.Errorf("path is required")
			}

			absolutePath := ResolveToCwd(path, cwd)

			if err := fileOps.Access(absolutePath); err != nil {
				return agent.ToolResult{}, fmt.Errorf("file not found: %s", path)
			}

			data, err := fileOps.ReadFile(absolutePath)
			if err != nil {
				return agent.ToolResult{}, fmt.Errorf("failed to read file: %w", err)
			}

			content := string(data)

			// Normalize line endings for matching
			normalizedContent := strings.ReplaceAll(content, "\r\n", "\n")
			normalizedOld := strings.ReplaceAll(oldText, "\r\n", "\n")
			normalizedNew := strings.ReplaceAll(newText, "\r\n", "\n")

			// Find exact match
			idx := strings.Index(normalizedContent, normalizedOld)
			if idx == -1 {
				return agent.ToolResult{}, fmt.Errorf("could not find the exact text in %s. The old text must match exactly including all whitespace and newlines", path)
			}

			// Check for multiple occurrences
			count := strings.Count(normalizedContent, normalizedOld)
			if count > 1 {
				return agent.ToolResult{}, fmt.Errorf("found %d occurrences of the text in %s. The text must be unique. Please provide more context", count, path)
			}

			// Perform replacement
			newContent := normalizedContent[:idx] + normalizedNew + normalizedContent[idx+len(normalizedOld):]

			if normalizedContent == newContent {
				return agent.ToolResult{}, fmt.Errorf("no changes made to %s. The replacement produced identical content", path)
			}

			if err := fileOps.WriteFile(absolutePath, []byte(newContent)); err != nil {
				return agent.ToolResult{}, fmt.Errorf("failed to write file: %w", err)
			}

			return textResult(fmt.Sprintf("Successfully replaced text in %s.", path)), nil
		},
	}
}

// CreateWriteTool creates a file writing tool.
func CreateWriteTool(cwd string, ops *FileOps) agent.Tool {
	fileOps := DefaultFileOps()
	if ops != nil {
		fileOps = *ops
	}

	return agent.Tool{
		Tool: ai.Tool{
			Name:        "write",
			Description: "Write content to a file. Creates the file if it doesn't exist, overwrites if it does. Automatically creates parent directories.",
			Parameters:  writeSchema(),
		},
		Label: "write",
		Execute: func(ctx context.Context, _ string, args map[string]any, _ agent.ToolUpdateFunc) (agent.ToolResult, error) {
			path, _ := args["path"].(string)
			content, _ := args["content"].(string)

			if path == "" {
				return agent.ToolResult{}, fmt.Errorf("path is required")
			}

			absolutePath := ResolveToCwd(path, cwd)
			dir := filepath.Dir(absolutePath)

			if err := fileOps.MkdirAll(dir); err != nil {
				return agent.ToolResult{}, fmt.Errorf("failed to create directory: %w", err)
			}

			if err := fileOps.WriteFile(absolutePath, []byte(content)); err != nil {
				return agent.ToolResult{}, fmt.Errorf("failed to write file: %w", err)
			}

			return textResult(fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path)), nil
		},
	}
}

// CreateGrepTool creates a pattern search tool using ripgrep.
func CreateGrepTool(cwd string, ops *BashOps) agent.Tool {
	bashOps := DefaultBashOps()
	if ops != nil {
		bashOps = *ops
	}

	return agent.Tool{
		Tool: ai.Tool{
			Name:        "grep",
			Description: "Search file contents for patterns (respects .gitignore). Uses ripgrep for fast, recursive search.",
			Parameters:  grepSchema(),
		},
		Label: "grep",
		Execute: func(ctx context.Context, _ string, args map[string]any, _ agent.ToolUpdateFunc) (agent.ToolResult, error) {
			pattern, _ := args["pattern"].(string)
			if pattern == "" {
				return agent.ToolResult{}, fmt.Errorf("pattern is required")
			}

			searchDir := cwd
			if v, ok := args["path"].(string); ok && v != "" {
				searchDir = ResolveToCwd(v, cwd)
			}

			limit := 100
			if v, ok := args["limit"].(float64); ok && int(v) > 0 {
				limit = int(v)
			}

			// Build ripgrep command
			rgArgs := []string{"rg", "--line-number", "--no-heading", "--color=never"}

			if v, ok := args["ignoreCase"].(bool); ok && v {
				rgArgs = append(rgArgs, "--ignore-case")
			}
			if v, ok := args["literal"].(bool); ok && v {
				rgArgs = append(rgArgs, "--fixed-strings")
			}
			if v, ok := args["context"].(float64); ok && int(v) > 0 {
				rgArgs = append(rgArgs, fmt.Sprintf("--context=%d", int(v)))
			}
			if v, ok := args["glob"].(string); ok && v != "" {
				rgArgs = append(rgArgs, fmt.Sprintf("--glob=%s", v))
			}

			rgArgs = append(rgArgs, fmt.Sprintf("--max-count=%d", limit))
			rgArgs = append(rgArgs, "--", pattern, searchDir)

			command := strings.Join(rgArgs, " ")
			stdout, _, err := bashOps.Exec(ctx, command, cwd)
			if err != nil {
				return agent.ToolResult{}, fmt.Errorf("grep failed: %w", err)
			}

			if strings.TrimSpace(stdout) == "" {
				return textResult("No matches found."), nil
			}

			// Truncate long lines and overall output
			lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
			var truncatedLines []string
			for _, line := range lines {
				text, _ := TruncateLine(line, GrepMaxLineLength)
				truncatedLines = append(truncatedLines, text)
			}

			output := strings.Join(truncatedLines, "\n")
			result := TruncateHead(output, TruncationOptions{})
			if result.Truncated {
				output = result.Content + fmt.Sprintf("\n\n[Truncated: %d/%d lines]", result.OutputLines, result.TotalLines)
			} else {
				output = result.Content
			}

			return textResult(output), nil
		},
	}
}

// CreateFindTool creates a file search tool using find/fd.
func CreateFindTool(cwd string, ops *BashOps) agent.Tool {
	bashOps := DefaultBashOps()
	if ops != nil {
		bashOps = *ops
	}

	return agent.Tool{
		Tool: ai.Tool{
			Name:        "find",
			Description: "Find files by glob pattern (respects .gitignore). Returns matching file paths.",
			Parameters:  findSchema(),
		},
		Label: "find",
		Execute: func(ctx context.Context, _ string, args map[string]any, _ agent.ToolUpdateFunc) (agent.ToolResult, error) {
			pattern, _ := args["pattern"].(string)
			if pattern == "" {
				return agent.ToolResult{}, fmt.Errorf("pattern is required")
			}

			searchDir := cwd
			if v, ok := args["path"].(string); ok && v != "" {
				searchDir = ResolveToCwd(v, cwd)
			}

			limit := 1000
			if v, ok := args["limit"].(float64); ok && int(v) > 0 {
				limit = int(v)
			}

			// Try fd first, fall back to find
			command := fmt.Sprintf("fd --glob '%s' '%s' --max-results %d 2>/dev/null || find '%s' -name '%s' -maxdepth 10 2>/dev/null | head -%d",
				pattern, searchDir, limit, searchDir, pattern, limit)

			stdout, _, err := bashOps.Exec(ctx, command, cwd)
			if err != nil {
				return agent.ToolResult{}, fmt.Errorf("find failed: %w", err)
			}

			if strings.TrimSpace(stdout) == "" {
				return textResult("No files found matching the pattern."), nil
			}

			result := TruncateHead(strings.TrimRight(stdout, "\n"), TruncationOptions{})
			output := result.Content
			if result.Truncated {
				output += fmt.Sprintf("\n\n[Truncated: %d/%d results]", result.OutputLines, result.TotalLines)
			}

			return textResult(output), nil
		},
	}
}

// CreateLsTool creates a directory listing tool.
func CreateLsTool(cwd string, ops *FileOps) agent.Tool {
	fileOps := DefaultFileOps()
	if ops != nil {
		fileOps = *ops
	}

	return agent.Tool{
		Tool: ai.Tool{
			Name:        "ls",
			Description: fmt.Sprintf("List directory contents. Sorted alphabetically, '/' suffix for directories. Truncated to 500 entries or %s.", FormatSize(DefaultMaxBytes)),
			Parameters:  lsSchema(),
		},
		Label: "ls",
		Execute: func(ctx context.Context, _ string, args map[string]any, _ agent.ToolUpdateFunc) (agent.ToolResult, error) {
			dirPath := cwd
			if v, ok := args["path"].(string); ok && v != "" {
				dirPath = ResolveToCwd(v, cwd)
			}

			limit := 500
			if v, ok := args["limit"].(float64); ok && int(v) > 0 {
				limit = int(v)
			}

			info, err := fileOps.Stat(dirPath)
			if err != nil {
				return agent.ToolResult{}, fmt.Errorf("path not found: %s", dirPath)
			}
			if !info.IsDir() {
				return agent.ToolResult{}, fmt.Errorf("not a directory: %s", dirPath)
			}

			entries, err := fileOps.ReadDir(dirPath)
			if err != nil {
				return agent.ToolResult{}, fmt.Errorf("cannot read directory: %w", err)
			}

			// Sort case-insensitive
			sort.Slice(entries, func(i, j int) bool {
				return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
			})

			var results []string
			limitReached := false

			for _, entry := range entries {
				if len(results) >= limit {
					limitReached = true
					break
				}
				name := entry.Name()
				if entry.IsDir() {
					name += "/"
				}
				results = append(results, name)
			}

			if len(results) == 0 {
				return textResult("(empty directory)"), nil
			}

			rawOutput := strings.Join(results, "\n")
			truncation := TruncateHead(rawOutput, TruncationOptions{MaxLines: 1<<31 - 1}) // No line limit, only byte limit

			output := truncation.Content
			var notices []string

			if limitReached {
				notices = append(notices, fmt.Sprintf("%d entries limit reached", limit))
			}
			if truncation.Truncated {
				notices = append(notices, fmt.Sprintf("%s limit reached", FormatSize(DefaultMaxBytes)))
			}
			if len(notices) > 0 {
				output += fmt.Sprintf("\n\n[%s]", strings.Join(notices, ". "))
			}

			return textResult(output), nil
		},
	}
}

// --- Tool sets ---

// ToolName identifies a built-in tool.
type ToolName string

const (
	ToolRead  ToolName = "read"
	ToolBash  ToolName = "bash"
	ToolEdit  ToolName = "edit"
	ToolWrite ToolName = "write"
	ToolGrep  ToolName = "grep"
	ToolFind  ToolName = "find"
	ToolLs    ToolName = "ls"
)

// CreateCodingTools creates the standard coding tools for a working directory.
func CreateCodingTools(cwd string) []agent.Tool {
	return []agent.Tool{
		CreateReadTool(cwd, nil),
		CreateBashTool(cwd, nil),
		CreateEditTool(cwd, nil),
		CreateWriteTool(cwd, nil),
	}
}

// CreateReadOnlyTools creates read-only exploration tools.
func CreateReadOnlyTools(cwd string) []agent.Tool {
	return []agent.Tool{
		CreateReadTool(cwd, nil),
		CreateGrepTool(cwd, nil),
		CreateFindTool(cwd, nil),
		CreateLsTool(cwd, nil),
	}
}

// CreateAllTools creates all available tools for a working directory.
func CreateAllTools(cwd string) map[ToolName]agent.Tool {
	return map[ToolName]agent.Tool{
		ToolRead:  CreateReadTool(cwd, nil),
		ToolBash:  CreateBashTool(cwd, nil),
		ToolEdit:  CreateEditTool(cwd, nil),
		ToolWrite: CreateWriteTool(cwd, nil),
		ToolGrep:  CreateGrepTool(cwd, nil),
		ToolFind:  CreateFindTool(cwd, nil),
		ToolLs:    CreateLsTool(cwd, nil),
	}
}

// --- Helpers ---

func textResult(text string) agent.ToolResult {
	return agent.ToolResult{
		Content: []ai.ContentBlock{{Type: ai.ContentTypeText, Text: text}},
		Details: json.RawMessage("{}"),
	}
}
