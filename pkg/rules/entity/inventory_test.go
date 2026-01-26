package entity_test

import (
	"fmt"
	"testing"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
)

func TestBulkCalculation(t *testing.T) {
	inv := entity.NewInventory()

	// Add items
	inv.AddItem("rope", "Rope", entity.Bulk1, 2, nil)       // 2 Bulk
	inv.AddItem("torch", "Torch", entity.BulkLight, 5, nil) // 5 Light

	total := inv.TotalBulk()
	// 2 Bulk = 20, 5 Light = 5, total = 25 = 2 Bulk, 5 Light
	if total.ToBulk() != 2 {
		t.Errorf("Expected 2 Bulk, got %d", total.ToBulk())
	}
	if total.LightRemainder() != 5 {
		t.Errorf("Expected 5 Light remainder, got %d", total.LightRemainder())
	}
}

func TestCoinBulk(t *testing.T) {
	inv := entity.NewInventory()
	inv.AddCoins(500, 300, 200, 0) // 1000 coins = 1 Bulk

	total := inv.TotalBulk()
	if total.ToBulk() != 1 {
		t.Errorf("1000 coins should be 1 Bulk, got %d", total.ToBulk())
	}
}

func TestBulkLimits(t *testing.T) {
	ent := entity.NewEntity("test", "Test", 1)
	ent.Abilities.Strength = 14 // +2 mod

	// Bulk limit = 5 + 2 = 7
	if ent.BulkLimit() != 7 {
		t.Errorf("Expected bulk limit 7, got %d", ent.BulkLimit())
	}

	// Max bulk = 10 + 2 = 12
	if ent.MaxBulk() != 12 {
		t.Errorf("Expected max bulk 12, got %d", ent.MaxBulk())
	}
}

func TestEncumbranceAutoApply(t *testing.T) {
	ent := entity.NewEntity("test", "Test", 1)
	ent.Abilities.Strength = 10 // +0 mod
	// Bulk limit = 5, Max = 10

	// Add 6 Bulk worth of items (over limit)
	for i := 0; i < 6; i++ {
		ent.PickUpItem(fmt.Sprintf("item_%d", i), "Heavy Item", entity.Bulk1, 1, nil)
	}

	if !ent.Conditions.Has(condition.Encumbered) {
		t.Error("Should be Encumbered at 6 Bulk (limit 5)")
	}

	// Drop 2 items (now 4 Bulk)
	ent.DropItem("item_0", 1)
	ent.DropItem("item_1", 1)

	if ent.Conditions.Has(condition.Encumbered) {
		t.Error("Should not be Encumbered at 4 Bulk (limit 5)")
	}
}

func TestImmobilizedAtMaxBulk(t *testing.T) {
	ent := entity.NewEntity("test", "Test", 1)
	ent.Abilities.Strength = 10 // +0 mod
	// Max = 10

	// Add 11 Bulk worth of items
	for i := 0; i < 11; i++ {
		ent.PickUpItem(fmt.Sprintf("item_%d", i), "Heavy Item", entity.Bulk1, 1, nil)
	}

	if !ent.Conditions.Has(condition.Immobilized) {
		t.Error("Should be Immobilized at 11 Bulk (max 10)")
	}
	if !ent.Conditions.Has(condition.Encumbered) {
		t.Error("Should also be Encumbered")
	}
}

func TestEncumberedSpeedPenalty(t *testing.T) {
	ent := entity.NewEntity("test", "Test", 1)
	ent.BaseSpeed = 25
	ent.Abilities.Strength = 10

	normalSpeed := ent.GetSpeed()
	if normalSpeed != 25 {
		t.Errorf("Expected speed 25, got %d", normalSpeed)
	}

	// Become encumbered
	for i := 0; i < 6; i++ {
		ent.PickUpItem(fmt.Sprintf("item_%d", i), "Heavy Item", entity.Bulk1, 1, nil)
	}

	encumberedSpeed := ent.GetSpeed()
	if encumberedSpeed != 15 {
		t.Errorf("Expected encumbered speed 15 (25-10), got %d", encumberedSpeed)
	}
}

func TestWornArmorBulkReduction(t *testing.T) {
	ent := entity.NewEntity("test", "Test", 1)

	// Chain mail is typically 2 Bulk, worn = 1 Bulk
	chainMail := item.NewArmor("chain_mail", "Chain Mail", item.MediumArmor, 4, 1, -2, -5, 14, 2)
	ent.WornArmor = &chainMail

	// Worn armour bulk = 2 - 1 = 1 Bulk = 10 internal
	bulk := ent.CurrentBulk()
	if bulk.ToBulk() != 1 {
		t.Errorf("Worn chain mail should be 1 Bulk (reduced from 2), got %d", bulk.ToBulk())
	}
}

func TestContainerBulkReduction(t *testing.T) {
	ent := entity.NewEntity("test", "Test", 1)
	ent.Abilities.Strength = 10 // limit 5

	// Add Backpack (0 Bulk, 2 Bulk reduction)
	backpack := item.StandardItems["backpack"]
	ent.PickUpItem(backpack.ID, backpack.Name, entity.BulkValue(backpack.Bulk*10), 1, backpack)

	// Add 3 Bulk of rope (Rope is 1 Bulk each)
	ent.PickUpItem("rope", "Rope", entity.Bulk1, 3, nil)

	// Initial bulk: 0 (backpack) + 3 (rope) = 3 Bulk.
	if ent.CurrentBulkInWholeBulk() != 3 {
		t.Errorf("Expected 3 Bulk before moving to container, got %d", ent.CurrentBulkInWholeBulk())
	}

	// Move rope into backpack
	err := ent.Inventory.MoveItem("rope", "backpack")
	if err != nil {
		t.Fatalf("Failed to move item: %v", err)
	}

	// Recalculate encumbrance
	ent.UpdateEncumbranceConditions()

	// Effective bulk: 0 (backpack) + max(0, 3 (rope) - 2 (reduction)) = 1 Bulk
	if ent.CurrentBulkInWholeBulk() != 1 {
		t.Errorf("Expected 1 Bulk after moving to container, got %d", ent.CurrentBulkInWholeBulk())
	}
}
