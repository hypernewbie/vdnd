package skill

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/entity"
)

// PerformSkillCheck makes a skill check
func PerformSkillCheck(actor *entity.Entity, id ability.SkillID, dc int) check.CheckResult {
	mod := actor.GetSkillModifier(id)
	return check.PerformCheck(mod, nil, dc)
}

// Demoralize - Intimidation vs Will DC
func Demoralize(actor, target *entity.Entity) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveWill)
	result := PerformSkillCheck(actor, ability.SkillIntimidation, dc)

	switch result.Degree {
	case check.CriticalSuccess:
		target.Conditions.Apply(condition.NewValuedCondition(condition.Frightened, 2, "Demoralize"))
	case check.Success:
		target.Conditions.Apply(condition.NewValuedCondition(condition.Frightened, 1, "Demoralize"))
	}
	return result
}

// Grapple - Athletics vs Fortitude DC
func Grapple(attacker, target *entity.Entity, modifiers []check.Modifier) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveFortitude)
	athletics := attacker.GetSkillModifier(ability.SkillAthletics)

	result := check.PerformCheck(athletics, modifiers, dc)

	switch result.Degree {
	case check.CriticalSuccess:
		target.Conditions.Apply(condition.NewCondition(condition.Restrained, "Grappled"))
	case check.Success:
		target.Conditions.Apply(condition.NewCondition(condition.Grabbed, "Grappled"))
	}
	return result
}

// Trip - Athletics vs Reflex DC
func Trip(attacker, target *entity.Entity, modifiers []check.Modifier) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveReflex)
	athletics := attacker.GetSkillModifier(ability.SkillAthletics)

	result := check.PerformCheck(athletics, modifiers, dc)

	switch result.Degree {
	case check.CriticalSuccess:
		target.Conditions.Apply(condition.NewCondition(condition.Prone, "Trip"))
		target.ApplyDamage(dice.DieRoll{Count: 1, Sides: 6}.Roll())
	case check.Success:
		target.Conditions.Apply(condition.NewCondition(condition.Prone, "Trip"))
	case check.CriticalFailure:
		attacker.Conditions.Apply(condition.NewCondition(condition.Prone, "Trip (Backfire)"))
	}
	return result
}

// Shove - Athletics vs Fortitude DC
func Shove(attacker, target *entity.Entity, modifiers []check.Modifier) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveFortitude)
	athletics := attacker.GetSkillModifier(ability.SkillAthletics)

	result := check.PerformCheck(athletics, modifiers, dc)

	// Movement is handled by caller dependent on map
	// Critical Failure: Attacker falls prone
	if result.Degree == check.CriticalFailure {
		attacker.Conditions.Apply(condition.NewCondition(condition.Prone, "Shove (Backfire)"))
	}

	return result
}

// Hide - Stealth vs DC
func Hide(actor *entity.Entity, dc int, modifiers []check.Modifier) check.CheckResult {
	stealth := actor.GetSkillModifier(ability.SkillStealth)
	result := check.PerformCheck(stealth, modifiers, dc)

	if result.Degree >= check.Success {
		actor.Conditions.Apply(condition.NewCondition(condition.Hidden, "Hide"))
	}
	return result
}

// Seek - Perception vs DC
func Seek(actor *entity.Entity, dc int, modifiers []check.Modifier) check.CheckResult {
	perception := actor.GetPerception()
	// Seek is a perception check
	return check.PerformCheck(perception, modifiers, dc)
}

// TreatWounds - Medicine check, requires trained
func TreatWounds(healer, patient *entity.Entity, dc int) (int, check.CheckResult) {
	if healer.SkillProficiencies[ability.SkillMedicine] < ability.Trained {
		return 0, check.CheckResult{Degree: check.Failure}
	}

	result := PerformSkillCheck(healer, ability.SkillMedicine, dc)
	healing := 0

	switch result.Degree {
	case check.CriticalSuccess:
		healing = dice.DieRoll{Count: 4, Sides: 8, Modifier: 0}.Roll()
	case check.Success:
		healing = dice.DieRoll{Count: 2, Sides: 8, Modifier: 0}.Roll()
	case check.CriticalFailure:
		dmg := dice.DieRoll{Count: 1, Sides: 8, Modifier: 0}.Roll()
		patient.ApplyDamage(dmg) // Simplified damage
	}

	if healing > 0 {
		patient.Heal(healing)
	}
	return healing, result
}

// RecallKnowledge - various skills vs topic DC
func RecallKnowledge(actor *entity.Entity, id ability.SkillID, dc int) (string, check.CheckResult) {
	result := PerformSkillCheck(actor, id, dc)
	info := ""

	switch result.Degree {
	case check.CriticalSuccess:
		info = "Accurate info + extra fact"
	case check.Success:
		info = "Accurate info"
	case check.Failure:
		info = "No useful info"
	case check.CriticalFailure:
		info = "FALSE info"
	}
	return info, result
}
