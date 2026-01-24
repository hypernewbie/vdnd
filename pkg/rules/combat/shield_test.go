package combat_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/combat"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
)

func TestRaiseShieldAction(t *testing.T) {
	actor := entity.NewEntity("test", "Test", 1)
	actor.WornShield = item.NewShield("steel_shield", "Steel Shield", 2, 5, 20, 1)

	turn := combat.NewTurn(actor)
	action := &combat.RaiseShieldAction{}

	result := action.Execute(actor, nil, turn)
	if !result.Success {
		t.Errorf("Raise Shield should succeed: %s", result.Description)
	}
	if !actor.WornShield.IsRaised {
		t.Error("Shield should be raised after action")
	}
}

func TestShieldBlockReducesDamage(t *testing.T) {
	actor := entity.NewEntity("test", "Test", 1)
	actor.WornShield = item.NewShield("steel_shield", "Steel Shield", 2, 5, 20, 1)
	actor.WornShield.IsRaised = true

	reaction := &combat.ShieldBlockReaction{}
	event := &combat.ReactionEvent{
		Damage:     12,
		DamageType: item.Slashing,
	}

	if !reaction.CanUse(actor, *event) {
		t.Error("Should be able to use Shield Block with raised shield")
	}

	result := reaction.Execute(actor, event)

	// Damage to actor = 12 - 5 (hardness) = 7
    if result.DamageToActor != 7 {
        t.Errorf("Expected 7 damage to actor, got %d", result.DamageToActor)
    }

    // Shield takes 7 damage
    if result.DamageToShield != 7 {
        t.Errorf("Expected 7 damage to shield, got %d", result.DamageToShield)
    }

    // Shield HP = 20 - 7 = 13
    if actor.WornShield.CurrentHP != 13 {
        t.Errorf("Expected shield HP 13, got %d", actor.WornShield.CurrentHP)
    }
}

func TestShieldBlockRequiresRaised(t *testing.T) {
	actor := entity.NewEntity("test", "Test", 1)
	actor.WornShield = item.NewShield("steel_shield", "Steel Shield", 2, 5, 20, 1)
	// Shield NOT raised

	reaction := &combat.ShieldBlockReaction{}
	event := combat.ReactionEvent{
		Damage:     10,
		DamageType: item.Slashing,
	}

	if reaction.CanUse(actor, event) {
		t.Error("Should NOT be able to use Shield Block without raised shield")
	}
}

func TestBrokenShieldNoACBonus(t *testing.T) {
	actor := entity.NewEntity("test", "Test", 1)
	actor.Abilities.Dexterity = 14 // +2 mod
	actor.WornShield = item.NewShield("steel_shield", "Steel Shield", 2, 5, 20, 1)
	actor.WornShield.IsRaised = true

	acWithShield := actor.GetAC(nil)

	// Break the shield
	actor.WornShield.CurrentHP = 5 // Below BT of 10

	acBroken := actor.GetAC(nil)

	if acBroken >= acWithShield {
		t.Errorf("Broken shield should not grant AC bonus. With: %d, Broken: %d", acWithShield, acBroken)
	}
}
