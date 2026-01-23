package entity

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/item"
)

type SaveType int

const (
	SaveFortitude SaveType = iota
	SaveReflex
	SaveWill
)

// GetAC calculates current Armor Class
func (e *Entity) GetAC() int {
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

	// Condition modifiers (flat-footed, clumsy, etc.)
	conditionMods := e.Conditions.GetACModifiers()
	conditionTotal := check.CalculateTotal(conditionMods)

	return base + dexMod + profBonus + armorBonus + conditionTotal
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
	conditionMods := e.Conditions.GetModifiers()
	saveMods := e.Conditions.GetSaveModifiers()
	allMods := append(conditionMods, saveMods...)
	return conMod + profBonus + check.CalculateTotal(allMods)
}

// GetReflex calculates Reflex save modifier
func (e *Entity) GetReflex() int {
	dexMod := e.Abilities.Modifier(ability.Dexterity)
	profBonus := e.Reflex.Bonus(e.Level)
	conditionMods := e.Conditions.GetModifiers()
	saveMods := e.Conditions.GetSaveModifiers()
	allMods := append(conditionMods, saveMods...)
	return dexMod + profBonus + check.CalculateTotal(allMods)
}

// GetWill calculates Will save modifier
func (e *Entity) GetWill() int {
	wisMod := e.Abilities.Modifier(ability.Wisdom)
	profBonus := e.Will.Bonus(e.Level)
	conditionMods := e.Conditions.GetModifiers()
	saveMods := e.Conditions.GetSaveModifiers()
	allMods := append(conditionMods, saveMods...)
	return wisMod + profBonus + check.CalculateTotal(allMods)
}

// GetPerception calculates Perception modifier
func (e *Entity) GetPerception() int {
	wisMod := e.Abilities.Modifier(ability.Wisdom)
	profBonus := e.Perception.Bonus(e.Level)
	conditionMods := e.Conditions.GetModifiers()
	// Fascinated etc affect Perception
	return wisMod + profBonus + check.CalculateTotal(conditionMods)
}

func (e *Entity) GetSkillModifier(skill ability.Skill) int {
	var ab ability.Ability
	switch skill {
	case ability.SkillAthletics:
		ab = ability.Strength
	case ability.SkillAcrobatics, ability.SkillStealth, ability.SkillThievery:
		ab = ability.Dexterity
	case ability.SkillArcana, ability.SkillCrafting, ability.SkillOccultism, ability.SkillSociety:
		ab = ability.Intelligence
	case ability.SkillMedicine, ability.SkillNature, ability.SkillReligion, ability.SkillSurvival:
		ab = ability.Wisdom
	case ability.SkillDeception, ability.SkillDiplomacy, ability.SkillIntimidation, ability.SkillPerformance:
		ab = ability.Charisma
	default:
		ab = ability.Strength // Default
	}

	abilityMod := e.Abilities.Modifier(ab)
	prof := ability.Untrained
	if p, ok := e.SkillProficiencies[skill]; ok {
		prof = p
	}

	profBonus := prof.Bonus(e.Level)
	conditionMods := e.Conditions.GetModifiers()

	return abilityMod + profBonus + check.CalculateTotal(conditionMods)
}

func (e *Entity) GetSaveDC(save SaveType) int {
	var mod int
	switch save {
	case SaveFortitude:
		mod = e.GetFortitude()
	case SaveReflex:
		mod = e.GetReflex()
	case SaveWill:
		mod = e.GetWill()
	}
	return 10 + mod
}

func (e *Entity) GetSkillDC(skill ability.Skill) int {
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
	if e.WornArmor != nil {
		speed += e.WornArmor.EffectiveSpeedPenalty(e.Abilities.Strength)
	}
	if speed < 5 {
		speed = 5 // Minimum speed is 5ft unless immobilized
	}
	return speed
}
