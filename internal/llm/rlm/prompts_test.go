package rlm

import (
	"os"
	"strings"
	"testing"
)

func TestBuildDMSystemPrompt_ForbidsPythonStateMutation(t *testing.T) {
	// Setup vdm_prompt.txt for this test
	const testPrompt = "Rule: Never simulate combat/damage/healing in Python. ripgrep tool. Depth: %d"
	promptFile := "vdm_prompt.txt"
	err := os.WriteFile(promptFile, []byte(testPrompt), 0644)
	if err != nil {
		t.Fatalf("failed to write test vdm_prompt.txt: %v", err)
	}
	defer os.Remove(promptFile)

	builder, err := NewDMSystemPromptBuilder(promptFile)
	if err != nil {
		t.Fatalf("failed to create builder: %v", err)
	}
	prompt := builder(1000, 1)
	if !strings.Contains(prompt, "Never simulate combat/damage/healing in Python") {
		t.Errorf("DM prompt must forbid Python state mutation")
	}
	if !strings.Contains(prompt, "ripgrep") {
		t.Errorf("DM prompt should mention ripgrep tool")
	}
}

func TestBuildDMSystemPrompt_FromFile(t *testing.T) {
	const testPrompt = "Custom DM Prompt: %d"
	promptFile := "vdm_prompt.txt"
	err := os.WriteFile(promptFile, []byte(testPrompt), 0644)
	if err != nil {
		t.Fatalf("failed to write test vdm_prompt.txt: %v", err)
	}
	defer os.Remove(promptFile)

	builder, err := NewDMSystemPromptBuilder(promptFile)
	if err != nil {
		t.Fatalf("failed to create builder: %v", err)
	}
	prompt := builder(1000, 2)
	expected := "Custom DM Prompt: 2"
	if prompt != expected {
		t.Errorf("Expected prompt %q, got %q", expected, prompt)
	}
}

func TestBuildDMSystemPrompt_FromEnvVar(t *testing.T) {
	const testPrompt = "Env DM Prompt: %d"
	const tmpFile = "temp_prompt.txt"
	err := os.WriteFile(tmpFile, []byte(testPrompt), 0644)
	if err != nil {
		t.Fatalf("failed to write %s: %v", tmpFile, err)
	}
	defer os.Remove(tmpFile)

	os.Setenv("VDM_PROMPT_FILE", tmpFile)
	defer os.Unsetenv("VDM_PROMPT_FILE")

	builder, err := NewDMSystemPromptBuilder(tmpFile)
	if err != nil {
		t.Fatalf("failed to create builder: %v", err)
	}
	prompt := builder(1000, 3)
	expected := "Env DM Prompt: 3"
	if prompt != expected {
		t.Errorf("Expected prompt %q, got %q", expected, prompt)
	}
}
