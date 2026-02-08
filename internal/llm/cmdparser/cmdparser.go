package cmdparser

import (
	"strings"
	"unicode"
)

// Parse splits a command string into arguments respecting shell-like quoting.
func Parse(cmd string) []string {
	var args []string
	var current strings.Builder
	inSingle, inDouble, escaped := false, false, false

	for _, ch := range cmd {
		switch {
		case escaped:
			current.WriteRune(ch)
			escaped = false
		case ch == '\\':
			escaped = true
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case unicode.IsSpace(ch) && !inSingle && !inDouble:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}