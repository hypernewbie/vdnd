package cli

import (
	"strings"
	"testing"
	"uaa/vdnd/internal/state"
)

func TestErrorStates(t *testing.T) {
	deps := NewTestDeps(t)
	s := &state.GameState{
		Entities: map[string]*state.EntityState{
			"hero": {
				ID:   "hero",
				Name: "Hero",
				AC:   15,
			},
		},
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	t.Run("Strike Invalid Target", func(t *testing.T) {
		_, err := cmdActionStrike([]string{"hero", "ghost"}, deps)
		if err == nil || !strings.Contains(err.Error(), "target not found") {
			t.Errorf("Expected 'target not found' error, got: %v", err)
		}
	})

	t.Run("Strike Invalid Actor", func(t *testing.T) {
		_, err := cmdActionStrike([]string{"ghost", "hero"}, deps)
		if err == nil || !strings.Contains(err.Error(), "actor not found") {
			t.Errorf("Expected 'actor not found' error, got: %v", err)
		}
	})

	t.Run("Heal Non-existent Entity", func(t *testing.T) {
		_, err := cmdHeal([]string{"ghost", "10"}, deps)
		if err == nil || !strings.Contains(err.Error(), "entity not found") {
			t.Errorf("Expected 'entity not found' error, got: %v", err)
		}
	})

	t.Run("Roll Invalid Expression", func(t *testing.T) {
		_, err := cmdRoll([]string{"2d6+lol"}, deps)
		if err == nil || !strings.Contains(err.Error(), "invalid dice expression") {
			t.Errorf("Expected 'invalid dice expression' error, got: %v", err)
		}
	})
}

func TestSceneEdgeCases(t *testing.T) {
	deps := NewTestDeps(t)
	
	t.Run("Load Non-existent Scene", func(t *testing.T) {
		_, err := cmdSceneLoad([]string{"missing.json"}, deps)
		if err == nil {
			t.Error("Expected error loading non-existent scene")
		}
	})

	t.Run("Save Without Scene", func(t *testing.T) {
		// Store is empty initially
		_, err := cmdSceneSave([]string{"saved.json"}, deps)
		if err == nil {
			t.Error("Expected error saving non-existent scene")
		}
	})
}

func TestEntityEdgeCases(t *testing.T) {
	deps := NewTestDeps(t)
	s := &state.GameState{
		SceneName: "Test",
		Entities:  make(map[string]*state.EntityState),
		Positions: make(map[string]*state.Zone),
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	t.Run("Get Non-existent Field", func(t *testing.T) {
		s.Entities["hero"] = &state.EntityState{ID: "hero", Name: "Hero", HP: 10, MaxHP: 10, AC: 15}
		deps.Store.Save(s)

		_, err := cmdEntityGet([]string{"hero", "--field", "magic"}, deps)
		if err == nil || !strings.Contains(err.Error(), "unknown field") {
			t.Errorf("Expected 'unknown field' error, got: %v", err)
		}
	})

	t.Run("Set Non-existent Field", func(t *testing.T) {
		_, err := cmdEntitySet([]string{"hero", "magic", "yes"}, deps)
		if err == nil || !strings.Contains(err.Error(), "unsupported field") {
			t.Errorf("Expected 'unsupported field' error, got: %v", err)
		}
	})

	t.Run("Add Entity Without File", func(t *testing.T) {
		_, err := cmdEntityAdd([]string{"hero"}, deps)
		if err == nil || !strings.Contains(err.Error(), "missing --file") {
			t.Errorf("Expected 'missing --file' error, got: %v", err)
		}
	})

	t.Run("Add Entity Missing File", func(t *testing.T) {
		_, err := cmdEntityAdd([]string{"hero", "--file", "missing.md"}, deps)
		if err == nil {
			t.Error("Expected error adding from missing file")
		}
	})

	t.Run("Spawn Missing Template", func(t *testing.T) {
		_, err := cmdEntitySpawn([]string{"missing.md"}, deps)
		if err == nil {
			t.Error("Expected error spawning from missing template")
		}
	})
}

func TestSpellEdgeCases(t *testing.T) {
	deps := NewTestDeps(t)
	s := &state.GameState{
		Entities: map[string]*state.EntityState{
			"wizard": {ID: "wizard", Name: "Wizard", Position: "room_a", AC: 10},
		},
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	t.Run("Cast Without Targets", func(t *testing.T) {
		_, err := cmdActionCast([]string{"wizard", "fireball", "--dc", "20"}, deps)
		if err == nil || !strings.Contains(err.Error(), "no targets specified") {
			t.Errorf("Expected 'no targets specified' error, got: %v", err)
		}
	})

	t.Run("Cast Save Spell Without DC", func(t *testing.T) {
		_, err := cmdActionCast([]string{"wizard", "fireball", "--target", "wizard"}, deps)
		if err == nil || !strings.Contains(err.Error(), "--dc is required") {
			t.Errorf("Expected '--dc is required' error, got: %v", err)
		}
	})
}
