package spell_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/spell"
)

func TestRitualMaterialRefundLogic(t *testing.T) {
	// Setup ritual that refunds on failure
	effect := &spell.GenericRitualEffect{
		SuccessDesc:     "Success",
		FailureDesc:     "Failure",
		CritFailureDesc: "Crit Fail",
		RefundOnFailure: true,
	}
	ritual := spell.NewRitual("test", "Test Ritual", 1, ability.SkillArcana, 0)
	ritual.Effect = effect
	ritual.CostCP = 1000
	ritual.PrimaryDC = 20

	caster := entity.NewEntity("caster", "Caster", 1)
	caster.Abilities.Intelligence = 10 // +0 mod
	caster.SkillProficiencies[ability.SkillArcana] = ability.Trained // +3 (level 1+2)

	// 1. Failure (Natural 10 + 3 = 13. DC 20 = Failure)
	attempt, _ := spell.NewRitualCastAttempt(ritual, caster, nil)
	outcome := spell.CastRitual(attempt, 10, nil)

	if outcome.Success {
		t.Errorf("Should have failed (Total %d vs DC %d)", attempt.PrimaryResult.Total, attempt.PrimaryResult.DC)
	}
	if !outcome.RefundMaterials {
		t.Error("Materials should be refunded on failure")
	}

	// 2. Critical Failure (Natural 1. Total 4 vs DC 20 = Crit Fail)
	attempt2, _ := spell.NewRitualCastAttempt(ritual, caster, nil)
	outcome2 := spell.CastRitual(attempt2, 1, nil)

	if outcome2.Success {
		t.Error("Should have failed")
	}
	if outcome2.RefundMaterials {
		t.Error("Materials should NOT be refunded on critical failure")
	}
}

func TestRitualAttributeMismatch(t *testing.T) {
	caster := entity.NewEntity("clumsy", "Clumsy", 1)
	caster.Abilities.Charisma = 8 // -1 modifier
	caster.SkillProficiencies[ability.SkillPerformance] = ability.Trained // +3 bonus

	ritual := spell.NewRitual("ritual", "Social Ritual", 1, ability.SkillPerformance, 0)
	ritual.PrimaryDC = 20

	attempt, _ := spell.NewRitualCastAttempt(ritual, caster, nil)
	outcome := spell.CastRitual(attempt, 10, nil) // 10 - 1 + 3 = 12. Fail.

	if outcome.Success {
		t.Error("Ritual should have failed due to low attribute modifier")
	}
}

func TestCrowdedRitual(t *testing.T) {
	ritual := spell.NewRitual("crowded", "Crowded Ritual", 1, ability.SkillArcana, 1)
	ritual.WithSecondaryCheck(ability.SkillArcana, ability.Trained, "Support")
	ritual.PrimaryDC = 20
	
	primary := entity.NewEntity("p", "Primary", 1)
	primary.Abilities.Intelligence = 18 // +4 mod
	primary.SkillProficiencies[ability.SkillArcana] = ability.Trained // +3 (lvl 1+2)
	
	s1 := entity.NewEntity("s1", "S1", 1)
	s1.SkillProficiencies[ability.SkillArcana] = ability.Trained

	secondaries := []*entity.Entity{s1}
	
	attempt, _ := spell.NewRitualCastAttempt(ritual, primary, secondaries)
	
	// Primary: Natural 20 + 7 = 27 (Crit Success)
	// S1: Natural 20 (Crit Success) -> +1 step (already at max)
	// Final: Critical Success
	spell.CastRitual(attempt, 20, []int{20})
	
	if attempt.FinalDegree != check.CriticalSuccess {
		t.Errorf("Expected CriticalSuccess, got %v", attempt.FinalDegree)
	}
}