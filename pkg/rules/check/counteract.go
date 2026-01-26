package check

// CounteractResult extends CheckResult with counteract-specific info.
type CounteractResult struct {
	CheckResult
	CounteractLevel  int  // Your counteract level (input)
	MaxLevelAffected int  // Maximum target level you can counteract
	TargetLevel      int  // The effect's level (input)
	CanCounteract    bool // Whether the counteract succeeded
}

// CounteractCheck performs a counteract check.
//
// Parameters:
//   - counteractLevel: Your counteract level (spell rank or half caster level)
//   - counteractMod: Your counteract modifier (proficiency + ability + bonuses)
//   - targetLevel: The level of the effect being counteracted
//   - targetDC: The DC to beat (caster's DC or affliction DC)
//
// Returns a CounteractResult with the degree of success and whether the
// counteract succeeded.
//
// src: rules/rules/core-rulebook/chapter-9-playing-the-game.md (Counteracting)
func CounteractCheck(counteractLevel, counteractMod, targetLevel, targetDC int) CounteractResult {
	// Perform the underlying d20 check
	result := PerformCheck(counteractMod, nil, targetDC)

	return calculateCounteractResult(result, counteractLevel, targetLevel)
}

// CounteractCheckWithRoll allows injecting the d20 result for testing.
func CounteractCheckWithRoll(naturalRoll, counteractLevel, counteractMod, targetLevel, targetDC int) CounteractResult {
	result := PerformCheckWithRoll(naturalRoll, counteractMod, nil, targetDC)

	return calculateCounteractResult(result, counteractLevel, targetLevel)
}

func calculateCounteractResult(result CheckResult, counteractLevel, targetLevel int) CounteractResult {
	// Determine max level based on degree of success
	var maxLevel int
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

	// Can't go below 0
	if maxLevel < 0 {
		maxLevel = 0
	}

	return CounteractResult{
		CheckResult:      result,
		CounteractLevel:  counteractLevel,
		MaxLevelAffected: maxLevel,
		TargetLevel:      targetLevel,
		CanCounteract:    targetLevel <= maxLevel,
	}
}
