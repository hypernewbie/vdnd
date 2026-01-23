package damage

import (
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/item"
	"uaa/vdnd/pkg/rules/trait"
)

// DamageInstance represents a single source of damage
type DamageInstance struct {
	Amount      int
	Type        item.DamageType
	Source      string          // "Longsword", "Fireball", etc.
	IsPrecision bool            // Sneak Attack, etc.
	Traits      []trait.TraitID // For things like "magical" damage
}

// DamageRoll represents the components of a damage roll before resolution
type DamageRoll struct {
	BaseDice    dice.DieRoll   // e.g., 2d6
	Modifier    int            // Usually STR mod
	BonusDice   []dice.DieRoll // Extra dice (striking rune, etc.)
	DamageType  item.DamageType
	Source      string
	IsPrecision bool
	Traits      []trait.TraitID
}

// Roll evaluates the damage roll and returns a DamageInstance
func (d DamageRoll) Roll() DamageInstance {
	total := d.BaseDice.Roll() + d.Modifier
	for _, bonus := range d.BonusDice {
		total += bonus.Roll()
	}
	return DamageInstance{
		Amount:      total,
		Type:        d.DamageType,
		Source:      d.Source,
		IsPrecision: d.IsPrecision,
		Traits:      d.Traits,
	}
}

// RollCritical evaluates as a critical hit (double damage, extra deadly dice)
func (d DamageRoll) RollCritical(deadlyDie dice.DieRoll, fatalDie dice.DieRoll) DamageInstance {
	// If fatal: damage die changes AND adds extra die
	baseDice := d.BaseDice
	if fatalDie.Sides > 0 {
		baseDice = dice.DieRoll{Count: baseDice.Count, Sides: fatalDie.Sides, Modifier: baseDice.Modifier}
	}

	// Roll base + modifiers, then double
	baseTotal := baseDice.Roll() + d.Modifier
	for _, bonus := range d.BonusDice {
		baseTotal += bonus.Roll()
	}

	doubledTotal := baseTotal * 2

	// Add deadly dice (NOT doubled)
	if deadlyDie.Sides > 0 {
		deadlyCount := deadlyDie.Count
		if deadlyCount == 0 {
			deadlyCount = 1
		}
		doubledTotal += dice.DieRoll{Count: deadlyCount, Sides: deadlyDie.Sides, Modifier: 0}.Roll()
	}

	// Add fatal extra die (NOT doubled)
	if fatalDie.Sides > 0 {
		doubledTotal += dice.DieRoll{Count: 1, Sides: fatalDie.Sides, Modifier: 0}.Roll()
	}

	return DamageInstance{
		Amount:      doubledTotal,
		Type:        d.DamageType,
		Source:      d.Source,
		IsPrecision: d.IsPrecision,
		Traits:      d.Traits,
	}
}
