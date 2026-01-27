package cli

import (
	"strings"
	"testing"
	"uaa/vdnd/internal/state"
)

func TestQueryCommands(t *testing.T) {
	deps := NewTestDeps(t)
	s := &state.GameState{
		Positions: map[string]*state.Zone{
			"room1": {Name: "Room 1", Adjacent: []string{"hallway"}},
			"hallway": {Name: "Hallway", Adjacent: []string{"room1", "room2"}},
			"room2": {Name: "Room 2", Adjacent: []string{"hallway"}, Cover: "standard"},
		},
		Entities: map[string]*state.EntityState{
			"hero1": {
				ID:       "hero1",
				Name:     "Hero 1",
				Position: "room1",
			},
			"hero2": {
				ID:       "hero2",
				Name:     "Hero 2",
				Position: "room2",
				EngagedWith: []string{"goblin"},
			},
			"goblin": {
				ID:       "goblin",
				Name:     "Goblin",
				Position: "room2",
				EngagedWith: []string{"hero2", "hero3"},
			},
			"hero3": {
				ID:       "hero3",
				Name:     "Hero 3",
				Position: "room2",
				EngagedWith: []string{"goblin"},
			},
		},
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	t.Run("Distance", func(t *testing.T) {
		out, err := cmdQueryDistance([]string{"hero1", "hero2"}, deps)
		if err != nil {
			t.Fatalf("Distance failed: %v", err)
		}
		if !strings.Contains(out, "60 ft") || !strings.Contains(out, "2 Zones") {
			t.Errorf("Unexpected distance: %s", out)
		}
	})

	t.Run("Targets", func(t *testing.T) {
		out, err := cmdQueryTargets([]string{"hero1", "--range", "100"}, deps)
		if err != nil {
			t.Fatalf("Targets failed: %v", err)
		}
		if !strings.Contains(out, "goblin") {
			t.Errorf("Expected goblin in targets: %s", out)
		}
		if strings.Contains(out, "hero2") {
			t.Error("Hero2 should not be a target (ally)")
		}
	})

	t.Run("Flanking", func(t *testing.T) {
		out, err := cmdQueryFlanking([]string{"hero3", "goblin"}, deps)
		if err != nil {
			t.Fatalf("Flanking failed: %v", err)
		}
		if !strings.Contains(out, "IS flanked") || !strings.Contains(out, "Hero 2") {
			t.Errorf("Expected flanking: %s", out)
		}
	})

	t.Run("Cover", func(t *testing.T) {
		out, err := cmdQueryCover([]string{"hero1", "hero2"}, deps)
		if err != nil {
			t.Fatalf("Cover failed: %v", err)
		}
		if !strings.Contains(out, "**standard** cover") {
			t.Errorf("Expected standard cover: %s", out)
		}
	})

	t.Run("Distance No Path", func(t *testing.T) {
		s.Positions["isolated"] = &state.Zone{Name: "Isolated"}
		s.Entities["ghost"] = &state.EntityState{ID: "ghost", Position: "isolated"}
		deps.Store.Save(s)

		out, _ := cmdQueryDistance([]string{"hero1", "ghost"}, deps)
		if !strings.Contains(out, "Infinite") {
			t.Errorf("Expected Infinite distance, got: %s", out)
		}
	})

	t.Run("Flanking Not Engaged", func(t *testing.T) {
		out, _ := cmdQueryFlanking([]string{"hero1", "goblin"}, deps)
		if !strings.Contains(out, "not in melee range") {
			t.Errorf("Expected 'not in range' message, got: %s", out)
		}
	})
}
