package combat

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
)

func TestTurnActions(t *testing.T) {
	e := entity.NewEntity("e1", "Hero", 1)
	e.MaxHP = 20
	e.CurrentHP = 20
	turn := NewTurn(e)

	// Test SpendActions Success
	err := turn.SpendActions(ability.CostOne)
	if err != nil {
		t.Errorf("Unexpected error spending 1 action: %v", err)
	}
	if turn.ActionsRemaining != 2 {
		t.Errorf("Expected 2 actions remaining, got %d", turn.ActionsRemaining)
	}

	// Test SpendActions Failure
	err = turn.SpendActions(ability.CostThree)
	if err == nil {
		t.Error("Expected error spending 3 actions when only 2 remain")
	}

	// Test SpendReaction
	err = turn.SpendReaction()
	if err != nil {
		t.Errorf("Unexpected error spending reaction: %v", err)
	}
	if !turn.ReactionUsed {
		t.Error("Expected ReactionUsed to be true")
	}

	err = turn.SpendReaction()
	if err == nil {
		t.Error("Expected error spending second reaction")
	}

	// Test CostTwo and CostThree
	turn.ActionsRemaining = 3
	if err := turn.SpendActions(ability.CostTwo); err != nil {
		t.Errorf("Failed to spend 2 actions: %v", err)
	}
	if turn.ActionsRemaining != 1 {
		t.Errorf("Expected 1 action, got %d", turn.ActionsRemaining)
	}

	turn.ActionsRemaining = 3
	if err := turn.SpendActions(ability.CostThree); err != nil {
		t.Errorf("Failed to spend 3 actions: %v", err)
	}
	if turn.ActionsRemaining != 0 {
		t.Errorf("Expected 0 actions, got %d", turn.ActionsRemaining)
	}
}

func TestTurnConditions(t *testing.T) {
	e := entity.NewEntity("e1", "Hero", 1)
	e.MaxHP = 20
	e.CurrentHP = 20

	// Test Quickened
	e.Conditions.Apply(condition.NewCondition(condition.Quickened, "Haste"))
	turn := NewTurn(e)
	if turn.ActionsRemaining != 4 {
		t.Errorf("Expected 4 actions (3+1 quickened), got %d", turn.ActionsRemaining)
	}

	// Test Slowed
	e.Conditions.Remove(condition.Quickened)
	e.Conditions.Apply(condition.NewValuedCondition(condition.Slowed, 1, "Slow Spell"))
	turn = NewTurn(e)
	if turn.ActionsRemaining != 2 {
		t.Errorf("Expected 2 actions (3-1 slowed), got %d", turn.ActionsRemaining)
	}

	// Test Slowed > Actions
	e.Conditions.Apply(condition.NewValuedCondition(condition.Slowed, 5, "Mega Slow"))
	turn = NewTurn(e)
	if turn.ActionsRemaining != 0 {
		t.Errorf("Expected 0 actions when slowed > base, got %d", turn.ActionsRemaining)
	}

	// Test Stunned (loses actions AND reduces condition)
	e.Conditions.Remove(condition.Slowed)
	e.Conditions.Apply(condition.NewValuedCondition(condition.Stunned, 2, "Stunned Effect"))
	turn = NewTurn(e)
	if turn.ActionsRemaining != 1 {
		t.Errorf("Expected 1 action (3-2 stunned), got %d", turn.ActionsRemaining)
	}
	if e.Conditions.Value(condition.Stunned) != 0 {
		t.Errorf("Expected Stunned condition to be reduced to 0, got %d", e.Conditions.Value(condition.Stunned))
	}
}

func TestCanAct(t *testing.T) {
	e := entity.NewEntity("e1", "Hero", 1)
	e.MaxHP = 20
	e.CurrentHP = 20
	turn := NewTurn(e)

	if !turn.CanAct() {
		t.Error("Entity should be able to act by default")
	}

	// Test Paralyzed
	e.Conditions.Apply(condition.NewCondition(condition.Paralyzed, "Hold Person"))
	if turn.CanAct() {
		t.Error("Paralyzed entity should not be able to act")
	}

	// Test Unconscious
	e.Conditions.Remove(condition.Paralyzed)
	e.CurrentHP = 0 // Unconscious
	if turn.CanAct() {
		t.Error("Unconscious entity should not be able to act")
	}
}

func TestResetShieldState(t *testing.T) {
	e := entity.NewEntity("e1", "Hero", 1)
	shield := &item.Shield{ID: "s1", Name: "Steel Shield"}
	e.WornShield = shield
	shield.IsRaised = true

	ResetShieldState(e)
	if shield.IsRaised {
		t.Error("Shield should be lowered after reset")
	}

	// Test nil shield
	e.WornShield = nil
	ResetShieldState(e) // Should not panic
}
