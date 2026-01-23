# Phase 2: Ability Scores & Proficiency

## Agent Prompt

You are implementing the ability score and proficiency system for a Pathfinder 2E rules engine in Go. This builds on Phase 1 (dice & checks) and provides the foundation for calculating attack rolls, saves, skill checks, and AC.

**Your task:** Implement the `pkg/ability` package with full test coverage.

**Prerequisite:** Phase 1 must be complete (`pkg/dice`, `pkg/check`).

---

## Context

PF2E characters have six ability scores that represent their raw capabilities. Each ability score generates a **modifier** used in checks. Characters also have **proficiency ranks** in various skills, saves, and attacks that determine their bonus.

### Ability Scores (src: `rules/rules/core-rulebook/chapter-1-introduction.md:107`)

| Score | Description |
|-------|-------------|
| **Strength (STR)** | Physical power. Melee damage, Athletics. |
| **Dexterity (DEX)** | Agility. AC, Reflex saves, ranged attacks, Stealth. |
| **Constitution (CON)** | Health. HP, Fortitude saves. |
| **Intelligence (INT)** | Knowledge. Skills known, Arcana, Crafting. |
| **Wisdom (WIS)** | Awareness. Perception, Will saves, Medicine. |
| **Charisma (CHA)** | Personality. Diplomacy, Intimidation, spell DCs for some classes. |

### Ability Modifier Formula (src: `chapter-1-introduction.md:111`)
```
modifier = (score - 10) / 2   (round down, i.e., integer division)
```

| Score | Modifier |
|-------|----------|
| 1 | -5 |
| 6-7 | -2 |
| 8-9 | -1 |
| 10-11 | 0 |
| 12-13 | +1 |
| 14-15 | +2 |
| 18-19 | +4 |
| 20-21 | +5 |

### Proficiency Ranks (src: `chapter-9-playing-the-game.md:104`)

| Rank | Bonus |
|------|-------|
| Untrained | +0 |
| Trained | level + 2 |
| Expert | level + 4 |
| Master | level + 6 |
| Legendary | level + 8 |

### Calculating a Check Modifier
```
total modifier = ability modifier + proficiency bonus + item bonus + other bonuses/penalties
```

### Calculating a DC
```
DC = 10 + total modifier
```
Used for things like "your Perception DC" or "your Fortitude DC".

---

## File Structure

```
pkg/
└── rules/
    └── ability/
        ├── ability.go       # AbilityScores, Ability enum, modifier calculation
        ├── proficiency.go   # ProficiencyRank, bonus calculation
        └── ability_test.go  # All tests
```

---

## Implementation Plan

### 1. `pkg/rules/ability/ability.go`

```go
// Ability represents one of the six ability scores
type Ability int
const (
    Strength Ability = iota
    Dexterity
    Constitution
    Intelligence
    Wisdom
    Charisma
)

// AbilityScores holds all six scores for an entity
type AbilityScores struct {
    Strength     int
    Dexterity    int
    Constitution int
    Intelligence int
    Wisdom       int
    Charisma     int
}

// Get returns the score for a given ability
func (a AbilityScores) Get(ability Ability) int

// Modifier returns the modifier for a given ability
// Formula: (score - 10) / 2, using integer division (floor)
func (a AbilityScores) Modifier(ability Ability) int

// ModifierFromScore calculates modifier from a raw score
// Useful as a standalone helper
func ModifierFromScore(score int) int
```

**Pseudocode:**
```
func ModifierFromScore(score int) int:
    return (score - 10) / 2   // Go integer division floors toward zero
                              // But we need floor toward negative infinity!
                              // (score - 10) / 2 works for positive
                              // For negative: need adjustment
```

**Important:** Go's integer division truncates toward zero, not toward negative infinity.
- `(9 - 10) / 2` = `-1 / 2` = `0` in Go (wrong, should be -1)
- Fix: `(score - 10 - 1) / 2 + some_adjustment` OR use explicit floor logic

**Correct implementation:**
```go
func ModifierFromScore(score int) int {
    diff := score - 10
    if diff >= 0 {
        return diff / 2
    }
    // For negative: -1 and -2 should both give -1, -3 and -4 give -2, etc.
    return (diff - 1) / 2
}
```

Or simpler with math:
```go
func ModifierFromScore(score int) int {
    return int(math.Floor(float64(score-10) / 2.0))
}
```

### 2. `pkg/rules/ability/proficiency.go`

```go
type ProficiencyRank int
const (
    Untrained ProficiencyRank = iota  // 0
    Trained                            // 1
    Expert                             // 2
    Master                             // 3
    Legendary                          // 4
)

// Bonus calculates the proficiency bonus for a given rank and level
func (r ProficiencyRank) Bonus(level int) int

// String returns a human-readable name
func (r ProficiencyRank) String() string
```

**Pseudocode:**
```
func (r ProficiencyRank) Bonus(level int) int:
    if r == Untrained:
        return 0
    base := level + 2
    extra := (int(r) - 1) * 2   // Trained=0 extra, Expert=2, Master=4, Legendary=6
    return base + extra
```

