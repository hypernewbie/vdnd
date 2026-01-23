package cli

import "strings"

// ParseFlags extracts --key value and --flag from args.
// Returns remaining positional args and a map of flags.
// Note: This is a simplistic parser. It assumes:
// - Flags start with --
// - If the next arg doesn't start with --, it's the value
// - Otherwise it's a boolean flag (value "true")
func ParseFlags(args []string) (positional []string, flags map[string]string) {
	flags = make(map[string]string)
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			key := strings.TrimPrefix(args[i], "--")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				flags[key] = args[i+1]
				i++
			} else {
				flags[key] = "true"
			}
		} else {
			positional = append(positional, args[i])
		}
	}
	return
}
