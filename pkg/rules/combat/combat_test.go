package combat

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
)

func TestMAP(t *testing.T) {
	tests := []struct {
		attack int
		agile  bool
		want   int
	}{
		{1, false, 0},
		{1, true, 0},
		{2, false, -5},
		{2, true, -4},
		{3, false, -10},
		{3, true, -8},
		{4, false, -10},
		{4, true, -8},
	}

	for _, tt := range tests {
		if got := CalculateMAP(tt.attack, tt.agile); got != tt.want {
			t.Errorf("CalculateMAP(%d, %v) = %d, want %d", tt.attack, tt.agile, got, tt.want)
		}
	}
}

func TestTurnState(t *testing.T) {
	e := entity.NewEntity("e1", "Hero", 1)
	turn := NewTurn(e)

	if turn.ActionsRemaining != 3 {
		t.Errorf("Fresh turn should have 3 actions, got %d", turn.ActionsRemaining)
	}

	err := turn.SpendActions(ability.CostOne)
	if err != nil || turn.ActionsRemaining != 2 {
		t.Error("SpendActions(ability.CostOne) failed")
	}

	// Quickened
	e.Conditions.Apply(condition.NewCondition(condition.Quickened, "Haste"))
	turn2 := NewTurn(e)
	if turn2.ActionsRemaining != 4 {
		t.Errorf("Quickened should have 4 actions, got %d", turn2.ActionsRemaining)
	}

	// Slowed
	e.Conditions.Apply(condition.NewValuedCondition(condition.Slowed, 1, "Mud"))
	turn3 := NewTurn(e)
	// 3 (base) + 1 (quickened) - 1 (slowed) = 3
	if turn3.ActionsRemaining != 3 {
		t.Errorf("Quickened + Slowed 1 should have 3 actions, got %d", turn3.ActionsRemaining)
	}
}

func TestStrike(t *testing.T) {
	actor := entity.NewEntity("a1", "Attacker", 1)
	actor.Abilities.Strength = 18                                // +4
	actor.WeaponProficiencies[item.GroupSword] = ability.Trained // +3 at lvl 1

	target := entity.NewEntity("t1", "Target", 1)
	target.Abilities.Dexterity = 10
	target.UnarmoredDefense = ability.Trained // AC = 10 + 0 + 3 = 13

	strike := NewStrike(&item.Longsword)
	turn := NewTurn(actor)

	// Longsword: 1d8, Slashing, Versatile P
	// Attack: +4 (str) + 3 (prof) = +7
	// 1st attack: +7 vs 13

	// We can't easily control the random roll here without mocking check.PerformCheck
	// or using a predictable RNG if we had one.
	// For now, let's just run it to ensure no panics and state updates.
	res := strike.Execute(actor, target, turn)

	if turn.AttacksMade != 1 {
		t.Error("Strike did not increment AttacksMade")
	}

	if res.Success {
		// Verify something happened
	}
}

func TestDemoralize(t *testing.T) {
	actor := entity.NewEntity("a1", "Attacker", 1)
	actor.Abilities.Charisma = 16                                         // +3
	actor.SkillProficiencies[ability.SkillIntimidation] = ability.Trained // +3

	target := entity.NewEntity("t1", "Target", 1)
	target.Abilities.Wisdom = 10
	target.Will = ability.Trained // Will = +0 + 3 = +3, DC = 13

	demo := &DemoralizeAction{}
	turn := NewTurn(actor)

	// Intimidation +6 vs Will DC 13
	res := demo.Execute(actor, target, turn)

	if turn.ActionsRemaining != 2 {
		t.Errorf("Demoralize should cost 1 action, got %d remaining", turn.ActionsRemaining)
	}

	if res.Success {
		if !target.Conditions.Has(condition.Frightened) {
			t.Error("Successful Demoralize did not apply Frightened")
		}
	}
}

func TestTripAction(t *testing.T) {
	actor := entity.NewEntity("a1", "Attacker", 1)
	actor.Abilities.Strength = 18
	actor.SkillProficiencies[ability.SkillAthletics] = ability.Trained

	target := entity.NewEntity("t1", "Target", 1)
	target.Abilities.Dexterity = 10
	target.Reflex = ability.Trained // DC 13

	action := &TripAction{}
	turn := NewTurn(actor)

	// Trip vs Reflex DC 13
	res := action.Execute(actor, target, turn)

	if turn.ActionsRemaining != 2 {
		t.Errorf("Trip should cost 1 action")
	}
	if turn.AttacksMade != 1 {
		t.Errorf("Trip should increment MAP count")
	}
	_ = res
}