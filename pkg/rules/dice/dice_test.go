package dice

import (
	"math/rand"
	"testing"
)

func TestDieRoll_RollWithRNG(t *testing.T) {
	tests := []struct {
		name     string
		die      DieRoll
		expected int
	}{
		{
			name:     "Count 0",
			die:      DieRoll{Count: 0, Sides: 6, Modifier: 5},
			expected: 5,
		},
		{
			name:     "Count 1, Sides 1",
			die:      DieRoll{Count: 1, Sides: 1, Modifier: 5},
			expected: 6,
		},
		{
			name:     "Count 3, Sides 1",
			die:      DieRoll{Count: 3, Sides: 1, Modifier: 10},
			expected: 13,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(1))
			if got := tt.die.RollWithRNG(rng); got != tt.expected {
				t.Errorf("DieRoll.RollWithRNG() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		expr     string
		expected DieRoll
		wantErr  bool
	}{
		{"1d20", DieRoll{1, 20, 0}, false},
		{"2d6+4", DieRoll{2, 6, 4}, false},
		{"3d10-2", DieRoll{3, 10, -2}, false},
		{"invalid", DieRoll{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := Parse(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("Parse() = %v, want %v", got, tt.expected)
			}
		})
	}
}
