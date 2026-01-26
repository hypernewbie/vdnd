package entity

import (
	"testing"
	"uaa/vdnd/pkg/rules/condition"
)

func TestHeroPoints(t *testing.T) {
	e := NewPC("pc1", "Valeros", 1, "Human", "Fighter", "Noble")

	// PCs start with 1 hero point
	if e.HeroPoints != 1 {
		t.Errorf("Expected 1 initial Hero Point for PC, got %d", e.HeroPoints)
	}

	// Gain hero point
	e.GainHeroPoint()
	if e.HeroPoints != 2 {
		t.Errorf("Expected 2 Hero Points, got %d", e.HeroPoints)
	}

	// Gain capped at 3
	e.HeroPoints = 3
	e.GainHeroPoint()
	if e.HeroPoints != 3 {
		t.Errorf("Expected 3 Hero Points (capped), got %d", e.HeroPoints)
	}

	// Spend hero point
	e.HeroPoints = 2
	err := e.SpendHeroPoint()
	if err != nil {
		t.Errorf("Unexpected error spending hero point: %v", err)
	}
	if e.HeroPoints != 1 {
		t.Errorf("Expected 1 Hero Point remaining, got %d", e.HeroPoints)
	}

	// Spend with none
	e.HeroPoints = 0
	err = e.SpendHeroPoint()
	if err == nil {
		t.Error("Expected error spending hero point with none remaining")
	}

	// CanUseHeroPoints
	e.HeroPoints = 1
	if !e.CanUseHeroPoints() {
		t.Error("Expected CanUseHeroPoints to be true")
	}
	e.HeroPoints = 0
	if e.CanUseHeroPoints() {
		t.Error("Expected CanUseHeroPoints to be false")
	}
}

func TestHeroPointStabilize(t *testing.T) {
	e := NewPC("pc1", "Valeros", 1, "Human", "Fighter", "Noble")
	e.MaxHP = 20
	e.CurrentHP = 0
	e.HeroPoints = 2

	// Stabilize dying
	e.Conditions.Apply(condition.NewValuedCondition(condition.Dying, 2, "damage"))
	success := e.HeroPointStabilize()
	if !success {
		t.Error("HeroPointStabilize failed when it should have succeeded")
	}
	if e.HeroPoints != 0 {
		t.Errorf("Expected 0 Hero Points after stabilization, got %d", e.HeroPoints)
	}
	if e.Conditions.Has(condition.Dying) {
		t.Error("Expected Dying condition to be removed")
	}
	if e.CurrentHP != 0 {
		t.Errorf("Expected 0 HP after stabilization, got %d", e.CurrentHP)
	}

	// Stabilize not dying
	e.HeroPoints = 1
	e.CurrentHP = 10
	success = e.HeroPointStabilize()
	if success {
		t.Error("HeroPointStabilize succeeded when not dying")
	}
	if e.HeroPoints != 1 {
		t.Errorf("Expected Hero Points to remain unchanged, got %d", e.HeroPoints)
	}

	// Stabilize no points
	e.HeroPoints = 0
	e.Conditions.Apply(condition.NewValuedCondition(condition.Dying, 1, "damage"))
	success = e.HeroPointStabilize()
	if success {
		t.Error("HeroPointStabilize succeeded with 0 points")
	}
	if !e.Conditions.Has(condition.Dying) {
		t.Error("Dying condition should not have been removed")
	}
}

// Test stabilizing with exactly 1 hero point
func TestHeroPointStabilize_WithOnePoint(t *testing.T) {
	entity := &Entity{HeroPoints: 1, CurrentHP: 0}
	tr := condition.NewTracker()
	tr.SetOwner("test")
	entity.Conditions = tr
	entity.Conditions.Apply(condition.NewValuedCondition(condition.Dying, 2, "test"))

	success := entity.HeroPointStabilize()

	if !success {
		t.Error("Expected success with 1 hero point")
	}
	if entity.HeroPoints != 0 {
		t.Errorf("Expected 0 Hero Points, got %d", entity.HeroPoints)
	}
	if entity.CurrentHP != 0 {
		t.Errorf("Expected 0 HP, got %d", entity.CurrentHP)
	}
	if entity.Conditions.Has(condition.Dying) {
		t.Error("Expected Dying condition to be removed")
	}
}

// Test stabilizing with maximum hero points (spending all 3)
func TestHeroPointStabilize_WithMaxPoints(t *testing.T) {
	entity := &Entity{HeroPoints: 3, CurrentHP: 0}
	tr := condition.NewTracker()
	tr.SetOwner("test")
	entity.Conditions = tr
	entity.Conditions.Apply(condition.NewValuedCondition(condition.Dying, 1, "test"))

	success := entity.HeroPointStabilize()

	if !success {
		t.Error("Expected success with 3 hero points")
	}
	if entity.HeroPoints != 0 {
		t.Errorf("Expected 0 Hero Points (all spent), got %d", entity.HeroPoints)
	}
	if entity.CurrentHP != 0 {
		t.Errorf("Expected 0 HP, got %d", entity.CurrentHP)
	}
	if entity.Conditions.Has(condition.Dying) {
		t.Error("Expected Dying condition to be removed")
	}
}