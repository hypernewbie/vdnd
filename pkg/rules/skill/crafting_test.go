package skill_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
	"uaa/vdnd/pkg/rules/skill"
)

func TestEarnIncomeTable(t *testing.T) {
	// Level 5, Trained = 9 sp = 90 cp
	earned := skill.GetEarnIncomeAmount(5, ability.Trained)
	if earned != 90 {
		t.Errorf("Expected 90 cp at level 5 Trained, got %d", earned)
	}

	// Level 5, Expert = 1 gp = 100 cp
	earned = skill.GetEarnIncomeAmount(5, ability.Expert)
	if earned != 100 {
		t.Errorf("Expected 100 cp at level 5 Expert, got %d", earned)
	}
}

func setupActor(level int) *entity.Entity {
	actor := entity.NewEntity("crafter", "Crafter", level)
	actor.Abilities = ability.AbilityScores{
		Strength:     10,
		Dexterity:    10,
		Constitution: 10,
		Intelligence: 10,
		Wisdom:       10,
		Charisma:     10,
	}
	return actor
}

func TestEarnIncome(t *testing.T) {
	actor := setupActor(5)
	actor.SkillProficiencies[ability.SkillCrafting] = ability.Trained

	// Force a success roll
	// DC for level 3 is 18
	// Mod should be 5 (level) + 2 (trained) + 0 (int) = 7
	// Total with 15 = 22
	result, res := skill.EarnIncome(actor, ability.SkillCrafting, 3, 15)

	if res.Degree != check.Success {
		t.Errorf("Expected Success, got %v (Total: %d, DC: %d, Mod: %d, Level: %d)",
			res.Degree, res.Total, res.DC, res.Total-res.NaturalRoll, actor.Level)
	}

	if result.EarnedCP != 50 { // Level 3 Success = 5 sp = 50 cp
		t.Errorf("Expected 50 cp, got %d", result.EarnedCP)
	}

	if result.DaysWorked != 1 {
		t.Error("Should always be 1 day worked per check")
	}
}

func TestEarnIncomeRequiresTraining(t *testing.T) {
	actor := setupActor(5)
	// No skill proficiencies set = Untrained

	result, res := skill.EarnIncome(actor, ability.SkillCrafting, 1, 20)

	if res.Degree != check.Failure {
		t.Error("Untrained should automatically fail Earn Income")
	}
	if result.EarnedCP != 0 {
		t.Errorf("Untrained should earn 0, got %d", result.EarnedCP)
	}
}

func TestCraftingProject(t *testing.T) {
	// Create a 10 gp item (1000 cp)
	project := skill.NewCraftingProject("longsword", "Longsword", 0, 1000)

	if project.MaterialsSpent != 500 {
		t.Errorf("Materials should be half price (500), got %d", project.MaterialsSpent)
	}

	// Setup phase (4 days)
	for i := 0; i < 3; i++ {
		if project.CraftSetup() {
			t.Error("Setup should not complete before 4 days")
		}
	}
	if !project.CraftSetup() {
		t.Error("Setup should complete on day 4")
	}
}

func TestCraftDailyProgress(t *testing.T) {
	actor := setupActor(5)
	actor.SkillProficiencies[ability.SkillCrafting] = ability.Trained

	project := skill.NewCraftingProject("dagger", "Dagger", 0, 200) // 2 gp dagger

	// Force critical success
	progress, res := skill.CraftDailyCheck(actor, project, 20)

	if res.Degree != check.CriticalSuccess {
		t.Errorf("Degree was %v, want Crit Success", res.Degree)
	}

	if project.ProgressCP != progress {
		t.Error("Project progress should match returned progress")
	}
}

func TestRepair(t *testing.T) {
	actor := setupActor(5)
	actor.SkillProficiencies[ability.SkillCrafting] = ability.Trained

	// Force success
	result, res := skill.Repair(actor, 0, 15)

	if res.Degree < check.Success {
		t.Errorf("Expected Success, got %v", res.Degree)
	}
	if !result.Repaired {
		t.Error("Success should repair item")
	}
}

func TestRepairProficiency(t *testing.T) {
	// Trained: 5 HP on Success, 10 HP on Crit
	actor := setupActor(5)
	actor.SkillProficiencies[ability.SkillCrafting] = ability.Trained

	resTrained, _ := skill.Repair(actor, 0, 15) // Success
	if resTrained.HPRestored != 5 {
		t.Errorf("Trained success should restore 5 HP, got %d", resTrained.HPRestored)
	}

	resTrainedCrit, _ := skill.Repair(actor, 0, 20) // Crit Success
	if resTrainedCrit.HPRestored != 10 {
		t.Errorf("Trained crit success should restore 10 HP, got %d", resTrainedCrit.HPRestored)
	}

	// Expert: 10 HP on Success, 20 HP on Crit
	actor.SkillProficiencies[ability.SkillCrafting] = ability.Expert
	resExpert, _ := skill.Repair(actor, 0, 10) // Success (10+9=19 vs DC 14)
	if resExpert.HPRestored != 10 {
		t.Errorf("Expert success should restore 10 HP, got %d", resExpert.HPRestored)
	}
}

func TestRepairShield(t *testing.T) {
	actor := setupActor(5)
	actor.SkillProficiencies[ability.SkillCrafting] = ability.Trained

	shield := item.NewShield("steel_shield", "Steel Shield", 2, 5, 20, 1)
	shield.CurrentHP = 5

	// Force Success (15) -> should restore 5 HP (Trained)
	skill.RepairShield(actor, shield, 15)

	if shield.CurrentHP != 10 {
		t.Errorf("Expected 10 HP after repair (5+5), got %d", shield.CurrentHP)
	}

	// Force Crit Success (20) -> should restore 10 HP (Trained)
	skill.RepairShield(actor, shield, 20)

	if shield.CurrentHP != 20 {
		t.Errorf("Expected 20 HP after crit repair (10+10), got %d", shield.CurrentHP)
	}
}
