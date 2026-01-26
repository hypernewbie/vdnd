package check

import (
	"testing"
)

func TestCounteractCheck(t *testing.T) {
	// targetLevel 3, targetDC 20.
	// counteractLevel 3, counteractMod +10.
	// Total needs 20, so roll 10+.

	tests := []struct {
		name string
		roll int
		want bool
		max  int
	}{
		{"Critical Success (Lvl+3=6)", 20, true, 6},
		{"Success (Lvl+1=4)", 10, true, 4},
		{"Failure (Lvl-1=2)", 5, false, 2},
		{"Critical Failure (Lvl-3=0)", 1, false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := CounteractCheckWithRoll(tt.roll, 3, 10, 3, 20)
			if res.MaxLevelAffected != tt.max {
				t.Errorf("%s: MaxLevelAffected = %d, want %d", tt.name, res.MaxLevelAffected, tt.max)
			}
			if res.CanCounteract != tt.want {
				t.Errorf("%s: CanCounteract = %v, want %v", tt.name, res.CanCounteract, tt.want)
			}
		})
	}
}

func TestCounteractCheck_Clamping(t *testing.T) {
	// Counteract Level 1.
	// Failure: 1 - 1 = 0.
	// Crit Failure: 1 - 3 = -2 -> 0.
	res := CounteractCheckWithRoll(1, 1, 10, 1, 20)
	if res.MaxLevelAffected != 0 {
		t.Errorf("Expected MaxLevel 0 for Crit Failure with Level 1, got %d", res.MaxLevelAffected)
	}
	if res.CanCounteract {
		t.Error("Crit Failure should NEVER counteract")
	}

	res = CounteractCheckWithRoll(5, 1, 10, 0, 20)
	if res.MaxLevelAffected != 0 {
		t.Errorf("Expected MaxLevel 0 for Failure with Level 1, got %d", res.MaxLevelAffected)
	}
	if !res.CanCounteract {
		t.Error("Failure Level 1 should counteract Level 0")
	}
}

func TestCounteractCheck_NegativeInput(t *testing.T) {
	tests := []struct {
		name          string
		counteractLvl int
		targetLvl     int
		expectedMax   int // Expected MaxLevelAffected after clamping (assuming Crit Success +3)
	}{
		{"Negative counteract level", -5, 3, 3}, // -5 becomes 0, 0 + 3 = 3
		{"Negative target level", 5, -3, 8},     // 5 + 3 = 8
		{"Both negative", -2, -5, 3},            // -2 becomes 0, 0 + 3 = 3
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Using a roll of 20 that gives CriticalSuccess (+3)
			result := CounteractCheckWithRoll(20, tt.counteractLvl, 15, tt.targetLvl, 10)
			if result.MaxLevelAffected != tt.expectedMax {
				t.Errorf("%s: expected max level %d, got %d", tt.name, tt.expectedMax, result.MaxLevelAffected)
			}

			// Adjust targetLevel for comparison (since system treats negative as 0 for success check)
			effectiveTarget := tt.targetLvl
			if effectiveTarget < 0 {
				effectiveTarget = 0
			}

			if (effectiveTarget <= result.MaxLevelAffected && result.Degree > CriticalFailure) != result.CanCounteract {
				t.Errorf("%s: CanCounteract mismatch, expected %v, got %v",
					tt.name, effectiveTarget <= result.MaxLevelAffected, result.CanCounteract)
			}
		})
	}
}