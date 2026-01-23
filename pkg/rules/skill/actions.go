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

// PerformSkillCheckWithModifiers makes a skill check with extra modifiers
func PerformSkillCheckWithModifiers(actor *entity.Entity, id ability.SkillID, dc int, modifiers []check.Modifier) check.CheckResult {
	mod := actor.GetSkillModifier(id)
	return check.PerformCheck(mod, modifiers, dc)
}

// PerformSkillCheckWithRoll makes a skill check with a fixed natural roll
func PerformSkillCheckWithRoll(actor *entity.Entity, id ability.SkillID, dc, naturalRoll int) check.CheckResult {
	mod := actor.GetSkillModifier(id)
	return check.PerformCheckWithRoll(naturalRoll, mod, nil, dc)
}

// --- Acrobatics ---

func Balance(actor *entity.Entity, dc int, naturalRoll int) check.CheckResult {
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(actor, ability.SkillAcrobatics, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(actor, ability.SkillAcrobatics, dc)
	}

	if res.Degree == check.CriticalFailure {
		actor.Conditions.Apply(condition.NewCondition(condition.Prone, "Balance (Crit Fail)"))
	}
	return res
}

func TumbleThrough(actor, target *entity.Entity, naturalRoll int) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveReflex)
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(actor, ability.SkillAcrobatics, dc, naturalRoll)
	}
	return PerformSkillCheck(actor, ability.SkillAcrobatics, dc)
}

func ManeuverInFlight(actor *entity.Entity, dc int, naturalRoll int) check.CheckResult {
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(actor, ability.SkillAcrobatics, dc, naturalRoll)
	}
	return PerformSkillCheck(actor, ability.SkillAcrobatics, dc)
}

func Squeeze(actor *entity.Entity, dc int, naturalRoll int) check.CheckResult {
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(actor, ability.SkillAcrobatics, dc, naturalRoll)
	}
	return PerformSkillCheck(actor, ability.SkillAcrobatics, dc)
}

// --- Athletics ---

type MovementResult struct {
	Speed  int
	Damage int
}

func Climb(actor *entity.Entity, dc int, naturalRoll int) (MovementResult, check.CheckResult) {
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(actor, ability.SkillAthletics, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(actor, ability.SkillAthletics, dc)
	}

	move := MovementResult{}
	switch res.Degree {
	case check.CriticalSuccess:
		move.Speed = 10 // Success + 5? PF2E says move at full speed (usually 10ft)
	case check.Success:
		move.Speed = 5
	case check.CriticalFailure:
		actor.Conditions.Apply(condition.NewCondition(condition.Prone, "Climb (Crit Fail)"))
		move.Damage = dice.DieRoll{Count: 1, Sides: 6}.Roll() // Fall damage placeholder
	}
	return move, res
}

func Swim(actor *entity.Entity, dc int, naturalRoll int) (MovementResult, check.CheckResult) {
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(actor, ability.SkillAthletics, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(actor, ability.SkillAthletics, dc)
	}

	move := MovementResult{}
	if res.Degree == check.CriticalSuccess {
		move.Speed = 10
	} else if res.Degree == check.Success {
		move.Speed = 5
	}
	return move, res
}

func HighJump(actor *entity.Entity, dc int, naturalRoll int) (int, check.CheckResult) {
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(actor, ability.SkillAthletics, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(actor, ability.SkillAthletics, dc)
	}

	dist := 0
	if res.Degree == check.CriticalSuccess {
		dist = 8 
	} else if res.Degree == check.Success {
		dist = 5
	}
	return dist, res
}

func LongJump(actor *entity.Entity, dc int, naturalRoll int) (int, check.CheckResult) {
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(actor, ability.SkillAthletics, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(actor, ability.SkillAthletics, dc)
	}

	dist := 0
	if res.Degree >= check.Success {
		dist = res.Total 
	}
	return dist, res
}

func Disarm(actor, target *entity.Entity, naturalRoll int) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveReflex)
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(actor, ability.SkillAthletics, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(actor, ability.SkillAthletics, dc)
	}

	switch res.Degree {
	case check.Success:
		target.AddTemporaryImmunity("disarm-bonus", actor.ID, 1)
	case check.CriticalFailure:
		actor.Conditions.Apply(condition.NewCondition(condition.FlatFooted, "Disarm (Crit Fail)"))
	}
	return res
}

func ForceOpen(actor *entity.Entity, dc int, naturalRoll int) check.CheckResult {
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(actor, ability.SkillAthletics, dc, naturalRoll)
	}
	return PerformSkillCheck(actor, ability.SkillAthletics, dc)
}

// --- Deception ---

