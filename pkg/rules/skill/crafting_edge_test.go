package skill_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/skill"
)

func TestCraftingCatastrophicFailure(t *testing.T) {
	e := entity.NewEntity("crafter", "Crafter", 1)
	e.SkillProficiencies[ability.SkillCrafting] = ability.Trained
	
	project := skill.NewCraftingProject("sword", "Steel Sword", 1, 4000) // 40gp
	initialMaterials := project.MaterialsSpent // 2000 CP

	// Force critical failure (natural 1)
	_, res := skill.CraftDailyCheck(e, project, 1)
	
	if res.Degree != check.CriticalFailure {
		t.Errorf("Expected CriticalFailure, got %v", res.Degree)
	}
	
	if project.MaterialsSpent >= initialMaterials {
		t.Error("Materials should have been reduced on critical failure")
	}
	
	expectedLoss := initialMaterials / 10
	if project.MaterialsSpent != initialMaterials - expectedLoss {
		t.Errorf("Expected %d materials remaining, got %d", initialMaterials - expectedLoss, project.MaterialsSpent)
	}
}

func TestLevel20Economy(t *testing.T) {
	// Level 20 Legendary crafter Earns Income
	amount := skill.GetEarnIncomeAmount(20, ability.Legendary)
	expected := 20000 // 200 gp
	
	if amount != expected {
		t.Errorf("Level 20 Legendary should earn %d CP, got %d", expected, amount)
	}
}

func TestUnderLevelledCrafting(t *testing.T) {
	e := entity.NewEntity("noob", "Noob", 1)
	e.SkillProficiencies[ability.SkillCrafting] = ability.Trained
	
	// Level 10 item requires Expert
	project := skill.NewCraftingProject("fancy_item", "Fancy Item", 10, 10000)
	
	// Daily check should fail due to proficiency
	progress, res := skill.CraftDailyCheck(e, project, 20) // Natural 20 but prof is low
	
	if res.Degree != check.Failure {
		t.Errorf("Should fail due to insufficient proficiency, got %v", res.Degree)
	}
	if progress != 0 {
		t.Errorf("Should make 0 progress, got %d", progress)
	}
}
