package check

// CounteractResult contains the outcome of a counteract attempt.
type CounteractResult struct {
	CheckResult
	MaxLevelAffected int  // Maximum level of effect that can be counteracted
	CanCounteract    bool // Whether the counteract was successful against the target
}

// CounteractCheck performs a counteract check.
//
// Parameters:
//   - counteractLevel: Your counteract level (spell rank or half caster level) [0+]
//   - counteractMod: Your counteract modifier (proficiency + ability + bonuses)
//   - targetLevel: The level of the effect being counteracted [0+]
//   - targetDC: The DC to beat (caster's DC or affliction DC)
//
// Returns a CounteractResult with the degree of success and whether the
// counteract succeeded.
//
// src: rules/rules/core-rulebook/chapter-9-playing-the-game.md (Counteracting)
func CounteractCheck(counteractLevel, counteractMod, targetLevel, targetDC int) CounteractResult {
	// Input validation (optional defensive programming)
	if counteractLevel < 0 {
		counteractLevel = 0
	}
	if targetLevel < 0 {
		targetLevel = 0
	}

	result := PerformCheck(counteractMod, nil, targetDC)
	return calculateCounteractResult(result, counteractLevel, targetLevel)
}

// CounteractCheckWithRoll allows injecting the d20 result for testing.
func CounteractCheckWithRoll(naturalRoll, counteractLevel, counteractMod, targetLevel, targetDC int) CounteractResult {
	// Input validation (optional defensive programming)
	if counteractLevel < 0 {
		counteractLevel = 0
	}
	if targetLevel < 0 {
		targetLevel = 0
	}

	result := PerformCheckWithRoll(naturalRoll, counteractMod, nil, targetDC)
	return calculateCounteractResult(result, counteractLevel, targetLevel)
}

// calculateCounteractResult handles the shared logic for degree-based level calculation.
func calculateCounteractResult(result CheckResult, counteractLevel, targetLevel int) CounteractResult {
	maxLevel := 0

	switch result.Degree {
	case CriticalSuccess:
		maxLevel = counteractLevel + 3
	case Success:
		maxLevel = counteractLevel + 1
	case Failure:
		maxLevel = counteractLevel - 1
	case CriticalFailure:
		maxLevel = counteractLevel - 3
	}

	// Clamping: MaxLevelAffected is clamped to a minimum of 0.
	if maxLevel < 0 {
		maxLevel = 0
	}

	return CounteractResult{
		CheckResult:      result,
		MaxLevelAffected: maxLevel,
		CanCounteract:    targetLevel <= maxLevel && result.Degree > CriticalFailure,
	}
}
