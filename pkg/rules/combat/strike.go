package combat

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
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

func (s *StrikeAction) Name() string     { return "Strike" }
func (s *StrikeAction) Cost() ActionCost { return CostOne }
func (s *StrikeAction) HasTrait(id trait.TraitID) bool {
	if id == trait.TraitAttack {
		return true
	}
	return s.Weapon.HasTrait(id)
}

func (s *StrikeAction) Validate(actor, target *entity.Entity, turn *TurnState) error {
	if !turn.CanAct() {
		return interface{}(nil).(error) // placeholder for "cannot act" error
	}
	return nil
}

func (s *StrikeAction) Execute(actor, target *entity.Entity, turn *TurnState) ActionResult {
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

	// Target's AC
	targetAC := target.GetAC()

	// Perform the check
	res := check.PerformCheck(attackMod, modifiers, targetAC)

	// Record attack for MAP
	turn.RecordAttack()

	// Process result
	if res.Degree == check.CriticalSuccess {
		damage := s.rollDamage(actor, true)
		target.TakeDamage(damage, string(s.Weapon.DamageType))
		turn.RecordStrike(StrikeRecord{TargetID: target.ID, Hit: true, WeaponID: s.Weapon.ID})
		return ActionResult{Success: true, Degree: res.Degree, Damage: damage}
	} else if res.Degree == check.Success {
		damage := s.rollDamage(actor, false)
		target.TakeDamage(damage, string(s.Weapon.DamageType))
		turn.RecordStrike(StrikeRecord{TargetID: target.ID, Hit: true, WeaponID: s.Weapon.ID})
		return ActionResult{Success: true, Degree: res.Degree, Damage: damage}
	}

	turn.RecordStrike(StrikeRecord{TargetID: target.ID, Hit: false, WeaponID: s.Weapon.ID})
	return ActionResult{Success: false, Degree: res.Degree}
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

func (s *StrikeAction) rollDamage(actor *entity.Entity, isCrit bool) int {
	damage := s.Weapon.Damage.Roll()

	// Add STR to melee damage
	if s.Weapon.IsMelee {
		damage += actor.Abilities.Modifier(ability.Strength)
	}

	if isCrit {
		damage *= 2

		// Handle Deadly trait (simplified: add one extra die roll of param type)
		// We'd need to parse the Deadly parameter (e.g. "d8")
		// For now, let's just look for the trait and maybe hardcode or skip if complex.
	}

	if damage < 0 {
		damage = 0
	}
	return damage
}
