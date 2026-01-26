package check

import (
	"testing"
)

func TestCounteractCheck(t *testing.T) {
	tests := []struct {
		name            string
		counteractLvl  int
		roll            int
		mod             int
		dc              int
		expectedDegree  DegreeOfSuccess
		expectedMax     int
		targetLvl       int
		expectedSuccess bool
	}{
		// Core Counteract Tests
		{"Crit success, lower target", 5, 20, 15, 25, CriticalSuccess, 8, 5, true},
		{"Crit success, exact max", 5, 20, 15, 25, CriticalSuccess, 8, 8, true},
		{"Crit success, too high", 5, 20, 15, 25, CriticalSuccess, 8, 9, false},
		{"Success, lower target", 5, 15, 10, 20, Success, 6, 4, true},
		{"Success, exact max", 5, 15, 10, 20, Success, 6, 6, true},
		{"Success, too high", 5, 15, 10, 20, Success, 6, 7, false},
		{"Failure, low target", 5, 8, 5, 20, Failure, 4, 3, true},
		{"Failure, at limit", 5, 8, 5, 20, Failure, 4, 4, true},
		{"Failure, too high", 5, 8, 5, 20, Failure, 4, 5, false},
		{"Crit failure", 5, 1, 2, 25, CriticalFailure, 2, 2, true},
		{"Crit failure, too high", 5, 1, 2, 25, CriticalFailure, 2, 3, false},

		// Edge Cases
		{"Level 1, crit fail (clamped)", 1, 1, -10, 20, CriticalFailure, 0, 0, true},
		{"Level 2, failure", 2, 5, 0, 10, Failure, 1, 1, true},
		{"Level 0, crit fail", 0, 1, 0, 20, CriticalFailure, 0, 0, true},
		{"Level 10, crit success", 10, 20, 10, 10, CriticalSuccess, 13, 13, true},

		// Natural 1/20 Interaction
		{"Nat 20 upgrades success", 5, 20, 5, 20, CriticalSuccess, 8, 8, true},
		{"Nat 1 downgrades success", 5, 1, 19, 20, Failure, 4, 4, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CounteractCheckWithRoll(tt.roll, tt.counteractLvl, tt.mod, tt.targetLvl, tt.dc)

			if result.Degree != tt.expectedDegree {
				t.Errorf("%s: expected degree %v, got %v", tt.name, tt.expectedDegree, result.Degree)
			}
			if result.MaxLevelAffected != tt.expectedMax {
				t.Errorf("%s: expected max level %d, got %d", tt.name, tt.expectedMax, result.MaxLevelAffected)
			}
			if result.CanCounteract != tt.expectedSuccess {
				t.Errorf("%s: expected success %v, got %v", tt.name, tt.expectedSuccess, result.CanCounteract)
			}
		})
	}
}
