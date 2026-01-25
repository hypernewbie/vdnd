package combat

import (
	"testing"

	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/trait"
)

func TestInteractAction(t *testing.T) {
	actor := entity.NewEntity("actor", "Actor", 1)
	actor.Abilities = ability.AbilityScores{10, 10, 10, 10, 10, 10}
	turn := NewTurn(actor)

	action := &InteractAction{}

	// Test basic interact
	res := action.Execute(actor, nil, turn)
	if !res.Success {
		t.Errorf("expected success, got failure: %s", res.Description)
	}
	if turn.ActionsRemaining != 2 {
		t.Errorf("expected 2 actions remaining, got %d", turn.ActionsRemaining)
	}

	// Test interact with description
	actionWithDesc := &InteractAction{ObjectDescription: "door lever"}
	res = actionWithDesc.Execute(actor, nil, turn)
	if !res.Success {
		t.Errorf("expected success")
	}
	if res.Description != "Interacted with door lever" {
		t.Errorf("expected 'Interacted with door lever', got '%s'", res.Description)
	}

	// Test no actions remaining
	turn.ActionsRemaining = 0
	err := action.Validate(actor, nil, turn)
	if err == nil {
		t.Error("expected error for no actions, got nil")
	}
}

func TestDropProneAction(t *testing.T) {
	actor := entity.NewEntity("actor", "Actor", 1)
	actor.Abilities = ability.AbilityScores{10, 10, 10, 10, 10, 10}
	turn := NewTurn(actor)

	action := &DropProneAction{}

	// Test basic drop prone
	res := action.Execute(actor, nil, turn)
	if !res.Success {
		t.Errorf("expected success")
	}
	if !actor.Conditions.Has(condition.Prone) {
		t.Error("expected actor to have Prone condition")
	}
	if turn.ActionsRemaining != 3 {
		t.Errorf("expected 3 actions remaining (free action), got %d", turn.ActionsRemaining)
	}

	// Test already prone
	err := action.Validate(actor, nil, turn)
	if err == nil {
		t.Error("expected error for already prone, got nil")
	}
}

func TestStandAction(t *testing.T) {
	actor := entity.NewEntity("actor", "Actor", 1)
	actor.Abilities = ability.AbilityScores{10, 10, 10, 10, 10, 10}
	actor.Conditions.Apply(condition.NewCondition(condition.Prone, "setup"))
	turn := NewTurn(actor)

	action := &StandAction{}

	// Test stand from prone
	res := action.Execute(actor, nil, turn)
	if !res.Success {
		t.Errorf("expected success, got %s", res.Description)
	}
	if actor.Conditions.Has(condition.Prone) {
		t.Error("expected Prone condition to be removed")
	}
	if turn.ActionsRemaining != 2 {
		t.Errorf("expected 2 actions remaining, got %d", turn.ActionsRemaining)
	}

	// Test not prone
	err := action.Validate(actor, nil, turn)
	if err == nil {
		t.Error("expected error for not prone, got nil")
	}

	// Test trait
	if !action.HasTrait(trait.TraitMove) {
		t.Error("Stand action should have Move trait")
	}
}

func TestTakeCoverAction(t *testing.T) {
	actor := entity.NewEntity("actor", "Actor", 1)
	actor.Abilities = ability.AbilityScores{10, 10, 10, 10, 10, 10}
	turn := NewTurn(actor)

	action := &TakeCoverAction{}

	// Test basic take cover
	res := action.Execute(actor, nil, turn)
	if !res.Success {
		t.Errorf("expected success")
	}
	if !actor.Conditions.Has(condition.TakingCover) {
		t.Error("expected actor to have TakingCover condition")
	}
	if turn.ActionsRemaining != 2 {
		t.Errorf("expected 2 actions remaining, got %d", turn.ActionsRemaining)
	}

	// Test AC bonus
	ac := actor.GetAC(nil)
	expectedAC := 10 + 0 + 0 + 0 + 0 + 4 // base + dex + prof + armor + shield + condition
	if ac != expectedAC {
		t.Errorf("expected AC %d, got %d", expectedAC, ac)
	}

	// Test Reflex bonus
	ref := actor.GetReflex()
	if ref != 4 { // 0 dex + 0 prof + 4 cover
		t.Errorf("expected Reflex 4, got %d", ref)
	}

	// Test removed on stride
	stride := &StrideAction{}
	stride.Execute(actor, "zone2", turn)
	if actor.Conditions.Has(condition.TakingCover) {
		t.Error("TakingCover should be removed on Stride")
	}
}
