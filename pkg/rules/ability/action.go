package ability

import (
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/condition"
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
