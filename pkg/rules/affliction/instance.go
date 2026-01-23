package affliction

import (
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/item"
)

type AfflictionInstance struct {
	Affliction     *Affliction
	CurrentStage   int
	TimeToOnset    int // Countdown
	TimeToNextSave int
	Source         string // "Giant Centipede bite"
}

func NewInstance(aff *Affliction, source string) *AfflictionInstance {
	return &AfflictionInstance{
		Affliction:     aff,
		CurrentStage:   1, // Starts at stage 1 unless stated otherwise
		TimeToOnset:    aff.OnsetDelay,
		TimeToNextSave: aff.Interval,
		Source:         source,
	}
}

// IsCured returns true if stage reached 0
func (i *AfflictionInstance) IsCured() bool {
	return i.CurrentStage <= 0
}

// IsActive returns true if past onset and not cured
func (i *AfflictionInstance) IsActive() bool {
	return i.TimeToOnset <= 0 && !i.IsCured()
}

// GetCurrentEffects returns damage and conditions for current stage
func (i *AfflictionInstance) GetCurrentEffects() (dice.DieRoll, item.DamageType, []ConditionEffect) {
	stage := i.Affliction.GetStage(i.CurrentStage)
	if stage == nil {
		return dice.DieRoll{}, item.DamageType(""), nil
	}
	return stage.Damage, stage.DamageType, stage.Conditions
}
