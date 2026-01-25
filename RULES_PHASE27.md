# Phase 27: Counteract System

## Agent Prompt

You are implementing the Counteract system for a Pathfinder 2E rules engine in Go. Counteracting is a special check used to cancel ongoing magical effects, cure afflictions, or dispel spells. It uses a level-comparison mechanic where your degree of success determines the maximum level of effect you can counteract.

**Your task:** Implement `CounteractCheck` in `pkg/rules/check/counteract.go` with full test coverage.

---

## Context

Counteracting is used for:
- **Dispel Magic:** Cancel a spell effect
- **Remove Disease/Curse:** Cure an affliction
- **Neutralize Poison:** End a poison effect
- **Anti-magic effects:** Suppress magical abilities

Key rules (src: `rules/rules/core-rulebook/chapter-9-playing-the-game.md`, Counteracting section):

The counteract check formula:
1. Roll a d20 + your counteract modifier (usually your spellcasting proficiency + ability mod)
2. Compare to the target's DC (usually the caster's spell DC or an affliction's DC)
3. Based on your degree of success, determine the maximum level you can counteract:
   - **Critical Success:** Counteract level + 3
   - **Success:** Counteract level + 1
   - **Failure:** Counteract level − 1
   - **Critical Failure:** Counteract level − 3

The counteract level is typically:
- For spells: The spell's rank
- For abilities: Half the creature's level (rounded up)

If the target effect's level is ≤ your maximum, the counteract succeeds.

---

## File Structure

```
pkg/
└── rules/
    └── check/
        ├── counteract.go       # CounteractCheck function
        └── counteract_test.go
```

---

## Implementation Plan

### 1. `pkg/rules/check/counteract.go`

```go
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

// CounteractCheckWithRoll allows injecting the d20 result for testing.
func CounteractCheckWithRoll(naturalRoll, counteractLevel, counteractMod, targetLevel, targetDC int) CounteractResult {
    result := PerformCheckWithRoll(naturalRoll, counteractMod, nil, targetDC)

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
```

---

## Test Plan

### `pkg/rules/check/counteract_test.go`

#### Core Counteract Tests

| Test Case | Counteract Lvl | Roll | Mod | DC | Degree | Max Level | Target Lvl | Can Counteract? |
|-----------|----------------|------|-----|----|--------------------|-----------|------------|-----------------|
| Crit success, lower target | 5 | 20 | +15 | 25 | CriticalSuccess | 8 | 5 | ✅ Yes |
| Crit success, exact max | 5 | 20 | +15 | 25 | CriticalSuccess | 8 | 8 | ✅ Yes |
| Crit success, too high | 5 | 20 | +15 | 25 | CriticalSuccess | 8 | 9 | ❌ No |
| Success, lower target | 5 | 15 | +10 | 20 | Success | 6 | 4 | ✅ Yes |
| Success, exact max | 5 | 15 | +10 | 20 | Success | 6 | 6 | ✅ Yes |
| Success, too high | 5 | 15 | +10 | 20 | Success | 6 | 7 | ❌ No |
| Failure, low target | 5 | 8 | +5 | 20 | Failure | 4 | 3 | ✅ Yes |
| Failure, at limit | 5 | 8 | +5 | 20 | Failure | 4 | 4 | ✅ Yes |
| Failure, too high | 5 | 8 | +5 | 20 | Failure | 4 | 5 | ❌ No |
| Crit failure | 5 | 1 | +2 | 25 | CriticalFailure | 2 | 2 | ✅ Yes |
| Crit failure, too high | 5 | 1 | +2 | 25 | CriticalFailure | 2 | 3 | ❌ No |

#### Edge Cases

| Test Case | Counteract Lvl | Degree | Max Level | Notes |
|-----------|----------------|--------|-----------|-------|
| Level 1, crit fail | 1 | CriticalFailure | 0 | 1 - 3 = -2, clamped to 0 |
| Level 2, failure | 2 | Failure | 1 | 2 - 1 = 1 |
| Level 0, crit fail | 0 | CriticalFailure | 0 | 0 - 3 = -3, clamped to 0 |
| Level 10, crit success | 10 | CriticalSuccess | 13 | Can counteract very high level effects |

#### Natural 1/20 Interaction

| Test Case | Roll | Base Degree | Final Degree | Notes |
|-----------|------|-------------|--------------|-------|
| Nat 20 upgrades success | 20 | Success | CriticalSuccess | Max level gets +3 instead of +1 |
| Nat 1 downgrades | 1 | Success | Failure | Max level gets -1 instead of +1 |

---

## Validation Checklist

- [ ] All tests pass
- [ ] `go vet` reports no issues
- [ ] `go fmt` applied
- [ ] MaxLevelAffected is clamped to minimum 0
- [ ] Natural 1/20 rules are respected (via PerformCheck)
- [ ] CanCounteract is true when `targetLevel <= maxLevel`

---

## Notes for Implementation

1. **Counteract modifier:** Usually calculated as `proficiency + ability mod + item bonuses`. The CLI computes this before calling CounteractCheck.

2. **Counteract level:** For spell-based counteracting, this is the spell's rank. For ability-based (e.g., a monster's dispelling ability), it's typically half the creature's level, rounded up.

3. **Target level:** This is the rank of the spell being dispelled, or the level of the affliction/curse.

4. **Level 0 effects:** Cantrips are rank 0. A critical failure at counteract level 1 would give max level = -2 → 0, which can still counteract cantrips.

5. **CLI integration:** The `vd check counteract` command will compute:
   - `counteractLevel` from the spell rank or `(entity.Level + 1) / 2`
   - `counteractMod` from the entity's spellcasting proficiency
   - The LLM provides `targetLevel` and `targetDC`
