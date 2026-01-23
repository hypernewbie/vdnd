package spell

import (
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
)

// DamageEffect is a simple damage-dealing spell
type DamageEffect struct {
	DamageDice dice.DieRoll
	DamageType item.DamageType
}

func (d *DamageEffect) Roll(caster *entity.Entity) EffectRoll {
	return EffectRoll{Damage: d.DamageDice.Roll()}
}

func (d *DamageEffect) Apply(caster, target *entity.Entity, degree check.DegreeOfSuccess, roll EffectRoll) EffectResult {
	if degree == check.CriticalSuccess && !d.isBasic() {
		// Non-basic spells often do nothing on crit success save
		return EffectResult{}
	}
	return EffectResult{Damage: roll.Damage, DamageType: d.DamageType}
}

func (d *DamageEffect) isBasic() bool { return true } // Simplified

// HealEffect heals the target
type HealEffect struct {
	HealDice dice.DieRoll
}

func (h *HealEffect) Roll(caster *entity.Entity) EffectRoll {
	return EffectRoll{Healed: h.HealDice.Roll()}
}

func (h *HealEffect) Apply(caster, target *entity.Entity, degree check.DegreeOfSuccess, roll EffectRoll) EffectResult {
	return EffectResult{Healed: roll.Healed}
}

// FearEffect implements the Fear spell
type FearEffect struct{}

func (f *FearEffect) Roll(caster *entity.Entity) EffectRoll {
	return EffectRoll{}
}

func (f *FearEffect) Apply(caster, target *entity.Entity, degree check.DegreeOfSuccess, roll EffectRoll) EffectResult {
	res := EffectResult{}
	switch degree {
	case check.CriticalFailure:
		res.Conditions = append(res.Conditions, condition.NewValuedCondition(condition.Frightened, 3, "Fear (Crit Fail)"))
		// Fleeing for 1 round etc
	case check.Failure:
		res.Conditions = append(res.Conditions, condition.NewValuedCondition(condition.Frightened, 2, "Fear (Fail)"))
	case check.Success:
		res.Conditions = append(res.Conditions, condition.NewValuedCondition(condition.Frightened, 1, "Fear (Success)"))
	}
	return res
}
