package cli

import (
	"strings"
	"testing"
	"uaa/vdnd/internal/state"
)

func TestDamageCommands(t *testing.T) {
	deps := NewTestDeps(t)
	s := &state.GameState{
		Entities: map[string]*state.EntityState{
			"skeleton": {
				ID:    "skeleton",
				Name:  "Skeleton",
				HP:    20,
				MaxHP: 20,
				Resistances: map[string]int{
					"slashing": 5,
					"cold":     5,
				},
				Weaknesses: map[string]int{
					"bludgeoning": 5,
				},
			},
			"hero": {
				ID:    "hero",
				Name:  "Hero",
				HP:    1,
				MaxHP: 20,
			},
		},
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	t.Run("Resistant Damage", func(t *testing.T) {
		// 10 slashing -> 10 - 5 = 5
		out, err := cmdDamage([]string{"skeleton", "10", "slashing"}, deps)
		if err != nil {
			t.Fatalf("Damage failed: %v", err)
		}
		if !strings.Contains(out, "Resistance to **slashing**: -5 damage") {
			t.Errorf("Unexpected output: %s", out)
		}
		st, _ := deps.Store.Load()
		if st.Entities["skeleton"].HP != 15 {
			t.Errorf("Expected HP 15, got %d", st.Entities["skeleton"].HP)
		}
	})

	t.Run("Weakness Damage", func(t *testing.T) {
		// 10 bludgeoning -> 10 + 5 = 15
		out, err := cmdDamage([]string{"skeleton", "10", "bludgeoning"}, deps)
		if err != nil {
			t.Fatalf("Damage failed: %v", err)
		}
		if !strings.Contains(out, "Weakness to **bludgeoning**: +5 damage") {
			t.Errorf("Unexpected output: %s", out)
		}
		st, _ := deps.Store.Load()
		if st.Entities["skeleton"].HP != 0 {
			t.Errorf("Expected HP 0, got %d", st.Entities["skeleton"].HP)
		}
	})

	t.Run("Dying Mechanics", func(t *testing.T) {
		// Hero 1 HP -> 5 piercing -> Dying 1
		out, err := cmdDamage([]string{"hero", "5", "piercing"}, deps)
		if err != nil {
			t.Fatalf("Damage failed: %v", err)
		}
		if !strings.Contains(out, "falling! **Dying 1**") {
			t.Errorf("Unexpected output: %s", out)
		}
		
		st, _ := deps.Store.Load()
		found := false
		for _, c := range st.Entities["hero"].Conditions {
			if c.ID == "dying" && c.Value == 1 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Dying condition not added: %+v", st.Entities["hero"].Conditions)
		}
	})

	t.Run("Healing from Dying", func(t *testing.T) {
		// Hero Dying 1 -> Heal 10 -> Dying removed, Wounded 1
		out, err := cmdHeal([]string{"hero", "10"}, deps)
		if err != nil {
			t.Fatalf("Heal failed: %v", err)
		}
		if !strings.Contains(out, "**Dying** removed. **Wounded 1** added.") {
			t.Errorf("Unexpected output: %s", out)
		}

		st, _ := deps.Store.Load()
		if st.Entities["hero"].HP != 10 {
			t.Errorf("Expected HP 10, got %d", st.Entities["hero"].HP)
		}
		dying := false
		wounded := false
		for _, c := range st.Entities["hero"].Conditions {
			if c.ID == "dying" { dying = true }
			if c.ID == "wounded" && c.Value == 1 { wounded = true }
		}
		if dying || !wounded {
			t.Errorf("Conditions incorrect: dying=%v, wounded=%v", dying, wounded)
		}
	})

	t.Run("Temp HP", func(t *testing.T) {
		// Hero HP 10 -> Temp HP 5 -> 10 slashing -> 5 Temp absorbed, 5 HP taken
		cmdTempHP([]string{"hero", "5"}, deps)
		out, err := cmdDamage([]string{"hero", "10", "slashing"}, deps)
		if err != nil {
			t.Fatalf("Damage failed: %v", err)
		}
		if !strings.Contains(out, "**Temp HP absorbed:** 5") {
			t.Errorf("Unexpected output: %s", out)
		}
		
		st, _ := deps.Store.Load()
		if st.Entities["hero"].HP != 5 {
			t.Errorf("Expected HP 5, got %d", st.Entities["hero"].HP)
		}
		if st.Entities["hero"].TempHP != 0 {
			t.Errorf("Expected TempHP 0, got %d", st.Entities["hero"].TempHP)
		}
	})
}
