package cmdparser

import (
	"testing"
)

func TestParse_Simple(t *testing.T) {
	args := Parse("status")
	if len(args) != 1 || args[0] != "status" {
		t.Errorf("Parse('status') = %v, want ['status']", args)
	}
}

func TestParse_MultipleArgs(t *testing.T) {
	args := Parse("action strike hero goblin")
	want := []string{"action", "strike", "hero", "goblin"}
	if len(args) != len(want) {
		t.Errorf("got %v, want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("args[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

func TestParse_QuotedString(t *testing.T) {
	args := Parse(`action cast wizard "fire ball" --target goblin`)
	want := []string{"action", "cast", "wizard", "fire ball", "--target", "goblin"}
	if len(args) != len(want) {
		t.Errorf("got %v, want %v", args, want)
	}
}

func TestParse_SingleQuotes(t *testing.T) {
	args := Parse(`entity add valeros --file 'character sheet.md'`)
	want := []string{"entity", "add", "valeros", "--file", "character sheet.md"}
	if len(args) != len(want) {
		t.Errorf("got %v, want %v", args, want)
	}
}

func TestParse_MixedQuotes(t *testing.T) {
	args := Parse(`action cast wizard "fire 'ball'"`)
	// Expect the outer double quotes to be stripped, inner single quotes preserved.
	want := []string{"action", "cast", "wizard", "fire 'ball'"}
	if len(args) != len(want) || args[3] != want[3] {
		t.Errorf("got %v, want %v", args, want)
	}
}

func TestParse_EmptyString(t *testing.T) {
	args := Parse("")
	if len(args) != 0 {
		t.Errorf("Parse('') = %v, want []", args)
	}
}

func TestParse_FlagsWithValues(t *testing.T) {
	args := Parse("action strike hero orc --weapon longsword --map 2")
	// Ensure flags and their values stay as separate arguments
	want := []string{"action", "strike", "hero", "orc", "--weapon", "longsword", "--map", "2"}
	if len(args) != len(want) {
		t.Errorf("got %v, want %v", args, want)
	}
}
