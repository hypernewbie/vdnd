package combat

import (
	"testing"
	"uaa/vdnd/pkg/rules/entity"
)

// MockReaction for testing
type MockReaction struct {
	name    string
	trigger ReactionTrigger
	canUse  bool
}

func (m *MockReaction) Name() string                 { return m.name }
func (m *MockReaction) TriggerType() ReactionTrigger { return m.trigger }
func (m *MockReaction) CanUse(actor *entity.Entity, event ReactionEvent) bool {
	return m.canUse
}

func TestReactionQueue(t *testing.T) {
	actor := entity.NewEntity("a1", "Hero", 1)
	actor.MaxHP = 20
	actor.CurrentHP = 20
	reaction := &MockReaction{
		name:    "Attack of Opportunity",
		trigger: TriggerOnMovementInReach,
		canUse:  true,
	}

	event := ReactionEvent{
		Trigger: TriggerOnMovementInReach,
		Source:  actor,
	}

	queue := ReactionQueue{
		Event: event,
		Available: []AvailableReaction{
			{Actor: actor, Reaction: reaction},
		},
	}

	if queue.Event.Trigger != TriggerOnMovementInReach {
		t.Errorf("Expected trigger %v, got %v", TriggerOnMovementInReach, queue.Event.Trigger)
	}

	if len(queue.Available) != 1 {
		t.Errorf("Expected 1 available reaction, got %d", len(queue.Available))
	}
}
