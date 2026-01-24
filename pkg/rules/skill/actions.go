package skill

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
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

type TumbleResult struct {
	Success bool
	EndMove bool
}

func TumbleThrough(actor, target *entity.Entity, naturalRoll int) (TumbleResult, check.CheckResult) {
	dc := target.GetSaveDC(ability.SaveReflex)
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(actor, ability.SkillAcrobatics, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(actor, ability.SkillAcrobatics, dc)
	}

	out := TumbleResult{Success: res.Degree >= check.Success}
	if res.Degree < check.Success {
		out.EndMove = true
	}
	return out, res
}

func ManeuverInFlight(actor *entity.Entity, dc int, naturalRoll int) (int, check.CheckResult) {
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(actor, ability.SkillAcrobatics, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(actor, ability.SkillAcrobatics, dc)
	}
	dist := 0
	if res.Degree >= check.Success {
		dist = actor.BaseSpeed
	}
	return dist, res
}

func Squeeze(actor *entity.Entity, dc int, naturalRoll int) (int, check.CheckResult) {
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(actor, ability.SkillAcrobatics, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(actor, ability.SkillAcrobatics, dc)
	}
	dist := 0
	switch res.Degree {
	case check.CriticalSuccess:
		dist = 10
	case check.Success:
		dist = 5
	}
	return dist, res
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
		move.Speed = 8
	case check.Success:
		move.Speed = 5
	case check.CriticalFailure:
		actor.Conditions.Apply(condition.NewCondition(condition.Prone, "Climb (Crit Fail)"))
		move.Damage = dice.DieRoll{Count: 1, Sides: 6}.Roll()
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
		// Target has weakness to further disarm attempts
		target.Conditions.Apply(condition.NewCondition("DisarmWeakness", actor.ID))
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
		target.Conditions.Apply(condition.NewRelationalCondition(condition.FlatFooted, []string{actor.ID}, "Feint (Crit)"))
		if inst := target.Conditions.Get(condition.FlatFooted); inst != nil {
			inst.Duration = 2
		}
	case check.Success:
		target.Conditions.Apply(condition.NewRelationalCondition(condition.FlatFooted, []string{actor.ID}, "Feint"))
		if inst := target.Conditions.Get(condition.FlatFooted); inst != nil {
			inst.Duration = 1
		}
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
	dc := target.GetSaveDC(ability.SaveWill)
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(actor, ability.SkillDiplomacy, dc, naturalRoll)
	}
	return PerformSkillCheck(actor, ability.SkillDiplomacy, dc)
}

func Request(actor, target *entity.Entity, naturalRoll int) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveWill)
	if naturalRoll > 0 {
		return PerformSkillCheckWithRoll(actor, ability.SkillDiplomacy, dc, naturalRoll)
	}
	return PerformSkillCheck(actor, ability.SkillDiplomacy, dc)
}

// --- Intimidation ---

func Coerce(actor, target *entity.Entity, naturalRoll int) check.CheckResult {
	dc := target.GetSaveDC(ability.SaveWill)
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
	if healing > 0 {
		patient.Heal(healing)
	}
	return healing, res
}

func AdministerFirstAid(healer, patient *entity.Entity, dc int, stabilize bool, naturalRoll int) check.CheckResult {
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(healer, ability.SkillMedicine, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(healer, ability.SkillMedicine, dc)
	}

	if res.Degree >= check.Success {
		if stabilize {
			patient.Conditions.Remove(condition.Dying)
			// PF2e: stabilize usually makes them Unconscious but not dying.
		} else {
			// Stop bleeding (Remove persistent bleed if we had specific types, for now just generic)
		}
	}
	return res
}

func TreatPoison(healer, patient *entity.Entity, dc int, naturalRoll int) check.CheckResult {
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(healer, ability.SkillMedicine, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(healer, ability.SkillMedicine, dc)
	}

	if res.Degree >= check.Success {
		bonus := 2
		if res.Degree == check.CriticalSuccess {
			bonus = 4
		}
		// Apply a circumstance bonus for next save.
		// For now we use a temporary immunity or condition to represent this.
		patient.Conditions.Apply(condition.NewModifierCondition("Treat Poison", check.BonusCircumstance, bonus, "Next Poison Save"))
	}
	return res
}

