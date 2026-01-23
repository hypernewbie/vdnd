# Phase 1: Core Dice & Check System

## Agent Prompt

You are implementing the core dice rolling and check resolution system for a Pathfinder 2E rules engine in Go. This is the foundation that all other game mechanics will build upon.

**Your task:** Implement the `pkg/dice` and `pkg/check` packages with full test coverage.

---

## Context

PF2E uses a d20-based check system where:
1. Roll a d20
2. Add modifiers (ability scores, proficiency, bonuses, penalties)
3. Compare total to a Difficulty Class (DC)
4. Determine degree of success (critical failure, failure, success, critical success)

Key rules (src: `rules/rules/core-rulebook/chapter-9-playing-the-game.md`):
- **Bonus stacking (line 112):** Only the highest bonus of each type applies. Types are: circumstance, item, status, untyped.
- **Penalty stacking:** Typed penalties work like bonuses (only worst applies). Untyped penalties ALL stack.
- **Degrees of success (line 145):** 
  - Critical Success: Beat DC by 10+ OR natural 20 upgrades success → crit
  - Success: Meet or beat DC
  - Failure: Below DC
  - Critical Failure: Fail by 10+ OR natural 1 downgrades failure → crit fail
- **Natural 1/20 (line 152):** Nat 20 improves degree by 1, nat 1 worsens degree by 1. Applied AFTER calculating ±10 threshold.

---

## File Structure

```
pkg/
├── dice/
│   ├── dice.go        # DieRoll struct, Roll function
│   └── dice_test.go
└── check/
    ├── modifier.go    # Modifier, BonusType, stacking logic
    ├── check.go       # CheckResult, DegreeOfSuccess, PerformCheck
    ├── modifier_test.go
    └── check_test.go
```

---

## Implementation Plan

### 1. `pkg/dice/dice.go`

```go
type DieRoll struct {
    Count    int  // Number of dice (e.g., 2 in "2d6")
    Sides    int  // Die type (e.g., 6 in "2d6")
    Modifier int  // Flat bonus (e.g., 4 in "2d6+4")
}

// Roll evaluates the dice expression and returns the total.
// Uses crypto/rand or math/rand based on your preference.
func (d DieRoll) Roll() int

// RollWithRNG allows injecting a random source for testing.
func (d DieRoll) RollWithRNG(rng *rand.Rand) int

// Parse converts a string like "2d6+4" into a DieRoll.
// Required for CLI input.
func Parse(expr string) (DieRoll, error)
```

**Pseudocode for Roll:**
```
total = 0
for i in range(count):
    total += random(1, sides)
return total + modifier
```

### 2. `pkg/check/modifier.go`

```go
type BonusType int
const (
    BonusUntyped BonusType = iota
    BonusCircumstance
    BonusItem
    BonusStatus
)

type Modifier struct {
    Value  int
    Type   BonusType
    Source string  // For display/debugging: "Heroism", "Cover", etc.
}

// CalculateTotal applies PF2E stacking rules to a slice of modifiers.
// Returns the net modifier to add to a d20 roll.
func CalculateTotal(modifiers []Modifier) int
```

**Stacking Rules Pseudocode:**
```
bonuses = {circumstance: 0, item: 0, status: 0}
penalties = {circumstance: 0, item: 0, status: 0}
untyped_penalty_total = 0

for mod in modifiers:
    if mod.value > 0:  # bonus
        if mod.type == untyped:
            # untyped bonuses: take highest (same as typed)
            bonuses[untyped] = max(bonuses[untyped], mod.value)
        else:
            bonuses[mod.type] = max(bonuses[mod.type], mod.value)
    else:  # penalty (negative value)
        if mod.type == untyped:
            untyped_penalty_total += mod.value  # ALL untyped penalties stack
        else:
            penalties[mod.type] = min(penalties[mod.type], mod.value)

return sum(bonuses.values()) + sum(penalties.values()) + untyped_penalty_total
```

### 3. `pkg/check/check.go`

```go
type DegreeOfSuccess int
const (
    CriticalFailure DegreeOfSuccess = iota
    Failure
    Success
    CriticalSuccess
)

type CheckResult struct {
    NaturalRoll int
    Modifiers   int              // Total from CalculateTotal
    Total       int              // NaturalRoll + Modifiers
    DC          int
    Degree      DegreeOfSuccess
}

// PerformCheck rolls a d20, applies modifiers, and determines success.
func PerformCheck(baseModifier int, modifiers []Modifier, dc int) CheckResult

// PerformCheckWithRoll allows injecting the d20 result for testing.
func PerformCheckWithRoll(naturalRoll int, baseModifier int, modifiers []Modifier, dc int) CheckResult

// DetermineDegree calculates the degree of success given the numbers.
// Handles nat 1/20 adjustments.
func DetermineDegree(naturalRoll, total, dc int) DegreeOfSuccess
```

**DetermineDegree Pseudocode:**
```
# Step 1: Calculate base degree from numbers only
if total >= dc + 10:
    degree = CriticalSuccess
elif total >= dc:
    degree = Success
elif total <= dc - 10:
    degree = CriticalFailure
else:
    degree = Failure

# Step 2: Apply natural 1/20 adjustment
if naturalRoll == 20:
    degree = min(degree + 1, CriticalSuccess)  # Improve by 1
elif naturalRoll == 1:
    degree = max(degree - 1, CriticalFailure)  # Worsen by 1

return degree
```

