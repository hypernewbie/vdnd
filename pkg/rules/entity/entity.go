package entity

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/item"
	"uaa/vdnd/pkg/rules/trait"
)

type Entity struct {
	// Identity
	ID    string
	Name  string
	Level int
	Size  Size

	// Flavour (for PCs primarily)
	Ancestry   string
	Class      string
	Background string

	// Core Stats
	Abilities ability.AbilityScores

	// Hit Points
	MaxHP     int
	CurrentHP int
	TempHP    int

	// Proficiencies
	Perception       ability.ProficiencyRank
	Fortitude        ability.ProficiencyRank
	Reflex           ability.ProficiencyRank
	Will             ability.ProficiencyRank
	UnarmoredDefense ability.ProficiencyRank
	// Add other armor/weapon proficiencies as needed
	ArmorProficiencies map[item.ArmorCategory]ability.ProficiencyRank

	// Equipment
	WornArmor      *item.Armor
	WieldedWeapons []*item.Weapon // Up to 2 (or more for multi-limbed)

	// Runtime State
	Conditions *condition.ConditionTracker

	// Position (zone-based)
	Position    string   // Zone ID
	EngagedWith []string // Entity IDs currently in melee with

	// Defenses
	Immunities  []string       // Trait/damage type IDs
	Resistances map[string]int // type -> amount
	Weaknesses  map[string]int // type -> amount

	// Creature traits (for monsters)
	Traits trait.TraitSet
}

func NewEntity(id, name string, level int) *Entity {
	return &Entity{
		ID:                 id,
		Name:               name,
		Level:              level,
		Size:               Medium,
		ArmorProficiencies: make(map[item.ArmorCategory]ability.ProficiencyRank),
		Conditions:         condition.NewTracker(),
		Resistances:        make(map[string]int),
		Weaknesses:         make(map[string]int),
		WieldedWeapons:     make([]*item.Weapon, 0),
	}
}

func NewPC(id, name string, level int, ancestry, class, background string) *Entity {
	e := NewEntity(id, name, level)
	e.Ancestry = ancestry
	e.Class = class
	e.Background = background
	return e
}

// Clone creates a deep copy of the entity
func (e *Entity) Clone() *Entity {
	clone := *e

	// Deep copy maps and slices
	clone.ArmorProficiencies = make(map[item.ArmorCategory]ability.ProficiencyRank)
	for k, v := range e.ArmorProficiencies {
		clone.ArmorProficiencies[k] = v
	}

	clone.WieldedWeapons = make([]*item.Weapon, len(e.WieldedWeapons))
	copy(clone.WieldedWeapons, e.WieldedWeapons)

	clone.Immunities = make([]string, len(e.Immunities))
	copy(clone.Immunities, e.Immunities)

	clone.Resistances = make(map[string]int)
	for k, v := range e.Resistances {
		clone.Resistances[k] = v
	}

	clone.Weaknesses = make(map[string]int)
	for k, v := range e.Weaknesses {
		clone.Weaknesses[k] = v
	}

	// Conditions tracker needs a deep clone if implemented, but here we'll just start fresh or
	// we'd need a Clone() on ConditionTracker. Let's assume for now fresh or handle if needed.
	// For simplicity in this phase, we won't deep clone the tracker state perfectly yet.
	clone.Conditions = condition.NewTracker()
	for _, c := range e.Conditions.All() {
		clone.Conditions.Apply(c)
	}

	return &clone
}
