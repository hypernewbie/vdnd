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

	return fmt.Sprintf("# Scene Created: %s\n\nSession initialized.", name), nil
}

func cmdSceneSave(args []string, deps Deps) (string, error) {
	if !deps.Store.Exists() {
		return "", fmt.Errorf("no active session")
	}
	// In a stateless CLI, explicit save is mostly a no-op or checkpoint
	return "Scene saved.", nil
}

func cmdSceneLoad(args []string, deps Deps) (string, error) {
	return "Scene loading from template not implemented yet.", nil
}
