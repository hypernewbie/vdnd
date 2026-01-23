package affliction_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/affliction"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
)

func TestAfflictionInstance(t *testing.T) {
	aff := &affliction.GiantCentipedeVenom
	inst := affliction.NewInstance(aff, "bite")

	if inst.CurrentStage != 1 {
		t.Errorf("Expected stage 1, got %d", inst.CurrentStage)
	}
	if !inst.IsActive() {
		t.Error("Expected active (no onset)")
	}

	dmg, _, conds := inst.GetCurrentEffects()
	if dmg.Count != 1 || dmg.Sides != 6 {
		t.Errorf("Expected 1d6 damage for stage 1, got %v", dmg)
	}
	if len(conds) != 1 || conds[0].ID != condition.FlatFooted {
		t.Error("Expected FlatFooted for stage 1")
	}
}

func TestOnsetDelay(t *testing.T) {
	aff := &affliction.ZombieRot
	inst := affliction.NewInstance(aff, "scratch")

	if inst.IsActive() {
		t.Error("Should not be active during onset")
	}

	// Tracker tick
	tracker := affliction.NewTracker()
	// We can't access unexported fields from another package
	// We should use Add
	tracker.Add(aff, "scratch")
	
	tracker.Tick(ability.IntervalDays)
	inst2 := tracker.Get(aff.ID)
	if !inst2.IsActive() {
		t.Error("Should be active after 1 day tick (onset 1)")
	}
}

func TestStageProgression(t *testing.T) {
	tracker := affliction.NewTracker()
	aff := &affliction.GiantCentipedeVenom
	tracker.Add(aff, "bite")

	// Stage 1 -> Success -> Cured (0)
	tracker.ProcessSave(aff.ID, check.Success)
	if tracker.Has(aff.ID) {
		t.Error("Should be cured after success at stage 1")
	}

	// Stage 1 -> Failure -> Stage 2
	tracker.Add(aff, "bite")
	tracker.ProcessSave(aff.ID, check.Failure)
	if inst := tracker.Get(aff.ID); inst.CurrentStage != 2 {
		t.Errorf("Expected stage 2, got %d", inst.CurrentStage)
	}

	// Stage 2 -> Critical Success -> Cured (0)
	tracker.ProcessSave(aff.ID, check.CriticalSuccess)
	if tracker.Has(aff.ID) {
		t.Error("Should be cured after critical success at stage 2")
	}

	// Stage 1 -> Critical Failure -> Stage 3
	tracker.Add(aff, "bite")
	tracker.ProcessSave(aff.ID, check.CriticalFailure)
	if inst := tracker.Get(aff.ID); inst.CurrentStage != 3 {
		t.Errorf("Expected stage 3, got %d", inst.CurrentStage)
	}

	// Stage 3 -> Critical Failure -> Stage 3 (Capped)
	tracker.ProcessSave(aff.ID, check.CriticalFailure)
	if inst := tracker.Get(aff.ID); inst.CurrentStage != 3 {
		t.Errorf("Expected capped stage 3, got %d", inst.CurrentStage)
	}
}

func TestEntityIntegration(t *testing.T) {
	e := entity.NewEntity("e1", "Target", 1)
	e.MaxHP = 20
	e.CurrentHP = 20

	e.Afflictions.Add(&affliction.GiantCentipedeVenom, "bite")

	// Process tick
	results := e.ProcessAfflictions(ability.IntervalRounds)
	if len(results) != 1 || results[0].AfflictionID != "giant-centipede-venom" {
		t.Error("Tick should have processed centipede venom")
	}

	// Check conditions applied
	if !e.Conditions.Has(condition.FlatFooted) {
		t.Error("Should have applied FlatFooted from venom")
	}

	// Damage verification
	if results[0].Damage == 0 {
		t.Error("Tick should have rolled damage")
	}
}