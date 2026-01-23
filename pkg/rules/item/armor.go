package item

import "uaa/vdnd/pkg/rules/trait"

type ArmorCategory int

const (
	Unarmored ArmorCategory = iota
	LightArmor
	MediumArmor
	HeavyArmor
)

type Armor struct {
	ID           string
	Name         string
	Category     ArmorCategory
	ACBonus      int // Item bonus to AC
	DexCap       int // Max DEX to AC (-1 = no cap)
	CheckPenalty int // Penalty to STR/DEX checks (negative number)
	SpeedPenalty int // Penalty to Speed (negative number)
	Strength     int // Min STR to ignore penalties
	Bulk         int
	Traits       trait.TraitSet
}

// NewArmor creates armour with given stats
func NewArmor(id, name string, cat ArmorCategory, acBonus, dexCap, checkPen, speedPen int) Armor {
	return Armor{
		ID:           id,
		Name:         name,
		Category:     cat,
		ACBonus:      acBonus,
		DexCap:       dexCap,
		CheckPenalty: checkPen,
		SpeedPenalty: speedPen,
		Traits:       make(trait.TraitSet, 0),
	}
}

// EffectiveCheckPenalty returns 0 if character meets STR requirement
func (a Armor) EffectiveCheckPenalty(strength int) int {
	if strength >= a.Strength {
		return 0
	}
	return a.CheckPenalty
}

// EffectiveSpeedPenalty returns 0 if character meets STR requirement
func (a Armor) EffectiveSpeedPenalty(strength int) int {
	if strength >= a.Strength {
		return 0
	}
	return a.SpeedPenalty
}

// AppliedDexBonus caps DEX bonus at DexCap
func (a Armor) AppliedDexBonus(dexMod int) int {
	if a.DexCap == -1 {
		return dexMod
	}
	if dexMod > a.DexCap {
		return a.DexCap
	}
	return dexMod
}
