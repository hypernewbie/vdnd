package item_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/item"
)

func TestShieldBasics(t *testing.T) {
	shield := item.NewShield("steel_shield", "Steel Shield", 2, 5, 20, 1)

	if shield.IsBroken() {
		t.Error("New shield should not be broken")
	}
	if shield.IsDestroyed() {
		t.Error("New shield should not be destroyed")
	}

	// Take damage below hardness - actually no, hardness is for blocking, shield takes raw damage
	shield.TakeDamage(8)
	if shield.CurrentHP != 12 {
		t.Errorf("Expected HP 12, got %d", shield.CurrentHP)
	}
	if shield.IsBroken() {
		t.Error("Shield should not be broken at 12 HP (BT=10)")
	}

	// Take more damage to break
	shield.TakeDamage(3)
	if shield.CurrentHP != 9 {
		t.Errorf("Expected HP 9, got %d", shield.CurrentHP)
	}
	if !shield.IsBroken() {
		t.Error("Shield should be broken at 9 HP (BT=10)")
	}
}

func TestShieldACBonus(t *testing.T) {
	shield := item.NewShield("steel_shield", "Steel Shield", 2, 5, 20, 1)

	if shield.ACBonus != 2 {
		t.Errorf("Expected AC bonus 2, got %d", shield.ACBonus)
	}

	// Raised state
	shield.IsRaised = true
	if !shield.IsRaised {
		t.Error("Shield should be raised")
	}

	// Break the shield
	shield.CurrentHP = 5
	if !shield.IsBroken() {
		t.Error("Shield should be broken")
	}
	// Broken shields can still be raised but shouldn't grant AC (handled in Entity.GetAC)
}
