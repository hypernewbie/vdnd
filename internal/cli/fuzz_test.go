package cli

import (
	"strings"
	"testing"
)

func FuzzFlagParsing(f *testing.F) {
	f.Add("--foo bar --baz")
	f.Add("pos1 pos2 --key val")
	f.Add("--bool --key val pos3")
	f.Fuzz(func(t *testing.T, input string) {
		args := strings.Fields(input)
		ParseFlags(args)
	})
}
