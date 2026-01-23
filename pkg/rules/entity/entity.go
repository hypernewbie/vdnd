package entity

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/affliction"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/feat"
	"uaa/vdnd/pkg/rules/item"
	"uaa/vdnd/pkg/rules/trait"
)

type ResistanceEntry struct {
	Amount int
	Except []string
}

type Entity struct {
	// Identity
	ID    string
	Name  string
	Level int
	Size  Size

	// Movement
	BaseSpeed int

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

	ArmorProficiencies  map[item.ArmorCategory]ability.ProficiencyRank
	WeaponProficiencies map[item.WeaponGroup]ability.ProficiencyRank
	SkillProficiencies  map[ability.SkillID]ability.ProficiencyRank
	SpellProficiency    ability.ProficiencyRank
	SpellcastingAbility ability.Ability

	// Equipment
	WornArmor *item.Armor

	WieldedWeapons []*item.Weapon // Up to 2 (or more for multi-limbed)

	// Runtime State
	Conditions  *condition.ConditionTracker
	Afflictions *affliction.AfflictionTracker
	Feats       *feat.FeatTracker

	// Position (zone-based)
	Position    string   // Zone ID
	EngagedWith []string // Entity IDs currently in melee with

	// Defenses
	Immunities  []string                   // Trait/damage type IDs
	Resistances map[string]ResistanceEntry // type -> amount + exceptions
	Weaknesses  map[string]int             // type -> amount

	// Creature traits (for monsters)
	Traits trait.TraitSet
}

func NewEntity(id, name string, level int) *Entity {
	return &Entity{
		ID:                  id,
		Name:                name,
		Level:               level,
		Size:                Medium,
		BaseSpeed:           25,
		ArmorProficiencies:  make(map[item.ArmorCategory]ability.ProficiencyRank),
		WeaponProficiencies: make(map[item.WeaponGroup]ability.ProficiencyRank),
		SkillProficiencies:  make(map[ability.SkillID]ability.ProficiencyRank),
		SpellcastingAbility: ability.Intelligence, // Default
		Conditions:          condition.NewTracker(),
		Afflictions:         affliction.NewTracker(),
		Feats:               feat.NewFeatTracker(),
		Resistances:         make(map[string]ResistanceEntry),
		Weaknesses:          make(map[string]int),
		WieldedWeapons:      make([]*item.Weapon, 0),
	}
}

func NewPC(id, name string, level int, ancestry, class, background string) *Entity {
	e := NewEntity(id, name, level)
	e.Ancestry = ancestry
	e.Class = class
	e.Background = background
	return e
}

func (e *Entity) GetName() string {
	return e.Name
}

func (e *Entity) GetID() string {
	return e.ID
}

func (e *Entity) GetLevel() int {
	return e.Level
}

func (e *Entity) GetAbilityScore(ab ability.Ability) int {
	return e.Abilities.Get(ab)
}

func (e *Entity) HasFeat(featID string) bool {
	return e.Feats.Has(featID)
}

func (e *Entity) HasSkillRank(skillID ability.SkillID, rank ability.ProficiencyRank) bool {
	prof := ability.Untrained
	if p, ok := e.SkillProficiencies[skillID]; ok {
		prof = p
	}
	return prof >= rank
}

func (e *Entity) HasTrait(traitID trait.TraitID) bool {
	return e.Traits.HasTrait(traitID)
}

// Clone creates a deep copy of the entity
func (e *Entity) Clone() *Entity {
	clone := *e

	// Deep copy maps and slices
	clone.ArmorProficiencies = make(map[item.ArmorCategory]ability.ProficiencyRank)
	for k, v := range e.ArmorProficiencies {
		clone.ArmorProficiencies[k] = v
	}

	clone.WeaponProficiencies = make(map[item.WeaponGroup]ability.ProficiencyRank)
	for k, v := range e.WeaponProficiencies {
		clone.WeaponProficiencies[k] = v
	}

	clone.SkillProficiencies = make(map[ability.SkillID]ability.ProficiencyRank)
	for k, v := range e.SkillProficiencies {
		clone.SkillProficiencies[k] = v
	}

	clone.WieldedWeapons = make([]*item.Weapon, len(e.WieldedWeapons))
	copy(clone.WieldedWeapons, e.WieldedWeapons)

	clone.Immunities = make([]string, len(e.Immunities))
	copy(clone.Immunities, e.Immunities)

	clone.Resistances = make(map[string]ResistanceEntry)
	for k, v := range e.Resistances {
		// Deep copy entry slice
		entry := ResistanceEntry{Amount: v.Amount}
		if v.Except != nil {
			entry.Except = make([]string, len(v.Except))
			copy(entry.Except, v.Except)
		}
		clone.Resistances[k] = entry
	}

	clone.Weaknesses = make(map[string]int)
	for k, v := range e.Weaknesses {
		clone.Weaknesses[k] = v
	}

	// Conditions tracker needs a deep clone
	clone.Conditions = condition.NewTracker()
	for _, c := range e.Conditions.All() {
		clone.Conditions.Apply(c)
	}

	// Afflictions tracker needs a deep clone
	clone.Afflictions = affliction.NewTracker()
	for _, a := range e.Afflictions.All() {
		// This is a shallow copy of instances, but they are mostly read-only/managed by tracker
		// In a real system we'd want NewInstanceFromExisting.
		clone.Afflictions.Add(a.Affliction, a.Source)
		inst := clone.Afflictions.Get(a.Affliction.ID)
		if inst != nil {
			inst.CurrentStage = a.CurrentStage
			inst.TimeToOnset = a.TimeToOnset
			inst.TimeToNextSave = a.TimeToNextSave
		}
	}

	// Feats tracker needs a deep clone
	clone.Feats = feat.NewFeatTracker()
	for _, f := range e.Feats.All() {
		clone.Feats.Add(f)
	}

	return &clone
}
