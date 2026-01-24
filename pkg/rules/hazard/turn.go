package hazard

import (
	"fmt"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/damage"
	"uaa/vdnd/pkg/rules/entity"
)

// TurnResult contains all results from a hazard's turn
type TurnResult struct {
	HazardID      string
	HazardName    string
	ActionResults []ActionResult
	TotalDamage   int
	WasReset      bool
}

// ActionResult contains the result of a single routine action
type ActionResult struct {
	ActionName  string
	ActionType  RoutineActionType
	Targets     []TargetResult
	Description string
}

// TargetResult contains what happened to a specific target
type TargetResult struct {
	EntityID   string
	EntityName string
	Hit        bool
	SaveResult check.DegreeOfSuccess
	Damage     int
	Effect     string
}

// TakeTurn executes the hazard's routine
func (h *Hazard) TakeTurn(targets []*entity.Entity) TurnResult {
	result := TurnResult{
		HazardID:      h.ID,
		HazardName:    h.Name,
		ActionResults: make([]ActionResult, 0),
	}

	if h.IsDisabled || h.Routine == nil {
		return result
	}

	actionsRemaining := h.Routine.TotalActions

	for _, action := range h.Routine.Actions {
		if actionsRemaining < action.ActionCost {
			continue
		}
		actionsRemaining -= action.ActionCost

		actionResult := h.executeAction(action, targets)
		result.ActionResults = append(result.ActionResults, actionResult)

		// Tally damage
		for _, tr := range actionResult.Targets {
			result.TotalDamage += tr.Damage
		}

		if action.Type == RoutineReset {
			result.WasReset = true
		}
	}

	return result
}

// executeAction runs a single routine action
func (h *Hazard) executeAction(action RoutineAction, targets []*entity.Entity) ActionResult {
	result := ActionResult{
		ActionName: action.Name,
		ActionType: action.Type,
		Targets:    make([]TargetResult, 0),
	}

	// Filter targets by position if needed
	affectedTargets := h.filterTargetsByPosition(targets, action.AffectsPosition)

	switch action.Type {
	case RoutineAttack:
		result = h.executeAttack(action, affectedTargets)
	case RoutineSaveEffect:
		result = h.executeSaveEffect(action, affectedTargets)
	case RoutineAreaEffect:
		result = h.executeAreaEffect(action, affectedTargets)
	case RoutineReset:
		h.Reset()
		result.Description = "Hazard resets for another activation"
	case RoutineSpecial:
		if action.CustomEffect != nil {
			hazardResults := action.CustomEffect(h, affectedTargets)
			for _, hr := range hazardResults {
				result.Targets = append(result.Targets, TargetResult{
					EntityID:   hr.Target.ID,
					EntityName: hr.Target.Name,
					Damage:     hr.Damage,
					Effect:     hr.Description,
				})
			}
		}
	}

	return result
}

func (h *Hazard) executeAttack(action RoutineAction, targets []*entity.Entity) ActionResult {
	result := ActionResult{
		ActionName: action.Name,
		ActionType: RoutineAttack,
		Targets:    make([]TargetResult, 0),
	}

	for _, target := range targets {
		tr := TargetResult{
			EntityID:   target.ID,
			EntityName: target.Name,
		}

		// Roll attack
		attackRoll := check.PerformCheck(action.AttackBonus, nil, target.GetAC(nil))

		if attackRoll.Degree >= check.Success {
			tr.Hit = true
			dmgRoll := action.DamageDice.Roll()
			if attackRoll.Degree == check.CriticalSuccess {
				dmgRoll *= 2
			}
			tr.Damage = dmgRoll

			// Apply damage
			dmgInstance := damage.DamageInstance{
				Amount: dmgRoll,
				Type:   action.DamageType,
				Source: h.Name,
			}
			damage.ProcessDamage(target, dmgInstance, attackRoll.Degree == check.CriticalSuccess)

			tr.Effect = fmt.Sprintf("%d %s damage", tr.Damage, action.DamageType)
		} else {
			tr.Effect = "Missed"
		}

		result.Targets = append(result.Targets, tr)
	}

	return result
}

func (h *Hazard) executeSaveEffect(action RoutineAction, targets []*entity.Entity) ActionResult {
	result := ActionResult{
		ActionName: action.Name,
		ActionType: RoutineSaveEffect,
		Targets:    make([]TargetResult, 0),
	}

	for _, target := range targets {
		tr := TargetResult{
			EntityID:   target.ID,
			EntityName: target.Name,
		}

		// Roll save
		saveMod := h.getParticipantSaveModifier(target, action.SaveType)
		saveResult := check.PerformCheck(saveMod, nil, action.SaveDC)
		tr.SaveResult = saveResult.Degree

		switch saveResult.Degree {
		case check.CriticalSuccess:
			tr.Effect = "Unaffected"
		case check.Success:
			tr.Effect = action.SuccessEffect
		case check.Failure:
			tr.Effect = action.FailureEffect
		case check.CriticalFailure:
			tr.Effect = action.CritFailEffect
		}

		result.Targets = append(result.Targets, tr)
	}

	return result
}

func (h *Hazard) executeAreaEffect(action RoutineAction, targets []*entity.Entity) ActionResult {
	// Similar to save effect but always affects all targets at position
	return h.executeSaveEffect(action, targets)
}

func (h *Hazard) filterTargetsByPosition(targets []*entity.Entity, position string) []*entity.Entity {
	if position == "" {
		// Affects all targets (usually those at hazard's position)
		filtered := make([]*entity.Entity, 0)
		for _, t := range targets {
			if t.Position == h.Position {
				filtered = append(filtered, t)
			}
		}
		return filtered
	}

	// Filter to specific position
	filtered := make([]*entity.Entity, 0)
	for _, t := range targets {
		if t.Position == position {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func (h *Hazard) getParticipantSaveModifier(e *entity.Entity, saveType ability.SaveType) int {
	switch saveType {
	case ability.SaveFortitude:
		return e.GetFortitude()
	case ability.SaveReflex:
		return e.GetReflex()
	case ability.SaveWill:
		return e.GetWill()
		default:
			return 0
		}
	}
	
	// Reset prepares the hazard to trigger again
	func (h *Hazard) Reset() {
		h.IsTriggered = false
	}
	