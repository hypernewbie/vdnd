package vdhelpers

import (
	"encoding/json"
	"testing"
	"uaa/vdnd/internal/cli"
)

func TestExecuteGenericVD(t *testing.T) {
	// Mock cliRun
	var recordedArgs [][]string
	mockStdout := "Scene status..."
	mockExitCode := 0

	oldCliRun := cliRun
	cliRun = func(args []string, deps cli.Deps) (string, int) {
		recordedArgs = append(recordedArgs, args)
		return mockStdout, mockExitCode
	}
	defer func() { cliRun = oldCliRun }()

	deps := cli.Deps{}
	resultJSON := ExecuteGenericVD("status", deps)

	// Check that cli.Run was called with correct args
	if len(recordedArgs) != 1 {
		t.Fatalf("cli.Run called %d times, want 1", len(recordedArgs))
	}
	want := []string{"status"}
	for i, arg := range recordedArgs[0] {
		if arg != want[i] {
			t.Errorf("argv[%d] = %q, want %q", i, arg, want[i])
		}
	}

	// Verify JSON structure
	var result VDResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if result.Stdout != mockStdout {
		t.Errorf("Stdout = %q, want %q", result.Stdout, mockStdout)
	}
	if result.ExitCode != mockExitCode {
		t.Errorf("ExitCode = %d, want %d", result.ExitCode, mockExitCode)
	}
}

func TestExecuteGenericVD_QuotedCommand(t *testing.T) {
	var recordedArgs [][]string
	oldCliRun := cliRun
	cliRun = func(args []string, deps cli.Deps) (string, int) {
		recordedArgs = append(recordedArgs, args)
		return "", 0
	}
	defer func() { cliRun = oldCliRun }()

	ExecuteGenericVD(`action cast wizard "fire ball" --target goblin`, cli.Deps{})
	if len(recordedArgs) != 1 {
		t.FailNow()
	}
	// Expect quotes stripped
	want := []string{"action", "cast", "wizard", "fire ball", "--target", "goblin"}
	for i, arg := range recordedArgs[0] {
		if arg != want[i] {
			t.Errorf("argv[%d] = %q, want %q", i, arg, want[i])
		}
	}
}

func TestExecuteGenericVD_EmptyCommand(t *testing.T) {
	resultJSON := ExecuteGenericVD("", cli.Deps{})
	var result VDResult
	json.Unmarshal([]byte(resultJSON), &result)
	if result.Error == "" {
		t.Error("Empty command should produce an error field")
	}
}