package damage

import (
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
)

// CriticalEffects determines what extra effects happen on a crit
type CriticalEffects struct {
	DeadlyDie    dice.DieRoll // From deadly trait
	FatalDie     dice.DieRoll // From fatal trait
	ExtraEffects []string     // Critical specialization, etc.
}

// GetCriticalEffects inspects a weapon for crit-related traits
func GetCriticalEffects(weapon *item.Weapon) CriticalEffects {
	return CriticalEffects{
		DeadlyDie: weapon.DeadlyDie,
		FatalDie:  weapon.FatalDie,
	}
}

// ApplyCriticalSpecialization applies weapon group crit effects
func ApplyCriticalSpecialization(target *entity.Entity, group item.WeaponGroup) []condition.ConditionInstance {
	applied := []condition.ConditionInstance{}

	switch group {
	case item.GroupSword, item.GroupAxe, item.GroupSpear:
		// Target takes 1d6 persistent bleed
		// We'd need persistent damage implementation
	case item.GroupHammer, item.GroupFlail, item.GroupClub:
		// Target knocked prone
		cond := condition.NewCondition(condition.Prone, "Crit Spec")
		target.Conditions.Apply(cond)
		applied = append(applied, cond)
	case item.GroupKnife:
		// Target flat-footed until end of your next turn
		cond := condition.NewCondition(condition.FlatFooted, "Crit Spec")
		target.Conditions.Apply(cond)
		applied = append(applied, cond)
	case item.GroupBrawling:
		// Target slowed 1 until end of your next turn
		cond := condition.NewValuedCondition(condition.Slowed, 1, "Crit Spec")
		target.Conditions.Apply(cond)
		applied = append(applied, cond)
	}

	return applied
}
