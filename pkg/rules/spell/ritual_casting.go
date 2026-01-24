package spell

import (
	"fmt"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/skill"
)

// RitualCastAttempt tracks the casting process and results
type RitualCastAttempt struct {
	Ritual           *Ritual
	PrimaryCaster    *entity.Entity
	SecondaryCasters []*entity.Entity

	// Results
	PrimaryResult    check.CheckResult
	SecondaryResults []SecondaryCheckResult
	FinalDegree      check.DegreeOfSuccess

	// State
	MaterialsConsumed int
	IsComplete        bool
}

type SecondaryCheckResult struct {
	Caster      *entity.Entity
	Skill       ability.SkillID
	Result      check.CheckResult
	Contributed bool
}

// NewRitualCastAttempt starts a new ritual casting
func NewRitualCastAttempt(ritual *Ritual, primary *entity.Entity, secondaries []*entity.Entity) (*RitualCastAttempt, error) {
	// Validate primary caster has required proficiency
	primaryProf := ability.Untrained
	if p, ok := primary.SkillProficiencies[ritual.PrimaryCheck]; ok {
		primaryProf = p
	}
	if primaryProf < ritual.RequiredRank {
		return nil, fmt.Errorf("primary caster requires %v in %s", ritual.RequiredRank, ritual.PrimaryCheck)
	}

	// Validate enough secondary casters
	if len(secondaries) < ritual.SecondaryCasters {
		return nil, fmt.Errorf("ritual requires %d secondary casters, got %d", ritual.SecondaryCasters, len(secondaries))
	}

	return &RitualCastAttempt{
		Ritual:           ritual,
		PrimaryCaster:    primary,
		SecondaryCasters: secondaries,
		SecondaryResults: make([]SecondaryCheckResult, 0),
	}, nil
}

// CastRitual performs all the checks and determines outcome
func CastRitual(attempt *RitualCastAttempt, primaryRoll int, secondaryRolls []int) RitualOutcome {
	ritual := attempt.Ritual

	// Determine DC
	dc := ritual.PrimaryDC
	if dc == 0 {
		dc = skill.LevelBasedDC(int(ritual.Rank) * 2) // Approximate level from rank
	}

	// Primary check
	if primaryRoll > 0 {
		attempt.PrimaryResult = skill.PerformSkillCheckWithRoll(
			attempt.PrimaryCaster, ritual.PrimaryCheck, dc, primaryRoll)
	} else {
		attempt.PrimaryResult = skill.PerformSkillCheck(
			attempt.PrimaryCaster, ritual.PrimaryCheck, dc)
	}

	// Secondary checks
	for i, secondary := range attempt.SecondaryCasters {
		if i >= len(ritual.SecondaryChecks) {
			break // No more requirements
		}

		req := ritual.SecondaryChecks[i]
		roll := 0
		if i < len(secondaryRolls) {
			roll = secondaryRolls[i]
		}

		var res check.CheckResult
		if roll > 0 {
			res = skill.PerformSkillCheckWithRoll(secondary, req.Skill, dc, roll)
		} else {
			res = skill.PerformSkillCheck(secondary, req.Skill, dc)
		}

		attempt.SecondaryResults = append(attempt.SecondaryResults, SecondaryCheckResult{
			Caster:      secondary,
			Skill:       req.Skill,
			Result:      res,
			Contributed: res.Degree >= check.Success,
		})
	}

	// Calculate final degree with secondary modifiers
	attempt.FinalDegree = calculateFinalDegree(attempt)
	attempt.MaterialsConsumed = ritual.CostCP
	attempt.IsComplete = true

	// Apply effect
	var outcome RitualOutcome
	if ritual.Effect != nil {
		outcome = ritual.Effect.Apply(attempt, attempt.PrimaryCaster, nil)
	} else {
		outcome = RitualOutcome{
			Success:     attempt.FinalDegree >= check.Success,
			Description: fmt.Sprintf("Ritual completed with %v", attempt.FinalDegree),
		}
	}

	if outcome.RefundMaterials {
		attempt.MaterialsConsumed = 0
	}

	return outcome
}

// calculateFinalDegree adjusts primary result based on secondary casters
// src: rules/rules/core-rulebook/chapter-7-spells.md (Rituals section)
func calculateFinalDegree(attempt *RitualCastAttempt) check.DegreeOfSuccess {
	degree := attempt.PrimaryResult.Degree

	// Each secondary crit success = +1 step
	// Each secondary crit failure = -1 step
	modifier := 0
	for _, sec := range attempt.SecondaryResults {
		switch sec.Result.Degree {
		case check.CriticalSuccess:
			modifier++
		case check.CriticalFailure:
			modifier--
		}
	}

	// Apply modifier
	finalDegree := int(degree) + modifier
	if finalDegree > int(check.CriticalSuccess) {
		finalDegree = int(check.CriticalSuccess)
	}
	if finalDegree < int(check.CriticalFailure) {
		finalDegree = int(check.CriticalFailure)
	}

	return check.DegreeOfSuccess(finalDegree)
}
