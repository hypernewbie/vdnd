package spell

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/combat"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
)

func TestSpellAttackAndDC(t *testing.T) {
	caster := entity.NewEntity("c1", "Wizard", 1)
	caster.Abilities.Intelligence = 18        // +4
	caster.SpellProficiency = ability.Trained // +3 at lvl 1
	caster.SpellcastingAbility = ability.Intelligence

	mod := GetSpellAttackModifier(caster)
	if mod != 7 {
		t.Errorf("Expected spell attack +7, got %d", mod)
	}

	dc := GetSpellDC(caster)
	if dc != 17 {
		t.Errorf("Expected spell DC 17, got %d", dc)
	}
}

func TestBasicSaveDamage(t *testing.T) {
	tests := []struct {
		degree check.DegreeOfSuccess
		base   int
		want   int
	}{
		{check.CriticalSuccess, 20, 0},
		{check.Success, 20, 10},
		{check.Failure, 20, 20},
		{check.CriticalFailure, 20, 40},
	}

	for _, tt := range tests {
		if got := ApplyBasicSaveDamage(tt.base, tt.degree); got != tt.want {
			t.Errorf("ApplyBasicSaveDamage(%d, %v) = %d, want %d", tt.base, tt.degree, got, tt.want)
		}
	}
}

func TestSpellExecution(t *testing.T) {
	caster := entity.NewEntity("c1", "Wizard", 1)
	caster.Abilities.Intelligence = 18
	caster.SpellProficiency = ability.Trained
	caster.SpellcastingAbility = ability.Intelligence

	target := entity.NewEntity("t1", "Goblin", 1)
	target.MaxHP = 20
	target.CurrentHP = 20
	target.Reflex = ability.Trained // mod +3

	// Fireball (6d6, basic Reflex)
	// We'll mock a turn
	turn := combat.NewTurn(caster)
	action := NewCastSpell(&Fireball, caster, []*entity.Entity{target})
	// We can't control the random rolls easily, but we can verify execution flow
	results := action.Execute(turn)

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if target.CurrentHP >= 20 && results[0].Damage > 0 {
		t.Error("Target HP should have decreased if damage was dealt")
	}
}

func TestFearSpell(t *testing.T) {
	caster := entity.NewEntity("c1", "Sorcerer", 1)
	caster.Abilities.Charisma = 18
	caster.SpellProficiency = ability.Trained
	caster.SpellcastingAbility = ability.Charisma

	target := entity.NewEntity("t1", "Guard", 1)
	target.Will = ability.Untrained // mod +0

	// Use the Fear effect directly with a fixed degree to test logic
	effect := &FearEffect{}

	resFail := effect.Apply(caster, target, check.Failure, EffectRoll{})
	if len(resFail.Conditions) == 0 || resFail.Conditions[0].ID != condition.Frightened {
		t.Error("Fear on failure should apply Frightened")
	}
}
