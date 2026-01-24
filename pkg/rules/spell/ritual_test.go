package spell_test

import (
	"fmt"
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/spell"
)

func setupEntity(id string, level int) *entity.Entity {
	e := entity.NewEntity(id, id, level)
	e.Abilities = ability.AbilityScores{
		Strength:     10,
		Dexterity:    10,
		Constitution: 10,
		Intelligence: 10,
		Wisdom:       10,
		Charisma:     10,
	}
	return e
}

func TestRitualCreation(t *testing.T) {
	ritual := spell.NewRitual("test", "Test Ritual", 3, ability.SkillReligion, 2).
		WithCastingTime(1, spell.DurationHours).
		WithCost(5000).
		WithSecondaryCheck(ability.SkillMedicine, ability.Trained, "Assist")

	if ritual.Rank != 3 {
		t.Errorf("Expected rank 3, got %d", ritual.Rank)
	}
	if ritual.SecondaryCasters != 2 {
		t.Errorf("Expected 2 secondary casters, got %d", ritual.SecondaryCasters)
	}
	if ritual.CostCP != 5000 {
		t.Errorf("Expected cost 5000 cp, got %d", ritual.CostCP)
	}
	if len(ritual.SecondaryChecks) != 1 {
		t.Errorf("Expected 1 secondary check, got %d", len(ritual.SecondaryChecks))
	}
}

func TestRitualCastAttemptValidation(t *testing.T) {
	ritual := spell.GetRitual("resurrect")
	if ritual == nil {
		t.Fatal("Resurrect ritual not found")
	}

	primary := setupEntity("cleric", 10)
	primary.SkillProficiencies[ability.SkillReligion] = ability.Expert

	secondary1 := setupEntity("healer", 8)
	secondary1.SkillProficiencies[ability.SkillMedicine] = ability.Expert

	secondary2 := setupEntity("acolyte", 6)
	secondary2.SkillProficiencies[ability.SkillReligion] = ability.Trained

	attempt, err := spell.NewRitualCastAttempt(ritual, primary, []*entity.Entity{secondary1, secondary2})
	if err != nil {
		t.Fatalf("Should create valid attempt: %v", err)
	}

	if attempt.PrimaryCaster != primary {
		t.Error("Primary caster mismatch")
	}
}

func TestRitualCastAttemptInsufficientCasters(t *testing.T) {
	ritual := spell.GetRitual("resurrect") // Requires 2 secondary
	if ritual == nil {
		t.Fatal("Resurrect ritual not found")
	}

	primary := setupEntity("cleric", 10)
	primary.SkillProficiencies[ability.SkillReligion] = ability.Expert

	// Only 1 secondary when 2 required
	secondary1 := setupEntity("healer", 8)
	secondary1.SkillProficiencies[ability.SkillMedicine] = ability.Expert

	_, err := spell.NewRitualCastAttempt(ritual, primary, []*entity.Entity{secondary1})
	if err == nil {
		t.Error("Should fail with insufficient secondary casters")
	}
}

func TestRitualCasting(t *testing.T) {
	ritual := spell.GetRitual("commune") // No secondary casters needed
	if ritual == nil {
		t.Fatal("Commune ritual not found")
	}

	primary := setupEntity("oracle", 12)
	primary.SkillProficiencies[ability.SkillReligion] = ability.Master

	attempt, err := spell.NewRitualCastAttempt(ritual, primary, nil)
	if err != nil {
		t.Fatalf("Should create valid attempt: %v", err)
	}

	// Force a success
	outcome := spell.CastRitual(attempt, 18, nil)

	t.Logf("Ritual outcome: degree=%v, success=%v, desc=%s",
		attempt.FinalDegree, outcome.Success, outcome.Description)

	if !attempt.IsComplete {
		t.Error("Ritual should be marked complete")
	}
	if attempt.MaterialsConsumed != ritual.CostCP {
		t.Errorf("Materials consumed should be %d, got %d", ritual.CostCP, attempt.MaterialsConsumed)
	}
}

