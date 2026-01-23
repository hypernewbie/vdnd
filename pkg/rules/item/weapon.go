package item

import (
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/trait"
)

type WeaponCategory int

const (
	CategoryUnarmed WeaponCategory = iota
	CategorySimple
	CategoryMartial
	CategoryAdvanced
)

type WeaponGroup string

const (
	GroupSword    WeaponGroup = "sword"
	GroupAxe      WeaponGroup = "axe"
	GroupHammer   WeaponGroup = "hammer"
	GroupPolearm  WeaponGroup = "polearm"
	GroupSpear    WeaponGroup = "spear"
	GroupKnife    WeaponGroup = "knife"
	GroupFlail    WeaponGroup = "flail"
	GroupBrawling WeaponGroup = "brawling"
	GroupClub     WeaponGroup = "club"
	GroupBow      WeaponGroup = "bow"
	GroupCrossbow WeaponGroup = "crossbow"
	GroupDart     WeaponGroup = "dart"
	GroupSling    WeaponGroup = "sling"
)

type Weapon struct {
	ID             string
	Name           string
	Category       WeaponCategory
	Group          WeaponGroup
	Damage         dice.DieRoll // e.g., {1, 8, 0} for 1d8
	DamageType     DamageType
	Hands          int // 1 or 2
	Traits         trait.TraitSet
	RangeIncrement int // 0 for melee, feet for ranged/thrown

	// Derived from traits, cached for convenience
	IsRanged bool
	IsMelee  bool
}

// NewWeapon creates a weapon with basic stats
func NewWeapon(id, name string, cat WeaponCategory, group WeaponGroup,
	damage dice.DieRoll, damageType DamageType, hands int,
	traits ...trait.TraitID) Weapon {

	w := Weapon{
		ID:             id,
		Name:           name,
		Category:       cat,
		Group:          group,
		Damage:         damage,
		DamageType:     damageType,
		Hands:          hands,
		Traits:         traits,
		RangeIncrement: 0,
	}

	// Simple heuristic for IsMelee/IsRanged
	// In PF2E, some weapons are both (Thrown)
	if group == GroupBow || group == GroupCrossbow || group == GroupSling || group == GroupDart {
		w.IsRanged = true
	} else {
		w.IsMelee = true
	}

	if w.Traits.HasTrait(trait.TraitThrown) {
		w.IsRanged = true
	}

	return w
}

// HasTrait checks if weapon has a specific trait
func (w Weapon) HasTrait(id trait.TraitID) bool {
	return w.Traits.HasTrait(id)
}

// IsAgile returns true if weapon has agile trait
func (w Weapon) IsAgile() bool {
	return w.HasTrait(trait.TraitAgile)
}

// IsFinesse returns true if weapon has finesse trait
func (w Weapon) IsFinesse() bool {
	return w.HasTrait(trait.TraitFinesse)
}

// GetReach returns reach in feet (5 normally, 10 with reach trait)
func (w Weapon) GetReach() int {
	if w.HasTrait(trait.TraitReach) {
		return 10
	}
	return 5
}

// GetRangeIncrement returns range increment (0 for pure melee)
func (w Weapon) GetRangeIncrement() int {
	return w.RangeIncrement
}
