package parser

import (
	"strings"
	"testing"
)

func FuzzEntityParsing(f *testing.F) {
	f.Add("# Goblin\n- Level: 1\n- HP: 10/10")
	f.Add("# Hero\n- Level: 5\n- Str: 18\n- Dex: 14")
	f.Fuzz(func(t *testing.T, md string) {
		r := strings.NewReader(md)
		entity, err := ParseEntity(r)
		if err == nil {
			if entity == nil {
				t.Error("Parsed nil entity with no error")
			}
		}
	})
}
