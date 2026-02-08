package rlm

import (
	"strings"
	"testing"
)

func TestBuildDMSystemPrompt_ForbidsPythonStateMutation(t *testing.T) {
	prompt := BuildDMSystemPrompt(1000, 1)
	if !strings.Contains(prompt, "Do NOT attempt to simulate combat, damage, or healing in Python") {
		t.Errorf("DM prompt must forbid Python state mutation")
	}
	if !strings.Contains(prompt, "ripgrep") {
		t.Errorf("DM prompt should mention ripgrep tool")
	}
}

func TestBuildVDLMSystemPrompt_MentionsTools(t *testing.T) {
	prompt := BuildVDLMSystemPrompt(1000, 1)
	if !strings.Contains(prompt, "vd_action_strike") {
		t.Errorf("VDLM prompt should mention vd_action_strike")
	}
	if !strings.Contains(prompt, "vd_status") {
		t.Errorf("VDLM prompt should mention vd_status")
	}
}