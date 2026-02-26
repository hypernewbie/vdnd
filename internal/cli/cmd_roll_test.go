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

	t.Run("Roll Basic", func(t *testing.T) {
		roller.Results = []int{3, 5}
		roller.Index = 0
		out, err := cmdRoll([]string{"2d6+4"}, deps)
		if err != nil {
			t.Fatalf("Roll failed: %v", err)
		}
		// New format: Rolled **2d6+4**: [3, 5] +4 = **12**
		if !strings.Contains(out, "[3, 5] +4") || !strings.Contains(out, "**12**") {
			t.Errorf("Unexpected output format: %s", out)
		}
	})

	t.Run("Roll Shorthand", func(t *testing.T) {
		roller.Results = []int{15}
		roller.Index = 0
		out, err := cmdRoll([]string{"d20+5"}, deps)
		if err != nil {
			t.Fatalf("Roll failed: %v", err)
		}
		if !strings.Contains(out, "[15] +5") || !strings.Contains(out, "**20**") {
			t.Errorf("Unexpected output format: %s", out)
		}
	})

	t.Run("Roll Multiple Groups", func(t *testing.T) {
		roller.Results = []int{4, 3}
		roller.Index = 0
		out, err := cmdRoll([]string{"1d8+1d6"}, deps)
		if err != nil {
			t.Fatalf("Roll failed: %v", err)
		}
		if !strings.Contains(out, "[4] [3]") || !strings.Contains(out, "**7**") {
			t.Errorf("Unexpected output format: %s", out)
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
