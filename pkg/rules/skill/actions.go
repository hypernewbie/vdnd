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

// PerformSkillCheckWithRoll makes a skill check with a fixed natural roll
func PerformSkillCheckWithRoll(actor *entity.Entity, id ability.SkillID, dc, naturalRoll int) check.CheckResult {
	mod := actor.GetSkillModifier(id)
	return check.PerformCheckWithRoll(naturalRoll, mod, nil, dc)
}

// --- Acrobatics ---

func Balance(actor *entity.Entity, dc int) check.CheckResult {
	res := PerformSkillCheck(actor, ability.SkillAcrobatics, dc)
	if res.Degree == check.CriticalFailure {
		actor.Conditions.Apply(condition.NewCondition(condition.Prone, "Balance (Crit Fail)"))
	}
	return res
}

func TumbleThrough(actor, target *entity.Entity) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveReflex)
	res := PerformSkillCheck(actor, ability.SkillAcrobatics, dc)
	// Success: Move through enemy space. Failure: Movement ends.
	return res
}

func ManeuverInFlight(actor *entity.Entity, dc int) check.CheckResult {
	return PerformSkillCheck(actor, ability.SkillAcrobatics, dc)
}

func Squeeze(actor *entity.Entity, dc int) check.CheckResult {
	return PerformSkillCheck(actor, ability.SkillAcrobatics, dc)
}

// --- Athletics ---

func Climb(actor *entity.Entity, dc int) check.CheckResult {
	res := PerformSkillCheck(actor, ability.SkillAthletics, dc)
	if res.Degree == check.CriticalFailure {
		actor.Conditions.Apply(condition.NewCondition(condition.Prone, "Climb (Crit Fail)"))
		// Fall damage logic would be handled by caller based on height
	}
	return res
}

func Swim(actor *entity.Entity, dc int) check.CheckResult {
	return PerformSkillCheck(actor, ability.SkillAthletics, dc)
}

func HighJump(actor *entity.Entity, dc int) (int, check.CheckResult) {
	res := PerformSkillCheck(actor, ability.SkillAthletics, dc)
	dist := 0
	if res.Degree == check.CriticalSuccess {
		dist = 8 
	} else if res.Degree == check.Success {
		dist = 5
	}
	return dist, res
}

func LongJump(actor *entity.Entity, dc int) (int, check.CheckResult) {
	res := PerformSkillCheck(actor, ability.SkillAthletics, dc)
	dist := 0
	if res.Degree >= check.Success {
		dist = res.Total // Total result in feet
	}
	return dist, res
}

func Disarm(actor, target *entity.Entity) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveReflex)
	res := PerformSkillCheck(actor, ability.SkillAthletics, dc)
	switch res.Degree {
	case check.Success:
		// +2 bonus to further disarm until target turn (Caller handles or use temp immunity system for "penalty")
		target.AddTemporaryImmunity("disarm-bonus", actor.ID, 1)
	case check.CriticalFailure:
		actor.Conditions.Apply(condition.NewCondition(condition.FlatFooted, "Disarm (Crit Fail)"))
	}
	return res
}

func ForceOpen(actor *entity.Entity, dc int) check.CheckResult {
	return PerformSkillCheck(actor, ability.SkillAthletics, dc)
}

func Grapple(attacker, target *entity.Entity, modifiers []check.Modifier) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveFortitude)
	athletics := attacker.GetSkillModifier(ability.SkillAthletics)
	res := check.PerformCheck(athletics, modifiers, dc)
	switch res.Degree {
	case check.CriticalSuccess:
		target.Conditions.Apply(condition.NewCondition(condition.Restrained, "Grapple"))
	case check.Success:
		target.Conditions.Apply(condition.NewCondition(condition.Grabbed, "Grapple"))
	}
	return res
}

func Trip(attacker, target *entity.Entity, modifiers []check.Modifier) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveReflex)
	athletics := attacker.GetSkillModifier(ability.SkillAthletics)
	res := check.PerformCheck(athletics, modifiers, dc)
	switch res.Degree {
	case check.CriticalSuccess:
		target.Conditions.Apply(condition.NewCondition(condition.Prone, "Trip"))
		target.ApplyDamage(dice.DieRoll{Count: 1, Sides: 6}.Roll())
	case check.Success:
		target.Conditions.Apply(condition.NewCondition(condition.Prone, "Trip"))
	case check.CriticalFailure:
		attacker.Conditions.Apply(condition.NewCondition(condition.Prone, "Trip (Crit Fail)"))
	}
	return res
}

