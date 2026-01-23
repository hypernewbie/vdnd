package item

import (
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/trait"
)

// Core Melee Weapons
var (
	Fist = NewWeapon("fist", "Fist", CategoryUnarmed, GroupBrawling,
		dice.DieRoll{Count: 1, Sides: 4, Modifier: 0}, Bludgeoning, 1,
		trait.TraitAgile, trait.TraitFinesse)

	Dagger = NewWeapon("dagger", "Dagger", CategorySimple, GroupKnife,
		dice.DieRoll{Count: 1, Sides: 4, Modifier: 0}, Piercing, 1,
		trait.TraitAgile, trait.TraitFinesse, trait.TraitThrown, trait.TraitVersatile)

	Longsword = NewWeapon("longsword", "Longsword", CategoryMartial, GroupSword,
		dice.DieRoll{Count: 1, Sides: 8, Modifier: 0}, Slashing, 1,
		trait.TraitVersatile)

	Greatsword = NewWeapon("greatsword", "Greatsword", CategoryMartial, GroupSword,
		dice.DieRoll{Count: 1, Sides: 12, Modifier: 0}, Slashing, 2)

	Rapier = NewWeapon("rapier", "Rapier", CategoryMartial, GroupSword,
		dice.DieRoll{Count: 1, Sides: 6, Modifier: 0}, Piercing, 1,
		trait.TraitDeadly, trait.TraitFinesse)
)

// Core Ranged Weapons
var (
	Shortbow = NewWeapon("shortbow", "Shortbow", CategoryMartial, GroupBow,
		dice.DieRoll{Count: 1, Sides: 6, Modifier: 0}, Piercing, 2,
		trait.TraitDeadly)

	Crossbow = NewWeapon("crossbow", "Crossbow", CategorySimple, GroupCrossbow,
		dice.DieRoll{Count: 1, Sides: 8, Modifier: 0}, Piercing, 2)
)

// Core Armor
var (
	NoArmor      = NewArmor("unarmored", "Unarmored", Unarmored, 0, -1, 0, 0)
	LeatherArmor = NewArmor("leather", "Leather Armor", LightArmor, 1, 4, -1, 0)
	ChainShirt   = NewArmor("chain-shirt", "Chain Shirt", LightArmor, 2, 3, -1, 0)
	ChainMail    = NewArmor("chain-mail", "Chain Mail", MediumArmor, 4, 1, -2, -5)
	PlateArmor   = NewArmor("plate", "Full Plate", HeavyArmor, 6, 0, -3, -10)
)

func init() {
	// Initialize range increments for ranged weapons
	Shortbow.RangeIncrement = 60
	Crossbow.RangeIncrement = 120
	Dagger.RangeIncrement = 10

	// Initialize STR requirements for armor
	LeatherArmor.Strength = 10
	ChainShirt.Strength = 12
	ChainMail.Strength = 16
	PlateArmor.Strength = 18
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
