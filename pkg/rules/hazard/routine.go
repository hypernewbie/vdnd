package hazard

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
)

// RoutineActionType defines what the hazard does
type RoutineActionType int

const (
	RoutineAttack RoutineActionType = iota
	RoutineSaveEffect
	RoutineAreaEffect
	RoutineReset
	RoutineSpecial
)

// RoutineAction represents one action in a hazard's routine
type RoutineAction struct {
	Name       string
	Type       RoutineActionType
	ActionCost int             // 1, 2, or 3 actions
	TargetCount int            // How many targets it can affect (1 for most attacks, 0/all for AoE)

	// For attacks
	AttackBonus int
	DamageDice  dice.DieRoll
	DamageType  item.DamageType

	// For saves
	SaveType       ability.SaveType
	SaveDC         int
	SuccessEffect  string
	FailureEffect  string
	CritFailEffect string

	// For area effects
	AffectsPosition string // Which position(s) affected

	// Description for special actions
	Description string

	// Custom effect function
	CustomEffect func(h *Hazard, targets []*entity.Entity) []HazardResult
}

// HazardRoutine defines all actions a complex hazard takes per turn
type HazardRoutine struct {
	Actions      []RoutineAction
	TotalActions int // How many actions the hazard has (usually 3)
}

// NewRoutine creates an empty routine
func NewRoutine(totalActions int) *HazardRoutine {
	return &HazardRoutine{
		Actions:      make([]RoutineAction, 0),
		TotalActions: totalActions,
	}
}

// AddAttack adds an attack action to the routine
func (r *HazardRoutine) AddAttack(name string, cost int, attackBonus int, damage dice.DieRoll, damageType item.DamageType, targetCount int) *HazardRoutine {
	r.Actions = append(r.Actions, RoutineAction{
		Name:        name,
		Type:        RoutineAttack,
		ActionCost:  cost,
		AttackBonus: attackBonus,
		DamageDice:  damage,
		DamageType:  damageType,
		TargetCount: targetCount,
	})
	return r
}

// AddSaveEffect adds a saving throw effect
func (r *HazardRoutine) AddSaveEffect(name string, cost int, saveType ability.SaveType, dc int, success, failure, critFail string) *HazardRoutine {
	r.Actions = append(r.Actions, RoutineAction{
		Name:           name,
		Type:           RoutineSaveEffect,
		ActionCost:     cost,
		SaveType:       saveType,
		SaveDC:         dc,
		SuccessEffect:  success,
		FailureEffect:  failure,
		CritFailEffect: critFail,
	})
	return r
}

// AddReset adds a reset action
func (r *HazardRoutine) AddReset(name string) *HazardRoutine {
	r.Actions = append(r.Actions, RoutineAction{
		Name:       name,
		Type:       RoutineReset,
		ActionCost: 1,
	})
	return r
}