package affliction

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/item"
)

type AfflictionType int

const (
	TypePoison AfflictionType = iota
	TypeDisease
	TypeCurse
)

type ConditionEffect struct {
	ID    condition.ConditionID
	Value int // For valued conditions
}

type Stage struct {
	Number     int
	Damage     dice.DieRoll
	DamageType item.DamageType
	Conditions []ConditionEffect
	IsFatal    bool // If true, reaching this stage causes death
}

type Affliction struct {
	ID           string
	Name         string
	Type         AfflictionType
	DC           int
	Save         ability.SaveType
	OnsetDelay   int // 0 = immediate
	OnsetUnit    ability.IntervalUnit
	MaxStage     int
	Stages       []Stage // Indexed by stage number-1 (Stage 1 is index 0)
	Interval     int
	IntervalUnit ability.IntervalUnit
}

func (a *Affliction) GetStage(num int) *Stage {
	if num <= 0 || num > len(a.Stages) {
		return nil
	}
	return &a.Stages[num-1]
}
