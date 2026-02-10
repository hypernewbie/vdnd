package rlm

import (
	"fmt"
	"os"
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

	t.Run("Ripgrep", func(t *testing.T) {
		// Use the sandbox directory for testing ripgrep
		sandboxFile := filepath.Join(absRoot, "sandbox", "test_rg.md")
		os.WriteFile(sandboxFile, []byte("The secret word is Antigravity"), 0644)
		defer os.Remove(sandboxFile)

		result, err := executor.Execute("res = ripgrep('Antigravity', 'sandbox/'); print(res[0])")
		if err != nil {
			t.Errorf("Execute failed: %v", err)
		}
		if !strings.Contains(result.Stdout, "sandbox/test_rg.md") {
			t.Errorf("Expected filename in stdout, got %q", result.Stdout)
		}
		if !strings.Contains(result.Stdout, "The secret word is Antigravity") {
			t.Errorf("Expected match in stdout, got %q", result.Stdout)
		}
	})
	t.Run("Enumerate", func(t *testing.T) {
		result, err := executor.Execute("l = ['a', 'b']; print(list(enumerate(l)))")
		if err != nil {
			t.Errorf("Execute failed: %v", err)
		}
		if !strings.Contains(result.Stdout, "[(0, 'a'), (1, 'b')]") {
			t.Errorf("Expected enumerate output, got %q", result.Stdout)
		}
	})

	t.Run("Extended Builtins", func(t *testing.T) {
		code := `
print(f"type: {type([]) is list}")
print(f"hasattr: {hasattr([], 'append')}")
print(f"locals: {'x' in locals()}")
print(f"globals: {'list_dir' in globals()}")
`
		result, err := executor.Execute(code)
		if err != nil {
			t.Errorf("Execute failed: %v", err)
		}
		expected := []string{
			"type: True",
			"hasattr: True",
			"locals: True",
			"globals: True",
		}
		for _, e := range expected {
			if !strings.Contains(result.Stdout, e) {
				t.Errorf("Expected %q in stdout, got %q", e, result.Stdout)
			}
		}
	})
}
