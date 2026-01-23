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

// IsGlobal returns true if the condition applies to everyone
func (c *ConditionInstance) IsGlobal() bool {
	return len(c.SpecificTo) == 0
}

// AppliesTo returns true if the condition applies to the given observer
func (c *ConditionInstance) AppliesTo(observerID string) bool {
	if c.IsGlobal() {
		return true
	}
	for _, id := range c.SpecificTo {
		if id == observerID {
			return true
		}
	}
	return false
}

// Apply adds or updates a condition.
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

	for _, existing := range t.conditions {
		if existing.ID == c.ID && sameStrings(existing.SpecificTo, c.SpecificTo) {
			if c.Value > existing.Value {
				existing.Value = c.Value
			}
			return
		}
	}

	inst := c
	t.conditions = append(t.conditions, &inst)
}

// ApplyRelative is a helper to apply a condition to a specific observer
func (t *ConditionTracker) ApplyRelative(id ConditionID, observerID string, source string) {
	t.Apply(NewRelationalCondition(id, []string{observerID}, source))
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
	for _, c := range t.conditions {
		if c.ID == id {
			newSpecific := make([]string, 0)
			for _, sid := range c.SpecificTo {
				if sid != targetID {
					newSpecific = append(newSpecific, sid)
				}
			}
			c.SpecificTo = newSpecific
		}
	}
}

// Reduce decreases a valued condition's value
func (t *ConditionTracker) Reduce(id ConditionID, amount int) {
	for i, c := range t.conditions {
		if c.ID == id {
			c.Value -= amount
			if c.Value <= 0 {
				t.conditions = append(t.conditions[:i], t.conditions[i+1:]...)
			}
			return
		}
	}
}

// Has checks if the entity has a specific condition (global or any relative)
func (t *ConditionTracker) Has(id ConditionID) bool {
	for _, c := range t.conditions {
		if c.ID == id {
			return true
		}
	}
	return false
}

// HasRelative checks if the entity has a condition relative to a specific target (or Global)
func (t *ConditionTracker) HasRelative(id ConditionID, observerID string) bool {
	for _, c := range t.conditions {
		if c.ID == id && c.AppliesTo(observerID) {
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
	// 1. Reduce Frightened
	t.Reduce(Frightened, 1)

	// 2. Persistent Damage
	remainingPersistent := make([]*ConditionInstance, 0)
	for _, pd := range t.persistentDamage {
		if actor != nil {
			actor.ApplyDamage(pd.Value)
		}
		if !check.FlatCheck(15) {
			remainingPersistent = append(remainingPersistent, pd)
		}
	}
	t.persistentDamage = remainingPersistent

	// 3. Duration Decay
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
