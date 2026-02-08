package cmdparser

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"status", []string{"status"}},
		{"action strike hero gob_1", []string{"action", "strike", "hero", "gob_1"}},
		{"action cast wizard 'fire ball' --target goblin", []string{"action", "cast", "wizard", "fire ball", "--target", "goblin"}},
		{`action strike hero "goblin king"`, []string{"action", "strike", "hero", "goblin king"}},
		{`cmd --flag "value with spaces" 'another one'`, []string{"cmd", "--flag", "value with spaces", "another one"}},
		{`cmd "escaped \"quote\""`, []string{"cmd", `escaped "quote"`}},
		{`  extra   spaces  `, []string{"extra", "spaces"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Parse(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}