package combat

import (
	"errors"
	"fmt"

	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/trait"
)

// InteractAction is a generic 1-action object manipulation.
// The outcome is determined by the LLM; this just tracks the action cost.
type InteractAction struct {
	ObjectDescription string // Optional, for output
}

func (i *InteractAction) Name() string             { return "Interact" }
func (i *InteractAction) Cost() ability.ActionCost { return ability.CostOne }
func (i *InteractAction) HasTrait(t trait.TraitID) bool {
	return t == trait.TraitManipulate
}

func (i *InteractAction) Validate(actor, _ *entity.Entity, turn *TurnState) error {
	if turn.ActionsRemaining < 1 {
		return errors.New("no actions remaining")
	}
	return nil
}

func (i *InteractAction) Execute(actor, _ *entity.Entity, turn *TurnState) ability.ActionResult {
	if err := turn.SpendActions(i.Cost()); err != nil {
		return ability.ActionResult{Success: false, Description: err.Error()}
	}
	desc := "Interacted with an object"
	if i.ObjectDescription != "" {
		desc = fmt.Sprintf("Interacted with %s", i.ObjectDescription)
	}
	return ability.ActionResult{Success: true, Description: desc}
}

// DropProneAction is a free action to fall prone.
type DropProneAction struct{}

func (d *DropProneAction) Name() string             { return "Drop Prone" }
func (d *DropProneAction) Cost() ability.ActionCost { return ability.CostFree }
func (d *DropProneAction) HasTrait(_ trait.TraitID) bool { return false }

func (d *DropProneAction) Validate(actor, _ *entity.Entity, _ *TurnState) error {
	if actor.Conditions.Has(condition.Prone) {
		return errors.New("already prone")
	}
	return nil
}

func (d *DropProneAction) Execute(actor, _ *entity.Entity, _ *TurnState) ability.ActionResult {
	actor.Conditions.Apply(condition.NewCondition(condition.Prone, "Dropped prone"))
	return ability.ActionResult{
		Success:     true,
		Description: "Dropped prone. You are flat-footed (-2 AC), take -2 to attack rolls, and gain +1 AC vs ranged.",
	}
}

// StandAction costs 1 action and removes Prone. Has Move trait.
type StandAction struct{}

func (s *StandAction) Name() string             { return "Stand" }
func (s *StandAction) Cost() ability.ActionCost { return ability.CostOne }
func (s *StandAction) HasTrait(t trait.TraitID) bool {
	return t == trait.TraitMove
}

func (s *StandAction) Validate(actor, _ *entity.Entity, turn *TurnState) error {
	if !actor.Conditions.Has(condition.Prone) {
		return errors.New("not prone")
	}
	if turn.ActionsRemaining < 1 {
		return errors.New("no actions remaining")
	}
	return nil
}

func (s *StandAction) Execute(actor, _ *entity.Entity, turn *TurnState) ability.ActionResult {
	if err := turn.SpendActions(s.Cost()); err != nil {
		return ability.ActionResult{Success: false, Description: err.Error()}
	}
	actor.Conditions.Remove(condition.Prone)
	return ability.ActionResult{
		Success:     true,
		Description: "Stood up from prone.",
	}
}

// TakeCoverAction grants +4 circumstance bonus to AC and Reflex.
// This bonus is lost when the entity moves.
type TakeCoverAction struct{}

func (t *TakeCoverAction) Name() string             { return "Take Cover" }
func (t *TakeCoverAction) Cost() ability.ActionCost { return ability.CostOne }
func (t *TakeCoverAction) HasTrait(_ trait.TraitID) bool { return false }

func (t *TakeCoverAction) Validate(actor, _ *entity.Entity, turn *TurnState) error {
	if turn.ActionsRemaining < 1 {
		return errors.New("no actions remaining")
	}
	// GM/LLM determines if cover is available; we just apply the bonus
	return nil
}

func (t *TakeCoverAction) Execute(actor, _ *entity.Entity, turn *TurnState) ability.ActionResult {
	if err := turn.SpendActions(t.Cost()); err != nil {
		return ability.ActionResult{Success: false, Description: err.Error()}
	}
	actor.Conditions.Apply(condition.NewCondition(
		condition.TakingCover,
		"Taking Cover",
	))
	return ability.ActionResult{
		Success:     true,
		Description: "Taking cover (+4 circumstance bonus to AC and Reflex vs area effects). Lost on movement.",
	}
}
