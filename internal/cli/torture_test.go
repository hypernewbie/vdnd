package cli

import (
	"strings"
	"testing"
	"uaa/vdnd/internal/state"
	"uaa/vdnd/pkg/rules/ability"
)

func TestTortureDyingYoYo(t *testing.T) {
	deps := NewTestDeps(t)
	// roller := deps.Roller.(*FixedRoller)

	// Setup: Fighter at 1 HP
	s := &state.GameState{
		Entities: map[string]*state.EntityState{
			"fighter": {
				ID:    "fighter",
				Name:  "Fighter",
				HP:    1,
				MaxHP: 20,
			},
			"cleric": {
				ID:   "cleric",
				Name: "Cleric",
			},
		},
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	// 1. Damage: Fighter takes 5 dmg -> HP 0.
	t.Run("Fighter falls to 0 HP", func(t *testing.T) {
		out, err := cmdDamage([]string{"fighter", "5", "slashing"}, deps)
		if err != nil {
			t.Fatalf("Damage failed: %v", err)
		}
		if !strings.Contains(out, "falling! **Dying 1**") {
			t.Errorf("Expected Dying 1, got: %s", out)
		}
		st, _ := deps.Store.Load()
		if st.Entities["fighter"].HP != 0 {
			t.Errorf("Expected HP 0, got %d", st.Entities["fighter"].HP)
		}
	})

	// 2. Heal: Cleric heals Fighter (10 HP).
	t.Run("Fighter healed from Dying", func(t *testing.T) {
		out, err := cmdHeal([]string{"fighter", "10", "--from", "cleric"}, deps)
		if err != nil {
			t.Fatalf("Heal failed: %v", err)
		}
		if !strings.Contains(out, "**Dying** removed. **Wounded 1** added.") {
			t.Errorf("Expected Wounded 1, got: %s", out)
		}
		st, _ := deps.Store.Load()
		if st.Entities["fighter"].HP != 10 {
			t.Errorf("Expected HP 10, got %d", st.Entities["fighter"].HP)
		}
	})

	// 3. Damage: Enemy hits Fighter again (15 dmg).
	t.Run("Fighter falls again while Wounded", func(t *testing.T) {
		out, err := cmdDamage([]string{"fighter", "15", "slashing"}, deps)
		if err != nil {
			t.Fatalf("Damage failed: %v", err)
		}
		// Dying 1 (base) + Wounded 1 = Dying 2
		if !strings.Contains(out, "falling! **Dying 2**") {
			t.Errorf("Expected Dying 2, got: %s", out)
		}
	})

	// 4. Crit Fail: Fighter takes damage while dying
	t.Run("Fighter takes damage while dying", func(t *testing.T) {
		// Taking damage while dying increases dying by 1 (or 2 if crit)
		out, err := cmdDamage([]string{"fighter", "1", "slashing"}, deps)
		if err != nil {
			t.Fatalf("Damage failed: %v", err)
		}
		if !strings.Contains(out, "**Dying** increased to 3") {
			t.Errorf("Expected Dying 3, got: %s", out)
		}
	})

	// 5. Crit Damage while dying
	t.Run("Fighter takes crit damage while dying", func(t *testing.T) {
		out, err := cmdDamage([]string{"fighter", "1", "slashing", "--crit"}, deps)
		if err != nil {
			t.Fatalf("Damage failed: %v", err)
		}
		// Dying 3 -> Dying 5 (Dead is usually 4, but let's see what our logic does)
		if !strings.Contains(out, "**Dying** increased to 5") {
			t.Errorf("Expected Dying 5, got: %s", out)
		}
	})
}

func TestTortureDamageKitchenSink(t *testing.T) {
	deps := NewTestDeps(t)

	s := &state.GameState{
		Entities: map[string]*state.EntityState{
			"monster": {
				ID:   "monster",
				Name: "Monster",
				HP:   100,
				MaxHP: 100,
				Immunities: []string{"fire"},
				Weaknesses: map[string]int{"cold": 5},
				Resistances: map[string]int{"slashing": 5},
			},
		},
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	t.Run("Fire Immunity", func(t *testing.T) {
		out, err := cmdDamage([]string{"monster", "20", "fire"}, deps)
		if err != nil { t.Fatalf("Err: %v", err) }
		if !strings.Contains(out, "Immune to **fire**!") { t.Errorf("Expected immunity: %s", out) }
		st, _ := deps.Store.Load()
		if st.Entities["monster"].HP != 100 { t.Error("HP should not change") }
	})

	t.Run("Slashing Resistance + Cold Weakness", func(t *testing.T) {
		// Slashing (5) -> Resist 5 -> 0
		out, err := cmdDamage([]string{"monster", "5", "slashing"}, deps)
		if err != nil { t.Fatalf("Err: %v", err) }
		if !strings.Contains(out, "- Resistance to **slashing**: -5 damage") { t.Errorf("Expected resist: %s", out) }
		
		// Cold (2) -> Weakness 5 -> 7
		out, err = cmdDamage([]string{"monster", "2", "cold"}, deps)
		if err != nil { t.Fatalf("Err: %v", err) }
		if !strings.Contains(out, "- Weakness to **cold**: +5 damage") { t.Errorf("Expected weakness: %s", out) }

		st, _ := deps.Store.Load()
		if st.Entities["monster"].HP != 93 { t.Errorf("Expected 93 HP, got %d", st.Entities["monster"].HP) }
	})
}

func TestTortureReactionDogpile(t *testing.T) {
	deps := NewTestDeps(t)
	roller := deps.Roller.(*FixedRoller)

	s := &state.GameState{
		Positions: map[string]*state.Zone{
			"room_a": {Name: "Room A", Adjacent: []string{"room_b"}},
			"room_b": {Name: "Room B", Adjacent: []string{"room_a"}},
		},
		Entities: map[string]*state.EntityState{
			"hero": {
				ID: "hero", Name: "Hero", Position: "room_a", HP: 50, MaxHP: 50, AC: 15,
			},
			"g1": {
				ID: "g1", Name: "G1", Position: "room_a", Reactions: []string{"attack_of_opportunity"},
				Abilities: ability.AbilityScores{Strength: 2},
				WieldedWeapons: []state.WeaponState{{ID: "spear", Damage: "1d6", DamageType: "piercing"}},
			},
			"g2": {
				ID: "g2", Name: "G2", Position: "room_a", Reactions: []string{"attack_of_opportunity"},
				Abilities: ability.AbilityScores{Strength: 2},
				WieldedWeapons: []state.WeaponState{{ID: "spear", Damage: "1d6", DamageType: "piercing"}},
			},
		},
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	t.Run("Stride triggers two reactors", func(t *testing.T) {
		out, _ := cmdActionStride([]string{"hero", "--to", "room_b"}, deps)
		if !strings.Contains(out, "G1") || !strings.Contains(out, "G2") {
			t.Errorf("Expected both goblins to react: %s", out)
		}
	})

	t.Run("First reaction - Success", func(t *testing.T) {
		// Roll 15 (Hit) + 4 Damage
		roller.Results = []int{15, 4}
		roller.Index = 0
		out, _ := cmdReact([]string{"g1", "attack_of_opportunity"}, deps)
		if !strings.Contains(out, "Damage:** **6**") { // 4 + 2 STR
			t.Errorf("Expected 6 damage: %s", out)
		}
		st, _ := deps.Store.Load()
		if len(st.PendingEvents[0].Reactors) != 1 {
			t.Error("Should have 1 reactor left")
		}
	})

	t.Run("Second reaction - Critical Disrupts", func(t *testing.T) {
		// Roll 20 (Crit) + 6 Damage
		roller.Results = []int{20, 6}
		roller.Index = 0
		out, _ := cmdReact([]string{"g2", "attack_of_opportunity"}, deps)
		if !strings.Contains(out, "disrupted") {
			t.Errorf("Expected disruption: %s", out)
		}
		st, _ := deps.Store.Load()
		if len(st.PendingEvents) != 0 {
			t.Error("Pending event should be gone")
		}
		if st.Entities["hero"].Position != "room_a" {
			t.Error("Hero should not have moved")
		}
	})
}
