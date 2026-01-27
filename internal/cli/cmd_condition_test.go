package cli

import (
	"strings"
	"testing"
	"uaa/vdnd/internal/state"
)

func TestConditionCommands(t *testing.T) {
	deps := NewTestDeps(t)
	s := &state.GameState{
		Entities: map[string]*state.EntityState{
			"goblin": {
				ID:   "goblin",
				Name: "Goblin",
			},
		},
		ReactionsUsed: make(map[string]bool),
	}
	deps.Store.Save(s)

	t.Run("Add Condition", func(t *testing.T) {
		out, err := cmdConditionAdd([]string{"goblin", "frightened", "2"}, deps)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}
		if !strings.Contains(out, "Added condition **frightened** 2 to **Goblin**") {
			t.Errorf("Unexpected output: %s", out)
		}
		st, _ := deps.Store.Load()
		if len(st.Entities["goblin"].Conditions) != 1 || st.Entities["goblin"].Conditions[0].ID != "frightened" || st.Entities["goblin"].Conditions[0].Value != 2 {
			t.Errorf("Condition not added correctly: %+v", st.Entities["goblin"].Conditions)
		}
	})

	t.Run("Reduce Condition", func(t *testing.T) {
		out, err := cmdConditionReduce([]string{"goblin", "frightened", "1"}, deps)
		if err != nil {
			t.Fatalf("Reduce failed: %v", err)
		}
		if !strings.Contains(out, "Reduced condition **frightened** on **Goblin** by 1") {
			t.Errorf("Unexpected output: %s", out)
		}
		st, _ := deps.Store.Load()
		if st.Entities["goblin"].Conditions[0].Value != 1 {
			t.Errorf("Expected value 1, got %d", st.Entities["goblin"].Conditions[0].Value)
		}
	})

	t.Run("List Conditions", func(t *testing.T) {
		out, err := cmdConditionList([]string{"goblin"}, deps)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if !strings.Contains(out, "| frightened | 1 |") {
			t.Errorf("Unexpected list output: %s", out)
		}
	})

	t.Run("Remove Condition", func(t *testing.T) {
		out, err := cmdConditionRemove([]string{"goblin", "frightened"}, deps)
		if err != nil {
			t.Fatalf("Remove failed: %v", err)
		}
		if !strings.Contains(out, "Removed condition **frightened** from **Goblin**") {
			t.Errorf("Unexpected output: %s", out)
		}
		st, _ := deps.Store.Load()
		if len(st.Entities["goblin"].Conditions) != 0 {
			t.Errorf("Condition not removed: %+v", st.Entities["goblin"].Conditions)
		}
	})

	t.Run("Remove Non-existent Condition", func(t *testing.T) {
		out, _ := cmdConditionRemove([]string{"goblin", "blinded"}, deps)
		if !strings.Contains(out, "does not have condition **blinded**") {
			t.Errorf("Unexpected output: %s", out)
		}
	})

	t.Run("Reduce Non-existent Condition", func(t *testing.T) {
		out, _ := cmdConditionReduce([]string{"goblin", "blinded", "1"}, deps)
		if !strings.Contains(out, "does not have condition **blinded**") {
			t.Errorf("Unexpected output: %s", out)
		}
	})

	t.Run("List No Conditions", func(t *testing.T) {
		out, _ := cmdConditionList([]string{"goblin"}, deps)
		if !strings.Contains(out, "has no active conditions") {
			t.Errorf("Unexpected output: %s", out)
		}
	})
}
