package condition

type ConditionTracker struct {
	conditions       map[ConditionID]*ConditionInstance
	persistentDamage []*ConditionInstance
}

func NewTracker() *ConditionTracker {
	return &ConditionTracker{
		conditions:       make(map[ConditionID]*ConditionInstance),
		persistentDamage: make([]*ConditionInstance, 0),
	}
}

// Apply adds or updates a condition.
// For valued conditions: takes the HIGHER value if already present.
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

	existing, ok := t.conditions[c.ID]
	if !ok {
		inst := c
		t.conditions[c.ID] = &inst
	} else {
		if c.Value > existing.Value {
			existing.Value = c.Value
		}
		// Typically duration/source might update or not, keeping it simple for now.
	}
}

// Remove completely removes a condition
func (t *ConditionTracker) Remove(id ConditionID) {
	delete(t.conditions, id)
}

// Reduce decreases a valued condition's value; removes if reduced to 0
func (t *ConditionTracker) Reduce(id ConditionID, amount int) {
	existing, ok := t.conditions[id]
	if !ok {
		return
	}
	existing.Value -= amount
	if existing.Value <= 0 {
		t.Remove(id)
	}
}

// Has checks if the entity has a specific condition
func (t *ConditionTracker) Has(id ConditionID) bool {
	_, ok := t.conditions[id]
	return ok
}

// Get returns the condition instance, or nil if not present
func (t *ConditionTracker) Get(id ConditionID) *ConditionInstance {
	return t.conditions[id]
}

// Value returns the value of a condition (0 if not present or binary)
func (t *ConditionTracker) Value(id ConditionID) int {
	if inst, ok := t.conditions[id]; ok {
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

// EndTurn processes end-of-turn effects (reduce frightened, etc.)
func (t *ConditionTracker) EndTurn() {
	if t.Has(Frightened) {
		t.Reduce(Frightened, 1)
	}

	// Duration-based conditions tick down
	for id, cond := range t.conditions {
		if cond.Duration > 0 {
			cond.Duration--
			if cond.Duration == 0 {
				t.Remove(id)
			}
		}
	}
}

// StartTurn processes start-of-turn effects
func (t *ConditionTracker) StartTurn() {
	// Placeholder for start of turn logic (e.g. slowed/stunned action reduction)
}

// IsFlatFooted checks if any condition makes the entity flat-footed
func (t *ConditionTracker) IsFlatFooted() bool {
	return t.Has(FlatFooted) || t.Has(Prone) || t.Has(Grabbed) || t.Has(Restrained) || t.Has(Paralyzed) || t.Has(Unconscious)
}
