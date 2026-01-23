package hazard

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/affliction"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
)

type HazardResult struct {
	Target      *entity.Entity
	Damage      int
	DamageType  item.DamageType
	Conditions  []condition.ConditionInstance
	Description string
}

type HazardEffect interface {
	Apply(hazard *Hazard, targets []*entity.Entity) []HazardResult
}

// DamageEffect - deals damage, optionally with save
type DamageEffect struct {
	Damage      dice.DieRoll
	DamageType  item.DamageType
	SaveType    ability.SaveType
	SaveDC      int
	IsBasicSave bool
}

func (d *DamageEffect) Apply(hazard *Hazard, targets []*entity.Entity) []HazardResult {
	results := make([]HazardResult, 0)
	for _, target := range targets {
		res := HazardResult{Target: target, DamageType: d.DamageType}
		degree := check.Success

		if d.SaveType != ability.SaveNone {
			var mod int
			switch d.SaveType {
			case ability.SaveFortitude:
				mod = target.GetFortitude()
			case ability.SaveReflex:
				mod = target.GetReflex()
			case ability.SaveWill:
				mod = target.GetWill()
			}
			checkRes := check.PerformCheck(mod, nil, d.SaveDC)
			degree = checkRes.Degree
		}

		baseDamage := d.Damage.Roll()
		actualDamage := baseDamage

		if d.IsBasicSave {
			switch degree {
			case check.CriticalSuccess:
				actualDamage = 0
			case check.Success:
				actualDamage = baseDamage / 2
			case check.Failure:
				actualDamage = baseDamage
			case check.CriticalFailure:
				actualDamage = baseDamage * 2
			}
		} else if d.SaveType != ability.SaveNone {
			// Non-basic save: usually Failure/CritFailure means full damage, Success/CritSuccess means none
			if degree >= check.Success {
				actualDamage = 0
			}
		}

		if actualDamage > 0 {
			target.ApplyDamage(actualDamage)
			res.Damage = actualDamage
			res.Description = "Took damage"
		} else {
			res.Description = "Avoided damage"
		}

		results = append(results, res)
	}
	return results
}

// ConditionEffect - applies conditions
type ConditionEffect struct {
	ConditionID condition.ConditionID
	Value       int
	Duration    int
	SaveType    ability.SaveType
	SaveDC      int
}

func (c *ConditionEffect) Apply(hazard *Hazard, targets []*entity.Entity) []HazardResult {
	results := make([]HazardResult, 0)
	for _, target := range targets {
		degree := check.Success
		if c.SaveType != ability.SaveNone {
			var mod int
			switch c.SaveType {
			case ability.SaveFortitude:
				mod = target.GetFortitude()
			case ability.SaveReflex:
				mod = target.GetReflex()
			case ability.SaveWill:
				mod = target.GetWill()
			}
			checkRes := check.PerformCheck(mod, nil, c.SaveDC)
			degree = checkRes.Degree
		}

		if degree <= check.Failure {
			inst := condition.NewValuedCondition(c.ConditionID, c.Value, hazard.Name)
			inst.Duration = c.Duration
			target.Conditions.Apply(inst)
			results = append(results, HazardResult{
				Target:      target,
				Description: "Applied condition " + string(c.ConditionID),
			})
		}
	}
	return results
}

// MultiEffect - combines multiple effects
type MultiEffect struct {
	Effects []HazardEffect
}

func (m *MultiEffect) Apply(hazard *Hazard, targets []*entity.Entity) []HazardResult {
	results := make([]HazardResult, 0)
	for _, effect := range m.Effects {
		results = append(results, effect.Apply(hazard, targets)...)
	}
	return results
}

// AttackEffect - makes an attack roll
type AttackEffect struct {
	AttackBonus int
	Damage      dice.DieRoll
	DamageType  item.DamageType
}

func (a *AttackEffect) Apply(hazard *Hazard, targets []*entity.Entity) []HazardResult {
	results := make([]HazardResult, 0)
	for _, target := range targets {
		targetAC := target.GetAC()
		checkRes := check.PerformCheck(a.AttackBonus, nil, targetAC)

		res := HazardResult{Target: target, DamageType: a.DamageType}
		if checkRes.Degree >= check.Success {
			damage := a.Damage.Roll()
			if checkRes.Degree == check.CriticalSuccess {
				damage *= 2
			}
			target.ApplyDamage(damage)
			res.Damage = damage
			res.Description = "Hit by attack"
		} else {
			res.Description = "Attack missed"
		}
		results = append(results, res)
	}
	return results
}

// AfflictionEffect - applies an affliction
type AfflictionEffect struct {
	Affliction affliction.Affliction
	OnHit      bool // Only if another effect (Attack) hits
}

func (a *AfflictionEffect) Apply(hazard *Hazard, targets []*entity.Entity) []HazardResult {
	results := make([]HazardResult, 0)
	for _, target := range targets {
		// In a real system we'd check if an attack hit this target. 
		// For MVP, we'll just apply it or the caller handles the logic.
		target.Afflictions.Add(&a.Affliction, hazard.Name)
		results = append(results, HazardResult{
			Target:      target,
			Description: "Exposed to " + a.Affliction.Name,
		})
	}
	return results
}
