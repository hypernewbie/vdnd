package rlm

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestREPLExecutor(t *testing.T) {
	// Navigate up to find vdn root if necessary, but here we expect to be in internal/llm/rlm
	// Let's find the absolute path to restricted_python.py relative to the repo root.
	// We'll assume the test is run from the repo root or we can find it.

	absRoot, err := filepath.Abs("../../../")
	if err != nil {
		t.Fatal(err)
	}

	pythonPath := FindPythonPath(absRoot)
	scriptPath := filepath.Join(absRoot, "py", "restricted_python.py")

	executor, err := NewREPLExecutor(pythonPath, scriptPath)
	if err != nil {
		t.Fatalf("Failed to start REPL: %v", err)
	}
	defer executor.Close()

	t.Run("Basic print", func(t *testing.T) {
		result, err := executor.Execute("print('hello world')")
		if err != nil {
			t.Errorf("Execute failed: %v", err)
		}
		if result.Stdout != "hello world\n" {
			t.Errorf("Expected 'hello world\n', got %q", result.Stdout)
		}
	})

	t.Run("Persistent State", func(t *testing.T) {
		_, err := executor.Execute("x = 10")
		if err != nil {
			t.Errorf("Execute x=10 failed: %v", err)
		}
		result, err := executor.Execute("print(x)")
		if err != nil {
			t.Errorf("Execute print(x) failed: %v", err)
		}
		if result.Stdout != "10\n" {
			t.Errorf("Expected '10\n', got %q", result.Stdout)
		}
	})

	t.Run("Directory Permissions", func(t *testing.T) {
		// character_creation.md exists in rules_derived/
		path := filepath.Join(absRoot, "rules_derived", "character_creation.md")
		result, err := executor.Execute(fmt.Sprintf("with open('%s', 'r') as f: print(f.read()[:10])", path))
		if err != nil {
			t.Errorf("Execute failed: %v", err)
		}
		if result.Error != "" {
			t.Errorf("Expected no error, got %q", result.Error)
		}
		if result.Stdout == "" {
			t.Error("Expected some stdout content, got empty string")
		}
	})

	t.Run("Allowed Files List", func(t *testing.T) {
		result, err := executor.Execute("print(allowed_files())")
		if err != nil {
			t.Errorf("Execute failed: %v", err)
		}
		if result.Error != "" {
			t.Errorf("Expected no error, got %q", result.Error)
		}
		// Should contain at least character_creation.md
		if !strings.Contains(result.Stdout, "character_creation.md") {
			t.Errorf("Expected allowed_files to contain 'character_creation.md', got %q", result.Stdout)
		}
	})
}
