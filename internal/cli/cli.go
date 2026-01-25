package cli

import (
	"fmt"
	"strings"
)

// CommandHandler func(args []string, deps Deps) (string, error)
type CommandHandler func(args []string, deps Deps) (string, error)

var commands = map[string]CommandHandler{
	"help":       cmdHelp,
	"scene new":  cmdSceneNew,
	"scene save": cmdSceneSave,
	"scene load": cmdSceneLoad,
	"status":     cmdStatus,
}

// Run is the main entry point. Takes CLI args and dependencies, returns output and exit code.
// This is the function that main.go calls and tests exercise.
func Run(args []string, deps Deps) (stdout string, exitCode int) {
	if len(args) == 0 {
		return helpText(), 0
	}

	// Build command key from first 1-2 args
	cmd, cmdArgs := parseCommand(args)

	handler, ok := commands[cmd]
	if !ok {
		return fmt.Sprintf("unknown command: %s\n\nRun 'vd help' for usage.", cmd), 1
	}

	// Execute handler
	result, err := handler(cmdArgs, deps)
	if err != nil {
		fmt.Fprintln(deps.Stderr, "error:", err)
		return "", 1
	}

	return result, 0
}

func parseCommand(args []string) (string, []string) {
	if len(args) == 0 {
		return "", nil
	}

	// Check for 2-word command
	if len(args) >= 2 {
		potentialCmd := args[0] + " " + args[1]
		if _, ok := commands[potentialCmd]; ok {
			return potentialCmd, args[2:]
		}
	}

	// Check for 1-word command
	if _, ok := commands[args[0]]; ok {
		return args[0], args[1:]
	}

	// Default to first arg as command key (even if unknown, let map lookup fail)
	return args[0], args[1:]
}

func cmdHelp(args []string, deps Deps) (string, error) {
	return helpText(), nil
}

func cmdStatus(args []string, deps Deps) (string, error) {
	state, err := deps.Store.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load state: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Scene: %s\n\n", state.SceneName))

	if len(state.Entities) == 0 {
		sb.WriteString("No entities in scene.\n")
	} else {
		sb.WriteString("## Entities\n")
		for id, e := range state.Entities {
			sb.WriteString(fmt.Sprintf("- **%s** (%s): HP %d/%d\n", id, e.Name, e.HP, e.MaxHP))
		}
	}

	return sb.String(), nil
}

func helpText() string {
	return `vd - Pathfinder 2E CLI

Usage:
  vd <command> [args]

Commands:
  help    Show this help message
`
}
