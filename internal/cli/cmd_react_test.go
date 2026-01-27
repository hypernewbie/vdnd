package cli

import (
	"strings"
	"testing"
	"uaa/vdnd/internal/state"
	"uaa/vdnd/pkg/rules/ability"
)

func TestReactions(t *testing.T) {
	deps := NewTestDeps(t)
	roller := deps.Roller.(*FixedRoller)

	// Setup state: Fighter and Goblin in same room
	s := &state.GameState{
		SceneName: "Reaction Test",
		Positions: map[string]*state.Zone{
			"room1": {Name: "Room 1"},
			"room2": {Name: "Room 2"},
		},
		Entities: map[string]*state.EntityState{
			"fighter": {
				ID:       "fighter",
				Name:     "Fighter",
				Level:    1,
				Position: "room1",
				AC:       18,
				Abilities: ability.AbilityScores{Strength: 4},
				WieldedWeapons: []state.WeaponState{
					{ID: "sword", Damage: "1d8+0", DamageType: "slashing"},
				},
				Reactions: []string{"attack_of_opportunity"},
			},
			"goblin": {
				ID:       "goblin",
				Name:     "Goblin",
				Level:    1,
				Position: "room1",
				HP:       15,
				MaxHP:    15,
				AC:       13,
			},
		},
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	t.Run("Stride Triggers Reaction", func(t *testing.T) {
		out, err := cmdActionStride([]string{"goblin", "--to", "room2"}, deps)
		if err != nil {
			t.Fatalf("Stride failed: %v", err)
		}
		if !strings.Contains(out, "Movement paused") {
			t.Errorf("Expected movement to be paused, got: %s", out)
		}

		st, _ := deps.Store.Load()
		if len(st.PendingEvents) == 0 {
			t.Fatal("Expected a pending event")
		}
		if st.Entities["goblin"].Position != "room1" {
			t.Errorf("Goblin should still be in room1, got %s", st.Entities["goblin"].Position)
		}
	})

	t.Run("Pending Command", func(t *testing.T) {
		out, err := cmdPending([]string{}, deps)
		if err != nil {
			t.Fatalf("Pending failed: %v", err)
		}
		if !strings.Contains(out, "Pending Event: movement") {
			t.Errorf("Unexpected pending output: %s", out)
		}
		if !strings.Contains(out, "Fighter") {
			t.Errorf("Expected Fighter in reactors list: %s", out)
		}
	})

	t.Run("React - Critical Hit Disrupts", func(t *testing.T) {
		// Reset state for clean run
		s.PendingEvents = nil
		s.Entities["goblin"].Position = "room1"
		s.ReactionsUsed = make(map[string]bool)
		deps.Store.Save(s)

		// Trigger
		cmdActionStride([]string{"goblin", "--to", "room2"}, deps)

		// Roll natural 20 for AoO
		roller.Results = []int{20, 8} // Nat 20, Max Damage 8
		roller.Index = 0

		out, err := cmdReact([]string{"fighter", "attack_of_opportunity"}, deps)
		if err != nil {
			t.Fatalf("React failed: %v", err)
		}

		if !strings.Contains(strings.ToLower(out), "critical hit") || !strings.Contains(strings.ToLower(out), "disrupted") {
			t.Errorf("Expected disruption, got: %s", out)
		}

		st, _ := deps.Store.Load()
		if len(st.PendingEvents) != 0 {
			t.Error("Pending event should have been removed")
		}
		if st.Entities["goblin"].Position != "room1" {
			t.Error("Goblin should NOT have moved")
		}
		if st.ReactionsUsed["fighter"] != true {
			t.Error("Fighter's reaction should be marked as used")
		}
	})

	t.Run("React - Skip Resumes", func(t *testing.T) {
		// Reset state
		s.PendingEvents = nil
		s.Entities["goblin"].Position = "room1"
		s.ReactionsUsed = make(map[string]bool)
		deps.Store.Save(s)

		// Trigger
		cmdActionStride([]string{"goblin", "--to", "room2"}, deps)

		out, err := cmdReactSkipAll([]string{}, deps)
		if err != nil {
			t.Fatalf("ReactSkipAll failed: %v", err)
		}

		if !strings.Contains(out, "finished moving from **room1** to **room2**") {
			t.Errorf("Expected movement to finish, got: %s", out)
		}

		st, _ := deps.Store.Load()
		if len(st.PendingEvents) != 0 {
			t.Error("Pending event should have been removed")
		}
		if st.Entities["goblin"].Position != "room2" {
			t.Errorf("Goblin should have moved to room2, got %s", st.Entities["goblin"].Position)
		}
	})

	t.Run("React - Skip Specific", func(t *testing.T) {
		// Reset state
		s.PendingEvents = nil
		s.Entities["goblin"].Position = "room1"
		s.ReactionsUsed = make(map[string]bool)
		
		// Add another reactor
		s.Entities["fighter2"] = &state.EntityState{
			ID: "fighter2", Name: "Fighter 2", Position: "room1", Reactions: []string{"attack_of_opportunity"},
		}
		deps.Store.Save(s)

		// Trigger
		cmdActionStride([]string{"goblin", "--to", "room2"}, deps)

		// Skip one
		out, _ := cmdReactSkip([]string{"fighter"}, deps)
		if !strings.Contains(out, "1 reactors remaining") {
			t.Errorf("Expected 1 reactor left: %s", out)
		}

		// Skip last
		out, _ = cmdReactSkip([]string{"fighter2"}, deps)
		if !strings.Contains(out, "finished moving") {
			t.Errorf("Expected movement to finish: %s", out)
		}
	})

	t.Run("React Error States", func(t *testing.T) {
		// No pending events
		s.PendingEvents = nil
		deps.Store.Save(s)
		
		_, err := cmdReact([]string{"fighter", "aoo"}, deps)
		if err == nil { t.Error("Expected error with no pending events") }

		_, err = cmdReactSkip([]string{"fighter"}, deps)
		if err == nil { t.Error("Expected error with no pending events") }

		_, err = cmdReactSkipAll([]string{}, deps)
		if err == nil { t.Error("Expected error with no pending events") }
	})
}
