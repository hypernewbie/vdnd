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

func TestHeroPointStabilise(t *testing.T) {
	e := NewPC("pc1", "Valeros", 1, "Human", "Fighter", "Noble")
	e.MaxHP = 20
	e.CurrentHP = 0
	e.HeroPoints = 2

	// Stabilise dying
	e.Conditions.Apply(condition.NewValuedCondition(condition.Dying, 2, "damage"))
	success := e.HeroPointStabilise()
	if !success {
		t.Error("HeroPointStabilise failed when it should have succeeded")
	}
	if e.HeroPoints != 0 {
		t.Errorf("Expected 0 Hero Points after stabilisation, got %d", e.HeroPoints)
	}
	if e.Conditions.Has(condition.Dying) {
		t.Error("Expected Dying condition to be removed")
	}
	if e.CurrentHP != 0 {
		t.Errorf("Expected 0 HP after stabilisation, got %d", e.CurrentHP)
	}

	// Stabilise not dying
	e.HeroPoints = 1
	e.CurrentHP = 10
	success = e.HeroPointStabilise()
	if success {
		t.Error("HeroPointStabilise succeeded when not dying")
	}
	if e.HeroPoints != 1 {
		t.Errorf("Expected Hero Points to remain unchanged, got %d", e.HeroPoints)
	}

	// Stabilise no points
	e.HeroPoints = 0
	e.Conditions.Apply(condition.NewValuedCondition(condition.Dying, 1, "damage"))
	success = e.HeroPointStabilise()
	if success {
		t.Error("HeroPointStabilise succeeded with 0 points")
	}
	if !e.Conditions.Has(condition.Dying) {
		t.Error("Dying condition should not have been removed")
	}
}
