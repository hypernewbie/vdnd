package entity

import (
	"uaa/vdnd/pkg/rules/condition"
)

// TakeDamage applies damage after immunities/resistances/weaknesses.
// Returns actual damage taken.
func (e *Entity) TakeDamage(amount int, damageType string) int {
	// 1. Check immunity
	if e.IsImmuneTo(damageType) {
		return 0
	}

	// 2. Apply weakness
	weakness := e.GetWeakness(damageType)
	amount += weakness

	// 3. Apply resistance
	resistance := e.GetResistance(damageType)
	amount -= resistance
	if amount < 0 {
		amount = 0
	}

	if amount == 0 {
		return 0
	}

	initialDamage := amount

	// 4. Temp HP absorbs first
	if e.TempHP > 0 {
		absorbed := e.TempHP
		if amount < absorbed {
			absorbed = amount
		}
		e.TempHP -= absorbed
		amount -= absorbed
	}

	// 5. Apply to current HP
	e.CurrentHP -= amount
	if e.CurrentHP < 0 {
		e.CurrentHP = 0
	}

	return initialDamage
}

// Heal restores HP (cannot exceed MaxHP)
func (e *Entity) Heal(amount int) {
	if e.IsDead() {
		return
	}
	e.CurrentHP += amount
	if e.CurrentHP > e.MaxHP {
		e.CurrentHP = e.MaxHP
	}

	// If healed from 0, remove unconscious (from damage) and dying
	if e.CurrentHP > 0 {
		e.Conditions.Remove(condition.Dying)
		e.Conditions.Remove(condition.Unconscious)
	}
}

// AddTempHP adds temporary HP (takes higher if already has temp HP)
func (e *Entity) AddTempHP(amount int) {
	if amount > e.TempHP {
		e.TempHP = amount
	}
}

// IsDead returns true if entity has died
func (e *Entity) IsDead() bool {
	maxDying := 4 - e.Conditions.Value(condition.Doomed)
	return e.Conditions.Value(condition.Dying) >= maxDying
}

// IsDying returns true if entity has dying condition
func (e *Entity) IsDying() bool {
	return e.Conditions.Has(condition.Dying)
}

// IsUnconscious returns true if HP <= 0 or has unconscious condition
func (e *Entity) IsUnconscious() bool {
	return e.CurrentHP <= 0 || e.Conditions.Has(condition.Unconscious)
}

// CheckDying should be called after taking damage at 0 HP.
// Applies dying condition or increases dying value.
func (e *Entity) CheckDying(wasCritical bool) {
	if e.CurrentHP > 0 {
		return
	}

	increase := 1
	if wasCritical {
		increase = 2
	}

	if e.Conditions.Has(condition.Dying) {
		current := e.Conditions.Value(condition.Dying)
		e.Conditions.Apply(condition.NewValuedCondition(condition.Dying, current+increase, "damage at 0 HP"))
	} else {
		dyingValue := increase + e.Conditions.Value(condition.Wounded)
		e.Conditions.Apply(condition.NewValuedCondition(condition.Dying, dyingValue, "reduced to 0 HP"))
		e.Conditions.Apply(condition.NewCondition(condition.Unconscious, "dying"))
	}
}

// RecoveryCheck makes a dying recovery check.
// Simplified for now: just returns a placeholder or logic if DC provided.
// Real recovery check involves a flat check (Phase 1 logic).
func (e *Entity) RecoveryCheck(success bool, critical bool) {
	if !e.IsDying() {
		return
	}

	if success {
		reduce := 1
		if critical {
			reduce = 2
		}
		e.Conditions.Reduce(condition.Dying, reduce)
		if !e.IsDying() {
			// Stabilized
			e.Conditions.Apply(condition.NewValuedCondition(condition.Wounded, e.Conditions.Value(condition.Wounded)+1, "stabilized"))
		}
	} else {
		increase := 1
		if critical {
			increase = 2
		}
		current := e.Conditions.Value(condition.Dying)
		e.Conditions.Apply(condition.NewValuedCondition(condition.Dying, current+increase, "failed recovery check"))
	}
}
