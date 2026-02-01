package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunCLI_EchoLoop(t *testing.T) {
	// 1. Setup Input (User typing commands)
	input := "echo hello world\nexit\n"
	in := strings.NewReader(input)

	// 2. Setup Output (Capturing CLI response)
	out := new(bytes.Buffer)

	// 3. Mock Config (No LLM for this basic test)
	cfg := &Config{
		LLMProvider: "none",
	}

	// 4. Run the CLI
	runCLI(context.Background(), in, out, cfg, false)

	// 5. Assertions
	outputStr := out.String()

	// Check startup message
	if !strings.Contains(outputStr, "Standard CLI mode enabled") {
		t.Errorf("Expected startup message, got: %s", outputStr)
	}

	// Check echo response
	if !strings.Contains(outputStr, "Echo: hello world") {
		t.Errorf("Expected echo response, got: %s", outputStr)
	}
}
