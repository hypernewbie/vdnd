package affliction

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/item"
)

type AfflictionTracker struct {
	afflictions []*AfflictionInstance
}

func NewTracker() *AfflictionTracker {
	return &AfflictionTracker{
		afflictions: make([]*AfflictionInstance, 0),
	}
}

// Add applies a new affliction, starting at stage 1
func (t *AfflictionTracker) Add(aff *Affliction, source string) {
	if t.Has(aff.ID) {
		return
	}
	t.afflictions = append(t.afflictions, NewInstance(aff, source))
}

// AddInstance adds a pre-constructed instance
func (t *AfflictionTracker) AddInstance(inst *AfflictionInstance) {
	if t.Has(inst.Affliction.ID) {
		return
	}
	t.afflictions = append(t.afflictions, inst)
}

// Has checks if entity has a specific affliction
func (t *AfflictionTracker) Has(afflictionID string) bool {
	return t.Get(afflictionID) != nil
}

// Get returns an affliction instance by ID
func (t *AfflictionTracker) Get(afflictionID string) *AfflictionInstance {
	for _, inst := range t.afflictions {
		if inst.Affliction.ID == afflictionID {
			return inst
		}
	}
	return nil
}

// Remove cures/removes an affliction
func (t *AfflictionTracker) Remove(afflictionID string) {
	for i, inst := range t.afflictions {
		if inst.Affliction.ID == afflictionID {
			t.afflictions = append(t.afflictions[:i], t.afflictions[i+1:]...)
			return
		}
	}
}

// All returns all active affliction instances
func (t *AfflictionTracker) All() []*AfflictionInstance {
	return t.afflictions
}

// ProcessSave updates stage based on save result
func (t *AfflictionTracker) ProcessSave(afflictionID string, result check.DegreeOfSuccess) {
	inst := t.Get(afflictionID)
	if inst == nil {
		return
	}

	switch result {
	case check.CriticalSuccess:
		inst.CurrentStage -= 2
	case check.Success:
		inst.CurrentStage -= 1
	case check.Failure:
		inst.CurrentStage += 1
	case check.CriticalFailure:
		inst.CurrentStage += 2
	}

	// Clamp to valid range
	if inst.CurrentStage < 0 {
		inst.CurrentStage = 0
	}
	if inst.CurrentStage > inst.Affliction.MaxStage {
		inst.CurrentStage = inst.Affliction.MaxStage
	}

	// Reset interval timer on save
	inst.TimeToNextSave = inst.Affliction.Interval

	// Remove if cured
	if inst.CurrentStage == 0 {
		t.Remove(afflictionID)
	}
}

// TickResult contains information about what happened during a tick
type TickResult struct {
	AfflictionID string
	SaveNeeded   bool
	Damage       int
	DamageType   item.DamageType
	Conditions   []ConditionEffect
	IsFatal      bool
}

// Tick advances time and processes effects
func (t *AfflictionTracker) Tick(unit ability.IntervalUnit) []TickResult {
	results := make([]TickResult, 0)

	for _, inst := range t.afflictions {
		if inst.Affliction.IntervalUnit != unit {
			continue
		}

		// Check onset
		if inst.TimeToOnset > 0 {
			inst.TimeToOnset -= 1
			continue
		}

		// Apply current stage effects
		dmgRoll, dmgType, conditions := inst.GetCurrentEffects()
		damage := 0
		if dmgRoll.Count > 0 {
			damage = dmgRoll.Roll()
		}

		isFatal := false
		if stage := inst.Affliction.GetStage(inst.CurrentStage); stage != nil {
			isFatal = stage.IsFatal
		}

		// Update save timer
		inst.TimeToNextSave -= 1

		tickRes := TickResult{
			AfflictionID: inst.Affliction.ID,
			Damage:       damage,
			DamageType:   dmgType,
			Conditions:   conditions,
			IsFatal:      isFatal,
		}
		if inst.TimeToNextSave <= 0 {
			tickRes.SaveNeeded = true
		}
		results = append(results, tickRes)
	}

	return results
}