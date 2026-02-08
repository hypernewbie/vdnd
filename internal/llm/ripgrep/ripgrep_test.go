package ripgrep

import (
	"reflect"
	"testing"
)

func TestParseRGOutput(t *testing.T) {
	output := `rules/combat.md:10:Strike is a basic action.
rules/combat.md:15:You can also Dash.
rules/items.md:5:Potions are useful.
`
	expected := []Match{
		{File: "rules/combat.md", Line: 10, Text: "Strike is a basic action."},
		{File: "rules/combat.md", Line: 15, Text: "You can also Dash."},
		{File: "rules/items.md", Line: 5, Text: "Potions are useful."},
	}

	got := parseRGOutput(output)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("parseRGOutput() = %v, want %v", got, expected)
	}
}

func TestToJSON(t *testing.T) {
	res := &Result{
		RGInstalled: true,
		Matches: []Match{
			{File: "test.md", Line: 1, Text: "hello"},
		},
	}
	jsonStr := res.ToJSON()
	expected := `{"rg_installed":true,"matches":[{"file":"test.md","line":1,"text":"hello"}]}`
	if jsonStr != expected {
		t.Errorf("ToJSON() = %s, want %s", jsonStr, expected)
	}
}