func TestSecondaryCheckModifiers(t *testing.T) {
	ritual := spell.NewRitual("test", "Test", 3, ability.SkillReligion, 2).
		WithSecondaryCheck(ability.SkillMedicine, ability.Trained, "").
		WithSecondaryCheck(ability.SkillArcana, ability.Trained, "")
	ritual.Effect = &spell.GenericRitualEffect{
		SuccessDesc:     "Success",
		CritSuccessDesc: "Critical!",
	}

	primary := setupEntity("caster", 10)
	primary.SkillProficiencies[ability.SkillReligion] = ability.Expert

	sec1 := setupEntity("sec1", 5)
	sec1.SkillProficiencies[ability.SkillMedicine] = ability.Trained

	sec2 := setupEntity("sec2", 5)
	sec2.SkillProficiencies[ability.SkillArcana] = ability.Trained

	attempt, _ := spell.NewRitualCastAttempt(ritual, primary, []*entity.Entity{sec1, sec2})

	// Primary succeeds (not crit), both secondaries crit succeed
	// DC for rank 3 ritual (lvl 6) is 22.
	// Primary: lvl 10 + Expert 4 = 14 mod. 15 roll + 14 = 29. 29 vs 22 = Success.
	// Secondaries: lvl 5 + Trained 2 = 7 mod. 20 roll + 7 = 27. 27 vs 22 = Success (not crit unless nat 20 adjusts it).
	// Nat 20 on secondary should shift Success to Crit Success.
	
	outcome := spell.CastRitual(attempt, 15, []int{20, 20})

	// Two secondary crit successes should boost Primary Success to Critical Success.
	if attempt.FinalDegree != check.CriticalSuccess {
		t.Errorf("Expected final degree CriticalSuccess, got %v", attempt.FinalDegree)
	}
	
	if outcome.Description != "Critical!" {
		t.Errorf("Expected 'Critical!' description, got %s", outcome.Description)
	}
}

func TestRitualBacklash(t *testing.T) {
	ritual := spell.GetRitual("plane_shift")
	if ritual == nil {
		t.Fatal("Plane Shift ritual not found")
	}

	primary := setupEntity("wizard", 14)
	primary.SkillProficiencies[ability.SkillOccultism] = ability.Master

	secondaries := make([]*entity.Entity, 3)
	for i := 0; i < 3; i++ {
		sec := setupEntity(fmt.Sprintf("sec%d", i), 8)
		sec.SkillProficiencies[ability.SkillArcana] = ability.Trained
		sec.SkillProficiencies[ability.SkillOccultism] = ability.Trained
		sec.SkillProficiencies[ability.SkillSurvival] = ability.Trained
		secondaries[i] = sec
	}

	attempt, _ := spell.NewRitualCastAttempt(ritual, primary, secondaries)

	// Force critical failure
	outcome := spell.CastRitual(attempt, 1, []int{1, 1, 1})

	if attempt.FinalDegree != check.CriticalFailure {
		t.Errorf("Expected CriticalFailure, got %v", attempt.FinalDegree)
	}

	if outcome.Backlash == "" {
		t.Error("Expected backlash on critical failure")
	}
}

func TestRitualMaterialRefund(t *testing.T) {
	ritual := spell.GetRitual("resurrect") // RefundOnFailure: true
	primary := setupEntity("cleric", 10)
	primary.SkillProficiencies[ability.SkillReligion] = ability.Expert

	secondaries := []*entity.Entity{
		setupEntity("s1", 10),
		setupEntity("s2", 10),
	}
	secondaries[0].SkillProficiencies[ability.SkillMedicine] = ability.Expert
	secondaries[1].SkillProficiencies[ability.SkillReligion] = ability.Trained

	// 1. Test Failure (Refund)
	attempt, _ := spell.NewRitualCastAttempt(ritual, primary, secondaries)
	// Mod is lvl 10 + Expert 4 = 14. DC for rank 5 is lvl 10 DC = 27.
	// Roll 5 + 14 = 19 (Failure).
	// Secondaries: 
	// S1: lvl 10 + Expert 4 = 14 mod. DC 27. Roll 12 + 14 = 26 (Failure)
	// S2: lvl 10 + Trained 2 = 12 mod. DC 27. Roll 12 + 12 = 24 (Failure)
	spell.CastRitual(attempt, 5, []int{12, 12})

	if attempt.MaterialsConsumed != 0 {
		t.Errorf("Expected 0 materials consumed on failure, got %d", attempt.MaterialsConsumed)
	}

	// 2. Test Critical Failure (No Refund)
	attempt2, _ := spell.NewRitualCastAttempt(ritual, primary, secondaries)
	// Roll 1 is auto crit fail.
	spell.CastRitual(attempt2, 1, []int{10, 10})

	if attempt2.MaterialsConsumed == 0 {
		t.Error("Expected materials to be consumed on critical failure")
	}
}