---

## Test Plan

### `pkg/dice/dice_test.go`

| Test Case | Input | Expected |
|-----------|-------|----------|
| Single die | `DieRoll{1, 6, 0}` with RNG returning 4 | 4 |
| Multiple dice | `DieRoll{3, 6, 0}` with RNG returning 2,3,4 | 9 |
| With modifier | `DieRoll{1, 20, 5}` with RNG returning 10 | 15 |
| Negative modifier | `DieRoll{1, 20, -2}` with RNG returning 10 | 8 |
| Zero dice | `DieRoll{0, 6, 5}` | 5 (just the modifier) |

**Statistical test (optional):** Roll 1d6 10,000 times, verify distribution is roughly uniform across 1-6.

### `pkg/check/modifier_test.go`

| Test Case | Modifiers | Expected Total |
|-----------|-----------|----------------|
| Empty list | `[]` | 0 |
| Single bonus | `[{+2, Status, "Heroism"}]` | +2 |
| Same type bonuses | `[{+2, Status, "Heroism"}, {+1, Status, "Bless"}]` | +2 (highest only) |
| Different type bonuses | `[{+2, Status, "Heroism"}, {+2, Item, "Sword"}]` | +4 (both apply) |
| All types | `[{+1, Circumstance}, {+2, Item}, {+1, Status}]` | +4 |
| Single penalty | `[{-2, Status, "Sickened"}]` | -2 |
| Same type penalties | `[{-2, Status, "Sickened"}, {-1, Status, "Frightened"}]` | -2 (worst only) |
| Mixed bonus/penalty same type | `[{+2, Status, "Heroism"}, {-1, Status, "Frightened"}]` | +1 |
| Untyped penalties stack | `[{-5, Untyped, "MAP"}, {-2, Untyped, "Range"}]` | -7 |
| Complex mix | `[{+2, Status}, {+1, Circumstance}, {-5, Untyped}, {-2, Untyped}, {-1, Status}]` | +2 + 1 - 1 - 5 - 2 = -5 |

### `pkg/check/check_test.go`

| Test Case | Natural Roll | Base Mod | Extra Mods | DC | Expected Degree |
|-----------|--------------|----------|------------|-----|-----------------|
| Simple success | 15 | +5 | [] | 15 | Success |
| Simple failure | 10 | +3 | [] | 15 | Failure |
| Crit success by +10 | 12 | +8 | [] | 10 | CriticalSuccess |
| Crit failure by -10 | 3 | -2 | [] | 15 | CriticalFailure |
| Nat 20 upgrades success | 20 | +0 | [] | 25 | Success (was failure, +1) |
| Nat 20 upgrades crit | 20 | +5 | [] | 15 | CriticalSuccess |
| Nat 1 downgrades | 1 | +10 | [] | 10 | Failure (was success, -1) |
| Nat 1 can't go below crit fail | 1 | -5 | [] | 20 | CriticalFailure |
| Nat 20 can crit even vs high DC | 20 | +0 | [] | 35 | Failure (20 + 0 = 20, miss by 15 = fail, nat20 → success) |
| Exactly meet DC | 15 | +0 | [] | 15 | Success |
| One below DC | 14 | +0 | [] | 15 | Failure |
| Beat DC by exactly 10 | 10 | +10 | [] | 10 | CriticalSuccess |
| Fail DC by exactly 10 | 10 | -5 | [] | 15 | CriticalFailure (total 5, DC 15, diff = -10) |

### Edge Cases for Nat 1/20

| Scenario | Natural | Total | DC | Base Degree | After Nat Adj |
|----------|---------|-------|-----|-------------|---------------|
| Nat 20, would crit anyway | 20 | 35 | 20 | CriticalSuccess | CriticalSuccess (no change, already max) |
| Nat 20, would fail by 10+ | 20 | 22 | 40 | CriticalFailure | Failure (+1) |
| Nat 1, would succeed by 10+ | 1 | 25 | 15 | CriticalSuccess | Success (-1) |
| Nat 1, would fail | 1 | 8 | 15 | Failure | CriticalFailure (-1) |

---

## Validation Checklist

- [ ] All tests pass
- [ ] `go vet` reports no issues
- [ ] `go fmt` applied
- [ ] No panics on edge cases (empty slices, zero values)
- [ ] Source references in comments match rule citations

---

## Notes for Implementation

1. **Randomness:** Use `math/rand` seeded with time for production. Provide `WithRNG` variants for deterministic tests.

2. **Integer arithmetic:** PF2E uses integers only. No floats needed.

3. **Degree clamping:** When adjusting for nat 1/20, clamp to valid range [CriticalFailure, CriticalSuccess].

4. **Modifier sources:** The `Source` field is for debugging/display. It doesn't affect calculations.

5. **Future-proofing:** The check system will be extended later for:
   - Fortune/misfortune effects (roll twice, take better/worse)
   - Hero points (reroll)
   - Assurance (take 10)
   
   Keep the core simple but consider how these might fit.