// --- Deception ---

func CreateADiversion(actor *entity.Entity, observers []*entity.Entity) []check.CheckResult {
	results := make([]check.CheckResult, 0)
	for _, obs := range observers {
		res := PerformSkillCheck(actor, ability.SkillDeception, 10 + obs.GetPerception())
		if res.Degree >= check.Success {
			actor.Conditions.ApplyRelative(condition.Hidden, obs.ID, "Diversion")
		}
		results = append(results, res)
	}
	return results
}

func Feint(actor, target *entity.Entity) check.CheckResult {
	dc := 10 + target.GetPerception()
	res := PerformSkillCheck(actor, ability.SkillDeception, dc)
	switch res.Degree {
	case check.CriticalSuccess:
		target.Conditions.Apply(condition.NewRelationalCondition(condition.FlatFooted, []string{actor.ID}, "Feint (Crit)"))
	case check.Success:
		target.Conditions.Apply(condition.NewRelationalCondition(condition.FlatFooted, []string{actor.ID}, "Feint"))
	case check.CriticalFailure:
		actor.Conditions.Apply(condition.NewRelationalCondition(condition.FlatFooted, []string{target.ID}, "Feint (Crit Fail)"))
	}
	return res
}

// --- Intimidation ---

func Demoralize(actor, target *entity.Entity) check.CheckResult {
	if target.IsTemporarilyImmune("demoralize", actor.ID) {
		return check.CheckResult{Degree: check.Failure}
	}
	dc := target.GetSaveDC(ability.SaveWill)
	res := PerformSkillCheck(actor, ability.SkillIntimidation, dc)
	switch res.Degree {
	case check.CriticalSuccess:
		target.Conditions.Apply(condition.NewValuedCondition(condition.Frightened, 2, "Demoralize"))
	case check.Success:
		target.Conditions.Apply(condition.NewValuedCondition(condition.Frightened, 1, "Demoralize"))
	}
	target.AddTemporaryImmunity("demoralize", actor.ID, 100) // 10 mins
	return res
}

// --- Medicine ---

func TreatWounds(healer, patient *entity.Entity, dc int) (int, check.CheckResult) {
	if healer.SkillProficiencies[ability.SkillMedicine] < ability.Trained {
		return 0, check.CheckResult{Degree: check.Failure}
	}
	res := PerformSkillCheck(healer, ability.SkillMedicine, dc)
	healing := 0
	switch res.Degree {
	case check.CriticalSuccess:
		healing = dice.DieRoll{Count: 4, Sides: 8}.Roll()
	case check.Success:
		healing = dice.DieRoll{Count: 2, Sides: 8}.Roll()
	case check.CriticalFailure:
		patient.ApplyDamage(dice.DieRoll{Count: 1, Sides: 8}.Roll())
	}
	if healing > 0 { patient.Heal(healing) }
	return healing, res
}

func AdministerFirstAid(healer, patient *entity.Entity, dc int, stabilize bool) check.CheckResult {
	return PerformSkillCheck(healer, ability.SkillMedicine, dc)
}

// --- Stealth ---

func Sneak(actor *entity.Entity, observer *entity.Entity) check.CheckResult {
	dc := 10 + observer.GetPerception()
	res := PerformSkillCheck(actor, ability.SkillStealth, dc)
	if res.Degree >= check.Success {
		actor.Conditions.ApplyRelative(condition.Hidden, observer.ID, "Sneak")
	} else {
		actor.Conditions.RemoveRelative(condition.Hidden, observer.ID)
	}
	return res
}

// --- Thievery ---

func PickLock(actor *entity.Entity, dc int, successesRequired int) (int, check.CheckResult) {
	res := PerformSkillCheck(actor, ability.SkillThievery, dc)
	progress := 0
	if res.Degree == check.CriticalSuccess { progress = 2 } else if res.Degree == check.Success { progress = 1 }
	return progress, res
}

func LevelBasedDC(level int) int {
	dcs := []int{14, 15, 16, 18, 19, 20, 22, 23, 24, 26, 27, 28, 30, 31, 32, 34, 35, 36, 38, 39, 40, 42, 44, 46, 48, 50}
	if level < 0 { return 13 }
	if level >= len(dcs) { return dcs[len(dcs)-1] + (level-25)*2 }
	return dcs[level]
}