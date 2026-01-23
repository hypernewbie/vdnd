package combat

import (
	"fmt"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
)

type TurnState struct {
	Entity           *entity.Entity
	ActionsRemaining int // Starts at 3 (modified by slowed/quickened)
	ReactionUsed     bool
	AttacksMade      int // For MAP calculation

	// Track what happened this turn for sweep, forceful, etc.
	StrikesMade []StrikeRecord
}

type StrikeRecord struct {
	TargetID string
	Hit      bool
	WeaponID string
}

func NewTurn(e *entity.Entity) *TurnState {
	actions := 3

	// Quickened grants +1 action
	if e.Conditions.Has(condition.Quickened) {
		actions += 1
	}

	// Slowed reduces actions
	actions -= e.Conditions.Value(condition.Slowed)

	if actions < 0 {
		actions = 0
	}

	return &TurnState{
		Entity:           e,
		ActionsRemaining: actions,
		ReactionUsed:     false,
		AttacksMade:      0,
		StrikesMade:      make([]StrikeRecord, 0),
	}
}

// SpendActions deducts actions, returns error if not enough
func (t *TurnState) SpendActions(cost ActionCost) error {
	costInt := 0
	switch cost {
	case CostOne:
		costInt = 1
	case CostTwo:
		costInt = 2
	case CostThree:
		costInt = 3
	}

	if t.ActionsRemaining < costInt {
		return fmt.Errorf("not enough actions: have %d, need %d", t.ActionsRemaining, costInt)
	}

	t.ActionsRemaining -= costInt
	return nil
}

// SpendReaction marks reaction as used
func (t *TurnState) SpendReaction() error {
	if t.ReactionUsed {
		return fmt.Errorf("reaction already used")
	}
	t.ReactionUsed = true
	return nil
}

// GetMAP returns current Multiple Attack Penalty
func (t *TurnState) GetMAP(isAgile bool) int {
	return CalculateMAP(t.AttacksMade+1, isAgile)
}

// RecordAttack increments attack counter for MAP
func (t *TurnState) RecordAttack() {
	t.AttacksMade++
}

// RecordStrike records a strike for sweep/forceful traits
func (t *TurnState) RecordStrike(record StrikeRecord) {
	t.StrikesMade = append(t.StrikesMade, record)
}

// CanAct returns true if entity can take actions (not stunned, etc.)
func (t *TurnState) CanAct() bool {
	// Simplified: just check if they are paralyzed or unconscious
	if t.Entity.Conditions.Has(condition.Paralyzed) || t.Entity.IsUnconscious() {
		return false
	}
	return true
}
