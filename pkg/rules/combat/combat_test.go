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
	actor.WeaponProficiencies[item.GroupSword] = ability.Expert // +4 + 1 = +5 bonus

	target1 := entity.NewEntity("t1", "Orc A", 1)
	target1.UnarmoredDefense = ability.Untrained // AC 11
	target2 := entity.NewEntity("t2", "Orc B", 1)
	target2.UnarmoredDefense = ability.Untrained // AC 11

	// Custom weapon with Sweep and Forceful
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

	// Attack 1: Target 1
	res1 := strike.ExecuteWithRoll(actor, target1, turn, 10)
	if !res1.Success {
		t.Errorf("Attack 1 failed: %v", res1.Degree)
	}

	// Attack 2: Target 2 (Sweep applies)
	res2 := strike.ExecuteWithRoll(actor, target2, turn, 6)
	if !res2.Success {
		t.Errorf("Attack 2 with Sweep failed: %v", res2.Degree)
	}
	if res2.Damage < 6 {
		t.Errorf("Forceful damage bonus not applied? Damage: %d", res2.Damage)
	}

	// Attack 3: Target 2 (Sweep does NOT apply, Forceful is +2 now)
	res3 := strike.ExecuteWithRoll(actor, target2, turn, 12)
	if !res3.Success {
		t.Errorf("Attack 3 failed: %v", res3.Degree)
	}
	if res3.Damage < 7 {
		t.Errorf("Forceful stage 2 bonus not applied? Damage: %d", res3.Damage)
	}
}

func TestBackswing(t *testing.T) {
	actor := entity.NewEntity("a1", "Fighter", 1)
	actor.Abilities.Strength = 18
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

	// Attack 1: Miss (Mod +4 vs AC 10. Roll 1 -> Total 5 -> Failure)
	res1 := strike.ExecuteWithRoll(actor, target, turn, 1)
	if res1.Success {
		t.Error("Attack 1 should have failed")
	}

	// Attack 2: Should have +1 from Backswing
	// Mod: +4, MAP: -5, Backswing: +1 -> Total mod +0 vs AC 11.
	// Roll 11 -> Total 11 -> Success.
	res2 := strike.ExecuteWithRoll(actor, target, turn, 11)
	if !res2.Success {
		t.Errorf("Attack 2 with Backswing failed: %v", res2.Degree)
	}
}

func TestTurnState(t *testing.T) {
	e := entity.NewEntity("e1", "Hero", 1)
	turn := NewTurn(e)
	if turn.ActionsRemaining != 3 {
		t.Errorf("Fresh turn should have 3 actions, got %d", turn.ActionsRemaining)
	}
	e.Conditions.Apply(condition.NewCondition(condition.Quickened, "Haste"))
	turn2 := NewTurn(e)
	if turn2.ActionsRemaining != 4 {
		t.Errorf("Quickened should have 4 actions, got %d", turn2.ActionsRemaining)
	}
}