package encounter

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
)

func TestEncounterLifecycle(t *testing.T) {
	enc := NewEncounter("test-1")
	if enc.State != StateNotStarted {
		t.Errorf("Expected NotStarted, got %d", enc.State)
	}

	e1 := entity.NewEntity("e1", "Hero", 1)
	e2 := entity.NewEntity("e2", "Orc", 1)

	enc.AddParticipant(e1)
	enc.AddParticipant(e2)

	if len(enc.Participants) != 2 {
		t.Errorf("Expected 2 participants, got %d", len(enc.Participants))
	}

	// Fake initiative
	enc.Participants[0].Initiative = 10
	enc.Participants[1].Initiative = 20

	err := enc.Start()
	if err != nil {
		t.Fatalf("Failed to start encounter: %v", err)
	}

	if enc.State != StateInProgress {
		t.Errorf("Expected InProgress, got %d", enc.State)
	}

	if enc.GetCurrentParticipant().Entity.ID != "e2" {
		t.Errorf("Expected e2 to be first, got %s", enc.GetCurrentParticipant().Entity.ID)
	}
}

func TestInitiativeSorting(t *testing.T) {
	enc := NewEncounter("test-init")

	// e1 has higher Perception for tie-break
	e1 := entity.NewEntity("e1", "Hero", 1)
	e1.Perception = ability.Trained // +3

	e2 := entity.NewEntity("e2", "Orc", 1)
	e2.Perception = ability.Untrained // +0

	enc.AddParticipant(e1)
	enc.AddParticipant(e2)

	// Same initiative
	enc.Participants[0].Initiative = 15
	enc.Participants[1].Initiative = 15

	enc.SortByInitiative()

	if enc.Participants[0].Entity.ID != "e1" {
		t.Errorf("Expected e1 (higher Perception) to win tie, got %s", enc.Participants[0].Entity.ID)
	}
}

func TestTurnProgression(t *testing.T) {
	enc := NewEncounter("test-turns")
	e1 := entity.NewEntity("e1", "A", 1)
	e2 := entity.NewEntity("e2", "B", 1)

	enc.AddParticipant(e1)
	enc.AddParticipant(e2)

	enc.Participants[0].Initiative = 20
	enc.Participants[1].Initiative = 10

	enc.Start() // e1 then e2

	// Turn 1 (e1)
	ts, _ := enc.StartTurn()
	if ts.Entity.ID != "e1" {
		t.Errorf("Expected e1 turn, got %s", ts.Entity.ID)
	}

	enc.EndTurn()
	if enc.CurrentTurn != 1 {
		t.Errorf("Expected turn index 1, got %d", enc.CurrentTurn)
	}
	if !enc.Participants[0].HasActed {
		t.Error("e1 should have acted")
	}

	// Turn 2 (e2)
	ts2, _ := enc.StartTurn()
	if ts2.Entity.ID != "e2" {
		t.Errorf("Expected e2 turn, got %s", ts2.Entity.ID)
	}

	enc.EndTurn()

	// Round wrap
	if enc.CurrentRound != 2 {
		t.Errorf("Expected Round 2, got %d", enc.CurrentRound)
	}
	if enc.CurrentTurn != 0 {
		t.Errorf("Expected turn index 0, got %d", enc.CurrentTurn)
	}
	if enc.Participants[0].HasActed {
		t.Error("HasActed should be reset at end of round")
	}
}

func TestConditionIntegration(t *testing.T) {
	enc := NewEncounter("test-conditions")
	e1 := entity.NewEntity("e1", "A", 1)
	e1.Conditions.Apply(condition.NewValuedCondition(condition.Frightened, 2, "Fear"))

	enc.AddParticipant(e1)
	enc.Participants[0].Initiative = 20
	enc.Start()

	enc.StartTurn()
	enc.EndTurn()

	// Frightened reduces at end of turn
	if e1.Conditions.Value(condition.Frightened) != 1 {
		t.Errorf("Expected Frightened 1 after turn, got %d", e1.Conditions.Value(condition.Frightened))
	}
}

func TestDelayResumeEdgeCases(t *testing.T) {
	enc := NewEncounter("test-delay")
	e1 := entity.NewEntity("A", "A", 1)
	e2 := entity.NewEntity("B", "B", 1)
	e3 := entity.NewEntity("C", "C", 1)

	enc.AddParticipant(e1)
	enc.AddParticipant(e2)
	enc.AddParticipant(e3)

	enc.Start()

	// A delays.
	enc.Delay()

	if enc.CurrentTurn != 1 {
		t.Errorf("Expected CurrentTurn 1 (B) after A delays, got %d", enc.CurrentTurn)
	}

	// B delays.
	enc.Delay()
	if enc.CurrentTurn != 2 {
		t.Errorf("Expected CurrentTurn 2 (C), got %d", enc.CurrentTurn)
	}

	// C acts.
	enc.StartTurn()
	enc.EndTurn()

	// Round 2.
	_, err := enc.StartTurn()
	if err == nil {
		t.Error("Expected error starting turn for delaying participant")
	}

	// Resume B.
	err = enc.ResumeFromDelay("B")
	if err != nil {
		t.Errorf("Resume failed: %v", err)
	}

	if enc.Participants[0].Entity.ID != "B" {
		t.Errorf("Expected B at head, got %s", enc.Participants[0].Entity.ID)
	}
	if enc.GetCurrentParticipant().Entity.ID != "B" {
		t.Errorf("Expected current participant B, got %s", enc.GetCurrentParticipant().Entity.ID)
	}
}

func TestEventQueueReentrance(t *testing.T) {
	enc := NewEncounter("test-events")
	e1 := entity.NewEntity("e1", "Hero", 1)
	enc.AddParticipant(e1)

	// Register handler for Move that emits a Manipulate event
	enc.EventQueue.RegisterHandler(ability.EventMove, func(event ability.Event, e *Encounter) bool {
		// Emit secondary event
		e.EventQueue.Emit(ability.Event{Type: ability.EventManipulate, Actor: event.Actor})
		return true
	})

	// Emit initial event
	enc.EventQueue.Emit(ability.Event{Type: ability.EventMove, Actor: e1})

	// Process (Move)
	enc.EventQueue.Process(enc)

	// Queue should now contain Manipulate event
	if len(enc.EventQueue.events) != 1 {
		t.Errorf("Expected 1 event (Manipulate) remaining in queue, got %d", len(enc.EventQueue.events))
	}
	if enc.EventQueue.events[0].Type != ability.EventManipulate {
		t.Errorf("Expected Manipulate event, got %v", enc.EventQueue.events[0].Type)
	}
}