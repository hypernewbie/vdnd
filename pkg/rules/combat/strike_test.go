package combat

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
	"uaa/vdnd/pkg/rules/trait"
)

func TestStrikeTraitsComprehensive(t *testing.T) {
	actor := entity.NewEntity("a1", "Fighter", 1)
	actor.Abilities.Strength = 18 // +4

	target1 := entity.NewEntity("t1", "Target 1", 1)
	target2 := entity.NewEntity("t2", "Target 2", 1)

	// Weapon with Forceful
	maul := item.Weapon{
		ID:         "maul-1",
		Name:       "Maul",
		Damage:     dice.DieRoll{Count: 2, Sides: 6}, // 2d6
		DamageType: item.Bludgeoning,
		Traits:     []trait.TraitID{trait.TraitForceful},
		IsMelee:    true,
	}

	strike := NewStrike(&maul)
	turn := NewTurn(actor)

	// First Strike: No Forceful bonus
	res1, _ := strike.ExecuteWithRoll(actor, target1, turn, 10)
	// Base damage: 2d6 + 4. Min 6.
	if res1.Success && res1.Damage < 6 {
		t.Errorf("First strike damage too low: %d", res1.Damage)
	}

	// Second Strike: Forceful bonus = num dice (2)
	res2, _ := strike.ExecuteWithRoll(actor, target1, turn, 15)
	// Base damage: 2d6 + 4 + 2. Min 8.
	if res2.Success && res2.Damage < 8 {
		t.Errorf("Second strike (forceful) damage too low: %d", res2.Damage)
	}

	// Third Strike: Forceful bonus = 2 * num dice (4)
	res3, _ := strike.ExecuteWithRoll(actor, target1, turn, 15)
	// Base damage: 2d6 + 4 + 4. Min 10.
	if res3.Success && res3.Damage < 10 {
		t.Errorf("Third strike (forceful) damage too low: %d", res3.Damage)
	}

	// Test Sweep with Different Weapons
	turn = NewTurn(actor)
	axe := item.Weapon{
		ID:      "axe-1",
		Name:    "Axe",
		Traits:  []trait.TraitID{trait.TraitSweep},
		IsMelee: true,
		Damage:  dice.DieRoll{Count: 1, Sides: 8},
	}
	axeStrike := NewStrike(&axe)

	// Strike target 1 with axe
	axeStrike.ExecuteWithRoll(actor, target1, turn, 10)

	// Use target2 to avoid unused error
	axeStrike.ExecuteWithRoll(actor, target2, turn, 10)
}

func TestStrikeMetadata(t *testing.T) {
	w := &item.Weapon{ID: "w1", Name: "Sword"}
	s := NewStrike(w)

	if s.Name() != "Strike" {
		t.Errorf("Expected Strike, got %s", s.Name())
	}
	if s.Cost() != ability.CostOne {
		t.Errorf("Expected CostOne, got %v", s.Cost())
	}
	if !s.HasTrait(trait.TraitAttack) {
		t.Error("Strike should have Attack trait")
	}

	actor := entity.NewEntity("a1", "Hero", 1)
	actor.MaxHP = 20
	actor.CurrentHP = 20
	turn := NewTurn(actor)

	if err := s.Validate(actor, nil, turn); err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	actor.CurrentHP = 0 // Cannot act
	if err := s.Validate(actor, nil, turn); err == nil {
		t.Error("Validation should fail when actor cannot act")
	}
}
