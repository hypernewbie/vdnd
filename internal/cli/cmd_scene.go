package cli

import (
	"fmt"
	"strings"
	"uaa/vdnd/internal/state"
)

func cmdSceneNew(args []string, deps Deps) (string, error) {
	// usage: vd scene new "My Scene"
	if len(args) < 1 {
		return "", fmt.Errorf("usage: vd scene new <name>")
	}
	name := strings.Join(args, " ") // Allow "My Scene" as multiple args if unquoted

	if deps.Store.Exists() {
		return "", fmt.Errorf("session already exists in this directory")
	}

	newState := &state.GameState{
		SceneName: name,
		Positions: make(map[string]*state.Zone),
		Entities:  make(map[string]*state.EntityState),
	}

	if err := deps.Store.Save(newState); err != nil {
		return "", fmt.Errorf("failed to save scene: %w", err)
	}

	return "# Scene Created: " + name + "\n\nSession initialized.", nil
}

func cmdSceneSave(args []string, deps Deps) (string, error) {
	if !deps.Store.Exists() {
		return "", fmt.Errorf("no active session")
	}
	state, err := deps.Store.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load state: %w", err)
	}
	if err := deps.Store.Save(state); err != nil {
		return "", fmt.Errorf("failed to save: %w", err)
	}
	return "Scene saved.", nil
}

func cmdSceneLoad(args []string, deps Deps) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("usage: vd scene load <name>")
	}
	name := strings.Join(args, " ")
	return fmt.Sprintf("# Scene Loaded: %s\n\nLoaded from template (not yet implemented).", name), nil
}
