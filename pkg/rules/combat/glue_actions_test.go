package combat

import (
	"testing"

	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/trait"
)

func TestInteractAction(t *testing.T) {
	actor := entity.NewEntity("actor1", "Actor", 1)
	turn := NewTurnWithActions(actor, 3)

	action := &InteractAction{}

	if action.Cost() != ability.CostOne {
		t.Errorf("Expected CostOne, got %v", action.Cost())
	}

	// 1. Basic Interact
	res := action.Execute(actor, nil, turn)
	if !res.Success {
		t.Errorf("Interact failed: %v", res.Description)
	}
	if turn.ActionsRemaining != 2 {
		t.Errorf("Expected 2 actions remaining, got %d", turn.ActionsRemaining)
	}
	if !action.HasTrait(trait.TraitManipulate) {
		t.Error("Interact should have Manipulate trait")
	}

	// 2. Interact with description
	actionWithDesc := &InteractAction{ObjectDescription: "lever"}
	res = actionWithDesc.Execute(actor, nil, turn)
	if !res.Success {
		t.Errorf("Interact failed: %v", res.Description)
	}
	if res.Description != "Interacted with lever" {
		t.Errorf("Expected 'Interacted with lever', got '%s'", res.Description)
	}

	// 3. No actions
	turn.ActionsRemaining = 0
	err := action.Validate(actor, nil, turn)
	if err == nil {
		t.Error("Expected error when no actions remaining")
	}
}

func TestDropProneAction(t *testing.T) {
	actor := entity.NewEntity("actor1", "Actor", 1)
	turn := NewTurnWithActions(actor, 3)
	action := &DropProneAction{}

	// 1. Drop Prone
	res := action.Execute(actor, nil, turn)
	if !res.Success {
		t.Errorf("DropProne failed: %v", res.Description)
	}
	if !actor.Conditions.Has(condition.Prone) {
		t.Error("Actor should be prone")
	}
	if turn.ActionsRemaining != 3 {
		t.Errorf("DropProne should be free, got %d actions left", turn.ActionsRemaining)
	}

	// 2. Already Prone
	err := action.Validate(actor, nil, turn)
	if err == nil {
		t.Error("Expected error when already prone")
	}
}

func TestStandAction(t *testing.T) {
	actor := entity.NewEntity("actor1", "Actor", 1)
	actor.Conditions.Apply(condition.NewCondition(condition.Prone, "test"))
	turn := NewTurnWithActions(actor, 3)
	action := &StandAction{}

	// Verify Move trait
	if !action.HasTrait(trait.TraitMove) {
		t.Error("Stand should have Move trait")
	}

	// 1. Stand from Prone
	res := action.Execute(actor, nil, turn)
	if !res.Success {
		t.Errorf("Stand failed: %v", res.Description)
	}
	if actor.Conditions.Has(condition.Prone) {
		t.Error("Actor should not be prone after standing")
	}
	if turn.ActionsRemaining != 2 {
		t.Errorf("Expected 2 actions remaining, got %d", turn.ActionsRemaining)
	}

	// 2. Not Prone
	err := action.Validate(actor, nil, turn)
	if err == nil {
		t.Error("Expected error when not prone")
	}

	// 3. No actions
	actor.Conditions.Apply(condition.NewCondition(condition.Prone, "test"))
	turn.ActionsRemaining = 0
	err = action.Validate(actor, nil, turn)
	if err == nil {
		t.Error("Expected error when no actions")
	}
}

func TestTakeCoverAction(t *testing.T) {
	actor := entity.NewEntity("actor1", "Actor", 1)
	turn := NewTurnWithActions(actor, 3)
	action := &TakeCoverAction{}

	// 1. Take Cover
	res := action.Execute(actor, nil, turn)
	if !res.Success {
		t.Errorf("TakeCover failed: %v", res.Description)
	}
	if !actor.Conditions.Has(condition.TakingCover) {
		t.Error("Actor should have TakingCover condition")
	}
	if turn.ActionsRemaining != 2 {
		t.Errorf("Expected 2 actions remaining, got %d", turn.ActionsRemaining)
	}

	// 2. Cover Bonus
	// We check if modifiers are applied correctly via ConditionTracker.GetACModifiers
	// Note: We need a dummy attacker for AC modifiers usually, or pass nil if safe.
	// existing code in effects.go: GetACModifiers(attacker *ConditionTracker, attackerID string)
	// It checks for attacker != nil for hidden/undetected logic, but otherwise just appends modifiers.
	mods := actor.Conditions.GetACModifiers(nil, "")
	found := false
	for _, m := range mods {
		if m.Source == "Taking Cover" && m.Value == 4 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected +4 circumstance bonus to AC from Taking Cover")
	}

	// 3. Cover Removed on Stride
	stride := &StrideAction{}
	// Execute stride (we assume destination string "loc2" is valid enough for the logic)
	// StrideAction.Execute(actor, dest, turn)
	// We need a fresh turn or just reuse
	turn.ActionsRemaining = 2
	strideRes := stride.Execute(actor, "loc2", turn)
	if !strideRes.Success {
		t.Errorf("Stride failed: %v", strideRes.Description)
	}
	if actor.Conditions.Has(condition.TakingCover) {
		t.Error("TakingCover should be removed after Stride")
	}
}

func TestInteractAction_Validate(t *testing.T) {
	actor := entity.NewEntity("actor1", "Actor", 1)
	turn := NewTurnWithActions(actor, 0)
	action := &InteractAction{}

	if err := action.Validate(actor, nil, turn); err == nil {
		t.Error("Expected error for 0 actions")
	}
}