package combat

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/trait"
)

type StrideAction struct{}

func (s *StrideAction) Name() string            { return "Stride" }
func (s *StrideAction) Cost() ability.ActionCost { return ability.CostOne }
func (s *StrideAction) HasTrait(id trait.TraitID) bool {
	return id == trait.TraitMove
}

func (s *StrideAction) Validate(actor, target *entity.Entity, turn *TurnState) error {
	return nil
}

func (s *StrideAction) Execute(actor *entity.Entity, destination string, turn *TurnState) ability.ActionResult {
	if err := turn.SpendActions(s.Cost()); err != nil {
		return ability.ActionResult{Success: false, Description: err.Error()}
	}

	actor.Conditions.Remove(condition.TakingCover)

	actor.Position = destination
	return ability.ActionResult{Success: true, Description: "Strided to " + destination}
}

type StepAction struct{}

func (s *StepAction) Name() string            { return "Step" }
func (s *StepAction) Cost() ability.ActionCost { return ability.CostOne }
func (s *StepAction) HasTrait(id trait.TraitID) bool {
	return id == trait.TraitMove
}

func (s *StepAction) Validate(actor, target *entity.Entity, turn *TurnState) error {
	return nil
}

func (s *StepAction) Execute(actor *entity.Entity, direction string, turn *TurnState) ability.ActionResult {
	if err := turn.SpendActions(s.Cost()); err != nil {
		return ability.ActionResult{Success: false, Description: err.Error()}
	}

	actor.Conditions.Remove(condition.TakingCover)

	// Step doesn't change Position in our zone-based system usually,
	// or it moves to adjacent zone if zones are small.
	return ability.ActionResult{Success: true, Description: "Stepped"}
}