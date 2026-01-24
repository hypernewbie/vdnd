package entity

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/affliction"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/item"
)

// GetAC calculates current Armor Class relative to an attacker
func (e *Entity) GetAC(attacker *Entity) int {
	base := 10

	// DEX modifier, capped by armor
	dexMod := e.Abilities.Modifier(ability.Dexterity)
	if e.WornArmor != nil {
		dexMod = e.WornArmor.AppliedDexBonus(dexMod)
	}

	// Proficiency bonus
	armorProf := e.getArmorProficiency()
	profBonus := armorProf.Bonus(e.Level)

	// Armor item bonus
	armorBonus := 0
	if e.WornArmor != nil {
		armorBonus = e.WornArmor.ACBonus
	}

	// Circumstance bonus from raised shield
	shieldBonus := 0
	if e.WornShield != nil && e.WornShield.IsRaised && !e.WornShield.IsBroken() {
		shieldBonus = e.WornShield.ACBonus
	}

	// Condition modifiers (flat-footed, clumsy, etc.)
	var attackerID string
	var attackerConds *condition.ConditionTracker
	if attacker != nil {
		attackerID = attacker.ID
		attackerConds = attacker.Conditions
	}
	conditionMods := e.Conditions.GetACModifiers(attackerConds, attackerID)
	conditionTotal := check.CalculateTotal(conditionMods)

	return base + dexMod + profBonus + armorBonus + shieldBonus + conditionTotal
}

func (e *Entity) getArmorProficiency() ability.ProficiencyRank {
	if e.WornArmor == nil {
		return e.UnarmoredDefense
	}
	if prof, ok := e.ArmorProficiencies[e.WornArmor.Category]; ok {
		return prof
	}
	return ability.Untrained
}

func (e *Entity) GetWeaponProficiency(w *item.Weapon) ability.ProficiencyRank {
	if prof, ok := e.WeaponProficiencies[w.Group]; ok {
		return prof
	}
	// Also check category (simple, martial, etc.) if we want to be more thorough
	// but group-based is common for monsters/specific training.
	return ability.Untrained
}

// GetFortitude calculates Fortitude save modifier
func (e *Entity) GetFortitude() int {
	conMod := e.Abilities.Modifier(ability.Constitution)
	profBonus := e.Fortitude.Bonus(e.Level)
	saveMods := e.Conditions.GetSaveModifiers("Fortitude")
	return conMod + profBonus + check.CalculateTotal(saveMods)
}

// GetReflex calculates Reflex save modifier
func (e *Entity) GetReflex() int {
	dexMod := e.Abilities.Modifier(ability.Dexterity)
	profBonus := e.Reflex.Bonus(e.Level)
	saveMods := e.Conditions.GetSaveModifiers("Reflex")
	return dexMod + profBonus + check.CalculateTotal(saveMods)
}

// GetWill calculates Will save modifier
func (e *Entity) GetWill() int {
	wisMod := e.Abilities.Modifier(ability.Wisdom)
	profBonus := e.Will.Bonus(e.Level)
	saveMods := e.Conditions.GetSaveModifiers("Will")
	return wisMod + profBonus + check.CalculateTotal(saveMods)
}

// GetPerception calculates Perception modifier
func (e *Entity) GetPerception() int {
	wisMod := e.Abilities.Modifier(ability.Wisdom)
	profBonus := e.Perception.Bonus(e.Level)
	conditionMods := e.Conditions.GetModifiers()
	// Fascinated etc affect Perception
	return wisMod + profBonus + check.CalculateTotal(conditionMods)
}

func (e *Entity) GetSkillModifier(skill ability.SkillID) int {
	ab := ability.GetKeyAbility(skill)

	abilityMod := e.Abilities.Modifier(ab)
	prof := ability.Untrained
	if p, ok := e.SkillProficiencies[skill]; ok {
		prof = p
	}

	profBonus := prof.Bonus(e.Level)
	conditionMods := e.Conditions.GetModifiers()

	return abilityMod + profBonus + check.CalculateTotal(conditionMods)
}

func (e *Entity) GetSaveDC(save ability.SaveType) int {
	var mod int
	switch save {
	case ability.SaveFortitude:
		mod = e.GetFortitude()
	case ability.SaveReflex:
		mod = e.GetReflex()
	case ability.SaveWill:
		mod = e.GetWill()
	default:
		mod = 0
	}
	return 10 + mod
}

func (e *Entity) GetSkillDC(skill ability.SkillID) int {
	return 10 + e.GetSkillModifier(skill)
}

func (e *Entity) IsImmuneTo(id string) bool {
	for _, imm := range e.Immunities {
		if imm == id {
			return true
		}
	}
	return false
}

func (e *Entity) GetResistance(damageType string) int {
	return e.Resistances[damageType].Amount
}

func (e *Entity) GetResistanceEntry(damageType string) ResistanceEntry {
	return e.Resistances[damageType]
}

func (e *Entity) GetWeakness(damageType string) int {
	return e.Weaknesses[damageType]
}

func (e *Entity) GetSpeed() int {
	speed := e.BaseSpeed

	// Armour speed penalty (if STR too low)
	if e.WornArmor != nil {
		speed += e.WornArmor.EffectiveSpeedPenalty(e.Abilities.Get(ability.Strength))
	}

	// Shield speed penalty (tower shield)
	if e.WornShield != nil {
		speed += e.WornShield.SpeedPenalty
	}

	// Condition penalties
	speed += e.Conditions.GetSpeedPenalty()

	if speed < 0 {
		speed = 0
	}
	return speed
}

// IsFlanking checks if two attackers are flanking a target.
// Simplified logic for 1x1 creatures: attackers must be on opposite sides/corners.
// Center line must pass through target.
func IsFlanking(target, a, b *Entity) bool {
	// If they are on opposite sides, vector A to Target == -(vector B to Target)
	// For 1x1: a.X + b.X == 2*target.X and a.Y + b.Y == 2*target.Y
	return (a.X + b.X == 2*target.X) && (a.Y + b.Y == 2*target.Y)
}

// ProcessAfflictions advances the affliction tracker by one time unit.
// It automatically applies any CONDITIONS triggered by the new stage.
//
// WARNING: DAMAGE IS NOT APPLIED AUTOMATICALLY!
// This method returns TickResult objects containing the damage that occurred.
// The caller is responsible for reporting this damage to the user and
// calling e.ApplyDamage() explicitly.
func (e *Entity) ProcessAfflictions(unit ability.IntervalUnit) []affliction.TickResult {
	results := e.Afflictions.Tick(unit)
	for _, res := range results {
		// Apply conditions
		for _, cond := range res.Conditions {
			e.Conditions.Apply(condition.NewValuedCondition(cond.ID, cond.Value, res.AfflictionID))
		}

		if res.IsFatal {
			e.Kill(res.AfflictionID)
		}
	}
	return results
}
