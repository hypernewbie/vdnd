package combat

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/trait"
)

func TestSkillActions(t *testing.T) {
	actor := entity.NewEntity("a1", "Hero", 1)
	actor.MaxHP = 20
	actor.CurrentHP = 20
	target := entity.NewEntity("t1", "Target", 1)
	target.MaxHP = 20
	target.CurrentHP = 20
	turn := NewTurn(actor)

	// Demoralize
	demoralize := &DemoralizeAction{}
	if demoralize.Name() != "Demoralize" {
		t.Errorf("Expected Demoralize, got %s", demoralize.Name())
	}
	if !demoralize.HasTrait(trait.TraitFear) {
		t.Error("Demoralize should have Fear trait")
	}
	demoralize.Execute(actor, target, turn)

	// Grapple
	grapple := &GrappleAction{}
	if !grapple.HasTrait(trait.TraitAttack) {
		t.Error("Grapple should have Attack trait")
	}
	turn.ActionsRemaining = 3
	grapple.Execute(actor, target, turn)

	// Trip
	trip := &TripAction{}
	turn.ActionsRemaining = 3
	trip.Execute(actor, target, turn)

	// Shove
	shove := &ShoveAction{}
	turn.ActionsRemaining = 3
	shove.Execute(actor, target, turn)

	// Hide
	hide := &HideAction{}
	turn.ActionsRemaining = 3
	hide.Execute(actor, target, turn)

	// Seek
	seek := &SeekAction{}
	turn.ActionsRemaining = 3
	seek.Execute(actor, target, turn)

	// Recall Knowledge
	recall := &RecallKnowledgeAction{Skill: ability.SkillArcana}
	turn.ActionsRemaining = 3
	recall.Execute(actor, target, turn)
}
