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

// FlatCheck performs a roll against a DC with no modifiers.
func FlatCheck(dc int) bool {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	roll := rng.Intn(20) + 1
	return roll >= dc
}

func (d DegreeOfSuccess) String() string {
	switch d {
	case CriticalFailure:
		return "Critical Failure"
	case Failure:
		return "Failure"
	case Success:
		return "Success"
	case CriticalSuccess:
		return "Critical Success"
	default:
		return "Unknown"
	}
}

// Adjust returns a degree of success shifted by the given amount (e.g. +1 or -1).
// It clamps the result to CriticalFailure and CriticalSuccess.
func (d DegreeOfSuccess) Adjust(steps int) DegreeOfSuccess {
	newVal := int(d) + steps
	if newVal < int(CriticalFailure) {
		return CriticalFailure
	}
	if newVal > int(CriticalSuccess) {
		return CriticalSuccess
	}
	return DegreeOfSuccess(newVal)
}

// Counteract determines if an effect is counteracted based on counteract levels and degree of success.
// CRB p.458
func Counteract(sourceLevel, targetLevel int, degree DegreeOfSuccess) bool {
	switch degree {
	case CriticalSuccess:
		// Counteract if target level <= source level + 3
		return targetLevel <= sourceLevel+3
	case Success:
		// Counteract if target level <= source level + 1
		return targetLevel <= sourceLevel+1
	case Failure:
		// Counteract if target level < source level
		// "Counteract the target if its counteract level is lower than your effect's counteract level."
		return targetLevel < sourceLevel
	case CriticalFailure:
		return false
	}
	return false
}
