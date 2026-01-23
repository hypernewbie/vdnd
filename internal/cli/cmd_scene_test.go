package cli

import (
	"strings"
	"testing"
	"uaa/vdnd/internal/state"
)

func TestSceneNew(t *testing.T) {
	deps := DefaultDeps()
	deps.Store = &state.MemoryStore{} // Empty store

	out, err := cmdSceneNew([]string{"The", "Dungeon"}, deps)
	if err != nil {
		t.Fatalf("cmdSceneNew failed: %v", err)
	}

	if !strings.Contains(out, "Scene Created: The Dungeon") {
		t.Errorf("Unexpected output: %s", out)
	}

	// Verify state saved
	s, err := deps.Store.Load()
	if err != nil {
		t.Fatalf("Store check failed: %v", err)
	}
	if s.SceneName != "The Dungeon" {
		t.Errorf("SceneName = %q, want 'The Dungeon'", s.SceneName)
	}
}

func TestSceneNew_Exists(t *testing.T) {
	deps := DefaultDeps()
	deps.Store = &state.MemoryStore{
		State: &state.GameState{SceneName: "Existing"},
	}

	_, err := cmdSceneNew([]string{"New"}, deps)
	if err == nil {
		t.Error("Expected error when scene exists, got nil")
	}
}
