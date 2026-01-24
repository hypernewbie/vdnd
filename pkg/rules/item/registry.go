package item

import (
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/trait"
)

// Core Melee Weapons
var (
	Fist = NewWeapon("fist", "Fist", CategoryUnarmed, GroupBrawling,
		dice.DieRoll{Count: 1, Sides: 4, Modifier: 0}, Bludgeoning, 1, 0, 0,
		trait.TraitAgile, trait.TraitFinesse)

	Dagger = NewWeapon("dagger", "Dagger", CategorySimple, GroupKnife,
		dice.DieRoll{Count: 1, Sides: 4, Modifier: 0}, Piercing, 1, 10, 0,
		trait.TraitAgile, trait.TraitFinesse, trait.TraitThrown, trait.TraitVersatile)

	Longsword = NewWeapon("longsword", "Longsword", CategoryMartial, GroupSword,
		dice.DieRoll{Count: 1, Sides: 8, Modifier: 0}, Slashing, 1, 0, 1,
		trait.TraitVersatile)

	Greatsword = NewWeapon("greatsword", "Greatsword", CategoryMartial, GroupSword,
		dice.DieRoll{Count: 1, Sides: 12, Modifier: 0}, Slashing, 2, 0, 2)

	Rapier = NewWeapon("rapier", "Rapier", CategoryMartial, GroupSword,
		dice.DieRoll{Count: 1, Sides: 6, Modifier: 0}, Piercing, 1, 0, 1,
		trait.TraitDeadly, trait.TraitFinesse)
)

// Core Ranged Weapons
var (
	Shortbow = NewWeapon("shortbow", "Shortbow", CategoryMartial, GroupBow,
		dice.DieRoll{Count: 1, Sides: 6, Modifier: 0}, Piercing, 2, 60, 2,
		trait.TraitDeadly)

	Crossbow = NewWeapon("crossbow", "Crossbow", CategorySimple, GroupCrossbow,
		dice.DieRoll{Count: 1, Sides: 8, Modifier: 0}, Piercing, 2, 120, 2)
)

// Core Armor
var (
	NoArmor      = NewArmor("unarmored", "Unarmored", Unarmored, 0, -1, 0, 0, 0, 0)
	LeatherArmor = NewArmor("leather", "Leather Armor", LightArmor, 1, 4, -1, 0, 10, 1)
	ChainShirt   = NewArmor("chain-shirt", "Chain Shirt", LightArmor, 2, 3, -1, 0, 12, 1)
	ChainMail    = NewArmor("chain-mail", "Chain Mail", MediumArmor, 4, 1, -2, -5, 16, 2)
	PlateArmor   = NewArmor("plate", "Full Plate", HeavyArmor, 6, 0, -3, -10, 18, 4)
)

var StandardShields = map[string]*Shield{
	"buckler":       NewShield("buckler", "Buckler", 1, 3, 6, 1),
	"wooden_shield": NewShield("wooden_shield", "Wooden Shield", 2, 3, 12, 1),
	"steel_shield":  NewShield("steel_shield", "Steel Shield", 2, 5, 20, 1),
	"tower_shield":  NewShield("tower_shield", "Tower Shield", 2, 5, 20, 4),
}

func init() {
	// Initialize specialized fields
	Dagger.ThrownRange = 10
	Dagger.VersatileType = Slashing

	Longsword.VersatileType = Piercing

	Rapier.DeadlyDie = dice.DieRoll{Count: 1, Sides: 8, Modifier: 0}
	Shortbow.DeadlyDie = dice.DieRoll{Count: 1, Sides: 10, Modifier: 0}

	// Tower shield has special trait
	StandardShields["tower_shield"].SpeedPenalty = -5
}

var weaponRegistry = map[string]Weapon{
	"fist":       Fist,
	"dagger":     Dagger,
	"longsword":  Longsword,
	"greatsword": Greatsword,
	"rapier":     Rapier,
	"shortbow":   Shortbow,
	"crossbow":   Crossbow,
}

var armorRegistry = map[string]Armor{
	"unarmored":   NoArmor,
	"leather":     LeatherArmor,
	"chain-shirt": ChainShirt,
	"chain-mail":  ChainMail,
	"plate":       PlateArmor,
}

// GetWeapon returns a weapon by ID
func GetWeapon(id string) (Weapon, bool) {
	w, ok := weaponRegistry[id]
	return w, ok
}

// GetArmor returns armor by ID
func GetArmor(id string) (Armor, bool) {
	a, ok := armorRegistry[id]
	return a, ok
}

// GetShield returns shield by ID
func GetShield(id string) (*Shield, bool) {
	s, ok := StandardShields[id]
	if !ok {
		return nil, false
	}
	// Return a copy so state doesn't leak between entities
	copy := *s
	return &copy, true
}
