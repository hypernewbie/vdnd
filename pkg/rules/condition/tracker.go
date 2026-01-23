package condition

import (
	"uaa/vdnd/pkg/rules/check"
)

type PersistentDamageActor interface {
	ApplyDamage(amount int)
	GetID() string
}

type ConditionTracker struct {
	conditions       []*ConditionInstance
	persistentDamage []*ConditionInstance
}

func NewTracker() *ConditionTracker {
	return &ConditionTracker{
		conditions:       make([]*ConditionInstance, 0),
		persistentDamage: make([]*ConditionInstance, 0),
	}
}

// Apply adds or updates a condition.
// For valued conditions: takes the HIGHER value if already present.
// For relational conditions: allows multiple instances if TargetID is different.
func (t *ConditionTracker) Apply(c ConditionInstance) {
	if c.ID == PersistentDamage {
		for _, existing := range t.persistentDamage {
			if existing.DamageType == c.DamageType {
				if c.Value > existing.Value {
					existing.Value = c.Value
				}
				return
			}
		}
		inst := c
		t.persistentDamage = append(t.persistentDamage, &inst)
		return
	}

	// Find existing match
	for _, existing := range t.conditions {
		if existing.ID == c.ID && existing.TargetID == c.TargetID {
			if c.Value > existing.Value {
				existing.Value = c.Value
			}
			// Usually we don't stack non-valued conditions or those with same target
			return
		}
	}

	inst := c
	t.conditions = append(t.conditions, &inst)
}

// Remove completely removes a condition (all instances of this ID)
func (t *ConditionTracker) Remove(id ConditionID) {
	newConds := make([]*ConditionInstance, 0)
	for _, c := range t.conditions {
		if c.ID != id {
			newConds = append(newConds, c)
		}
	}
	t.conditions = newConds
}

// RemoveRelative removes a condition relative to a specific target
func (t *ConditionTracker) RemoveRelative(id ConditionID, targetID string) {
	newConds := make([]*ConditionInstance, 0)
	for _, c := range t.conditions {
		if !(c.ID == id && c.TargetID == targetID) {
			newConds = append(newConds, c)
		}
	}
	t.conditions = newConds
}

// RemovePersistentDamage removes a specific type of persistent damage
func (t *ConditionTracker) RemovePersistentDamage(damageType string) {
	remaining := make([]*ConditionInstance, 0)
	for _, c := range t.persistentDamage {
		if c.DamageType != damageType {
			remaining = append(remaining, c)
		}
	}
	t.persistentDamage = remaining
}

// Reduce decreases a valued condition's value; removes if reduced to 0
func (t *ConditionTracker) Reduce(id ConditionID, amount int) {
	for i, c := range t.conditions {
		if c.ID == id {
			c.Value -= amount
			if c.Value <= 0 {
				// Remove this one
				t.conditions = append(t.conditions[:i], t.conditions[i+1:]...)
			}
			return // Only reduce one (typically there's only one valued global condition)
		}
	}
}

// Has checks if the entity has a specific condition (ignores TargetID)
func (t *ConditionTracker) Has(id ConditionID) bool {
	for _, c := range t.conditions {
		if c.ID == id {
			return true
		}
	}
	return false
}

// HasRelative checks if the entity has a condition relative to a specific target
func (t *ConditionTracker) HasRelative(id ConditionID, targetID string) bool {
	for _, c := range t.conditions {
		if c.ID == id && (c.TargetID == "" || c.TargetID == targetID) {
			return true
		}
	}
	return false
}

// Get returns the first condition instance found with this ID
func (t *ConditionTracker) Get(id ConditionID) *ConditionInstance {
	for _, c := range t.conditions {
		if c.ID == id {
			return c
		}
	}
	return nil
}

// Value returns the value of the first instance found with this ID
func (t *ConditionTracker) Value(id ConditionID) int {
	if inst := t.Get(id); inst != nil {
		return inst.Value
	}
	return 0
}

// All returns all active conditions
func (t *ConditionTracker) All() []ConditionInstance {
	all := make([]ConditionInstance, 0, len(t.conditions)+len(t.persistentDamage))
	for _, c := range t.conditions {
		all = append(all, *c)
	}
	for _, c := range t.persistentDamage {
		all = append(all, *c)
	}
	return all
}

// EndTurn processes end-of-turn effects
func (t *ConditionTracker) EndTurn(actor PersistentDamageActor) {
	t.Reduce(Frightened, 1)

	// 1. Persistent Damage: Apply damage, then flat check DC 15 to remove
	remainingPersistent := make([]*ConditionInstance, 0)
	for _, pd := range t.persistentDamage {
		// Apply damage (entity logic handles resistances/weaknesses if we use ApplyDamage)
		if actor != nil {
			actor.ApplyDamage(pd.Value)
		}

		// Flat check DC 15
		if !check.FlatCheck(15) {
			remainingPersistent = append(remainingPersistent, pd)
		}
	}
	t.persistentDamage = remainingPersistent

	// 2. Duration-based conditions tick down
	remaining := make([]*ConditionInstance, 0)
	for _, cond := range t.conditions {
		if cond.Duration > 0 {
			cond.Duration--
			if cond.Duration > 0 {
				remaining = append(remaining, cond)
			}
		} else if cond.Duration == -1 {
			remaining = append(remaining, cond)
		}
	}
	t.conditions = remaining
}

// StartTurn processes start-of-turn effects
func (t *ConditionTracker) StartTurn() {
}

// IsFlatFooted checks if any condition makes the entity flat-footed
func (t *ConditionTracker) IsFlatFooted() bool {
	return t.Has(FlatFooted) || t.Has(Prone) || t.Has(Grabbed) || t.Has(Restrained) || t.Has(Paralyzed) || t.Has(Unconscious)
}
