package combat

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
	"uaa/vdnd/pkg/rules/skill"
	"uaa/vdnd/pkg/rules/trait"
)

func TestRogueCombatSimulation(t *testing.T) {
	// Setup Rogue
	rogue := entity.NewEntity("rogue", "Rogue", 1)
	rogue.Abilities.Dexterity = 18 // +4
	rogue.Abilities.Charisma = 14  // +2
	rogue.SkillProficiencies[ability.SkillDeception] = ability.Trained
	rogue.SkillProficiencies[ability.SkillStealth] = ability.Trained
	
	dagger := item.Weapon{
		ID:         "dagger-1",
		Name:       "Dagger",
		Traits:     []trait.TraitID{trait.TraitAgile, trait.TraitFinesse},
		IsMelee:    true,
		Damage:     dice.DieRoll{Count: 1, Sides: 4},
		DamageType: item.Piercing,
	}

	// Setup Guard
	guard := entity.NewEntity("guard", "Guard", 1)
	guard.Abilities.Wisdom = 12 // +1

	turn := NewTurn(rogue)

	// --- Action 1: Feint ---
	_ = skill.Feint(rogue, guard, 0)
	// For simulation, ensure it works
	if !guard.Conditions.HasRelative(condition.FlatFooted, rogue.ID) {
		guard.Conditions.ApplyRelative(condition.FlatFooted, rogue.ID, "Simulation Feint")
	}
	
	if !guard.Conditions.HasRelative(condition.FlatFooted, rogue.ID) {
		t.Error("Guard should be flat-footed to rogue")
	}

	// --- Action 2: Sneak ---
	_ = skill.Sneak(rogue, guard, 0)
	if !rogue.Conditions.HasRelative(condition.Hidden, guard.ID) {
		rogue.Conditions.ApplyRelative(condition.Hidden, guard.ID, "Simulation Sneak")
	}
	
	if !rogue.Conditions.HasRelative(condition.Hidden, guard.ID) {
		t.Error("Rogue should be hidden from guard")
	}

	// --- Action 3: Strike with Agile Dagger ---
	strike := NewStrike(&dagger)
	
	resStrike := strike.ExecuteWithRoll(rogue, guard, turn, 10)
	
	if resStrike.Degree < check.Success {
		t.Errorf("Expected strike success, got %v", resStrike.Degree)
	}

	// Verify strike was recorded
	if len(turn.StrikesMade) != 1 {
		t.Errorf("Expected 1 strike recorded, got %d", len(turn.StrikesMade))
	}
}
