package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestGameStateJSONRoundTrip tests that GameState can be marshaled and unmarshaled correctly.
func TestGameStateJSONRoundTrip(t *testing.T) {
	original := &GameState{
		SceneName:        "Test Scene",
		Positions:        make(map[string]*Zone),
		Entities:         make(map[string]*EntityState),
		InCombat:         false,
		Round:            0,
		InitiativeOrder:  []string{},
		CurrentTurn:      "",
		TurnIndex:        0,
		ActionsRemaining: 0,
		ReactionsUsed:    make(map[string]bool),
		AttacksMade:      0,
		PendingEvents:    []PendingEvent{},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal GameState: %v", err)
	}

	// Unmarshal back
	var restored GameState
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal GameState: %v", err)
	}

	// Validate
	if err := restored.Validate(); err != nil {
		t.Errorf("Restored GameState is invalid: %v", err)
	}

	// Check key fields
	if restored.SceneName != original.SceneName {
		t.Errorf("SceneName mismatch: got %q, want %q", restored.SceneName, original.SceneName)
	}
	if restored.Positions == nil {
		t.Error("Positions should not be nil after unmarshal")
	}
	if restored.Entities == nil {
		t.Error("Entities should not be nil after unmarshal")
	}
	if restored.ReactionsUsed == nil {
		t.Error("ReactionsUsed should not be nil after unmarshal")
	}
}

// TestGameStateValidation tests various validation scenarios.
func TestGameStateValidation(t *testing.T) {
	tests := []struct {
		name    string
		state   *GameState
		wantErr bool
	}{
		{
			name: "valid state",
			state: &GameState{
				SceneName:     "Test",
				Positions:     make(map[string]*Zone),
				Entities:      make(map[string]*EntityState),
				ReactionsUsed: make(map[string]bool),
			},
			wantErr: false,
		},
		{
			name: "empty scene name",
			state: &GameState{
				SceneName:     "",
				Positions:     make(map[string]*Zone),
				Entities:      make(map[string]*EntityState),
				ReactionsUsed: make(map[string]bool),
			},
			wantErr: true,
		},
		{
			name: "nil positions",
			state: &GameState{
				SceneName:     "Test",
				Positions:     nil,
				Entities:      make(map[string]*EntityState),
				ReactionsUsed: make(map[string]bool),
			},
			wantErr: true,
		},
		{
			name: "nil entities",
			state: &GameState{
				SceneName:     "Test",
				Positions:     make(map[string]*Zone),
				Entities:      nil,
				ReactionsUsed: make(map[string]bool),
			},
			wantErr: true,
		},
		{
			name: "nil reactions used",
			state: &GameState{
				SceneName:     "Test",
				Positions:     make(map[string]*Zone),
				Entities:      make(map[string]*EntityState),
				ReactionsUsed: nil,
			},
			wantErr: true,
		},
		{
			name: "entity with invalid stats",
			state: &GameState{
				SceneName:     "Test",
				Positions:     make(map[string]*Zone),
				Entities:      map[string]*EntityState{"e1": {ID: "e1", Name: "Test", MaxHP: 0, AC: 15}},
				ReactionsUsed: make(map[string]bool),
			},
			wantErr: true,
		},
		{
			name: "entity with invalid AC",
			state: &GameState{
				SceneName:     "Test",
				Positions:     make(map[string]*Zone),
				Entities:      map[string]*EntityState{"e1": {ID: "e1", Name: "Test", MaxHP: 10, AC: 0}},
				ReactionsUsed: make(map[string]bool),
			},
			wantErr: true,
		},
		{
			name: "nil entity in map",
			state: &GameState{
				SceneName:     "Test",
				Positions:     make(map[string]*Zone),
				Entities:      map[string]*EntityState{"e1": nil},
				ReactionsUsed: make(map[string]bool),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.state.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestFileStoreIntegration tests FileStore with actual file I/O.
func TestFileStoreIntegration(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	store := &FileStore{Root: tmpDir}

	// Test that state doesn't exist initially
	if store.Exists() {
		t.Error("Store should not exist initially")
	}

	// Create and save a state
	original := &GameState{
		SceneName:        "Test Scene",
		Positions:        make(map[string]*Zone),
		Entities:         make(map[string]*EntityState),
		InCombat:         false,
		Round:            0,
		InitiativeOrder:  []string{},
		CurrentTurn:      "",
		TurnIndex:        0,
		ActionsRemaining: 0,
		ReactionsUsed:    make(map[string]bool),
		AttacksMade:      0,
		PendingEvents:    []PendingEvent{},
	}

	if err := store.Save(original); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Verify file was created
	statePath := filepath.Join(tmpDir, "state.json")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("state.json should exist after Save")
	}

	// Verify store.Exists() returns true
	if !store.Exists() {
		t.Error("Store.Exists() should return true after saving")
	}

	// Load the state back
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Validate loaded state
	if err := loaded.Validate(); err != nil {
		t.Errorf("Loaded state is invalid: %v", err)
	}

	// Verify content
	if loaded.SceneName != original.SceneName {
		t.Errorf("SceneName mismatch: got %q, want %q", loaded.SceneName, original.SceneName)
	}
}

// TestMemoryStore tests the in-memory store implementation.
func TestMemoryStore(t *testing.T) {
	store := &MemoryStore{}

	// Test that state doesn't exist initially
	if store.Exists() {
		t.Error("MemoryStore should not exist initially")
	}

	// Test loading from empty store
	_, err := store.Load()
	if err == nil {
		t.Error("Expected error when loading from empty MemoryStore")
	}

	// Create and save a state
	state := &GameState{
		SceneName:     "Test",
		Positions:     make(map[string]*Zone),
		Entities:      make(map[string]*EntityState),
		ReactionsUsed: make(map[string]bool),
	}

	if err := store.Save(state); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Verify store.Exists() returns true
	if !store.Exists() {
		t.Error("MemoryStore.Exists() should return true after saving")
	}

	// Load the state back
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Verify it's the same state
	if loaded != state {
		t.Error("MemoryStore should return the same pointer")
	}
}
