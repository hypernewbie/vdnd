package combat

import (
	"fmt"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/trait"
)

// RaiseShieldAction - 1 action, grants shield's AC bonus until start of next turn
// src: rules/rules/actions/raise-a-shield.md
type RaiseShieldAction struct{}

func (r *RaiseShieldAction) Name() string             { return "Raise a Shield" }
func (r *RaiseShieldAction) Cost() ability.ActionCost { return ability.CostOne }

func (r *RaiseShieldAction) HasTrait(id trait.TraitID) bool {
	return false // No special traits
}

func (r *RaiseShieldAction) Validate(actor, target *entity.Entity, turn *TurnState) error {
	if actor.WornShield == nil {
		return fmt.Errorf("no shield equipped")
	}
	if actor.WornShield.IsBroken() {
		return fmt.Errorf("shield is broken")
	}
	if actor.WornShield.IsRaised {
		return fmt.Errorf("shield already raised")
	}
	return nil
}

func (r *RaiseShieldAction) Execute(actor, _ *entity.Entity, turn *TurnState) ability.ActionResult {
	if err := turn.SpendActions(r.Cost()); err != nil {
		return ability.ActionResult{Success: false, Description: err.Error()}
	}

	if err := r.Validate(actor, nil, turn); err != nil {
		return ability.ActionResult{Success: false, Description: err.Error()}
	}

	actor.WornShield.IsRaised = true

	desc := fmt.Sprintf("Shield raised (+%d AC)", actor.WornShield.ACBonus)
	return ability.ActionResult{Success: true, Description: desc}
}

// ShieldBlockReaction - Reaction to reduce incoming physical damage
// src: rules/rules/actions/shield-block.md
type ShieldBlockReaction struct{}

func (s *ShieldBlockReaction) Name() string { return "Shield Block" }

func (s *ShieldBlockReaction) TriggerType() ReactionTrigger {
	return TriggerOnDamageTaken
}

func (s *ShieldBlockReaction) CanUse(actor *entity.Entity, event ReactionEvent) bool {
	// Must have shield raised
	if actor.WornShield == nil || !actor.WornShield.IsRaised {
		return false
	}
	// Shield must not be broken
	if actor.WornShield.IsBroken() {
		return false
	}
	// Damage must be physical (slashing, piercing, bludgeoning)
	if !event.DamageType.IsPhysical() {
		return false
	}
	return true
}

// Execute reduces damage by Hardness, deals remainder to both target and shield
func (s *ShieldBlockReaction) Execute(actor *entity.Entity, event *ReactionEvent) ShieldBlockResult {
	shield := actor.WornShield
	hardness := shield.Hardness

	// Reduce damage by hardness
	reducedDamage := event.Damage - hardness
	if reducedDamage < 0 {
		reducedDamage = 0
	}

	// Shield takes damage equal to the damage dealt (after hardness, but shield takes from original)
	// PF2E: "The shield prevents you from taking an amount of damage up to the shield's Hardness.
	//        You and the shield each take any remaining damage."
	shieldDamage := event.Damage - hardness
	if shieldDamage < 0 {
		shieldDamage = 0
	}
	shield.TakeDamage(shieldDamage)

	return ShieldBlockResult{
		DamageToActor:   reducedDamage,
		DamageToShield:  shieldDamage,
		ShieldBroken:    shield.IsBroken(),
		ShieldDestroyed: shield.IsDestroyed(),
	}
}

type ShieldBlockResult struct {
	DamageToActor   int
	DamageToShield  int
	ShieldBroken    bool
	ShieldDestroyed bool
}
