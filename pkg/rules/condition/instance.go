package condition

type ConditionInstance struct {
	ID         ConditionID
	Value      int    // For valued conditions; 0 for binary conditions
	Duration   int    // Rounds remaining; -1 = until removed
	Source     string // What caused this: "Demoralize", "Dragon Breath", etc.
	DamageType string // "fire", "bleed", etc. (only for persistent damage)
	TargetID   string // For relational conditions (e.g., "concealed from [TargetID]")
}

// NewCondition creates a binary condition instance
func NewCondition(id ConditionID, source string) ConditionInstance {
	return ConditionInstance{
		ID:       id,
		Value:    0,
		Duration: -1,
		Source:   source,
	}
}

// NewRelationalCondition creates a condition relative to another entity
func NewRelationalCondition(id ConditionID, targetID, source string) ConditionInstance {
	return ConditionInstance{
		ID:       id,
		TargetID: targetID,
		Duration: -1,
		Source:   source,
	}
}

// NewValuedCondition creates a valued condition instance
func NewValuedCondition(id ConditionID, value int, source string) ConditionInstance {
	return ConditionInstance{
		ID:       id,
		Value:    value,
		Duration: -1,
		Source:   source,
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
	}
}
