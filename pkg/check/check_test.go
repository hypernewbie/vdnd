package check

import "testing"

func TestPerformCheckWithRoll(t *testing.T) {
	tests := []struct {
		name         string
		naturalRoll  int
		baseModifier int
		extraMods    []Modifier
		dc           int
		expectedDeg  DegreeOfSuccess
	}{
		{"Simple success", 15, 5, []Modifier{}, 15, Success},
		{"Simple failure", 10, 3, []Modifier{}, 15, Failure},
		{"Crit success by +10", 12, 8, []Modifier{}, 10, CriticalSuccess},
		{"Crit failure by -10", 3, -2, []Modifier{}, 15, CriticalFailure},
		{"Nat 20 upgrades success", 20, 0, []Modifier{}, 25, Success}, // 20 vs 25 = fail, nat 20 -> success
		{"Nat 20 upgrades crit", 20, 5, []Modifier{}, 15, CriticalSuccess},
		{"Nat 1 downgrades", 1, 10, []Modifier{}, 10, Failure}, // 11 vs 10 = success, nat 1 -> failure
		{"Nat 1 can't go below crit fail", 1, -5, []Modifier{}, 20, CriticalFailure},
		{"Nat 20 can success even vs high DC", 20, 0, []Modifier{}, 35, Failure}, // 20 vs 35 = crit fail, nat 20 -> failure
		{"Exactly meet DC", 15, 0, []Modifier{}, 15, Success},
		{"One below DC", 14, 0, []Modifier{}, 15, Failure},
		{"Beat DC by exactly 10", 10, 10, []Modifier{}, 10, CriticalSuccess},
		{"Fail DC by exactly 10", 10, -5, []Modifier{}, 15, CriticalFailure},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PerformCheckWithRoll(tt.naturalRoll, tt.baseModifier, tt.extraMods, tt.dc)
			if got.Degree != tt.expectedDeg {
				t.Errorf("PerformCheckWithRoll() degree = %v, want %v (total %d vs DC %d, natural %d)", got.Degree, tt.expectedDeg, got.Total, tt.dc, tt.naturalRoll)
			}
		})
	}
}

func TestDetermineDegree(t *testing.T) {
	tests := []struct {
		name        string
		naturalRoll int
		total       int
		dc          int
		expected    DegreeOfSuccess
	}{
		{"Nat 20, would crit anyway", 20, 35, 20, CriticalSuccess},
		{"Nat 20, would fail by 10+", 20, 22, 40, Failure},              // Total 22 vs DC 40 is crit fail, +1 = fail
		{"Nat 1, would succeed by 10+", 1, 25, 15, Success},             // Total 25 vs DC 15 is crit success, -1 = success
		{"Nat 1, would fail", 1, 8, 15, CriticalFailure},                // Total 8 vs DC 15 is fail, -1 = crit fail
		{"Nat 20, normal success to crit", 20, 20, 15, CriticalSuccess}, // Total 20 vs DC 15 is success, +1 = crit success
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetermineDegree(tt.naturalRoll, tt.total, tt.dc); got != tt.expected {
				t.Errorf("DetermineDegree() = %v, want %v", got, tt.expected)
			}
		})
	}
}
