package condition

import (
	"testing"
)

func TestRelationalVisibility(t *testing.T) {
	tracker := NewTracker()
	guardA := "guard-a"
	guardB := "guard-b"

	// Rogue is Hidden relative to Guard A
	tracker.ApplyRelative(Hidden, guardA, "Stealth")

	// Check logic
	if !tracker.HasRelative(Hidden, guardA) {
		t.Error("Should be Hidden relative to Guard A")
	}

	if tracker.HasRelative(Hidden, guardB) {
		t.Error("Should NOT be Hidden relative to Guard B")
	}

	// Rogue becomes Invisible (Global)
	tracker.Apply(NewCondition(Invisible, "Potion"))

	// PF2E logic: Invisible implies Hidden to everyone
	if !tracker.HasRelative(Invisible, guardB) {
		t.Error("Should be Invisible relative to Guard B (Global)")
	}
}

func TestGlobalVsSpecific(t *testing.T) {
	tracker := NewTracker()
	target := "target-1"

	// Apply global frightened
	tracker.Apply(NewValuedCondition(Frightened, 1, "Fear Spell"))

	if !tracker.HasRelative(Frightened, target) {
		t.Error("Global condition should apply to specific target")
	}

	if !tracker.HasRelative(Frightened, "any-other") {
		t.Error("Global condition should apply to anyone")
	}
}

func TestRelationalAttitudes(t *testing.T) {
	tracker := NewTracker()
	merchant := "merchant-id"

	tracker.ApplyRelative(Friendly, merchant, "Diplomacy")

	if !tracker.HasRelative(Friendly, merchant) {
		t.Error("Should be Friendly to the merchant")
	}

	if tracker.HasRelative(Friendly, "random-npc") {
		t.Error("Should not be Friendly to random NPCs")
	}
}
