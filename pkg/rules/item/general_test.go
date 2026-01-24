package item_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/item"
)

func TestGeneralItemCreation(t *testing.T) {
	rope := item.NewGeneralItem("rope", "Rope (50 ft)", item.CategoryAdventuringGear, 1, 100)

	if rope.Bulk != 1 {
		t.Errorf("Expected bulk 1, got %d", rope.Bulk)
	}
	if rope.Price != 100 {
		t.Errorf("Expected price 100cp, got %d", rope.Price)
	}
}

func TestBackpackCapacity(t *testing.T) {
	backpack := item.StandardItems["backpack"]

	if backpack.BulkCapacity != 40 { // 4 Bulk in internal units
		t.Errorf("Backpack should hold 4 Bulk worth, got %d", backpack.BulkCapacity/10)
	}
	if backpack.BulkReduction != 20 { // 2 Bulk reduction
		t.Errorf("Backpack should reduce 2 Bulk, got %d", backpack.BulkReduction/10)
	}
}
