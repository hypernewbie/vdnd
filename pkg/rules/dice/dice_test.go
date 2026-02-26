package dice

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		expr     string
		wantMod  int
		wantGrps []DiceGroup
		wantErr  bool
	}{
		{"1d20", 0, []DiceGroup{{1, 20}}, false},
		{"2d6+4", 4, []DiceGroup{{2, 6}}, false},
		{"d20+7", 7, []DiceGroup{{1, 20}}, false},
		{"1d8+1d6+2", 2, []DiceGroup{{1, 8}, {1, 6}}, false},
		{"+5", 5, nil, false},
		{"-2", -2, nil, false},
		{"1D12", 0, []DiceGroup{{1, 12}}, false},
		{"2d4 + 2d4", 0, []DiceGroup{{2, 4}, {2, 4}}, false},
		{"d10 - 1", -1, []DiceGroup{{1, 10}}, false},
		{"", 0, nil, true},
		{"invalid", 0, nil, true},
		{"2d6+-", 0, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := Parse(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Modifier != tt.wantMod {
				t.Errorf("Parse() gotMod = %v, want %v", got.Modifier, tt.wantMod)
			}
			if len(got.Groups) != len(tt.wantGrps) {
				t.Errorf("Parse() got %d groups, want %d", len(got.Groups), len(tt.wantGrps))
				return
			}
			for i, g := range got.Groups {
				if g != tt.wantGrps[i] {
					t.Errorf("Parse() group[%d] = %v, want %v", i, g, tt.wantGrps[i])
				}
			}
		})
	}
}
