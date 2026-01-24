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

type Language string

const (
	LangCommon   Language = "common"
	LangDraconic Language = "draconic"
	LangElven    Language = "elven"
	LangDwarven  Language = "dwarven"
	LangGoblin   Language = "goblin"
	LangOrcish   Language = "orcish"
	LangAbyssal  Language = "abyssal"
	LangCelestial Language = "celestial"
	LangInfernal Language = "infernal"
	LangSylvan   Language = "sylvan"
)

type Alignment string

const (
	AlignLG Alignment = "LG"
	AlignNG Alignment = "NG"
	AlignCG Alignment = "CG"
	AlignLN Alignment = "LN"
	AlignN  Alignment = "N"
	AlignCN Alignment = "CN"
	AlignLE Alignment = "LE"
	AlignNE Alignment = "NE"
	AlignCE Alignment = "CE"
)

type LightLevel int

const (
	LightBright LightLevel = iota
	LightDim
	LightDarkness
)

type Entity struct {
	// Identity
	ID        string
	Name      string
	Level     int
	Size      Size
	Alignment Alignment
	Languages []Language

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
	WornArmor  *item.Armor
	WornShield *item.Shield

	WieldedWeapons []*item.Weapon // Up to 2 (or more for multi-limbed)
	Inventory      *Inventory

	// Runtime State
	Conditions          *condition.ConditionTracker
	Afflictions         *affliction.AfflictionTracker
	Feats               *feat.FeatTracker
	TemporaryImmunities map[string]int // "ActionID:SourceID" -> rounds remaining

	// Position (zone-based or grid-based)
	Position    string   // Zone ID
	X, Y        int      // Grid coordinates
	EngagedWith []string // Entity IDs currently in melee with

	// Defenses
	Immunities  []string                   // Trait/damage type IDs
	Resistances map[string]ResistanceEntry // type -> amount + exceptions
	Weaknesses  map[string]int             // type -> amount

	// Creature traits (for monsters)
	Traits trait.TraitSet
}

func NewEntity(id, name string, level int) *Entity {
	tr := condition.NewTracker()
	tr.SetOwner(id)
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
		Conditions:          tr,
		Afflictions:         affliction.NewTracker(),
		Feats:               feat.NewFeatTracker(),
		TemporaryImmunities: make(map[string]int),
		Resistances:         make(map[string]ResistanceEntry),
		Weaknesses:          make(map[string]int),
		WieldedWeapons:      make([]*item.Weapon, 0),
		Inventory:           NewInventory(),
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

// AddTemporaryImmunity sets immunity for a number of rounds (e.g. 10 mins = 100 rounds)
func (e *Entity) AddTemporaryImmunity(actionID, sourceID string, rounds int) {
	key := actionID + ":" + sourceID
	e.TemporaryImmunities[key] = rounds
}

// IsTemporarilyImmune checks if entity is currently immune to an action from a source
func (e *Entity) IsTemporarilyImmune(actionID, sourceID string) bool {
	key := actionID + ":" + sourceID
	rounds, ok := e.TemporaryImmunities[key]
	return ok && rounds > 0
}

// TickAdvances advances time for the entity
func (e *Entity) TickAdvances() {
	for key, rounds := range e.TemporaryImmunities {
		if rounds > 0 {
			e.TemporaryImmunities[key] = rounds - 1
			if e.TemporaryImmunities[key] == 0 {
				delete(e.TemporaryImmunities, key)
			}
		}
	}
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

	clone.TemporaryImmunities = make(map[string]int)
	for k, v := range e.TemporaryImmunities {
		clone.TemporaryImmunities[k] = v
	}

	if e.WornShield != nil {
		shieldCopy := *e.WornShield
		clone.WornShield = &shieldCopy
	}

	if e.Inventory != nil {
		clone.Inventory = &Inventory{
			Items:   make([]InventoryItem, len(e.Inventory.Items)),
			CoinsCP: e.Inventory.CoinsCP,
			CoinsSP: e.Inventory.CoinsSP,
			CoinsGP: e.Inventory.CoinsGP,
			CoinsPP: e.Inventory.CoinsPP,
		}
		copy(clone.Inventory.Items, e.Inventory.Items)
	}

	return &clone
}