package combat_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/combat"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
)

func TestShieldBreakageMidCombat(t *testing.T) {
	e := entity.NewEntity("test", "Test", 1)
	shield := &item.Shield{
		Name:      "Steel Shield",
		ACBonus:   2,
		Hardness:  5,
		MaxHP:     20,
		CurrentHP: 2, // Already damaged, near breaking
	}
	e.WornShield = shield

	// 1. Raise shield
	raise := &combat.RaiseShieldAction{}
	turn := combat.NewTurn(e)
	raise.Execute(e, nil, turn)

	if !shield.IsRaised {
		t.Fatal("Shield should be raised")
	}

	// 2. Take heavy damage and block
	// Damage 15, Hardness 5. 
	// Shield takes 15 - 5 = 10 damage.
	
	shield.TakeDamage(10)

	if !shield.IsDestroyed() {
		t.Error("Shield should be destroyed")
	}

	// Mid-combat check: Should not be able to raise it if it's destroyed
	err := raise.Validate(e, nil, turn)
	if err == nil {
		t.Error("Should not be able to validate RaiseShield with destroyed shield")
	}
}

func TestShieldSnapshottingAC(t *testing.T) {
	e := entity.NewEntity("test", "Test", 1)
	shield := &item.Shield{
		Name:      "Steel Shield",
		ACBonus:   2,
		MaxHP:     20,
		CurrentHP: 20,
	}
	e.WornShield = shield

	baseAC := e.GetAC(nil)

	// Raise
	raise := &combat.RaiseShieldAction{}
	turn := combat.NewTurn(e)
	raise.Execute(e, nil, turn)

	if e.GetAC(nil) != baseAC+2 {
		t.Errorf("Expected AC %d, got %d", baseAC+2, e.GetAC(nil))
	}

	// Simulate Turn Reset (Next Round or Start of Turn)
	combat.ResetShieldState(e)
	
	if shield.IsRaised {
		t.Error("Shield should be lowered after reset")
	}
	if e.GetAC(nil) != baseAC {
		t.Errorf("AC should return to %d, got %d", baseAC, e.GetAC(nil))
	}
}

func TestShieldDamageTypes(t *testing.T) {
	e := entity.NewEntity("test", "Test", 1)
	shield := &item.Shield{
		Name:      "Wooden Shield",
		Hardness:  3,
		MaxHP:     10,
		CurrentHP: 10,
	}
	e.WornShield = shield
	shield.IsRaised = true

	reaction := &combat.ShieldBlockReaction{}

	// Physical damage
	physEvent := combat.ReactionEvent{
		Trigger:    combat.TriggerOnDamageTaken,
		DamageType: item.Slashing,
	}
	if !reaction.CanUse(e, physEvent) {
		t.Errorf("Should be able to block physical damage. Raised: %v, Broken: %v, Phys: %v",
			e.WornShield.IsRaised, e.WornShield.IsBroken(), physEvent.DamageType.IsPhysical())
	}

	// Fire damage (Energy)
	fireEvent := combat.ReactionEvent{
		Trigger:    combat.TriggerOnDamageTaken,
		DamageType: item.Fire,
	}
	if reaction.CanUse(e, fireEvent) {
		t.Error("Should NOT be able to block fire damage")
	}
}