package hazard_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/hazard"
	"uaa/vdnd/pkg/rules/item"
)

func TestComplexHazardCreation(t *testing.T) {
	h := hazard.GetComplexHazard("spinning_blade_pillar")
	if h == nil {
		t.Fatal("Failed to create spinning blade pillar")
	}

	if h.Complexity != hazard.ComplexityComplex {
		t.Error("Should be complex hazard")
	}

	if h.Routine == nil {
		t.Error("Complex hazard should have routine")
	}

	if len(h.Routine.Actions) != 2 {
		t.Errorf("Expected 2 routine actions, got %d", len(h.Routine.Actions))
	}
}

func TestHazardTurn(t *testing.T) {
	h := hazard.GetComplexHazard("spinning_blade_pillar")
	h.Position = "trap_room"

	target := entity.NewEntity("victim", "Unfortunate Adventurer", 5)
	target.MaxHP = 40
	target.CurrentHP = 40
	target.Position = "trap_room"

	result := h.TakeTurn([]*entity.Entity{target})

	if result.HazardID != h.ID {
		t.Error("Result should have hazard ID")
	}

	if len(result.ActionResults) == 0 {
		t.Error("Should have action results")
	}

	t.Logf("Hazard dealt %d total damage", result.TotalDamage)
}

func TestHazardDisable(t *testing.T) {
	h := hazard.GetComplexHazard("spinning_blade_pillar")

	rogue := entity.NewEntity("rogue", "Skilled Rogue", 5)
	rogue.SkillProficiencies[ability.SkillThievery] = ability.Expert

	option := h.DisableOptions[0]
	result := h.AttemptDisable(rogue, option)

	t.Logf("Disable attempt: %v", result.Degree)

	if result.Degree >= check.Success && !h.IsDisabled {
		t.Error("Successful disable should disable hazard")
	}
}

func TestHazardReset(t *testing.T) {
	h := hazard.GetComplexHazard("spinning_blade_pillar")
	h.IsTriggered = true

	h.Reset()

	if h.IsTriggered {
		t.Error("Reset should clear triggered state")
	}
}

func TestHazardPositionFiltering(t *testing.T) {
	h := hazard.NewHazard("test", "Test Hazard", 1)
	h.Position = "room_a"

	inRoom := entity.NewEntity("in", "In Room", 1)
	inRoom.Position = "room_a"

	outOfRoom := entity.NewEntity("out", "Out of Room", 1)
	outOfRoom.Position = "room_b"

	targets := []*entity.Entity{inRoom, outOfRoom}

	// Internal filter method (tested via TakeTurn behavior)
	// Only inRoom should be affected

	h.Routine = hazard.NewRoutine(1).
		AddAttack("Test Attack", 1, 10, dice.DieRoll{Count: 1, Sides: 4}, item.Bludgeoning, 1)

	result := h.TakeTurn(targets)

	if len(result.ActionResults[0].Targets) != 1 {
		t.Errorf("Should only affect 1 target, got %d", len(result.ActionResults[0].Targets))
	}

	if result.ActionResults[0].Targets[0].EntityID != "in" {
		t.Error("Wrong target affected")
	}
}
