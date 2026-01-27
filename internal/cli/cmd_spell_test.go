package cli

import (
	"strings"
	"testing"
	"uaa/vdnd/internal/state"
)

func TestActionCast(t *testing.T) {
	deps := NewTestDeps(t)
	roller := deps.Roller.(*FixedRoller)

	s := &state.GameState{
		Entities: map[string]*state.EntityState{
			"wizard": {
				ID:   "wizard",
				Name: "Wizard",
				Position: "backstage",
			},
			"goblin1": {
				ID:       "goblin1",
				Name:     "Goblin 1",
				Position: "room_a",
				HP:       15,
				MaxHP:    15,
				Reflex:   7, // DC 17
			},
			"goblin2": {
				ID:       "goblin2",
				Name:     "Goblin 2",
				Position: "room_a",
				HP:       15,
				MaxHP:    15,
				Reflex:   7,
			},
		},
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	t.Run("Fireball AoE", func(t *testing.T) {
		// Damage roll (6d6): [3, 4, 2, 5, 1, 5] = 20
		// Goblin 1: Roll 5 (Total 12) -> Fail. Takes 20.
		// Goblin 2: Roll 15 (Total 22) -> Success. Takes 10.
		roller.Results = []int{3, 4, 2, 5, 1, 5, 5, 15}
		roller.Index = 0

		out, err := cmdActionCast([]string{"wizard", "fireball", "--zone", "room_a", "--dc", "20"}, deps)
		if err != nil {
			t.Fatalf("Cast failed: %v", err)
		}

		if !strings.Contains(out, "Goblin 1") || !strings.Contains(out, "Goblin 2") {
			t.Errorf("Expected both goblins in output: %s", out)
		}
		if !strings.Contains(out, "Damage:** **20**") {
			t.Errorf("Expected Goblin 1 to take 20 damage: %s", out)
		}
		if !strings.Contains(out, "Damage:** **10**") {
			t.Errorf("Expected Goblin 2 to take 10 damage: %s", out)
		}

		st, _ := deps.Store.Load()
		if st.Entities["goblin1"].HP != 0 {
			t.Errorf("Goblin 1 HP should be 0, got %d", st.Entities["goblin1"].HP)
		}
		if st.Entities["goblin2"].HP != 5 {
			t.Errorf("Goblin 2 HP should be 5, got %d", st.Entities["goblin2"].HP)
		}
	})

	t.Run("Magic Missile Auto-hit", func(t *testing.T) {
		// Reset state
		s.Entities["goblin1"].HP = 15
		deps.Store.Save(s)

		// Damage roll (1d4+1): [3] + 1 = 4
		roller.Results = []int{3}
		roller.Index = 0

		out, err := cmdActionCast([]string{"wizard", "magic_missile", "--target", "goblin1"}, deps)
		if err != nil {
			t.Fatalf("Cast failed: %v", err)
		}

		if !strings.Contains(out, "Damage:** **4**") {
			t.Errorf("Expected 4 damage, got: %s", out)
		}

		st, _ := deps.Store.Load()
		if st.Entities["goblin1"].HP != 11 {
			t.Errorf("Expected 11 HP, got %d", st.Entities["goblin1"].HP)
		}
	})

    t.Run("Chilling Darkness Attack Roll", func(t *testing.T) {
        // Goblin 1 AC: 10 (base for tests if not set, wait, EntityState.AC defaults to 0 if not set, but Validate requires > 0)
        s.Entities["goblin1"].AC = 13
        deps.Store.Save(s)

        // Attack roll: 10 + 5 (mod) = 15 vs AC 13 -> Success
        // Damage (5d6): [1, 2, 3, 4, 5] = 15
        roller.Results = []int{10, 1, 2, 3, 4, 5}
        roller.Index = 0

        out, err := cmdActionCast([]string{"wizard", "chilling_darkness", "--target", "goblin1", "--attack_mod", "5"}, deps)
        if err != nil {
            t.Fatalf("Cast failed: %v", err)
        }

        if !strings.Contains(out, "Success") || !strings.Contains(out, "Damage:** **15**") {
            t.Errorf("Unexpected output: %s", out)
        }
    })

	t.Run("Heal Utility", func(t *testing.T) {
		s.Entities["goblin1"].HP = 5
		deps.Store.Save(s)

		// Heal (1d8+8): [4] + 8 = 12
		roller.Results = []int{4}
		roller.Index = 0

		out, err := cmdActionCast([]string{"wizard", "heal", "--target", "goblin1"}, deps)
		if err != nil { t.Fatalf("Err: %v", err) }
		if !strings.Contains(out, "healed for **12**") { t.Errorf("Unexpected: %s", out) }

		st, _ := deps.Store.Load()
		if st.Entities["goblin1"].HP != 15 { t.Errorf("Expected 15 HP, got %d", st.Entities["goblin1"].HP) }
	})

	t.Run("Generic Spell", func(t *testing.T) {
		// Generic spell: rolls damage (1d6) then save (1d20)
		// Damage: 4
		// Roll 4 + 7 = 11 vs DC 15 -> Failure
		roller.Results = []int{4, 4}
		roller.Index = 0
		out, _ := cmdActionCast([]string{"wizard", "custom", "--target", "goblin1", "--type", "save", "--save", "reflex", "--dc", "15", "--damage", "1d6", "--dmg_type", "fire"}, deps)
		if !strings.Contains(out, "Failure") { t.Errorf("Unexpected: %s", out) }
	})
}
