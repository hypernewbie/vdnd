package cli

import (
	"strings"
	"testing"
	"uaa/vdnd/internal/state"
	"uaa/vdnd/pkg/rules/ability"
)

func TestActionStrike(t *testing.T) {
	deps := NewTestDeps(t)
	roller := deps.Roller.(*FixedRoller)

	// Setup state
	s := &state.GameState{
		SceneName: "Test Scene",
		Entities: map[string]*state.EntityState{
			"hero": {
				ID:    "hero",
				Name:  "Hero",
				Level: 1,
				HP:    20,
				MaxHP: 20,
				AC:    15,
				Abilities: ability.AbilityScores{
					Strength: 4,
				},
				WieldedWeapons: []state.WeaponState{
					{ID: "sword", Damage: "1d8+0", DamageType: "slashing"},
				},
			},
			"goblin": {
				ID:    "goblin",
				Name:  "Goblin",
				Level: 1,
				HP:    15,
				MaxHP: 15,
				AC:    13,
			},
		},
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	t.Run("Strike Hit", func(t *testing.T) {
		roller.Results = []int{10, 5}
		roller.Index = 0
		out, err := cmdActionStrike([]string{"hero", "goblin"}, deps)
		if err != nil { t.Fatalf("cmdActionStrike failed: %v", err) }
		if !strings.Contains(out, "Success") { t.Errorf("Expected Success, got: %s", out) }
		if !strings.Contains(out, "**9** slashing") { t.Errorf("Expected 9 damage, got: %s", out) }
		st, _ := deps.Store.Load()
		if st.Entities["goblin"].HP != 6 { t.Errorf("Expected Goblin HP 6, got %d", st.Entities["goblin"].HP) }
		if st.AttacksMade != 1 { t.Errorf("Expected AttacksMade 1, got %d", st.AttacksMade) }
	})

	t.Run("Strike Critical", func(t *testing.T) {
		s.Entities["goblin"].HP = 15
		s.AttacksMade = 0
		deps.Store.Save(s)
		roller.Results = []int{20, 8}
		roller.Index = 0
		out, err := cmdActionStrike([]string{"hero", "goblin"}, deps)
		if err != nil { t.Fatalf("cmdActionStrike failed: %v", err) }
		if !strings.Contains(out, "Critical Success") { t.Errorf("Expected Critical Success, got: %s", out) }
		if !strings.Contains(out, "**24** slashing") { t.Errorf("Expected 24 damage, got: %s", out) }
		st, _ := deps.Store.Load()
		if st.Entities["goblin"].HP != 0 { t.Errorf("Expected Goblin HP 0, got %d", st.Entities["goblin"].HP) }
	})

	t.Run("Strike with MAP", func(t *testing.T) {
		s.Entities["goblin"].HP = 15
		s.AttacksMade = 1
		deps.Store.Save(s)
		roller.Results = []int{10}
		roller.Index = 0
		out, err := cmdActionStrike([]string{"hero", "goblin"}, deps)
		if err != nil { t.Fatalf("cmdActionStrike failed: %v", err) }
		if !strings.Contains(out, "Failure") { t.Errorf("Expected Failure, got: %s", out) }
		st, _ := deps.Store.Load()
		if st.AttacksMade != 2 { t.Errorf("Expected AttacksMade 2, got %d", st.AttacksMade) }
	})

	t.Run("Strike with MAP -10", func(t *testing.T) {
		s.Entities["goblin"].HP = 15
		s.AttacksMade = 2
		deps.Store.Save(s)
		roller.Results = []int{10}
		roller.Index = 0
		out, err := cmdActionStrike([]string{"hero", "goblin"}, deps)
		if err != nil { t.Fatalf("cmdActionStrike failed: %v", err) }
		if !strings.Contains(out, "Failure") { t.Errorf("Expected Failure, got: %s", out) }
		st, _ := deps.Store.Load()
		if st.AttacksMade != 3 { t.Errorf("Expected AttacksMade 3, got %d", st.AttacksMade) }
	})
}

func TestActionMovement(t *testing.T) {
	deps := NewTestDeps(t)
	s := &state.GameState{
		SceneName: "Test Scene",
		Positions: map[string]*state.Zone{
			"A": {Name: "Zone A", Adjacent: []string{"B"}},
			"B": {Name: "Zone B", Adjacent: []string{"A"}},
		},
		Entities: map[string]*state.EntityState{
			"hero": {
				ID:       "hero",
				Name:     "Hero",
				Position: "A",
			},
		},
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	t.Run("Stride", func(t *testing.T) {
		out, err := cmdActionStride([]string{"hero", "--to", "B"}, deps)
		if err != nil { t.Fatalf("Stride failed: %v", err) }
		if !strings.Contains(out, "strided from **A** to **B**") { t.Errorf("Unexpected output: %s", out) }
		st, _ := deps.Store.Load()
		if st.Entities["hero"].Position != "B" { t.Errorf("Expected position B, got %s", st.Entities["hero"].Position) }
	})

	t.Run("Step", func(t *testing.T) {
		out, err := cmdActionStep([]string{"hero", "--to", "A"}, deps)
		if err != nil { t.Fatalf("Step failed: %v", err) }
		if !strings.Contains(out, "stepped from **B** to **A**") { t.Errorf("Unexpected output: %s", out) }
		st, _ := deps.Store.Load()
		if st.Entities["hero"].Position != "A" { t.Errorf("Expected position A, got %s", st.Entities["hero"].Position) }
	})
}

func TestActionRaiseShield(t *testing.T) {
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

	out, err := cmdActionRaiseShield([]string{"hero"}, deps)
	if err != nil { t.Fatalf("RaiseShield failed: %v", err) }
	if !strings.Contains(out, "raised their shield (+2 AC)") { t.Errorf("Unexpected output: %s", out) }

	st, _ := deps.Store.Load()
	if st.Entities["hero"].GetAC() != 17 { t.Errorf("Expected effective AC 17, got %d", st.Entities["hero"].GetAC()) }
	if st.Entities["hero"].AC != 15 { t.Errorf("Expected base AC 15, got %d", st.Entities["hero"].AC) }
	if !st.Entities["hero"].RaisedShield { t.Error("Expected RaisedShield true") }
	
	found := false
	for _, c := range st.Entities["hero"].Conditions {
		if c.ID == "raised_shield" { found = true; break }
	}
	if !found { t.Error("Condition raised_shield not found") }
}