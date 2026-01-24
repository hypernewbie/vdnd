package spell

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/combat"
	"uaa/vdnd/pkg/rules/damage"
	"uaa/vdnd/pkg/rules/entity"
)

type CastSpellAction struct {
	Spell   *Spell
	Caster  *entity.Entity
	Targets []*entity.Entity
}

func NewCastSpell(spell *Spell, caster *entity.Entity, targets []*entity.Entity) *CastSpellAction {
	return &CastSpellAction{
		Spell:   spell,
		Caster:  caster,
		Targets: targets,
	}
}

func (c *CastSpellAction) Name() string            { return "Cast a Spell: " + c.Spell.Name }
func (c *CastSpellAction) Cost() ability.ActionCost { return c.Spell.ActionCost }

func (c *CastSpellAction) Execute(turn *combat.TurnState) []EffectResult {
	results := []EffectResult{}
	dc := GetSpellDC(c.Caster)

	// Roll effect once (e.g. damage roll)
	rollResult := c.Spell.Effect.Roll(c.Caster)

	for _, target := range c.Targets {
		var degree check.DegreeOfSuccess

		if c.Spell.Save != ability.SaveNone {
			// Target makes save
			saveResult := TargetMakesSave(target, c.Spell.Save, dc)
			degree = saveResult.Degree
		} else if c.Spell.RequiresAttackRoll {
			// Spell attack
			attackResult := RollSpellAttack(c.Caster, target, nil)
			degree = attackResult.Degree
		} else {
			// Auto-hit
			degree = check.Success
		}

		// Apply spell effect using the pre-rolled result
		result := c.Spell.Effect.Apply(c.Caster, target, degree, rollResult)

		// Handle basic save damage adjustment
		if c.Spell.IsBasicSave {
			result.Damage = ApplyBasicSaveDamage(result.Damage, degree)
		}

		// Apply damage through pipeline
		if result.Damage > 0 {
			damage.ProcessDamage(target, damage.DamageInstance{
				Amount: result.Damage,
				Type:   result.DamageType,
				Source: c.Spell.Name,
			}, degree == check.CriticalSuccess)
		}

		// Apply healing
		if result.Healed > 0 {
			target.Heal(result.Healed)
		}

		// Apply conditions
		for _, cond := range result.Conditions {
			target.Conditions.Apply(cond)
		}

		results = append(results, result)
	}

	return results
}

func GetSpellAttackModifier(caster *entity.Entity) int {
	abilityMod := caster.Abilities.Modifier(caster.SpellcastingAbility)
	profBonus := caster.SpellProficiency.Bonus(caster.Level)
	// Add condition modifiers
	conditionMods := caster.Conditions.GetModifiers()
	return abilityMod + profBonus + check.CalculateTotal(conditionMods)
}

func GetSpellDC(caster *entity.Entity) int {
	return 10 + GetSpellAttackModifier(caster)
}

func RollSpellAttack(caster, target *entity.Entity, modifiers []check.Modifier) check.CheckResult {
	attackMod := GetSpellAttackModifier(caster)
	// Spell attack vs AC
	targetAC := target.GetAC(caster)
	return check.PerformCheck(attackMod, modifiers, targetAC)
}

func TargetMakesSave(target *entity.Entity, saveType ability.SaveType, dc int) check.CheckResult {
	var mod int
	switch saveType {
	case ability.SaveFortitude:
		mod = target.GetFortitude()
	case ability.SaveReflex:
		mod = target.GetReflex()
	case ability.SaveWill:
		mod = target.GetWill()
	default:
		mod = 0
	}

	// Saving throws are checks
	return check.PerformCheck(mod, nil, dc)
}

func ApplyBasicSaveDamage(baseDamage int, degree check.DegreeOfSuccess) int {
	switch degree {
	case check.CriticalSuccess:
		return 0
	case check.Success:
		return baseDamage / 2
	case check.Failure:
		return baseDamage
	case check.CriticalFailure:
		return baseDamage * 2
	}
	return baseDamage
}
