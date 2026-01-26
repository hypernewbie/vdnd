package entity_test

import (
	"fmt"
	"testing"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
)

func TestBagOfRocksStress(t *testing.T) {
	e := entity.NewEntity("stress", "Stress Tester", 1)
	// Base Str 10 -> MaxBulk 10, Encumbered 5
	e.Abilities.Strength = 10

	// Add 100 rocks of 1 Bulk each
	for i := 0; i < 100; i++ {
		e.PickUpItem(fmt.Sprintf("rock_%d", i), fmt.Sprintf("Rock %d", i), entity.Bulk1, 1, nil)
	}

	totalBulk := e.CurrentBulkInWholeBulk()
	if totalBulk < 100 {
		t.Errorf("Expected bulk ~100, got %d", totalBulk)
	}

	if !e.Conditions.Has(condition.Immobilized) {
		t.Error("Entity should be immobilized")
	}

	if !e.Conditions.Has(condition.Encumbered) {
		t.Error("Entity should be encumbered")
	}
}

func TestFractionalBulkArithmetic(t *testing.T) {
	e := entity.NewEntity("math", "Math Tester", 1)

	// 9 Light items = 0 Bulk (10 L = 1 Bulk)
	for i := 0; i < 9; i++ {
		e.PickUpItem(fmt.Sprintf("light_%d", i), "Light Item", entity.BulkLight, 1, nil)
	}
	if e.CurrentBulkInWholeBulk() != 0 {
		t.Errorf("9 light items should be 0 bulk, got %d", e.CurrentBulkInWholeBulk())
	}

	// 10th Light item -> 1 Bulk
	e.PickUpItem("light_9", "10th Light", entity.BulkLight, 1, nil)
	if e.CurrentBulkInWholeBulk() != 1 {
		t.Errorf("10 light items should be 1 bulk, got %d", e.CurrentBulkInWholeBulk())
	}

	// 1000 coins -> 1 Bulk
	e.Inventory.CoinsCP = 1000
	if e.CurrentBulkInWholeBulk() != 2 {
		t.Errorf("1000 coins + 10L should be 2 bulk, got %d", e.CurrentBulkInWholeBulk())
	}
}

func TestDropToMove(t *testing.T) {
	e := entity.NewEntity("drop", "Dropper", 1)
	e.Abilities.Strength = 10 // Max 10

	// Add very heavy item
	e.PickUpItem("boulder", "Boulder", entity.BulkValue(500), 1, nil)

	if !e.Conditions.Has(condition.Immobilized) {
		t.Fatal("Should be immobilized")
	}

	// Drop item
	e.DropItem("boulder", 1)

	if e.Conditions.Has(condition.Immobilized) {
		t.Error("Immobilized should be removed after dropping boulder")
	}
}