Alternatively, lookup table:
```
rank_bonus = [0, 2, 4, 6, 8]  // indexed by ProficiencyRank
if rank == Untrained: return 0
return level + rank_bonus[rank]
```

### 3. Helper: Calculate Skill/Save Modifier

This is a convenience function that combines ability + proficiency. It might live in `ability.go` or a separate file.

```go
// CalculateModifier returns the total modifier for a check
// given an ability score, proficiency rank, and character level.
// Does NOT include item/circumstance/status bonuses.
func CalculateModifier(abilityScore int, rank ProficiencyRank, level int) int {
    return ModifierFromScore(abilityScore) + rank.Bonus(level)
}

// CalculateDC returns 10 + the modifier (used for save DCs, Perception DC, etc.)
func CalculateDC(modifier int) int {
    return 10 + modifier
}
```

---

## Test Plan

### `pkg/rules/ability/ability_test.go`

#### Modifier Calculation Tests

| Score | Expected Modifier |
|-------|-------------------|
| 1 | -5 |
| 2 | -4 |
| 3 | -4 |
| 4 | -3 |
| 5 | -3 |
| 6 | -2 |
| 7 | -2 |
| 8 | -1 |
| 9 | -1 |
| 10 | 0 |
| 11 | 0 |
| 12 | +1 |
| 13 | +1 |
| 14 | +2 |
| 15 | +2 |
| 16 | +3 |
| 17 | +3 |
| 18 | +4 |
| 19 | +4 |
| 20 | +5 |
| 21 | +5 |
| 22 | +6 |

**Table-driven test recommended!**

#### AbilityScores.Get() Tests

| Input | Ability | Expected |
|-------|---------|----------|
| `{10, 14, 12, 8, 16, 18}` | Strength | 10 |
| `{10, 14, 12, 8, 16, 18}` | Dexterity | 14 |
| `{10, 14, 12, 8, 16, 18}` | Charisma | 18 |

#### AbilityScores.Modifier() Tests

| Scores | Ability | Expected |
|--------|---------|----------|
| `{10, 14, 12, 8, 16, 18}` | Strength | 0 |
| `{10, 14, 12, 8, 16, 18}` | Dexterity | +2 |
| `{10, 14, 12, 8, 16, 18}` | Intelligence | -1 |
| `{10, 14, 12, 8, 16, 18}` | Wisdom | +3 |
| `{10, 14, 12, 8, 16, 18}` | Charisma | +4 |

### Proficiency Bonus Tests

| Rank | Level | Expected Bonus |
|------|-------|----------------|
| Untrained | 1 | 0 |
| Untrained | 10 | 0 |
| Untrained | 20 | 0 |
| Trained | -1 | 1 |
| Trained | 1 | 3 |
| Trained | 5 | 7 |
| Trained | 10 | 12 |
| Trained | 20 | 22 |
| Expert | 1 | 5 |
| Expert | 5 | 9 |
| Expert | 10 | 14 |
| Master | 1 | 7 |
| Master | 10 | 16 |
| Legendary | 1 | 9 |
| Legendary | 10 | 18 |
| Legendary | 20 | 28 |

### CalculateModifier Tests

| Ability Score | Rank | Level | Expected |
|---------------|------|-------|----------|
| 10 | Untrained | 1 | 0 |
| 14 | Trained | 1 | 2 + 3 = 5 |
| 18 | Expert | 5 | 4 + 9 = 13 |
| 8 | Master | 10 | -1 + 16 = 15 |
| 20 | Legendary | 20 | 5 + 28 = 33 |

### CalculateDC Tests

| Modifier | Expected DC |
|----------|-------------|
| 0 | 10 |
| 5 | 15 |
| 15 | 25 |
| -2 | 8 |
| 33 | 43 |

---

## Validation Checklist

- [ ] All tests pass
- [ ] `go vet` reports no issues
- [ ] `go fmt` applied
- [ ] Negative ability scores handled correctly (floor division)
- [ ] Edge case: level 0 (if applicable)
- [ ] String methods for enums work correctly

---

## Notes for Implementation

1. **Integer division gotcha:** Go truncates toward zero, not toward -∞. For scores below 10, you need special handling. Test with score=9 (should be -1, not 0).

2. **Level 0:** Some creatures might be level 0. Ensure proficiency bonus calculates correctly (`0 + 2 = 2` for Trained).

3. **AbilityScores as value type:** Keep it a simple struct, easily copied. No pointers needed.

4. **Integration with Phase 1:** This package doesn't directly depend on `pkg/check`, but will be used alongside it. For example:
   ```go
   // Future usage
   totalMod := ability.CalculateModifier(char.Abilities.Get(ability.Strength), char.AthleticsProficiency, char.Level)
   result := check.PerformCheck(totalMod, bonuses, dc)
   ```

5. **Enum String methods:** Useful for CLI output:
   ```go
   func (a Ability) String() string {
       return [...]string{"Strength", "Dexterity", "Constitution", "Intelligence", "Wisdom", "Charisma"}[a]
   }
   ```
