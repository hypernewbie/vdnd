package hazard

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/trait"
)

type HazardType int

const (
	HazardTrap HazardType = iota
	HazardHaunt
	HazardEnvironmental
)

type Complexity int

const (
	ComplexitySimple Complexity = iota
	ComplexityComplex
)

type DisableOption struct {
	Skill       ability.SkillID
	DC          int
	Description string
}

type Hazard struct {
	ID         string
	Name       string
	Level      int
	Type       HazardType
	Complexity Complexity
	Traits     trait.TraitSet

	// Detection
	StealthDC int // DC to notice with Perception

	// Defenses (for hazards that can be attacked)
	AC         int
	Fortitude  int
	Reflex     int
	Will       int
	HP         int
	Hardness   int // Damage reduction
	Immunities []string

	// Disabling
	DisableOptions []DisableOption

	// Trigger
	Trigger TriggerCondition

	// Effects
	Effect HazardEffect

	// Complex hazard initiative
	// TODO: ENCOUNTER INTEGRATION - Checks for Complex hazards must roll initiative using this modifier
	// and add the hazard to the turn order.
	Initiative int // Modifier for complex hazards

	// State
	IsTriggered bool
	IsDisabled  bool
	CurrentHP   int
	Position    string

	// Routine for complex hazards
	Routine *HazardRoutine
}

func NewHazard(id, name string, level int) *Hazard {
	return &Hazard{
		ID:             id,
		Name:           name,
		Level:          level,
		Traits:         trait.TraitSet{},
		DisableOptions: make([]DisableOption, 0),
		CurrentHP:      0, // Will be set if HP > 0
	}
}

// Detect attempts to notice the hazard with Perception
func (h *Hazard) Detect(observer *entity.Entity) bool {
	// Secret check: Perception vs Stealth DC
	perceptionMod := observer.GetPerception()
	res := check.PerformCheck(perceptionMod, nil, h.StealthDC)
	return res.Degree >= check.Success
}

// CanDisable checks if entity can attempt to disable
func (h *Hazard) CanDisable(actor *entity.Entity, method DisableOption) bool {
	// Typically requires training in the skill
	prof := ability.Untrained
	if p, ok := actor.SkillProficiencies[method.Skill]; ok {
		prof = p
	}
	return prof >= ability.Trained
}

// AttemptDisable tries to disable the hazard
func (h *Hazard) AttemptDisable(actor *entity.Entity, method DisableOption) check.CheckResult {
	skillMod := actor.GetSkillModifier(method.Skill)
	res := check.PerformCheck(skillMod, nil, method.DC)

	if res.Degree >= check.Success {
		h.IsDisabled = true
	}
	return res
}

// CheckTrigger determines if a hazard triggers
func (h *Hazard) CheckTrigger(event ability.Event) bool {
	if h.IsDisabled || h.IsTriggered {
		// Simple hazards only trigger once usually
		if h.Complexity == ComplexitySimple && h.IsTriggered {
			return false
		}
	}

	if h.Trigger.Matches(event, h.Position) {
		h.IsTriggered = true
		return true
	}
	return false
}

// Activate fires the hazard effect
func (h *Hazard) Activate(targets []*entity.Entity) []HazardResult {
	if h.Effect == nil {
		return nil
	}
	return h.Effect.Apply(h, targets)
}

// TakeDamage for attackable hazards
func (h *Hazard) TakeDamage(amount int, damageType string) int {
	if h.HP <= 0 {
		return 0
	}

	// Apply hardness
	actualDamage := amount - h.Hardness
	if actualDamage < 0 {
		actualDamage = 0
	}

	h.CurrentHP -= actualDamage
	if h.CurrentHP <= 0 {
		h.CurrentHP = 0
		h.IsDisabled = true // Destroyed
	}

	return actualDamage
}
