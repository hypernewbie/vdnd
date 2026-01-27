package cli

import (
	"strings"
	"testing"
	"uaa/vdnd/internal/state"
)

func TestSkillActions(t *testing.T) {
	deps := NewTestDeps(t)
	roller := deps.Roller.(*FixedRoller)

	s := &state.GameState{
		Entities: map[string]*state.EntityState{
			"fighter": {
				ID:    "fighter",
				Name:  "Fighter",
				Level: 1,
				Skills: map[string]int{
					"athletics":    10,
					"intimidation": 8,
				},
			},
			"goblin": {
				ID:      "goblin",
				Name:    "Goblin",
				Reflex:  7,  // DC 17
				Fortitude: 6, // DC 16
				Will:    5,  // DC 15
				HP:      15,
			},
		},
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	t.Run("Trip Success", func(t *testing.T) {
		// Roll 8 + 10 = 18 vs Reflex DC 17 -> Success
		roller.Results = []int{8}
		roller.Index = 0

		out, err := cmdActionTrip([]string{"fighter", "goblin"}, deps)
		if err != nil {
			t.Fatalf("Trip failed: %v", err)
		}
		if !strings.Contains(out, "Success") || !strings.Contains(out, "Prone") {
			t.Errorf("Unexpected output: %s", out)
		}

		st, _ := deps.Store.Load()
		found := false
		for _, c := range st.Entities["goblin"].Conditions {
			if c.ID == "prone" { found = true }
		}
		if !found { t.Error("Goblin should be Prone") }
	})

	t.Run("Demoralize Crit Success", func(t *testing.T) {
		// Roll 19 + 8 = 27 vs Will DC 15 -> Critical Success
		roller.Results = []int{19}
		roller.Index = 0

		out, err := cmdActionDemoralize([]string{"fighter", "goblin"}, deps)
		if err != nil {
			t.Fatalf("Demoralize failed: %v", err)
		}
		if !strings.Contains(out, "Critical Success") || !strings.Contains(out, "Frightened 2") {
			t.Errorf("Unexpected output: %s", out)
		}

		st, _ := deps.Store.Load()
		found := false
		for _, c := range st.Entities["goblin"].Conditions {
			if c.ID == "frightened" && c.Value == 2 { found = true }
		}
		if !found { t.Error("Goblin should be Frightened 2") }
	})

	t.Run("Grapple Fail", func(t *testing.T) {
		// Roll 2 + 10 = 12 vs Fort DC 16 -> Failure
		roller.Results = []int{2}
		roller.Index = 0

		out, err := cmdActionGrapple([]string{"fighter", "goblin"}, deps)
		if err != nil {
			t.Fatalf("Grapple failed: %v", err)
		}
		if !strings.Contains(out, "Failure") {
			t.Errorf("Unexpected output: %s", out)
		}

		st, _ := deps.Store.Load()
		for _, c := range st.Entities["goblin"].Conditions {
			if c.ID == "grabbed" { t.Error("Goblin should NOT be Grabbed") }
		}
	})

	t.Run("Shove Success", func(t *testing.T) {
		// Roll 15 + 10 = 25 vs Fort DC 16 -> Success
		roller.Results = []int{15}
		roller.Index = 0
		out, _ := cmdActionSkill([]string{"shove", "fighter", "goblin"}, deps)
		if !strings.Contains(out, "Success") || !strings.Contains(out, "5ft back") {
			t.Errorf("Unexpected output: %s", out)
		}
	})

	t.Run("Hide and Seek", func(t *testing.T) {
		// Hide: Roll 15 + 0 (fallback mod) = 15 vs DC 15 -> Success
		roller.Results = []int{15}
		roller.Index = 0
		cmdActionSkill([]string{"hide", "fighter"}, deps)

		st, _ := deps.Store.Load()
		found := false
		for _, c := range st.Entities["fighter"].Conditions {
			if c.ID == "hidden" { found = true }
		}
		if !found { t.Error("Fighter should be hidden") }

		// Seek: Roll 15 + 0 vs Stealth DC 10 (fallback) -> Success
		roller.Results = []int{15}
		roller.Index = 0
		out, _ := cmdActionSkill([]string{"seek", "goblin", "fighter"}, deps)
		if !strings.Contains(out, "no longer **Hidden**") {
			t.Errorf("Unexpected output: %s", out)
		}
	})

	t.Run("Unknown Action", func(t *testing.T) {
		_, err := cmdActionSkill([]string{"dance", "fighter"}, deps)
		if err == nil || !strings.Contains(err.Error(), "unknown skill action") {
			t.Errorf("Expected 'unknown skill action' error, got: %v", err)
		}
	})
}
