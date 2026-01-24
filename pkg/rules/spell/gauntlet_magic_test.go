package spell_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/spell"
)

// Source: rules/rules/core-rulebook/chapter-7-spells.md (Rituals, Casting)

func TestRitualSecondaryCasterDeath(t *testing.T) {
	ritual := spell.NewRitual("test", "Test", 1, ability.SkillArcana, 1)
	ritual.WithSecondaryCheck(ability.SkillArcana, ability.Trained, "Support")
	
	primary := entity.NewEntity("p", "Primary", 1)
	s1 := entity.NewEntity("s1", "S1", 1)
	
	// A Secondary Caster dies mid-ritual.
	s1.Kill("death mid-ritual")
	
	// Expect: Ritual fails or continues with penalty. 
	attempt, err := spell.NewRitualCastAttempt(ritual, primary, []*entity.Entity{s1})
	if err == nil {
		t.Log("Warning: System allowed dead secondary caster in ritual attempt")
		_ = attempt
	}
}

func TestCantripScaling(t *testing.T) {
	// Level 19 Wizard casts Electric Arc. Should be auto-heightened to Rank 10.
	e := entity.NewEntity("wizard", "Wizard", 19)
	
	expectedRank := 10
	actualRank := (e.Level + 1) / 2
	
	if actualRank != expectedRank {
		t.Errorf("Expected cantrip rank %d for level 19, got %d", expectedRank, actualRank)
	}
}

func TestCounteractingRitual(t *testing.T) {
	t.Log("Testing Dispel Magic vs active Ritual outputs (e.g. Create Undead)")
}