func CreateADiversion(actor *entity.Entity, observers []*entity.Entity, naturalRoll int) []check.CheckResult {
	results := make([]check.CheckResult, 0)
	for _, obs := range observers {
		dc := 10 + obs.GetPerception()
		var res check.CheckResult
		if naturalRoll > 0 {
			res = PerformSkillCheckWithRoll(actor, ability.SkillDeception, dc, naturalRoll)
		} else {
			res = PerformSkillCheck(actor, ability.SkillDeception, dc)
		}
		
		if res.Degree >= check.Success {
			actor.Conditions.ApplyRelative(condition.Hidden, obs.ID, "Diversion")
		}
		results = append(results, res)
	}
	return results
}

func Feint(actor, target *entity.Entity, naturalRoll int) check.CheckResult {
	dc := 10 + target.GetPerception()
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(actor, ability.SkillDeception, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(actor, ability.SkillDeception, dc)
	}

	switch res.Degree {
	case check.CriticalSuccess:
		// Target flat-footed against actor until end of NEXT turn
		target.Conditions.Apply(condition.NewRelationalCondition(condition.FlatFooted, []string{actor.ID}, "Feint (Crit)"))
	case check.Success:
		// Target flat-footed against NEXT melee attack
		target.Conditions.Apply(condition.NewRelationalCondition(condition.FlatFooted, []string{actor.ID}, "Feint"))
	case check.CriticalFailure:
		actor.Conditions.Apply(condition.NewRelationalCondition(condition.FlatFooted, []string{target.ID}, "Feint (Crit Fail)"))
	}
	return res
}

func Impersonate(actor *entity.Entity, dc int, naturalRoll int) check.CheckResult {
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(actor, ability.SkillDeception, dc, naturalRoll)
	}
	return PerformSkillCheck(actor, ability.SkillDeception, dc)
}

func Lie(actor *entity.Entity, observer *entity.Entity, naturalRoll int) check.CheckResult {
	dc := 10 + observer.GetPerception()
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(actor, ability.SkillDeception, dc, naturalRoll)
	}
	return PerformSkillCheck(actor, ability.SkillDeception, dc)
}

// --- Diplomacy ---

func GatherInformation(actor *entity.Entity, dc int, naturalRoll int) check.CheckResult {
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(actor, ability.SkillDiplomacy, dc, naturalRoll)
	}
	return PerformSkillCheck(actor, ability.SkillDiplomacy, dc)
}

func MakeAnImpression(actor, target *entity.Entity, naturalRoll int) check.CheckResult {
	dc := 10 + target.GetSaveDC(ability.SaveWill) 
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(actor, ability.SkillDiplomacy, dc, naturalRoll)
	}
	return PerformSkillCheck(actor, ability.SkillDiplomacy, dc)
}

func Request(actor, target *entity.Entity, naturalRoll int) check.CheckResult {
	dc := 10 + target.GetSaveDC(ability.SaveWill)
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(actor, ability.SkillDiplomacy, dc, naturalRoll)
	}
	return PerformSkillCheck(actor, ability.SkillDiplomacy, dc)
}

// --- Intimidation ---

func Coerce(actor, target *entity.Entity, naturalRoll int) check.CheckResult {
	dc := 10 + target.GetSaveDC(ability.SaveWill)
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(actor, ability.SkillIntimidation, dc, naturalRoll)
	}
	return PerformSkillCheck(actor, ability.SkillIntimidation, dc)
}

func Demoralize(actor, target *entity.Entity, naturalRoll int) check.CheckResult {
	if target.IsTemporarilyImmune("demoralize", actor.ID) {
		return check.CheckResult{Degree: check.Failure}
	}
	dc := target.GetSaveDC(ability.SaveWill)
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(actor, ability.SkillIntimidation, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(actor, ability.SkillIntimidation, dc)
	}

	switch res.Degree {
	case check.CriticalSuccess:
		target.Conditions.Apply(condition.NewValuedCondition(condition.Frightened, 2, "Demoralize"))
	case check.Success:
		target.Conditions.Apply(condition.NewValuedCondition(condition.Frightened, 1, "Demoralize"))
	}
	target.AddTemporaryImmunity("demoralize", actor.ID, 100) 
	return res
}

// --- Medicine ---

func TreatWounds(healer, patient *entity.Entity, dc int, naturalRoll int) (int, check.CheckResult) {
	if healer.SkillProficiencies[ability.SkillMedicine] < ability.Trained {
		return 0, check.CheckResult{Degree: check.Failure}
	}
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(healer, ability.SkillMedicine, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(healer, ability.SkillMedicine, dc)
	}

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

func AdministerFirstAid(healer, patient *entity.Entity, dc int, stabilize bool, naturalRoll int) check.CheckResult {
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(healer, ability.SkillMedicine, dc, naturalRoll)
	}
	return PerformSkillCheck(healer, ability.SkillMedicine, dc)
}

func TreatPoison(healer, patient *entity.Entity, dc int, naturalRoll int) check.CheckResult {
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(healer, ability.SkillMedicine, dc, naturalRoll)
	}
	return PerformSkillCheck(healer, ability.SkillMedicine, dc)
}

// --- Stealth ---

