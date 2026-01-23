package combat

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/trait"
)

// Re-export or use ability.ActionCost directly.
// For now let's just use ability.ActionCost in the interface.

// Action interface for all executable actions
type Action interface {
	Name() string
	Cost() ability.ActionCost
	HasTrait(trait.TraitID) bool
	// Validate checks if action can be performed
	Validate(actor *entity.Entity, target *entity.Entity, turn *TurnState) error
	// Execute performs the action and returns result
	Execute(actor *entity.Entity, target *entity.Entity, turn *TurnState) ability.ActionResult
}