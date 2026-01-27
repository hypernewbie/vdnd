package cli

import (
	"strings"
	"testing"
	"uaa/vdnd/internal/state"
)

func TestRollAndCheck(t *testing.T) {
	deps := NewTestDeps(t)
	roller := deps.Roller.(*FixedRoller)

	s := &state.GameState{
		Entities: map[string]*state.EntityState{
			"hero": {
				ID:   "hero",
				Name: "Hero",
				Skills: map[string]int{
					"stealth": 7,
				},
			},
		},
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	t.Run("Roll", func(t *testing.T) {
		roller.Results = []int{3, 5}
		roller.Index = 0
		out, err := cmdRoll([]string{"2d6+4"}, deps)
		if err != nil {
			t.Fatalf("Roll failed: %v", err)
		}
		if !strings.Contains(out, "**12**") {
			t.Errorf("Expected 12, got: %s", out)
		}
	})

	t.Run("Check with DC", func(t *testing.T) {
		// Roll 18 + 7 = 25 vs DC 20 -> Success
		roller.Results = []int{18}
		roller.Index = 0
		out, err := cmdCheck([]string{"hero", "stealth", "--dc", "20"}, deps)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		if !strings.Contains(out, "Success") || !strings.Contains(out, "**25**") {
			t.Errorf("Unexpected output: %s", out)
		}
	})

	t.Run("Check without DC", func(t *testing.T) {
		roller.Results = []int{10}
		roller.Index = 0
		out, err := cmdCheck([]string{"hero", "stealth"}, deps)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		if !strings.Contains(out, "**17**") {
			t.Errorf("Expected 17, got: %s", out)
		}
	})
}
