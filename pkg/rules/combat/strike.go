package combat

import (
	"fmt"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/damage"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
	"uaa/vdnd/pkg/rules/trait"
)

type StrikeAction struct {
	Weapon *item.Weapon
}

func NewStrike(weapon *item.Weapon) *StrikeAction {
	return &StrikeAction{Weapon: weapon}
}

func (s *StrikeAction) Name() string             { return "Strike" }
func (s *StrikeAction) Cost() ability.ActionCost { return ability.CostOne }
func (s *StrikeAction) HasTrait(id trait.TraitID) bool {
	if id == trait.TraitAttack {
		return true
	}
	return s.Weapon.HasTrait(id)
}

func (s *StrikeAction) Validate(actor, target *entity.Entity, turn *TurnState) error {
	if !turn.CanAct() {
		return fmt.Errorf("actor cannot act")
	}
	return nil
}

func (s *StrikeAction) Execute(actor, target *entity.Entity, turn *TurnState) ability.ActionResult {
	// Calculate attack modifier
	abilityMod := s.calculateAttackAbilityModifier(actor)
	profBonus := actor.GetWeaponProficiency(s.Weapon).Bonus(actor.Level)
	attackMod := abilityMod + profBonus

	// Apply MAP
	mapPenalty := turn.GetMAP(s.Weapon.IsAgile())

	// Get all bonuses/penalties
	modifiers := []check.Modifier{
		{Value: mapPenalty, Type: check.BonusUntyped, Source: "MAP"},
	}
	modifiers = append(modifiers, actor.Conditions.GetAttackModifiers(s.Weapon.IsMelee)...)

	// Sweep Trait logic
	if s.Weapon.HasTrait(trait.TraitSweep) {
		differentTarget := false
		for _, prev := range turn.StrikesMade {
			if prev.WeaponID == s.Weapon.ID && prev.TargetID != target.ID {
				differentTarget = true
				break
			}
		}
		if differentTarget {
			modifiers = append(modifiers, check.Modifier{Value: 1, Type: check.BonusCircumstance, Source: "Sweep"})
		}
	}

	// Target's AC
	targetAC := target.GetAC()

	// Perform the check
	res := check.PerformCheck(attackMod, modifiers, targetAC)

	// Record attack for MAP
	turn.RecordAttack()

	// Process result
	if res.Degree == check.CriticalSuccess {
		dmg := s.rollDamageInstance(actor, turn, true)
		pipelineRes := damage.ProcessDamage(target, dmg, true)
		turn.RecordStrike(StrikeRecord{TargetID: target.ID, Hit: true, WeaponID: s.Weapon.ID})
		return ability.ActionResult{Success: true, Degree: res.Degree, Damage: pipelineRes.FinalDamage}
	} else if res.Degree == check.Success {
		dmg := s.rollDamageInstance(actor, turn, false)
		pipelineRes := damage.ProcessDamage(target, dmg, false)
		turn.RecordStrike(StrikeRecord{TargetID: target.ID, Hit: true, WeaponID: s.Weapon.ID})
		return ability.ActionResult{Success: true, Degree: res.Degree, Damage: pipelineRes.FinalDamage}
	}

	turn.RecordStrike(StrikeRecord{TargetID: target.ID, Hit: false, WeaponID: s.Weapon.ID})
	return ability.ActionResult{Success: false, Degree: res.Degree}
}

func (s *StrikeAction) calculateAttackAbilityModifier(actor *entity.Entity) int {
	if s.Weapon.IsMelee {
		strMod := actor.Abilities.Modifier(ability.Strength)
		if s.Weapon.IsFinesse() {
			dexMod := actor.Abilities.Modifier(ability.Dexterity)
			if dexMod > strMod {
				return dexMod
			}
		}
		return strMod
	}
	// Ranged
	return actor.Abilities.Modifier(ability.Dexterity)
}

func (s *StrikeAction) rollDamageInstance(actor *entity.Entity, turn *TurnState, isCrit bool) damage.DamageInstance {
	dr := damage.DamageRoll{
		BaseDice:   s.Weapon.Damage,
		Modifier:   0,
		DamageType: s.Weapon.DamageType,
		Source:     s.Weapon.Name,
		Traits:     s.Weapon.Traits,
	}

	// Add STR to melee damage
	if s.Weapon.IsMelee {
		dr.Modifier = actor.Abilities.Modifier(ability.Strength)
	}

	// Forceful Trait logic
	if s.Weapon.HasTrait(trait.TraitForceful) {
		strikesWithWeapon := 0
		for _, prev := range turn.StrikesMade {
			if prev.WeaponID == s.Weapon.ID {
				strikesWithWeapon++
			}
		}

		if strikesWithWeapon == 1 {
			// Second attack: bonus = number of damage dice
			dr.Modifier += s.Weapon.Damage.Count
		} else if strikesWithWeapon >= 2 {
			// Third or more: bonus = 2 * number of damage dice
			dr.Modifier += 2 * s.Weapon.Damage.Count
		}
	}

	if isCrit {
		return dr.RollCritical(s.Weapon.DeadlyDie, s.Weapon.FatalDie)
	}
	return dr.Roll()
}
