package combat

import (
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/trait"
)

type ActionCost int

const (
	CostFree ActionCost = iota
	CostReaction
	CostOne
	CostTwo
	CostThree
)

// ActionResult represents the outcome of performing an action
type ActionResult struct {
	Success     bool
	Degree      check.DegreeOfSuccess         // For checks
	Description string                        // Human-readable outcome
	Damage      int                           // If damage was dealt
	Conditions  []condition.ConditionInstance // Conditions applied
}

// Action interface for all executable actions
type Action interface {
	Name() string
	Cost() ActionCost
	HasTrait(trait.TraitID) bool
	// Validate checks if action can be performed
	Validate(actor *entity.Entity, target *entity.Entity) error
	// Execute performs the action and returns result
	Execute(actor *entity.Entity, target *entity.Entity, turn *TurnState) ActionResult
}