func TreatDisease(healer, patient *entity.Entity, dc int, naturalRoll int) check.CheckResult {
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(healer, ability.SkillMedicine, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(healer, ability.SkillMedicine, dc)
	}

	if res.Degree >= check.Success {
		bonus := 2
		if res.Degree == check.CriticalSuccess {
			bonus = 4
		}
		patient.Conditions.Apply(condition.NewModifierCondition("Treat Disease", check.BonusCircumstance, bonus, "Next Disease Save"))
	}
	return res
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
		actor.Conditions.ApplyRelative(condition.Undetected, observer.ID, "Sneak")
		actor.Conditions.RemoveRelative(condition.Hidden, observer.ID)
	} else {
		actor.Conditions.ApplyRelative(condition.Observed, observer.ID, "Sneak (Fail)")
		actor.Conditions.RemoveRelative(condition.Hidden, observer.ID)
		actor.Conditions.RemoveRelative(condition.Undetected, observer.ID)
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
	if res.Degree == check.CriticalSuccess {
		progress = 2
	} else if res.Degree == check.Success {
		progress = 1
	}
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

// --- Crafting ---

// EarnIncomeResult contains the outcome of an Earn Income check
type EarnIncomeResult struct {
	EarnedCP   int // Copper pieces earned
	TaskLevel  int // Level of task attempted
	DaysWorked int // Always 1 per check
}

// EarnIncome performs a skill check to earn money during downtime
// src: rules/compendium/skills.md "Earn Income"
// Can use: Crafting, Lore, Performance
func EarnIncome(actor *entity.Entity, skillID ability.SkillID, taskLevel int, naturalRoll int) (EarnIncomeResult, check.CheckResult) {
	result := EarnIncomeResult{TaskLevel: taskLevel, DaysWorked: 1}

	// Must be trained in the skill
	prof := ability.Untrained
	if p, ok := actor.SkillProficiencies[skillID]; ok {
		prof = p
	}
	if prof < ability.Trained {
		return result, check.CheckResult{Degree: check.Failure}
	}

	// Task level cannot exceed your level
	if taskLevel > actor.Level {
		taskLevel = actor.Level
	}

	dc := LevelBasedDC(taskLevel)
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(actor, skillID, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(actor, skillID, dc)
	}

	switch res.Degree {
	case check.CriticalSuccess:
		// Earn as if task level was 1 higher
		result.EarnedCP = GetEarnIncomeAmount(taskLevel+1, prof)
	case check.Success:
		result.EarnedCP = GetEarnIncomeAmount(taskLevel, prof)
	case check.Failure:
		// Earn for a task 1 level lower (minimum 0)
		fallbackLevel := taskLevel - 1
		if fallbackLevel < 0 {
			fallbackLevel = 0
		}
		result.EarnedCP = GetEarnIncomeAmount(fallbackLevel, prof)
	case check.CriticalFailure:
		// No earnings, may have wasted resources
		result.EarnedCP = 0
	}

	return result, res
}

// RepairResult contains the outcome of a Repair check
type RepairResult struct {
	Repaired     bool
	HPRestored   int // For items with HP (shields, etc.)
	MaterialCost int // Copper spent on materials (if any)
}

// Repair fixes a damaged item (10 minute activity)
// src: rules/compendium/skills.md "Repair"
func Repair(actor *entity.Entity, itemLevel int, naturalRoll int) (RepairResult, check.CheckResult) {
	result := RepairResult{}

	// Must be trained in Crafting
	prof := ability.Untrained
	if p, ok := actor.SkillProficiencies[ability.SkillCrafting]; ok {
		prof = p
	}
	if prof < ability.Trained {
		return result, check.CheckResult{Degree: check.Failure}
	}

	dc := LevelBasedDC(itemLevel)
	var res check.CheckResult
	if naturalRoll > 0 {
		res = PerformSkillCheckWithRoll(actor, ability.SkillCrafting, dc, naturalRoll)
	} else {
		res = PerformSkillCheck(actor, ability.SkillCrafting, dc)
	}

	switch res.Degree {
	case check.CriticalSuccess:
		result.Repaired = true
		// 10 HP + 10 per rank above Trained
		rankBonus := int(prof - ability.Trained)
		result.HPRestored = 10 + (rankBonus * 10)
	case check.Success:
		result.Repaired = true
		// 5 HP + 5 per rank above Trained
		rankBonus := int(prof - ability.Trained)
		result.HPRestored = 5 + (rankBonus * 5)
	case check.Failure:
		result.Repaired = false
	case check.CriticalFailure:
		result.Repaired = false
	}

	return result, res
}

// RepairShield specifically repairs a shield
func RepairShield(actor *entity.Entity, s *item.Shield, naturalRoll int) (RepairResult, check.CheckResult) {
	if s == nil {
		return RepairResult{}, check.CheckResult{Degree: check.Failure}
	}

	// Use item level 0 for basic shields (could be parameterised)
	itemLevel := 0 // Or derive from shield type

	result, res := Repair(actor, itemLevel, naturalRoll)

	if result.Repaired {
		s.CurrentHP += result.HPRestored
		if s.CurrentHP > s.MaxHP {
			s.CurrentHP = s.MaxHP
		}
	} else if res.Degree == check.CriticalFailure {
		// Deal 2d6 damage to shield
		damage := dice.DieRoll{Count: 2, Sides: 6}.Roll()
		s.TakeDamage(damage)
	}

	return result, res
}

