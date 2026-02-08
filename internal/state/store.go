package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Store interface {
	Load() (*GameState, error)
	Save(state *GameState) error
	Exists() bool
	GetManual() (string, error)
}

type FileStore struct {
	Root string
}

func (s *FileStore) GetManual() (string, error) {
	content, err := os.ReadFile(filepath.Join(s.Root, "vd_manual.md"))
	if err != nil {
		content, err = os.ReadFile("vd_manual.md")
	}
	return string(content), err
}

func (s *FileStore) path() string {
	return filepath.Join(s.Root, "state.json")
}

func (s *FileStore) Exists() bool {
	_, err := os.Stat(s.path())
	return err == nil
}

func (s *FileStore) Load() (*GameState, error) {
	f, err := os.Open(s.path())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var state GameState
	if err := json.NewDecoder(f).Decode(&state); err != nil {
		return nil, err
	}
	if state.ReactionsUsed == nil {
		state.ReactionsUsed = make(map[string]bool)
	}
	if err := state.Validate(); err != nil {
		return nil, fmt.Errorf("invalid state: %w", err)
	}
	return &state, nil
}

func (s *FileStore) Save(state *GameState) error {
	f, err := os.Create(s.path())
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(state)
}

type MemoryStore struct {
	State *GameState
}

func (s *MemoryStore) Load() (*GameState, error) {
	if s.State == nil {
		return nil, errors.New("state not found")
	}
	return s.State, nil
}

func (s *MemoryStore) Save(state *GameState) error {
	s.State = state
	return nil
}

func (s *MemoryStore) Exists() bool {
	return s.State != nil
}

func (s *MemoryStore) GetManual() (string, error) {
	return "# Mock Manual", nil
}
