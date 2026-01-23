package check

import "testing"

func TestCalculateTotal(t *testing.T) {
	tests := []struct {
		name      string
		modifiers []Modifier
		expected  int
	}{
		{
			name:      "Empty list",
			modifiers: []Modifier{},
			expected:  0,
		},
		{
			name: "Single bonus",
			modifiers: []Modifier{
				{2, BonusStatus, "Heroism"},
			},
			expected: 2,
		},
		{
			name: "Same type bonuses",
			modifiers: []Modifier{
				{2, BonusStatus, "Heroism"},
				{1, BonusStatus, "Bless"},
			},
			expected: 2,
		},
		{
			name: "Different type bonuses",
			modifiers: []Modifier{
				{2, BonusStatus, "Heroism"},
				{2, BonusItem, "Sword"},
			},
			expected: 4,
		},
		{
			name: "All types",
			modifiers: []Modifier{
				{1, BonusCircumstance, ""},
				{2, BonusItem, ""},
				{1, BonusStatus, ""},
			},
			expected: 4,
		},
		{
			name: "Single penalty",
			modifiers: []Modifier{
				{-2, BonusStatus, "Sickened"},
			},
			expected: -2,
		},
		{
			name: "Same type penalties",
			modifiers: []Modifier{
				{-2, BonusStatus, "Sickened"},
				{-1, BonusStatus, "Frightened"},
			},
			expected: -2,
		},
		{
			name: "Mixed bonus/penalty same type",
			modifiers: []Modifier{
				{2, BonusStatus, "Heroism"},
				{-1, BonusStatus, "Frightened"},
			},
			expected: 1,
		},
		{
			name: "Untyped penalties stack",
			modifiers: []Modifier{
				{-5, BonusUntyped, "MAP"},
				{-2, BonusUntyped, "Range"},
			},
			expected: -7,
		},
		{
			name: "Complex mix",
			modifiers: []Modifier{
				{2, BonusStatus, ""},
				{1, BonusCircumstance, ""},
				{-5, BonusUntyped, ""},
				{-2, BonusUntyped, ""},
				{-1, BonusStatus, ""},
			},
			expected: -5, // (2 status bonus) + (1 circum bonus) + (-1 status penalty) + (-5 untyped) + (-2 untyped) = 2 + 1 - 1 - 7 = -5
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateTotal(tt.modifiers); got != tt.expected {
				t.Errorf("CalculateTotal() = %v, want %v", got, tt.expected)
			}
		})
	}
}