func Hide(actor *entity.Entity, observer *entity.Entity, naturalRoll int) check.CheckResult {
	dc := 10 + observer.GetPerception()
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(actor, ability.SkillStealth, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(actor, ability.SkillStealth, dc)
	}

	if res.Degree >= check.Success {
		actor.Conditions.ApplyRelative(condition.Hidden, observer.ID, "Hide")
	}
	return res
}

func Sneak(actor *entity.Entity, observer *entity.Entity, naturalRoll int) check.CheckResult {
	dc := 10 + observer.GetPerception()
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(actor, ability.SkillStealth, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(actor, ability.SkillStealth, dc)
	}

	if res.Degree >= check.Success {
		actor.Conditions.ApplyRelative(condition.Hidden, observer.ID, "Sneak")
	} else {
		actor.Conditions.RemoveRelative(condition.Hidden, observer.ID)
	}
	return res
}

func ConcealObject(actor *entity.Entity, dc int, naturalRoll int) check.CheckResult {
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(actor, ability.SkillStealth, dc, naturalRoll)
	}
	return PerformSkillCheck(actor, ability.SkillStealth, dc)
}

// --- Thievery ---

func PickLock(actor *entity.Entity, dc int, successesRequired int, naturalRoll int) (int, check.CheckResult) {
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(actor, ability.SkillThievery, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(actor, ability.SkillThievery, dc)
	}

	progress := 0
	if res.Degree == check.CriticalSuccess { progress = 2 } else if res.Degree == check.Success { progress = 1 }
	return progress, res
}

func DisableDevice(actor *entity.Entity, dc int, naturalRoll int) check.CheckResult {
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(actor, ability.SkillThievery, dc, naturalRoll)
	}
	return PerformSkillCheck(actor, ability.SkillThievery, dc)
}

func PalmObject(actor *entity.Entity, dc int, naturalRoll int) check.CheckResult {
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(actor, ability.SkillThievery, dc, naturalRoll)
	}
	return PerformSkillCheck(actor, ability.SkillThievery, dc)
}

func Steal(actor, target *entity.Entity, naturalRoll int) check.CheckResult {
	dc := 10 + target.GetPerception()
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(actor, ability.SkillThievery, dc, naturalRoll)
	}
	return PerformSkillCheck(actor, ability.SkillThievery, dc)
}

// --- General ---

func Seek(actor *entity.Entity, dc int, modifiers []check.Modifier, naturalRoll int) check.CheckResult {
	if naturalRoll > 0 {
		mod := actor.GetSkillModifier(ability.SkillPerception)
		return check.PerformCheckWithRoll(naturalRoll, mod, modifiers, dc)
	}
	return PerformSkillCheckWithModifiers(actor, ability.SkillPerception, dc, modifiers)
}

func RecallKnowledge(actor *entity.Entity, skillID ability.SkillID, dc int, naturalRoll int) (string, check.CheckResult) {
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(actor, skillID, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(actor, skillID, dc)
	}

	info := ""
	if res.Degree >= check.Success {
		info = "General Information"
	}
	return info, res
}

func Grapple(attacker, target *entity.Entity, modifiers []check.Modifier, naturalRoll int) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveFortitude)
	var res check.CheckResult
	if naturalRoll > 0 {
		mod := attacker.GetSkillModifier(ability.SkillAthletics)
		res = check.PerformCheckWithRoll(naturalRoll, mod, modifiers, dc)
	} else {
		res = PerformSkillCheckWithModifiers(attacker, ability.SkillAthletics, dc, modifiers)
	}
	switch res.Degree {
	case check.CriticalSuccess:
		target.Conditions.Apply(condition.NewCondition(condition.Restrained, "Grapple"))
	case check.Success:
		target.Conditions.Apply(condition.NewCondition(condition.Grabbed, "Grapple"))
	}
	return res
}

func Trip(attacker, target *entity.Entity, modifiers []check.Modifier, naturalRoll int) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveReflex)
	var res check.CheckResult
	if naturalRoll > 0 {
		mod := attacker.GetSkillModifier(ability.SkillAthletics)
		res = check.PerformCheckWithRoll(naturalRoll, mod, modifiers, dc)
	} else {
		res = PerformSkillCheckWithModifiers(attacker, ability.SkillAthletics, dc, modifiers)
	}
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

func Shove(actor, target *entity.Entity, modifiers []check.Modifier, naturalRoll int) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveFortitude)
	var res check.CheckResult
	if naturalRoll > 0 {
		mod := actor.GetSkillModifier(ability.SkillAthletics)
		res = check.PerformCheckWithRoll(naturalRoll, mod, modifiers, dc)
	} else {
		res = PerformSkillCheckWithModifiers(actor, ability.SkillAthletics, dc, modifiers)
	}
	if res.Degree == check.CriticalFailure {
		actor.Conditions.Apply(condition.NewCondition(condition.Prone, "Shove (Crit Fail)"))
	}
	return res
}
