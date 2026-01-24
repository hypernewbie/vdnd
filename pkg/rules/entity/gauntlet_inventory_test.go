package entity_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/entity"
)

// mockReducer implements the reducer interface for BulkReduction tests
type mockReducer struct {
	reduction int
}
func (m mockReducer) GetBulkReduction() int { return m.reduction }

// Source: rules/rules/core-rulebook/chapter-6-equipment.md (Bulk, Carry Limit)

func TestRecursiveContainers(t *testing.T) {
	e := entity.NewEntity("test", "Test", 1)
	
	// Backpack A reduces bulk by 2 (20 BulkValue)
	bpA := mockReducer{reduction: 20}
	
	// id, name, bulk, quantity, itemRef
	e.Inventory.AddItem("bpA", "Backpack A", 0, 1, bpA)
	
	// Put 50 rocks (10 bulk each -> 5 bulk total -> 50 BulkValue) in bpA
	e.Inventory.AddItem("rock", "Rock", 10, 5, nil)
	e.Inventory.MoveItem("rock", "bpA")
	
	// Total Bulk = 50 (rocks) - 20 (bpA reduction) = 30 (3 Bulk)
	bulk := e.Inventory.TotalBulk()
	if bulk.ToBulk() != 3 {
		t.Errorf("Expected 3 bulk after reduction, got %d (Value: %d)", bulk.ToBulk(), bulk)
	}
}

func TestDroppingContainer(t *testing.T) {
	e := entity.NewEntity("test", "Test", 1)
	
	bp := mockReducer{reduction: 20}
	e.Inventory.AddItem("bp", "Backpack", 0, 1, bp)
	e.Inventory.AddItem("anvil", "Anvil", 100, 1, nil) // 10 Bulk (100 BulkValue)
	e.Inventory.MoveItem("anvil", "bp")
	
	initialBulk := e.Inventory.TotalBulk().ToBulk()
	if initialBulk != 8 { // 10 - 2
		t.Errorf("Expected 8 bulk, got %d", initialBulk)
	}
	
	// Drop backpack
	e.Inventory.RemoveItem("bp", 1)
	
	// Orphaned items (whose parent was removed) should still count towards total bulk.
	afterDrop := e.Inventory.TotalBulk().ToBulk()
	if afterDrop != 10 {
		t.Errorf("Expected 10 bulk after losing container, got %d", afterDrop)
	}
}

func TestCoinAccumulationOverflow(t *testing.T) {
	e := entity.NewEntity("test", "Test", 1)
	
	// 1,000,000,000 copper pieces -> 1,000,000 Bulk
	e.Inventory.CoinsCP = 1000000000
	
	bulk := e.Inventory.TotalBulk().ToBulk()
	if bulk != 1000000 {
		t.Errorf("Expected 1,000,000 bulk, got %d", bulk)
	}
	
	if !e.IsEncumbered() {
		t.Error("Entity should be encumbered")
	}
}