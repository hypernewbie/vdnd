package skill

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
)

func TestKeyAbilities(t *testing.T) {
	tests := []struct {
		id   ability.SkillID
		want ability.Ability
	}{
		{ability.SkillAthletics, ability.Strength},
		{ability.SkillAcrobatics, ability.Dexterity},
		{ability.SkillArcana, ability.Intelligence},
		{ability.SkillMedicine, ability.Wisdom},
		{ability.SkillDiplomacy, ability.Charisma},
	}

	for _, tt := range tests {
		if got := GetKeyAbility(tt.id); got != tt.want {
			t.Errorf("GetKeyAbility(%s) = %v, want %v", tt.id, got, tt.want)
		}
	}
}

func TestSkillModifiers(t *testing.T) {
	e := entity.NewEntity("e1", "Hero", 1)
	e.Abilities.Dexterity = 16 // +3

	// Untrained
	mod := e.GetSkillModifier(ability.SkillAcrobatics)
	if mod != 3 {
		t.Errorf("Expected mod 3, got %d", mod)
	}

	// Trained L1: 3 (dex) + 2 (trained) + 1 (lvl) = 6
	e.SkillProficiencies[ability.SkillAcrobatics] = ability.Trained
	mod = e.GetSkillModifier(ability.SkillAcrobatics)
	if mod != 6 {
		t.Errorf("Expected mod 6, got %d", mod)
	}

	// Expert L5: 4 (str 18) + 4 (expert) + 5 (lvl) = 13
	e2 := entity.NewEntity("e2", "Expert", 5)
	e2.Abilities.Strength = 18
	e2.SkillProficiencies[ability.SkillAthletics] = ability.Expert
	mod2 := e2.GetSkillModifier(ability.SkillAthletics)
	if mod2 != 13 {
		t.Errorf("Expected mod 13, got %d", mod2)
	}
}

func TestDCValues(t *testing.T) {
	if got := DifficultyDC(DifficultyTrained); got != 15 {
		t.Errorf("DifficultyTrained DC = %d, want 15", got)
	}
	if got := LevelBasedDC(1); got != 15 {
		t.Errorf("Level 1 DC = %d, want 15", got)
	}
	if got := LevelBasedDC(10); got != 27 {
		t.Errorf("Level 10 DC = %d, want 27", got)
	}
	if got := AdjustedDC(20, AdjustHard); got != 22 {
		t.Errorf("Adjusted DC = %d, want 22", got)
	}
}

func TestSkillActions(t *testing.T) {
	actor := entity.NewEntity("a1", "Actor", 1)
	actor.Abilities.Charisma = 16
	actor.SkillProficiencies[ability.SkillIntimidation] = ability.Trained

	target := entity.NewEntity("t1", "Target", 1)
	target.Abilities.Wisdom = 10
	target.Will = ability.Trained // DC 13

	target.Conditions.Remove(condition.Frightened)
	Demoralize(actor, target)
}

func TestTreatWounds(t *testing.T) {
	healer := entity.NewEntity("h1", "Healer", 1)
	healer.Abilities.Wisdom = 14
	patient := entity.NewEntity("p1", "Patient", 1)
	patient.MaxHP = 20
	patient.CurrentHP = 10

	// Untrained should fail
	healing, res := TreatWounds(healer, patient, 15)
	if res.Degree != check.Failure || healing != 0 {
		t.Error("Untrained TreatWounds should return Failure")
	}

	healer.SkillProficiencies[ability.SkillMedicine] = ability.Trained
	healer.SkillProficiencies[ability.SkillMedicine] = ability.Trained
	TreatWounds(healer, patient, 15)
}

func TestCombatManeuvers(t *testing.T) {
	attacker := entity.NewEntity("a1", "Attacker", 1)
	attacker.Abilities.Strength = 18                                      // +4
	attacker.SkillProficiencies[ability.SkillAthletics] = ability.Trained // +7 total at lvl 1

	target := entity.NewEntity("t1", "Target", 1)
	target.Abilities.Dexterity = 14    // +2
	target.Abilities.Constitution = 14 // +2
	target.Reflex = ability.Trained    // +2+2+1 = +5, DC 15
	target.Fortitude = ability.Trained // +2+2+1 = +5, DC 15

	// Trip vs Reflex DC 15
	// We can't guarantee success without fixed rng, but can run code paths
	Trip(attacker, target, nil)

	// Grapple vs Fort DC 15
	Grapple(attacker, target, nil)

	// Shove vs Fort DC 15
	Shove(attacker, target, nil)
}

func TestHideSeek(t *testing.T) {
	hider := entity.NewEntity("h1", "Hider", 1)
	hider.Abilities.Dexterity = 14
	hider.SkillProficiencies[ability.SkillStealth] = ability.Trained // +5 at lvl 1

	seeker := entity.NewEntity("s1", "Seeker", 1)
	seeker.Abilities.Wisdom = 14
	seeker.Perception = ability.Trained // +5

	// Hide vs seeker Perception DC
	Hide(hider, seeker)

	// Seek vs standard DC 15
	Seek(seeker, 15, nil)
}
