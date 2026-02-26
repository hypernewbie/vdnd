package subagents

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPythonReadOnlySubagent_Restriction(t *testing.T) {
	wd, _ := os.Getwd()
	// Project root is 3 levels up from internal/llm/subagents
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(wd)))
	
	// Change CWD to project root for the test
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("Failed to change CWD: %v", err)
	}
	defer os.Chdir(wd)

	python := FindPythonPath(".")
	script := filepath.Join("py", "restricted_python.py")

	agent, err := NewPythonReadOnlySubagent(python, script)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.repl.Close()

	// 1. Test reading should work (if file exists)
	// Create a dummy file in sandbox
	sandboxDir := filepath.Join(projectRoot, "sandbox")
	os.MkdirAll(sandboxDir, 0755)
	testFile := filepath.Join(sandboxDir, "test_readonly.md")
	os.WriteFile(testFile, []byte("readonly test content"), 0644)
	defer os.Remove(testFile)

	readCode := `print(open("sandbox/test_readonly.md").read())`
	result, err := agent.Run(context.Background(), readCode, nil)
	if err != nil {
		t.Errorf("Read failed: %v", err)
	}
	if !strings.Contains(result, "readonly test content") {
		t.Errorf("Expected content not found in result: %q", result)
	}

	// 2. Test writing should FAIL
	writeCode := `open("sandbox/should_fail.md", "w").write("fail")`
	result, err = agent.Run(context.Background(), writeCode, nil)
	if err != nil {
		t.Errorf("Execution failed unexpectedly: %v", err)
	}
	if !strings.Contains(result, "PermissionError") && !strings.Contains(result, "Read-only mode") {
		t.Errorf("Expected PermissionError for writing in read-only mode, got: %q", result)
	}
}
