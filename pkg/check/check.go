package check

import (
	"math/rand"
	"time"
)

type DegreeOfSuccess int

const (
	CriticalFailure DegreeOfSuccess = iota
	Failure
	Success
	CriticalSuccess
)

type CheckResult struct {
	NaturalRoll int
	Modifiers   int // Total from CalculateTotal
	Total       int // NaturalRoll + Modifiers
	DC          int
	Degree      DegreeOfSuccess
}

// PerformCheck rolls a d20, applies modifiers, and determines success.
func PerformCheck(baseModifier int, modifiers []Modifier, dc int) CheckResult {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	naturalRoll := rng.Intn(20) + 1
	return PerformCheckWithRoll(naturalRoll, baseModifier, modifiers, dc)
}

// PerformCheckWithRoll allows injecting the d20 result for testing.
func PerformCheckWithRoll(naturalRoll int, baseModifier int, modifiers []Modifier, dc int) CheckResult {
	totalModifiers := baseModifier + CalculateTotal(modifiers)
	total := naturalRoll + totalModifiers
	degree := DetermineDegree(naturalRoll, total, dc)

	return CheckResult{
		NaturalRoll: naturalRoll,
		Modifiers:   totalModifiers,
		Total:       total,
		DC:          dc,
		Degree:      degree,
	}
}

// DetermineDegree calculates the degree of success given the numbers.
// Handles nat 1/20 adjustments.
func DetermineDegree(naturalRoll, total, dc int) DegreeOfSuccess {
	var degree DegreeOfSuccess

	// Step 1: Calculate base degree from numbers only
	if total >= dc+10 {
		degree = CriticalSuccess
	} else if total >= dc {
		degree = Success
	} else if total <= dc-10 {
		degree = CriticalFailure
	} else {
		degree = Failure
	}

	// Step 2: Apply natural 1/20 adjustment
	if naturalRoll == 20 {
		if degree < CriticalSuccess {
			degree++
		}
	} else if naturalRoll == 1 {
		if degree > CriticalFailure {
			degree--
		}
	}

	return degree
}
