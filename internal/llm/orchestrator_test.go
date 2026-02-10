package llm

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm/llmtypes"
	"uaa/vdnd/internal/llm/vdengine"
	"uaa/vdnd/internal/state"
)

func TestOrchestrator_MapToolToArgs(t *testing.T) {
	engine := vdengine.New(cli.Deps{
		Store: &state.MemoryStore{State: &state.GameState{
			Entities:      make(map[string]*state.EntityState),
			ReactionsUsed: make(map[string]bool),
		}},
		Stderr: io.Discard,
	})

	tests := []struct {
		name     string
		call     llmtypes.ToolCall
		expected []string
	}{
		{
			name: "vd_action_strike basic",
			call: llmtypes.ToolCall{
				Name:      "vd_action_strike",
				Arguments: `{"actor": "hero", "target": "goblin"}`,
			},
			expected: []string{"action", "strike", "hero", "goblin"},
		},
		{
			name: "vd_action_strike with weapon and map",
			call: llmtypes.ToolCall{
				Name:      "vd_action_strike",
				Arguments: `{"actor": "hero", "target": "goblin", "weapon": "longsword", "map": 1}`,
			},
			expected: []string{"action", "strike", "hero", "goblin", "--weapon", "longsword", "--map", "1"},
		},
		{
			name: "vd_damage",
			call: llmtypes.ToolCall{
				Name:      "vd_damage",
				Arguments: `{"id": "goblin", "amount": 10, "type": "fire"}`,
			},
			expected: []string{"damage", "goblin", "10", "fire"},
		},
		{
			name: "vd_heal",
			call: llmtypes.ToolCall{
				Name:      "vd_heal",
				Arguments: `{"id": "hero", "amount": 5}`,
			},
			expected: []string{"heal", "hero", "5"},
		},
		{
			name: "vd_condition_add with flags",
			call: llmtypes.ToolCall{
				Name:      "vd_condition_add",
				Arguments: `{"id": "hero", "condition": "frightened", "value": 1, "duration": 2, "source": "scary monster"}`,
			},
			expected: []string{"condition", "add", "hero", "frightened", "1", "--duration", "2", "--source", "scary monster"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, got, _ := engine.ExecuteTool(tt.call)
			if len(got) != len(tt.expected) {
				t.Fatalf("mapToolToArgs() returned %v, want %v", got, tt.expected)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("mapToolToArgs()[%d] = %s, want %s", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestOrchestrator_TruncateHistory(t *testing.T) {
	o := &Orchestrator{}
	initialMessage := "I am ready."
	o.history = []llmtypes.Message{
		{Role: "model", Content: initialMessage},
	}

	// Add many large messages to exceed 10KB
	// 10KB is 10240 bytes. Let's add 15 messages of 1KB each.
	for i := 0; i < 15; i++ {
		content := fmt.Sprintf("Message %d: %s", i, strings.Repeat("a", 1024))
		o.history = append(o.history, llmtypes.Message{Role: "user", Content: content})
	}

	// Verify history size before truncation (should be 16 messages total)
	if len(o.history) != 16 {
		t.Fatalf("Expected 16 messages before truncation, got %d", len(o.history))
	}

	// Run truncation
	o.truncateHistory()

	// Verify history size after truncation (should be <= 10KB)
	if len(o.history) >= 16 {
		t.Errorf("History was not truncated, still has %d messages", len(o.history))
	}

	// Verify the earliest message (initial model message) was removed as it's at index 0
	foundInitial := false
	for _, msg := range o.history {
		if msg.Content == initialMessage {
			foundInitial = true
			break
		}
	}
	if foundInitial {
		t.Errorf("Initial message at index 0 was expected to be removed during truncation")
	}

	// Verify total size is under 10KB (rough check)
	totalSize := 0
	for _, msg := range o.history {
		totalSize += len(msg.Content)
	}
	if totalSize > 10*1024 {
		t.Errorf("Total size %d exceeds 10KB limit", totalSize)
	}
}
