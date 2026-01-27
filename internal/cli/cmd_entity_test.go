package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"uaa/vdnd/internal/state"
)

func TestEntityCommands(t *testing.T) {
	deps := NewTestDeps(t)
	
	// Initialise state with a scene
	s := &state.GameState{
		SceneName: "Test Scene",
		Entities:  make(map[string]*state.EntityState),
		Positions: make(map[string]*state.Zone),
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	// Create a temp entity file
	entityMD := `# Goblin Warrior
- Level: 1
- HP: 15/15
- AC: 16
- Speed: 25ft
`
	tmpFile := filepath.Join(t.TempDir(), "goblin.md")
	os.WriteFile(tmpFile, []byte(entityMD), 0644)

	// Test Entity Add
	t.Run("Add Entity", func(t *testing.T) {
		out, err := cmdEntityAdd([]string{"goblin1", "--file", tmpFile}, deps)
		if err != nil {
			t.Fatalf("cmdEntityAdd failed: %v", err)
		}
		if !strings.Contains(out, "Added entity: **Goblin Warrior** (goblin1)") {
			t.Errorf("Unexpected output: %s", out)
		}

		// Verify in state
		st, _ := deps.Store.Load()
		if _, ok := st.Entities["goblin1"]; !ok {
			t.Error("Entity goblin1 not found in state")
		}
	})

	// Test Entity Get
	t.Run("Get Entity", func(t *testing.T) {
		out, err := cmdEntityGet([]string{"goblin1"}, deps)
		if err != nil {
			t.Fatalf("cmdEntityGet failed: %v", err)
		}
		if !strings.Contains(out, "# Goblin Warrior (goblin1)") {
			t.Errorf("Unexpected output: %s", out)
		}
		if !strings.Contains(out, "**HP:** 15/15") {
			t.Errorf("Unexpected output: %s", out)
		}
	})

	// Test Entity Set
	t.Run("Set Entity", func(t *testing.T) {
		_, err := cmdEntitySet([]string{"goblin1", "hp", "10"}, deps)
		if err != nil {
			t.Fatalf("cmdEntitySet failed: %v", err)
		}

		// Verify in state
		st, _ := deps.Store.Load()
		if st.Entities["goblin1"].HP != 10 {
			t.Errorf("Expected HP 10, got %d", st.Entities["goblin1"].HP)
		}
	})

	// Test Entity List
	t.Run("List Entities", func(t *testing.T) {
		out, err := cmdEntityList([]string{}, deps)
		if err != nil {
			t.Fatalf("cmdEntityList failed: %v", err)
		}
		if !strings.Contains(out, "| goblin1 | Goblin Warrior | 1 | 10/15 |") {
			t.Errorf("Unexpected list output: %s", out)
		}

		// List with zone filter
		out, _ = cmdEntityList([]string{"--zone", "dungeon"}, deps)
		if !strings.Contains(out, "No entities found in zone") {
			t.Errorf("Expected 'no entities' message, got: %s", out)
		}
	})

	// Test Entity Spawn
	t.Run("Spawn Entities", func(t *testing.T) {
		out, err := cmdEntitySpawn([]string{tmpFile, "--count", "2", "--prefix", "g"}, deps)
		if err != nil {
			t.Fatalf("cmdEntitySpawn failed: %v", err)
		}
		if !strings.Contains(out, "Spawned 2 entities with prefix **g**") {
			t.Errorf("Unexpected spawn output: %s", out)
		}

		// Verify in state
		st, _ := deps.Store.Load()
		if _, ok := st.Entities["g_1"]; !ok {
			t.Error("Entity g_1 not found in state")
		}
		if _, ok := st.Entities["g_2"]; !ok {
			t.Error("Entity g_2 not found in state")
		}
		if st.Entities["g_1"].Name != "Goblin Warrior 1" {
			t.Errorf("Expected name 'Goblin Warrior 1', got '%s'", st.Entities["g_1"].Name)
		}
	})
}
