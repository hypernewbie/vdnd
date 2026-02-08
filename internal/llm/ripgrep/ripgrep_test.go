package ripgrep

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// NOTE: execLookPath and execCommand are defined in ripgrep.go
// Do NOT redeclare them here, or we shadow the package variables and mocks won't work.

func TestSearch_RGInstalled(t *testing.T) {
	// Save original
	oldLookPath := execLookPath
	oldExecCommand := execCommand
	defer func() {
		execLookPath = oldLookPath
		execCommand = oldExecCommand
	}()

	// Temporarily replace LookPath to simulate rg present
	execLookPath = func(name string) (string, error) {
		if name == "rg" {
			return "/usr/bin/rg", nil
		}
		return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
	}

	// Mock command execution
	execCommand = func(name string, arg ...string) *exec.Cmd {
		// Return a command that writes a fake rg output to stdout
		// We call the test binary again, but filtering for TestHelperProcess
		cs := []string{"-test.run=^TestHelperProcess$", "--"}
		cs = append(cs, name)
		cs = append(cs, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		return cmd
	}

	result, err := Search("fireball", "rules/")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if !result.RGInstalled {
		t.Error("RGInstalled should be true")
	}
	if result.Warning != "" {
		t.Error("Warning should be empty when rg installed")
	}
	// Verify matches parsing (if helper process outputs sample lines)
	if len(result.Matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(result.Matches))
	}
	if len(result.Matches) > 0 && result.Matches[0].Text != "Fireball deals 6d6 fire damage." {
		t.Errorf("Unexpected match text: %q", result.Matches[0].Text)
	}
}

func TestSearch_RGNotInstalled(t *testing.T) {
	// Save original
	oldLookPath := execLookPath
	defer func() { execLookPath = oldLookPath }()

	execLookPath = func(name string) (string, error) {
		return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
	}

	// Capture stderr to check warning
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	result, err := Search("fireball", "")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	w.Close()
	captured, _ := io.ReadAll(r)
	output := string(captured)

	if result.RGInstalled {
		t.Error("RGInstalled should be false")
	}
	if !strings.Contains(result.Warning, "RIPGREP NOT INSTALLED") {
		t.Error("Warning should contain installation notice")
	}
	// Check that warning was printed to stderr with ANSI colors/emojis
	if !strings.Contains(output, "🔥🔥🔥") {
		t.Error("Loud warning with emojis should be printed to stderr")
	}
}

// Helper process for mocking rg output
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// Simulate rg output: file:line:text
	fmt.Fprintf(os.Stdout, "rules/spells.md:42:Fireball deals 6d6 fire damage.\n")
	fmt.Fprintf(os.Stdout, "rules/spells.md:45:Heightened Fireball increases damage.\n")
	os.Exit(0)
}
