package combat

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
	"uaa/vdnd/pkg/rules/trait"
)

func TestMAP(t *testing.T) {
	tests := []struct {
		attack int
		agile  bool
		want   int
	}{
		{1, false, 0}, {1, true, 0},
		{2, false, -5}, {2, true, -4},
		{3, false, -10}, {3, true, -8},
	}
	for _, tt := range tests {
		if got := CalculateMAP(tt.attack, tt.agile); got != tt.want {
			t.Errorf("MAP(%d, %v) = %d, want %d", tt.attack, tt.agile, got, tt.want)
		}
	}
}

func TestSweepAndForceful(t *testing.T) {
	actor := entity.NewEntity("a1", "Fighter", 1)
	actor.Abilities.Strength = 18 // +4
	// Total attack bonus = 4 (str) + 0 (trained bonus at lvl 1) = 4

	target1 := entity.NewEntity("t1", "Orc A", 1)
	target1.UnarmoredDefense = ability.Untrained // AC 11
	target2 := entity.NewEntity("t2", "Orc B", 1)
	target2.UnarmoredDefense = ability.Untrained // AC 11

	falchion := item.Weapon{
		ID:         "falchion-1",
		Name:       "Falchion",
		Group:      item.GroupSword,
		Damage:     dice.DieRoll{Count: 1, Sides: 10},
		DamageType: item.Slashing,
		Traits:     []trait.TraitID{trait.TraitSweep, trait.TraitForceful},
		IsMelee:    true,
	}

	strike := NewStrike(&falchion)
	turn := NewTurn(actor)

	// Attack 1: Target 1. Roll 7. Total 11 vs 11 AC -> Success.
	res1 := strike.ExecuteWithRoll(actor, target1, turn, 7)
	if !res1.Success {
		t.Errorf("Attack 1 failed: %v", res1.Degree)
	}

	// Attack 2: Target 2. 
	// Mod: +4 (STR), -5 (MAP), +1 (Sweep) = +0.
	// Roll 11. Total 11 vs 11 AC -> Success.
	res2 := strike.ExecuteWithRoll(actor, target2, turn, 11)
	if !res2.Success {
		t.Errorf("Attack 2 with Sweep failed: %v", res2.Degree)
	}
	// Forceful: +1 damage (1 die)
	// Base damage: 1d10 (min 1) + 4 (STR) + 1 (Forceful) = min 6
	if res2.Damage < 6 {
		t.Errorf("Forceful damage bonus not applied? Damage: %d", res2.Damage)
	}

	// Attack 3: Target 2.
	// Mod: +4 (STR), -10 (MAP), Sweep: 0 (same target as last) = -6.
	// Roll 17. Total 11 vs 11 AC -> Success.
	res3 := strike.ExecuteWithRoll(actor, target2, turn, 17)
	if !res3.Success {
		t.Errorf("Attack 3 failed: %v", res3.Degree)
	}
	// Forceful: +2 damage (2 * 1 die)
	// Base damage: 1d10 (min 1) + 4 (STR) + 2 (Forceful) = min 7
	if res3.Damage < 7 {
		t.Errorf("Forceful stage 2 bonus not applied? Damage: %d", res3.Damage)
	}
}

func TestBackswing(t *testing.T) {
	actor := entity.NewEntity("a1", "Fighter", 1)
	actor.Abilities.Strength = 18 // +4
	target := entity.NewEntity("t1", "Orc", 1)
	target.UnarmoredDefense = ability.Untrained // AC 11

	mace := item.Weapon{
		ID:         "mace-1",
		Name:       "Mace",
		Group:      item.GroupClub,
		Damage:     dice.DieRoll{Count: 1, Sides: 6},
		DamageType: item.Bludgeoning,
		Traits:     []trait.TraitID{trait.TraitBackswing},
		IsMelee:    true,
	}

	strike := NewStrike(&mace)
	turn := NewTurn(actor)

	// Attack 1: Miss. Mod +4. Roll 1. Total 5 failure.
	res1 := strike.ExecuteWithRoll(actor, target, turn, 1)
	if res1.Success {
		t.Error("Attack 1 should have failed")
	}

	// Attack 2: Should have +1 from Backswing.
	// Mod: +4 (STR), -5 (MAP), +1 (Backswing) = +0.
	// Roll 11. Total 11 vs 11 AC -> Success.
	res2 := strike.ExecuteWithRoll(actor, target, turn, 11)
	if !res2.Success {
		t.Errorf("Attack 2 with Backswing failed: %v", res2.Degree)
	}
}

func TestAgileWeapon(t *testing.T) {
	actor := entity.NewEntity("a1", "Rogue", 1)
	actor.Abilities.Strength = 14 // +2
	target := entity.NewEntity("t1", "Orc", 1)
	target.UnarmoredDefense = ability.Untrained // AC 11

	dagger := item.Weapon{
		ID:         "dagger-1",
		Name:       "Dagger",
		Traits:     []trait.TraitID{trait.TraitAgile},
		IsMelee:    true,
		Damage:     dice.DieRoll{Count: 1, Sides: 4},
	}

	strike := NewStrike(&dagger)
	turn := NewTurn(actor)

	// Attack 1: Roll 9. Total 11 -> Success.
	_ = strike.ExecuteWithRoll(actor, target, turn, 9)

	// Attack 2: MAP should be -4.
	// Mod: +2 (STR), -4 (MAP) = -2.
	// Roll 13. Total 11 -> Success.
	res2 := strike.ExecuteWithRoll(actor, target, turn, 13)
	if !res2.Success {
		t.Errorf("Agile MAP -4 should have succeeded, got %v", res2.Degree)
	}

	// Attack 3: MAP should be -8.
	// Mod: +2 (STR), -8 (MAP) = -6.
	// Roll 17. Total 11 -> Success.
	res3 := strike.ExecuteWithRoll(actor, target, turn, 17)
	if !res3.Success {
		t.Errorf("Agile MAP -8 should have succeeded, got %v", res3.Degree)
	}
}

func TestCombatPersistentDamage(t *testing.T) {
	actor := entity.NewEntity("a1", "Hero", 1)
	actor.MaxHP = 100
	actor.CurrentHP = 100
	
	actor.Conditions.Apply(condition.NewPersistentDamage(10, "fire", "Burn"))
	
	// Simulate end of turn
	actor.Conditions.EndTurn(actor)
	
	if actor.CurrentHP >= 100 {
		t.Errorf("Expected damage from persistent fire, got %d", actor.CurrentHP)
	}
}
