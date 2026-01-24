package subsystem

import (
	"fmt"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/entity"
)

// InfluenceTarget represents an NPC to influence
type InfluenceTarget struct {
	Entity      *entity.Entity
	CurrentVP   int
	InfluenceVP int               // VP needed to influence
	Resistances []ability.SkillID // Skills they resist
	Weaknesses  []ability.SkillID // Skills that work well
	Discovery   InfluenceDiscovery
}

// InfluenceDiscovery tracks what PCs know about the target
type InfluenceDiscovery struct {
	Discovered       bool
	ResistancesKnown bool
	WeaknessesKnown  bool
}

// Influence tracks a social influence encounter
type Influence struct {
	*Subsystem
	Targets        []InfluenceTarget
	RoundsPerPhase int // Social rounds before checking progress
}

// NewInfluence creates an influence encounter
// src: rules/rules/gamemastery-guide/chapter-3-subsystems.md (Influence)
func NewInfluence(id, name string, rounds int) *Influence {
	return &Influence{
		Subsystem:      NewSubsystem(id, name, SubsystemInfluence, 0, -99, rounds),
		Targets:        make([]InfluenceTarget, 0),
		RoundsPerPhase: 3,
	}
}

// AddTarget adds an NPC target
func (inf *Influence) AddTarget(e *entity.Entity, vpNeeded int, resists, weaknesses []ability.SkillID) {
	inf.Targets = append(inf.Targets, InfluenceTarget{
		Entity:      e,
		InfluenceVP: vpNeeded,
		Resistances: resists,
		Weaknesses:  weaknesses,
	})
	inf.TargetVP += vpNeeded // Total VP is sum of all targets
}

// Discover attempts to learn about a target
func (inf *Influence) Discover(actor *entity.Entity, targetID string, skillID ability.SkillID, naturalRoll int) DiscoverResult {
	target := inf.getTarget(targetID)
	if target == nil {
		return DiscoverResult{Description: "Target not found"}
	}

	dc := target.Entity.GetSaveDC(ability.SaveWill)
	res := inf.ContributeWithCheck(actor, skillID, dc, 0, 0, naturalRoll)

	result := DiscoverResult{
		CheckResult: res,
	}

	switch res.Degree {
	case check.CriticalSuccess:
		target.Discovery.Discovered = true
		target.Discovery.ResistancesKnown = true
		target.Discovery.WeaknessesKnown = true
		result.Learned = "Learned all resistances and weaknesses"
	case check.Success:
		target.Discovery.Discovered = true
		// Learn one or the other
		if len(target.Weaknesses) > 0 {
			target.Discovery.WeaknessesKnown = true
			result.Learned = "Learned weaknesses"
		} else {
			target.Discovery.ResistancesKnown = true
			result.Learned = "Learned resistances"
		}
	case check.Failure:
		result.Description = "Failed to learn anything useful"
	case check.CriticalFailure:
		result.Description = "Target is now suspicious"
	}

	return result
}

type DiscoverResult struct {
	CheckResult ContributionResult
	Learned     string
	Description string
}

// InfluenceTarget attempts to sway a target
func (inf *Influence) InfluenceTarget(actor *entity.Entity, targetID string, skillID ability.SkillID, naturalRoll int) InfluenceResult {
	target := inf.getTarget(targetID)
	if target == nil {
		return InfluenceResult{Description: "Target not found"}
	}

	dc := target.Entity.GetSaveDC(ability.SaveWill)

	// Apply modifiers for weaknesses/resistances
	modifier := 0
	for _, r := range target.Resistances {
		if r == skillID {
			modifier -= 2 // Harder if they resist this approach
			break
		}
	}
	for _, w := range target.Weaknesses {
		if w == skillID {
			modifier += 2 // Easier if this is their weakness
			break
		}
	}

	effectiveDC := dc - modifier // Lower DC = easier
	res := inf.ContributeWithCheck(actor, skillID, effectiveDC, 1, 2, naturalRoll)

	target.CurrentVP += res.VPEarned

	result := InfluenceResult{
		CheckResult: res,
		VPEarned:    res.VPEarned,
		TargetVP:    target.CurrentVP,
		TargetMax:   target.InfluenceVP,
	}

	if target.CurrentVP >= target.InfluenceVP {
		result.Influenced = true
		result.Description = fmt.Sprintf("%s has been influenced!", target.Entity.Name)
	}

	return result
}

type InfluenceResult struct {
	CheckResult ContributionResult
	VPEarned    int
	TargetVP    int
	TargetMax   int
	Influenced  bool
	Description string
}

func (inf *Influence) getTarget(entityID string) *InfluenceTarget {
	for i := range inf.Targets {
		if inf.Targets[i].Entity.ID == entityID {
			return &inf.Targets[i]
		}
	}
	return nil
}

// GetInfluencedCount returns how many targets have been influenced
func (inf *Influence) GetInfluencedCount() int {
	count := 0
	for _, t := range inf.Targets {
		if t.CurrentVP >= t.InfluenceVP {
			count++
		}
	}
	return count
}
