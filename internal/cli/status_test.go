package cli

import (
	"strings"
	"testing"
	"uaa/vdnd/internal/state"
)

func TestStatus(t *testing.T) {
	deps := NewTestDeps(t)
	s := &state.GameState{
		SceneName: "The Grand Hall",
		Entities: map[string]*state.EntityState{
			"hero": {ID: "hero", Name: "Hero", HP: 20, MaxHP: 20, AC: 15},
		},
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	out, err := cmdStatus([]string{}, deps)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if !strings.Contains(out, "# Scene: The Grand Hall") {
		t.Errorf("Unexpected output: %s", out)
	}
	if !strings.Contains(out, "Hero") {
		t.Errorf("Expected Hero in status: %s", out)
	}
}

func TestStatusEmpty(t *testing.T) {
	deps := NewTestDeps(t)
	s := &state.GameState{
		SceneName: "Empty Room",
		Entities:  make(map[string]*state.EntityState),
		Positions: make(map[string]*state.Zone),
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	out, err := cmdStatus([]string{}, deps)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if !strings.Contains(out, "No entities in scene.") {
		t.Errorf("Unexpected output: %s", out)
	}
}
