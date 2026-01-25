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

// GetACModifiers returns modifiers that apply to AC relative to an attacker
func (t *ConditionTracker) GetACModifiers(attacker *ConditionTracker, attackerID string) []check.Modifier {
	mods := t.GetModifiers()

	isFlatFooted := t.IsFlatFooted() || t.HasRelative(FlatFooted, attackerID)

	// Attacker being hidden/undetected makes target flat-footed
	if !isFlatFooted && attacker != nil {
		if attacker.HasRelative(Hidden, t.GetID()) || attacker.HasRelative(Undetected, t.GetID()) {
			isFlatFooted = true
		}
	}

	if isFlatFooted {
		mods = append(mods, check.Modifier{
			Value:  -2,
			Type:   check.BonusCircumstance,
			Source: "Flat-footed",
		})
	}

	clumsyVal := t.Value(Clumsy)
	if encClumsy := t.GetEncumbranceClumsy(); encClumsy > clumsyVal {
		clumsyVal = encClumsy
	}

	if clumsyVal > 0 {
		mods = append(mods, check.Modifier{
			Value:  -clumsyVal,
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

	if t.Has(StandardCover) {
		mods = append(mods, check.Modifier{
			Value:  2,
			Type:   check.BonusCircumstance,
			Source: "Standard Cover",
		})
	}

	if t.Has(TakingCover) {
		mods = append(mods, check.Modifier{
			Value:  4,
			Type:   check.BonusCircumstance,
			Source: "Taking Cover",
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

	clumsyVal := t.Value(Clumsy)
	if encClumsy := t.GetEncumbranceClumsy(); encClumsy > clumsyVal {
		clumsyVal = encClumsy
	}

	if clumsyVal > 0 && !isMelee {
		mods = append(mods, check.Modifier{
			Value:  -clumsyVal,
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
func (t *ConditionTracker) GetSaveModifiers(saveType string) []check.Modifier {
	mods := t.GetModifiers()

	if t.Has(StandardCover) && saveType == "Reflex" {
		mods = append(mods, check.Modifier{
			Value:  2,
			Type:   check.BonusCircumstance,
			Source: "Standard Cover",
		})
	}

	if t.Has(TakingCover) && saveType == "Reflex" {
		mods = append(mods, check.Modifier{
			Value:  4,
			Type:   check.BonusCircumstance,
			Source: "Taking Cover",
		})
	}

	clumsyVal := t.Value(Clumsy)
	if encClumsy := t.GetEncumbranceClumsy(); encClumsy > clumsyVal {
		clumsyVal = encClumsy
	}

	if clumsyVal > 0 && saveType == "Reflex" {
		mods = append(mods, check.Modifier{
			Value:  -clumsyVal,
			Type:   check.BonusStatus,
			Source: "Clumsy",
		})
	}

	if t.Has(Drained) && saveType == "Fortitude" {
		mods = append(mods, check.Modifier{
			Value:  -t.Value(Drained),
			Type:   check.BonusStatus,
			Source: "Drained",
		})
	}

	if t.Has(Stupefied) && saveType == "Will" {
		mods = append(mods, check.Modifier{
			Value:  -t.Value(Stupefied),
			Type:   check.BonusStatus,
			Source: "Stupefied",
		})
	}

	return mods
}

// GetSpeedPenalty returns speed penalty from conditions
func (t *ConditionTracker) GetSpeedPenalty() int {
	penalty := 0

	if t.Has(Encumbered) {
		penalty -= 10
	}

	// Immobilized = speed 0
	if t.Has(Immobilized) {
		return -9999 // Effectively 0 speed
	}

	return penalty
}

// GetEncumbranceClumsy returns clumsy value from encumbrance
func (t *ConditionTracker) GetEncumbranceClumsy() int {
	if t.Has(Encumbered) {
		return 1
	}
	return 0
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
