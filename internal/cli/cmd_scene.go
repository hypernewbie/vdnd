package cli

import (
	"fmt"
	"os"
	"strings"
	"uaa/vdnd/internal/state"
)

func cmdSceneNew(args []string, deps Deps) (string, error) {
	// usage: vd scene new "My Scene" [--force]
	positional, flags := ParseFlags(args)
	if len(positional) < 1 {
		return "", NewUsageError("missing scene name", "vd scene new <name> [--force]")
	}
	name := strings.Join(positional, " ") 
	force := flags["force"] == "true"

	if deps.Store.Exists() && !force {
		return "", NewStateError("session already exists in this directory", "Use the --force flag to overwrite the existing session, or use a different directory.")
	}

	newState := &state.GameState{
		SceneName:     name,
		Positions:     make(map[string]*state.Zone),
		Entities:      make(map[string]*state.EntityState),
		ReactionsUsed: make(map[string]bool),
	}

	if err := deps.Store.Save(newState); err != nil {
		return "", WrapSystemError(err, "failed to save scene")
	}

	return "# Scene Created: " + name + "\n\nSession initialized.", nil
}

func cmdSceneSave(args []string, deps Deps) (string, error) {
	if !deps.Store.Exists() {
		return "", NewStateError("no active session", "Start a new scene first with 'vd scene new <name>'.")
	}
	state, err := deps.Store.Load()
	if err != nil {
		return "", WrapSystemError(err, "failed to load state")
	}
	if err := deps.Store.Save(state); err != nil {
		return "", WrapSystemError(err, "failed to save")
	}
	return "Scene saved.", nil
}

func cmdSceneLoad(args []string, deps Deps) (string, error) {
	if len(args) < 1 {
		return "", NewUsageError("missing file path", "vd scene load <path>")
	}
	path := args[0]

	// For now, let's just check if the file exists as a mock "load"
	// In a real implementation we might copy it to state.json
	if _, err := os.Stat(path); err != nil {
		return "", WrapSystemError(err, fmt.Sprintf("failed to load scene from: %s", path))
	}

	return fmt.Sprintf("# Scene Loaded: %s\n\n(Note: This currently only validates the file exists)", path), nil
}
