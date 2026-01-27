package cli

import (
	"encoding/json"
	"reflect"
	"testing"
	"uaa/vdnd/internal/state"
	"uaa/vdnd/pkg/rules/ability"
)

func TestPersistenceCycle(t *testing.T) {
	s1 := &state.GameState{
		SceneName: "Complex Scene",
		Positions: map[string]*state.Zone{
			"z1": {Name: "Zone 1", Adjacent: []string{"z2"}},
		},
		Entities: map[string]*state.EntityState{
			"e1": {
				ID:    "e1",
				Name:  "Entity 1",
				Level: 5,
				Abilities: ability.AbilityScores{Strength: 4},
				Conditions: []state.ConditionInstance{
					{ID: "frightened", Value: 1, Duration: 5},
				},
			},
		},
		PendingEvents: []state.PendingEvent{
			{
				ID:   "evt1",
				Type: "movement",
				Payload: map[string]string{"to": "z2"},
				Reactors: []state.AvailableReaction{
					{EntityID: "e2", Reaction: "aoo"},
				},
			},
		},
		ReactionsUsed: make(map[string]bool),
	}

	data, err := json.MarshalIndent(s1, "", "  ")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var s2 state.GameState
	if err := json.Unmarshal(data, &s2); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Validate field by field or deep equal
	if !reflect.DeepEqual(s1, &s2) {
		// reflect.DeepEqual can be tricky with empty vs nil maps/slices
		if s1.SceneName != s2.SceneName {
			t.Errorf("SceneName mismatch: %s vs %s", s1.SceneName, s2.SceneName)
		}
		if len(s1.Entities) != len(s2.Entities) {
			t.Errorf("Entities count mismatch: %d vs %d", len(s1.Entities), len(s2.Entities))
		}
		// ... more checks if needed, but DeepEqual is the goal
		t.Errorf("DeepEqual failed. Data: %s", string(data))
	}
}
