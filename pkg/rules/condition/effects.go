package condition

import (
	"uaa/vdnd/pkg/rules/check"
)

// GetModifiers returns all universal modifiers from active conditions
func (t *ConditionTracker) GetModifiers() []check.Modifier {
	mods := make([]check.Modifier, 0)

	if t.Has(Frightened) {
		mods = append(mods, check.Modifier{
			Value:  -t.Value(Frightened),
			Type:   check.BonusStatus,
			Source: "Frightened",
		})
	}

	if t.Has(Sickened) {
		mods = append(mods, check.Modifier{
			Value:  -t.Value(Sickened),
			Type:   check.BonusStatus,
			Source: "Sickened",
		})
	}

	if t.Has(Fatigued) {
		mods = append(mods, check.Modifier{
			Value:  -1,
			Type:   check.BonusStatus,
			Source: "Fatigued",
		})
	}

	if t.Has(Fascinated) {
		mods = append(mods, check.Modifier{
			Value:  -2,
			Type:   check.BonusStatus,
			Source: "Fascinated",
		})
	}

	return mods
}

// GetACModifiers returns modifiers that apply to AC
func (t *ConditionTracker) GetACModifiers() []check.Modifier {
	mods := t.GetModifiers()

	if t.Has(FlatFooted) || t.Has(Prone) || t.Has(Grabbed) || t.Has(Restrained) || t.Has(Paralyzed) {
		mods = append(mods, check.Modifier{
			Value:  -2,
			Type:   check.BonusCircumstance,
			Source: "Flat-footed",
		})
	}

	if t.Has(Clumsy) {
		mods = append(mods, check.Modifier{
			Value:  -t.Value(Clumsy),
			Type:   check.BonusStatus,
			Source: "Clumsy",
		})
	}

	if t.Has(Unconscious) {
		mods = append(mods, check.Modifier{
			Value:  -4,
			Type:   check.BonusStatus,
			Source: "Unconscious",
		})
	}

	return mods
}

// GetAttackModifiers returns modifiers that apply to attack rolls
func (t *ConditionTracker) GetAttackModifiers(isMelee bool) []check.Modifier {
	mods := t.GetModifiers()

	if t.Has(Enfeebled) && isMelee {
		mods = append(mods, check.Modifier{
			Value:  -t.Value(Enfeebled),
			Type:   check.BonusStatus,
			Source: "Enfeebled",
		})
	}

	if t.Has(Clumsy) && !isMelee {
		mods = append(mods, check.Modifier{
			Value:  -t.Value(Clumsy),
			Type:   check.BonusStatus,
			Source: "Clumsy",
		})
	}

	if t.Has(Prone) {
		mods = append(mods, check.Modifier{
			Value:  -2,
			Type:   check.BonusCircumstance,
			Source: "Prone",
		})
	}

	return mods
}

// GetSaveModifiers returns modifiers that apply to saving throws
func (t *ConditionTracker) GetSaveModifiers() []check.Modifier {
	mods := t.GetModifiers()

	// Note: Specific conditions like Clumsy (Reflex) or Drained (Fortitude)
	// would normally be handled by more specific check lookups.
	// For now, we return the general ones.

	return mods
}

// GetActionsLost returns how many actions are lost (from slowed/stunned)
func (t *ConditionTracker) GetActionsLost() int {
	lost := 0
	if t.Has(Slowed) {
		lost += t.Value(Slowed)
	}
	if t.Has(Stunned) {
		lost += t.Value(Stunned)
	}
	return lost
}
