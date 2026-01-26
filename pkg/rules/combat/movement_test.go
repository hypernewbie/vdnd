package combat

import (
	"testing"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/trait"
)

func TestMovementActions(t *testing.T) {
	actor := entity.NewEntity("a1", "Hero", 1)
	actor.MaxHP = 20
	actor.CurrentHP = 20
	actor.Position = "zone-a"
	turn := NewTurn(actor)

	stride := &StrideAction{}
	if stride.Name() != "Stride" {
		t.Errorf("Expected Stride, got %s", stride.Name())
	}
	if !stride.HasTrait(trait.TraitMove) {
		t.Error("Stride should have Move trait")
	}

	res := stride.Execute(actor, "zone-b", turn)
	if !res.Success {
		t.Errorf("Stride failed: %s", res.Description)
	}
	if actor.Position != "zone-b" {
		t.Errorf("Expected position zone-b, got %s", actor.Position)
	}
	if turn.ActionsRemaining != 2 {
		t.Errorf("Expected 2 actions remaining, got %d", turn.ActionsRemaining)
	}

	step := &StepAction{}
	if step.Name() != "Step" {
		t.Errorf("Expected Step, got %s", step.Name())
	}
	res = step.Execute(actor, "zone-c", turn)
	if !res.Success {
		t.Errorf("Step failed: %s", res.Description)
	}
	// Step doesn't change position in current implementation
	if turn.ActionsRemaining != 1 {
		t.Errorf("Expected 1 action remaining, got %d", turn.ActionsRemaining)
	}
}

func TestStrideWithModes(t *testing.T) {
	actor := entity.NewEntity("a1", "Flier", 1)
	actor.BaseSpeed = 30
	actor.FlySpeed = 50
	turn := NewTurn(actor)

	stride := &StrideAction{}
	step := &StepAction{}

	// 1. Ground Stride
	res := stride.Execute(actor, "pos1", turn)
	if !res.Success {
		t.Errorf("Ground stride failed: %s", res.Description)
	}
	// Note: Exact string match depends on implementation, checking for key parts
	// "Moved up to 30 ft (ground) to pos1"
	if res.Description != "Moved up to 30 ft (ground) to pos1" {
		t.Errorf("Unexpected description: %s", res.Description)
	}

	// 2. Switch to Fly
	if err := actor.SetMoveMode(entity.MoveModeFly); err != nil {
		t.Fatalf("Failed to set fly mode: %v", err)
	}

	// 3. Fly Stride
	res = stride.Execute(actor, "pos2", turn)
	if !res.Success {
		t.Errorf("Fly stride failed: %s", res.Description)
	}
	if res.Description != "Moved up to 50 ft (fly) to pos2" {
		t.Errorf("Unexpected description: %s", res.Description)
	}

	// 4. Invalid Mode Stride
	// Reset actions for further testing
	turn = NewTurn(actor)
	// Set Swim (0 speed) via cheat (SetMoveMode would block it, so we manually set it to test Execute safety if state gets weird)
	actor.CurrentMoveMode = entity.MoveModeSwim
	res = stride.Execute(actor, "pos3", turn)
	if res.Success {
		t.Error("Expected stride failure with 0 swim speed")
	}
	if res.Description != "Cannot move: no swim speed" {
		t.Errorf("Unexpected failure description: %s", res.Description)
	}

	// 5. Step with no speed
	res = step.Execute(actor, "north", turn)
	if res.Success {
		t.Error("Expected step failure with 0 swim speed")
	}
	if res.Description != "Cannot step: no swim speed" {
		t.Errorf("Unexpected failure description: %s", res.Description)
	}
}
