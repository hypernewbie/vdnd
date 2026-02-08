package llm

import (
	"fmt"
	"strings"
	"testing"
)

func TestOrchestrator_TruncateHistory(t *testing.T) {
	o := &Orchestrator{}
	initialMessage := "I am ready."
	o.history = []Message{
		{Role: "model", Content: initialMessage},
	}

	// Add many large messages to exceed 10KB
	// 10KB is 10240 bytes. Let's add 15 messages of 1KB each.
	for i := 0; i < 15; i++ {
		content := fmt.Sprintf("Message %d: %s", i, strings.Repeat("a", 1024))
		o.history = append(o.history, Message{Role: "user", Content: content})
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
