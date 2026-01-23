package affliction

import (
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/item"
)

type AfflictionInstance struct {
	Affliction     *Affliction
	CurrentStage   int
	TimeToOnset    int // Countdown in intervals
	TimeToNextSave int
	HasOnsetted    bool
	Source         string // "Giant Centipede bite"
}

func NewInstance(aff *Affliction, source string) *AfflictionInstance {
	return &AfflictionInstance{
		Affliction:     aff,
		CurrentStage:   1, 
		TimeToOnset:    aff.OnsetDelay,
		TimeToNextSave: aff.Interval,
		HasOnsetted:    false,
		Source:         source,
	}
}

// IsCured returns true if stage reached 0
func (i *AfflictionInstance) IsCured() bool {
	return i.CurrentStage <= 0
}

// IsActive returns true if past onset and not cured
func (i *AfflictionInstance) IsActive() bool {
	return i.HasOnsetted && !i.IsCured()
}

// GetCurrentEffects returns damage and conditions for current stage
func (i *AfflictionInstance) GetCurrentEffects() (dice.DieRoll, item.DamageType, []ConditionEffect) {
	stage := i.Affliction.GetStage(i.CurrentStage)
	if stage == nil {
		return dice.DieRoll{}, item.DamageType(""), nil
	}
	return stage.Damage, stage.DamageType, stage.Conditions
}

// ProcessSave advances or reduces stage based on Degree of Success
func (i *AfflictionInstance) ProcessSave(degree check.DegreeOfSuccess) {
	switch degree {
	case check.CriticalSuccess:
		i.CurrentStage -= 2
	case check.Success:
		i.CurrentStage -= 1
	case check.Failure:
		i.CurrentStage += 1
	case check.CriticalFailure:
		i.CurrentStage += 2
	}

	if i.CurrentStage < 0 {
		i.CurrentStage = 0
	}
	if i.CurrentStage > i.Affliction.MaxStage {
		i.CurrentStage = i.Affliction.MaxStage
	}
}

// Tick handles time passing. Returns true if a save is required.
func (i *AfflictionInstance) Tick() bool {
	if i.IsCured() {
		return false
	}

	if !i.HasOnsetted {
		if i.TimeToOnset > 0 {
			i.TimeToOnset--
			return false
		}
		i.HasOnsetted = true
		i.TimeToNextSave = i.Affliction.Interval 
		return true
	}

	i.TimeToNextSave--
	if i.TimeToNextSave <= 0 {
		i.TimeToNextSave = i.Affliction.Interval
		return true
	}

	return false
}

// TickWithRoll is a helper for testing that processes time and a save in one go if needed
func (i *AfflictionInstance) TickWithRoll(naturalRoll int, dc int) bool {
	if i.Tick() {
		res := check.PerformCheckWithRoll(naturalRoll, 0, nil, dc)
		i.ProcessSave(res.Degree)
		return true
	}
	return false
}
