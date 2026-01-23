package condition

import (
	"uaa/vdnd/pkg/rules/check"
)

type ConditionInstance struct {
	ID         ConditionID
	Value      int            // For valued conditions; 0 for binary conditions
	Duration   int            // Rounds remaining; -1 = until removed
	Source     string         // What caused this: "Demoralize", "Dragon Breath", etc.
	DamageType string         // "fire", "bleed", etc. (only for persistent damage)
	SpecificTo []string       // For relational conditions (e.g., "hidden from [Entity IDs]")
	Modifier   check.Modifier // For conditions that grant a bonus/penalty
}

// NewModifierCondition creates a condition that grants a modifier
func NewModifierCondition(source string, bonusType check.BonusType, value int, id string) ConditionInstance {
	return ConditionInstance{
		ID:       ConditionID(id),
		Value:    value,
		Duration: -1,
		Source:   source,
		Modifier: check.Modifier{
			Value:  value,
			Type:   bonusType,
			Source: source,
		},
		SpecificTo: make([]string, 0),
	}
}

// NewCondition creates a binary condition instance
func NewCondition(id ConditionID, source string) ConditionInstance {
	return ConditionInstance{
		ID:         id,
		Value:      0,
		Duration:   -1,
		Source:     source,
		SpecificTo: make([]string, 0),
	}
}

// NewValuedCondition creates a valued condition instance
func NewValuedCondition(id ConditionID, value int, source string) ConditionInstance {
	return ConditionInstance{
		ID:         id,
		Value:      value,
		Duration:   -1,
		Source:     source,
		SpecificTo: make([]string, 0),
	}
}

// NewRelationalCondition creates a condition relative to specific entities
func NewRelationalCondition(id ConditionID, specificTo []string, source string) ConditionInstance {
	return ConditionInstance{
		ID:         id,
		SpecificTo: specificTo,
		Duration:   -1,
		Source:     source,
	}
}

// NewPersistentDamage creates a persistent damage condition
func NewPersistentDamage(amount int, damageType, source string) ConditionInstance {
	return ConditionInstance{
		ID:         PersistentDamage,
		Value:      amount,
		DamageType: damageType,
		Duration:   -1,
		Source:     source,
		SpecificTo: make([]string, 0),
	}
}