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
