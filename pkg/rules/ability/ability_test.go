package ability

import "testing"

func TestModifierFromScore(t *testing.T) {
	tests := []struct {
		score    int
		expected int
	}{
		{1, -5}, {2, -4}, {3, -4}, {4, -3}, {5, -3},
		{6, -2}, {7, -2}, {8, -1}, {9, -1}, {10, 0},
		{11, 0}, {12, 1}, {13, 1}, {14, 2}, {15, 2},
		{16, 3}, {17, 3}, {18, 4}, {19, 4}, {20, 5},
		{21, 5}, {22, 6},
	}
	for _, tt := range tests {
		if got := ModifierFromScore(tt.score); got != tt.expected {
			t.Errorf("ModifierFromScore(%d) = %d, want %d", tt.score, got, tt.expected)
		}
	}
}

func TestAbilityScores_Get(t *testing.T) {
	scores := AbilityScores{10, 14, 12, 8, 16, 18}
	tests := []struct {
		ability  Ability
		expected int
	}{
		{Strength, 10},
		{Dexterity, 14},
		{Constitution, 12},
		{Intelligence, 8},
		{Wisdom, 16},
		{Charisma, 18},
	}
	for _, tt := range tests {
		if got := scores.Get(tt.ability); got != tt.expected {
			t.Errorf("AbilityScores.Get(%s) = %d, want %d", tt.ability, got, tt.expected)
		}
	}
}

func TestAbilityScores_Modifier(t *testing.T) {
	scores := AbilityScores{10, 14, 12, 8, 16, 18}
	tests := []struct {
		ability  Ability
		expected int
	}{
		{Strength, 0},
		{Dexterity, 2},
		{Intelligence, -1},
		{Wisdom, 3},
		{Charisma, 4},
	}
	for _, tt := range tests {
		if got := scores.Modifier(tt.ability); got != tt.expected {
			t.Errorf("AbilityScores.Modifier(%s) = %d, want %d", tt.ability, got, tt.expected)
		}
	}
}

func TestProficiencyRank_Bonus(t *testing.T) {
	tests := []struct {
		rank     ProficiencyRank
		level    int
		expected int
	}{
		{Untrained, 1, 0},
		{Untrained, 10, 0},
		{Untrained, 20, 0},
		{Trained, -1, 1},
		{Trained, 1, 3},
		{Trained, 5, 7},
		{Trained, 10, 12},
		{Trained, 20, 22},
		{Expert, 1, 5},
		{Expert, 5, 9},
		{Expert, 10, 14},
		{Master, 1, 7},
		{Master, 10, 16},
		{Legendary, 1, 9},
		{Legendary, 10, 18},
		{Legendary, 20, 28},
	}
	for _, tt := range tests {
		t.Run(tt.rank.String(), func(t *testing.T) {
			if got := tt.rank.Bonus(tt.level); got != tt.expected {
				t.Errorf("%s.Bonus(%d) = %d, want %d", tt.rank, tt.level, got, tt.expected)
			}
		})
	}
}

func TestCalculateModifier(t *testing.T) {
	tests := []struct {
		score    int
		rank     ProficiencyRank
		level    int
		expected int
	}{
		{10, Untrained, 1, 0},
		{14, Trained, 1, 5},
		{18, Expert, 5, 13},
		{8, Master, 10, 15},
		{20, Legendary, 20, 33},
	}
	for _, tt := range tests {
		if got := CalculateModifier(tt.score, tt.rank, tt.level); got != tt.expected {
			t.Errorf("CalculateModifier(%d, %s, %d) = %d, want %d", tt.score, tt.rank, tt.level, got, tt.expected)
		}
	}
}

func TestCalculateDC(t *testing.T) {
	tests := []struct {
		mod      int
		expected int
	}{
		{0, 10},
		{5, 15},
		{15, 25},
		{-2, 8},
		{33, 43},
	}
	for _, tt := range tests {
		if got := CalculateDC(tt.mod); got != tt.expected {
			t.Errorf("CalculateDC(%d) = %d, want %d", tt.mod, got, tt.expected)
		}
	}
}
