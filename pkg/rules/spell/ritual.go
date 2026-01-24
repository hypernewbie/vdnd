package spell

import (
	"fmt"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/trait"
)

// RitualRank is the spell rank of the ritual (1-10)
type RitualRank int

// CastingDuration represents how long the ritual takes
type CastingDuration struct {
	Value int
	Unit  DurationUnit
}

type DurationUnit int

const (
	DurationMinutes DurationUnit = iota
	DurationHours
	DurationDays
)

func (d CastingDuration) String() string {
	switch d.Unit {
	case DurationMinutes:
		return fmt.Sprintf("%d minutes", d.Value)
	case DurationHours:
		return fmt.Sprintf("%d hours", d.Value)
	case DurationDays:
		return fmt.Sprintf("%d days", d.Value)
	}
	return fmt.Sprintf("%d units", d.Value)
}

// SecondaryCheckRequirement defines what secondary casters need
type SecondaryCheckRequirement struct {
	Skill          ability.SkillID
	MinProficiency ability.ProficiencyRank
	Description    string
}

// Ritual represents a ritual spell
// src: rules/compendium/spells/rituals/
type Ritual struct {
	ID               string
	Name             string
	Rank             RitualRank
	Traits           trait.TraitSet
	CastingTime      CastingDuration
	CostCP           int // Material cost in copper
	SecondaryCasters int // Minimum required (can be 0)

	// Check requirements
	PrimaryCheck    ability.SkillID
	PrimaryDC       int // 0 = use level-based DC
	SecondaryChecks []SecondaryCheckRequirement

	// Requirements
	RequiredRank ability.ProficiencyRank // Minimum proficiency in primary check

	// Effect
	Effect           RitualEffect
	HeightenedCostCP int // Additional cost per rank above base
}

func NewRitual(id, name string, rank RitualRank, primarySkill ability.SkillID, secondaryCasters int) *Ritual {
	return &Ritual{
		ID:               id,
		Name:             name,
		Rank:             rank,
		PrimaryCheck:     primarySkill,
		SecondaryCasters: secondaryCasters,
		SecondaryChecks:  make([]SecondaryCheckRequirement, 0),
		Traits:           make(trait.TraitSet, 0),
	}
}

// WithCastingTime sets the casting duration
func (r *Ritual) WithCastingTime(value int, unit DurationUnit) *Ritual {
	r.CastingTime = CastingDuration{Value: value, Unit: unit}
	return r
}

// WithCost sets the material cost
func (r *Ritual) WithCost(costCP int) *Ritual {
	r.CostCP = costCP
	return r
}

// WithSecondaryCheck adds a secondary check requirement
func (r *Ritual) WithSecondaryCheck(skill ability.SkillID, minProf ability.ProficiencyRank, desc string) *Ritual {
	r.SecondaryChecks = append(r.SecondaryChecks, SecondaryCheckRequirement{
		Skill:          skill,
		MinProficiency: minProf,
		Description:    desc,
	})
	return r
}

// RitualOutcome represents the result of casting a ritual
type RitualOutcome struct {
	Success         bool
	RefundMaterials bool
	Description     string
	Backlash        string // Effect on critical failure
	TargetEffect    string // What happens to the target
}

// RitualEffect defines what the ritual does
type RitualEffect interface {
	// Apply processes the ritual's effect based on the casting result
	Apply(attempt *RitualCastAttempt, caster *entity.Entity, targets []*entity.Entity) RitualOutcome
}

// GenericRitualEffect provides a simple effect implementation
type GenericRitualEffect struct {
	SuccessDesc      string
	CritSuccessDesc  string
	FailureDesc      string
	CritFailureDesc  string
	BacklashDesc     string
	RefundOnFailure  bool
}

func (e *GenericRitualEffect) Apply(attempt *RitualCastAttempt, caster *entity.Entity, targets []*entity.Entity) RitualOutcome {
	outcome := RitualOutcome{}

	switch attempt.FinalDegree {
	case check.CriticalSuccess:
		outcome.Success = true
		outcome.Description = e.CritSuccessDesc
	case check.Success:
		outcome.Success = true
		outcome.Description = e.SuccessDesc
	case check.Failure:
		outcome.Success = false
		outcome.Description = e.FailureDesc
		if e.RefundOnFailure {
			outcome.RefundMaterials = true
		}
	case check.CriticalFailure:
		outcome.Success = false
		outcome.Description = e.CritFailureDesc
		outcome.Backlash = e.BacklashDesc
	}

	return outcome
}
