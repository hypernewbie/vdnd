package vdhelpers

import (
	"encoding/json"
	"io"
	"testing"
	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/state"
)

func TestExecuteGenericVD(t *testing.T) {
	// Setup deps with memory store
	gs := &state.GameState{
		SceneName:     "Test Scene",
		Positions:     make(map[string]*state.Zone),
		Entities:      make(map[string]*state.EntityState),
		ReactionsUsed: make(map[string]bool),
	}
	deps := cli.Deps{
		Roller: &cli.FixedRoller{Results: []int{10, 10}},
		Store:  &state.MemoryStore{State: gs},
		Clock:  &cli.FixedClock{},
		Stderr: io.Discard,
	}

	// Test a simple command
	resStr := ExecuteGenericVD("status", deps)
	var res VDResult
	if err := json.Unmarshal([]byte(resStr), &res); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// We expect status to return something in stdout
	if res.Stdout == "" && res.Error == "" {
		t.Errorf("expected stdout or error, got empty result")
	}

	// Test empty command
	resStr = ExecuteGenericVD("", deps)
	if err := json.Unmarshal([]byte(resStr), &res); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if res.Error != "empty command" {
		t.Errorf("expected 'empty command' error, got %q", res.Error)
	}
}