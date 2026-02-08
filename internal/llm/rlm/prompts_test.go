package rlm

import (
	"strings"
	"testing"
)

func TestBuildDMSystemPrompt_ForbidsPythonStateMutation(t *testing.T) {
	prompt := BuildDMSystemPrompt(1000, 1)
	if !strings.Contains(prompt, "Never simulate combat/damage/healing in Python") {
		t.Errorf("DM prompt must forbid Python state mutation")
	}
	if strings.Contains(prompt, "recursive_llm") {
		t.Errorf("DM prompt should not mention recursive_llm (depth=1)")
	}
	if !strings.Contains(prompt, "ripgrep") {
		t.Errorf("DM prompt should mention ripgrep tool")
	}
}
