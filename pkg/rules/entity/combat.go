package entity

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
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

func (e *Entity) IsImmuneTo(id string) bool {
	for _, imm := range e.Immunities {
		if imm == id {
			return true
		}
	}
	return false
}

func (e *Entity) GetResistance(damageType string) int {
	return e.Resistances[damageType]
}

func (e *Entity) GetWeakness(damageType string) int {
	return e.Weaknesses[damageType]
}
