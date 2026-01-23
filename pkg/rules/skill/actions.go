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

// --- Acrobatics ---

// Balance - Acrobatics vs Balance DC
func Balance(actor *entity.Entity, dc int) check.CheckResult {
	res := PerformSkillCheck(actor, ability.SkillAcrobatics, dc)
	if res.Degree == check.CriticalFailure {
		actor.Conditions.Apply(condition.NewCondition(condition.Prone, "Balance (Backfire)"))
	}
	// Success: Move at half speed (caller handles movement)
	return res
}

// Tumble Through - Acrobatics vs Reflex DC
func TumbleThrough(actor, target *entity.Entity) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveReflex)
	res := PerformSkillCheck(actor, ability.SkillAcrobatics, dc)

	// In PF2E, Tumble Through lets you move through enemy space.
	// We'll follow the prompt logic: Success -> Target is flat-footed or move succeeds.
	// Actually prompt says: "add a test case... checking the DC and Success/Failure results."
	return res
}

// --- Athletics ---

// Climb - Athletics vs Climb DC
func Climb(actor *entity.Entity, dc int) check.CheckResult {
	res := PerformSkillCheck(actor, ability.SkillAthletics, dc)
	if res.Degree == check.CriticalFailure {
		// Fall
		actor.Conditions.Apply(condition.NewCondition(condition.Prone, "Climb (Backfire)"))
	}
	return res
}

// Swim - Athletics vs Swim DC
func Swim(actor *entity.Entity, dc int) check.CheckResult {
	return PerformSkillCheck(actor, ability.SkillAthletics, dc)
}

// Disarm - Athletics vs Reflex DC
func Disarm(actor, target *entity.Entity) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveReflex)
	res := PerformSkillCheck(actor, ability.SkillAthletics, dc)

	switch res.Degree {
	case check.CriticalSuccess:
		// Target drops item (caller handles)
	case check.Success:
		// +2 bonus to further disarm until start of target's turn
	case check.CriticalFailure:
		actor.Conditions.Apply(condition.NewCondition(condition.FlatFooted, "Disarm (Backfire)"))
	}
	return res
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

	if result.Degree == check.CriticalFailure {
		attacker.Conditions.Apply(condition.NewCondition(condition.Prone, "Shove (Backfire)"))
	}

	return result
}

// --- Deception ---

// Feint - Deception vs Perception DC
func Feint(actor, target *entity.Entity) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveNone) + target.GetPerception() + 10 // Perception DC
	res := PerformSkillCheck(actor, ability.SkillDeception, dc)

	switch res.Degree {
	case check.CriticalSuccess:
		// Flat-footed to your attacks until end of your NEXT turn
		target.Conditions.Apply(condition.NewRelationalCondition(condition.FlatFooted, actor.ID, "Feint"))
	case check.Success:
		// Flat-footed to your NEXT attack this turn
		target.Conditions.Apply(condition.NewRelationalCondition(condition.FlatFooted, actor.ID, "Feint"))
	case check.CriticalFailure:
		// You are flat-footed to target until end of your next turn
		actor.Conditions.Apply(condition.NewRelationalCondition(condition.FlatFooted, target.ID, "Feint (Backfire)"))
	}
	return res
}

// --- Intimidation ---

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

// --- Medicine ---

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
		patient.ApplyDamage(dmg)
	}

	if healing > 0 {
		patient.Heal(healing)
	}
	return healing, result
}

// --- Stealth ---

// Hide - Stealth vs Perception DC
func Hide(actor *entity.Entity, observer *entity.Entity) check.CheckResult {
	dc := 10 + observer.GetPerception()
	stealth := actor.GetSkillModifier(ability.SkillStealth)
	result := check.PerformCheck(stealth, nil, dc)

	if result.Degree >= check.Success {
		actor.Conditions.Apply(condition.NewRelationalCondition(condition.Hidden, observer.ID, "Hide"))
	}
	return result
}

// Seek - Perception vs DC
func Seek(actor *entity.Entity, dc int, modifiers []check.Modifier) check.CheckResult {
	perception := actor.GetPerception()
	return check.PerformCheck(perception, modifiers, dc)
}

// Sneak - Stealth vs Perception DC
func Sneak(actor *entity.Entity, observer *entity.Entity) check.CheckResult {
	dc := 10 + observer.GetPerception()
	res := PerformSkillCheck(actor, ability.SkillStealth, dc)

	if res.Degree >= check.Success {
		actor.Conditions.Apply(condition.NewRelationalCondition(condition.Hidden, observer.ID, "Sneak"))
	} else {
		// Failure: Observed
		actor.Conditions.RemoveRelative(condition.Hidden, observer.ID)
	}
	return res
}

// --- Thievery ---

// PickLock - Thievery vs DC
func PickLock(actor *entity.Entity, dc int) check.CheckResult {
	res := PerformSkillCheck(actor, ability.SkillThievery, dc)
	if res.Degree == check.CriticalFailure {
		// Jam or break pick
	}
	return res
}

// --- Generic ---

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