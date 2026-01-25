# Phase 28: Odds & Ends

## Agent Prompt

You are implementing the final "cleanup" mechanics for a Pathfinder 2E rules engine in Go. These are small but important features that complete the tactical engine: Hero Points, Recall Knowledge, and Treat Wounds.

**Your task:** Implement Hero Points on Entity, Recall Knowledge skill action, and Treat Wounds skill action with full test coverage.

---

## Context

### Hero Points
Hero Points are a meta-currency for player characters (not NPCs/monsters):
- PCs typically start each session with 1 Hero Point
- Can earn more through heroic actions (LLM awards these)
- Capped at 3 Hero Points maximum
- Two uses:
  1. **Reroll:** Spend 1 to reroll any check (must use the new result)
  2. **Stabilise:** When dying, spend ALL remaining Hero Points to immediately stabilise at 0 HP

### Recall Knowledge
A 1-action skill check to identify a creature, object, or concept:
- The appropriate skill depends on the subject (Nature for beasts, Arcana for constructs, etc.)
- The LLM determines which skill and the DC
- Success = learn one useful fact; Critical Success = additional/specific info
- Critical Failure = receive false information (LLM should mislead)

### Treat Wounds
A 10-minute exploration activity using Medicine:
- Requires Trained in Medicine
- DC 15 (standard), DC 20 (expert), DC 30 (master), DC 40 (legendary)
- Heals 2d8 on success, 4d8 on critical success
- Higher DCs grant bonus healing (+10 at DC 30, +30 at DC 40)
- Critical failure deals 1d8 damage
- Target gains immunity for 1 hour (prevents repeated healing spam)

Key sources:
- Hero Points: `rules/rules/core-rulebook/chapter-1-introduction.md`
- Recall Knowledge: `rules/compendium/skills.md`
- Treat Wounds: `rules/compendium/skills.md`

---

## File Structure

```
pkg/
└── rules/
    ├── entity/
    │   ├── entity.go           # Add HeroPoints field
    │   ├── hero_points.go      # Hero Point methods
    │   └── hero_points_test.go
    ├── skill/
    │   ├── actions.go          # Add RecallKnowledge, TreatWounds
    │   └── actions_test.go
    └── condition/
        └── conditions.go       # Add TreatWoundsImmunity
```

---

## Implementation Plan

### 1. Hero Points on Entity

**Target:** `pkg/rules/entity/entity.go`

```go
type Entity struct {
    // ... existing fields ...

    // Hero Points (for PCs only, NPCs typically have 0)
    HeroPoints int
}
```

**Target:** `pkg/rules/entity/hero_points.go`

```go
package entity

import (
    "errors"

    "vdnd/pkg/rules/condition"
)

const MaxHeroPoints = 3

// GainHeroPoint adds a hero point, capped at 3.
func (e *Entity) GainHeroPoint() {
    e.HeroPoints++
    if e.HeroPoints > MaxHeroPoints {
        e.HeroPoints = MaxHeroPoints
    }
}

// SpendHeroPoint removes one hero point.
// Returns an error if no hero points are available.
func (e *Entity) SpendHeroPoint() error {
    if e.HeroPoints <= 0 {
        return errors.New("no hero points available")
    }
    e.HeroPoints--
    return nil
}

// HeroPointStabilise spends ALL hero points to stabilise when dying.
// Returns false if no hero points to spend or not dying.
func (e *Entity) HeroPointStabilise() bool {
    if e.HeroPoints == 0 {
        return false
    }
    if !e.Conditions.Has(condition.Dying) {
        return false
    }

    e.HeroPoints = 0
    e.Conditions.Remove(condition.Dying)
    e.HP = 0 // Stabilise at exactly 0 HP
    return true
}

// CanUseHeroPoints returns true if the entity has hero points.
func (e *Entity) CanUseHeroPoints() bool {
    return e.HeroPoints > 0
}
```

### 2. Add TreatWoundsImmunity Condition

**Target:** `pkg/rules/condition/conditions.go`

```go
const TreatWoundsImmunity ConditionID = "treat_wounds_immunity"

// TreatWoundsImmunity prevents being treated again for 1 hour.
// The duration is tracked by the LLM/session, not the engine.
```

### 3. Recall Knowledge Action

**Target:** `pkg/rules/skill/actions.go`

```go
// RecallKnowledge attempts to identify a creature, object, or concept.
//
// The LLM determines:
//   - Which skill to use (Arcana, Nature, Occultism, Religion, Society, etc.)
//   - The DC based on the subject's level/rarity
//
// Returns:
//   - learned: true if any information was gained (Success or CritSuccess)
//   - result: the check result for degree of success
//
// On Critical Failure, the LLM should provide false information.
//
// src: rules/compendium/skills.md "Recall Knowledge"
func RecallKnowledge(actor *entity.Entity, skillID ability.SkillID, dc int) (learned bool, result check.CheckResult) {
    result = PerformSkillCheck(actor, skillID, dc)
    learned = result.Degree >= check.Success
    return
}

// RecallKnowledgeWithRoll allows injecting the d20 result for testing.
func RecallKnowledgeWithRoll(actor *entity.Entity, skillID ability.SkillID, dc, naturalRoll int) (learned bool, result check.CheckResult) {
    result = PerformSkillCheckWithRoll(actor, skillID, dc, naturalRoll)
    learned = result.Degree >= check.Success
    return
}

// RecallKnowledgeSkillFor returns the typical skill for identifying a subject.
// This is a helper for the LLM to determine which skill to use.
func RecallKnowledgeSkillFor(subjectType string) ability.SkillID {
    switch subjectType {
    case "aberration", "spirit", "esoterica":
        return ability.SkillOccultism
    case "animal", "beast", "fey", "plant":
        return ability.SkillNature
    case "construct", "dragon", "ooze", "magic":
        return ability.SkillArcana
    case "undead", "celestial", "fiend", "divine":
        return ability.SkillReligion
    case "humanoid", "history", "culture":
        return ability.SkillSociety
    default:
        return ability.SkillSociety // Default to Society
    }
}
```

### 4. Treat Wounds Action

**Target:** `pkg/rules/skill/actions.go`

```go
// TreatWoundsResult contains the outcome of a Treat Wounds attempt.
type TreatWoundsResult struct {
    check.CheckResult
    HealingAmount int  // Positive for healing, negative for damage on crit fail
    Applied       bool // Whether healing/damage was applied to the patient
}

// TreatWounds attempts to heal a patient using Medicine.
//
// This is a 10-minute exploration activity.
// DC determines the difficulty and potential bonus healing:
//   - DC 15: Base healing
//   - DC 20: Base healing (requires Expert)
//   - DC 30: +10 bonus healing (requires Master)
//   - DC 40: +30 bonus healing (requires Legendary)
//
// Outcomes:
//   - Critical Success: 4d8 + bonuses
//   - Success: 2d8 + bonuses
//   - Critical Failure: 1d8 damage to patient
//
// The patient gains TreatWoundsImmunity for 1 hour afterward.
//
// src: rules/compendium/skills.md "Treat Wounds"
func TreatWounds(healer, patient *entity.Entity, dc int, roller dice.Roller) TreatWoundsResult {
    // Must be trained in Medicine
    if healer.SkillProficiencies[ability.SkillMedicine] < ability.Trained {
        return TreatWoundsResult{
            CheckResult: check.CheckResult{Degree: check.Failure},
            Applied:     false,
        }
    }

    // Check for immunity
    if patient.Conditions.Has(condition.TreatWoundsImmunity) {
        return TreatWoundsResult{
            CheckResult: check.CheckResult{Degree: check.Failure},
            Applied:     false,
        }
    }

    result := PerformSkillCheck(healer, ability.SkillMedicine, dc)

    // Calculate bonus based on DC
    bonus := 0
    if dc >= 40 {
        bonus = 30
    } else if dc >= 30 {
        bonus = 10
    }

    var healingAmount int
    switch result.Degree {
    case check.CriticalSuccess:
        healingAmount = rollDice(roller, 4, 8) + bonus
    case check.Success:
        healingAmount = rollDice(roller, 2, 8) + bonus
    case check.CriticalFailure:
        damage := rollDice(roller, 1, 8)
        healingAmount = -damage
    default:
        healingAmount = 0
    }

    // Apply healing or damage
    applied := false
    if healingAmount > 0 {
        patient.Heal(healingAmount)
        applied = true
    } else if healingAmount < 0 {
        patient.TakeDamage(-healingAmount, "slashing")
        applied = true
    }

    // Apply immunity (even on failure, the attempt was made)
    patient.Conditions.Apply(condition.NewCondition(
        condition.TreatWoundsImmunity,
        "Treated by "+healer.ID,
    ))

    return TreatWoundsResult{
        CheckResult:   result,
        HealingAmount: healingAmount,
        Applied:       applied,
    }
}

// Helper for rolling dice
func rollDice(roller dice.Roller, count, sides int) int {
    results := roller.Roll(count, sides)
    total := 0
    for _, r := range results {
        total += r
    }
    return total
}
```

---

## Test Plan

### `pkg/rules/entity/hero_points_test.go`

| Test Case | Setup | Action | Expected |
|-----------|-------|--------|----------|
| Gain hero point | 0 HP | GainHeroPoint() | 1 HP |
| Gain capped at 3 | 3 HP | GainHeroPoint() | 3 HP (no change) |
| Spend hero point | 2 HP | SpendHeroPoint() | 1 HP, no error |
| Spend with none | 0 HP | SpendHeroPoint() | Error: "no hero points" |
| Stabilise dying | 2 HP, Dying 2 | HeroPointStabilise() | true, 0 HP, 0 Hero Points, no Dying |
| Stabilise not dying | 2 HP, no Dying | HeroPointStabilise() | false, 2 HP remains |
| Stabilise no points | 0 HP, Dying 1 | HeroPointStabilise() | false, still Dying |

### `pkg/rules/skill/actions_test.go` (Recall Knowledge)

| Test Case | Roll | DC | Degree | Learned? |
|-----------|------|----|--------------------|----------|
| Success | 15 | 15 | Success | ✅ Yes |
| Critical success | 20 | 15 | CriticalSuccess | ✅ Yes |
| Failure | 10 | 20 | Failure | ❌ No |
| Critical failure | 1 | 20 | CriticalFailure | ❌ No (LLM gives false info) |

### `pkg/rules/skill/actions_test.go` (Treat Wounds)

| Test Case | Roll | DC | Degree | Healing | Notes |
|-----------|------|----|--------------------|---------|-------|
| Success DC 15 | 15 | 15 | Success | 2d8 | Base healing |
| Crit success DC 15 | 20 | 15 | CriticalSuccess | 4d8 | Double dice |
| Success DC 30 | 18 | 30 | Success | 2d8 + 10 | Bonus healing |
| Crit success DC 40 | 25 | 40 | CriticalSuccess | 4d8 + 30 | Max bonus |
| Critical failure | 1 | 15 | CriticalFailure | -1d8 | Damage! |
| Failure | 10 | 20 | Failure | 0 | No effect |
| Immune patient | - | - | - | 0 | Already treated |
| Untrained healer | - | - | Failure | 0 | Requires Trained |

### Immunity Tests

| Test Case | Setup | Action | Expected |
|-----------|-------|--------|----------|
| Immunity applied | TreatWounds success | Check patient | Has TreatWoundsImmunity |
| Immunity blocks | Has immunity | TreatWounds | Failure, Applied: false |
| Immunity from failure | TreatWounds failure | Check patient | Still has immunity |

---

## Validation Checklist

- [ ] All tests pass
- [ ] `go vet` reports no issues
- [ ] `go fmt` applied
- [ ] Hero Points capped at 3
- [ ] Stabilise requires Dying condition
- [ ] Stabilise sets HP to exactly 0
- [ ] Treat Wounds requires Trained Medicine
- [ ] Treat Wounds applies immunity even on failure
- [ ] Critical failure deals damage
- [ ] DC 30+ adds bonus healing

---

## Notes for Implementation

1. **Hero Point rerolls:** The actual reroll happens in the CLI layer. The engine just tracks spending. The CLI will call the check twice and let the player see both results (must use the second).

2. **RecallKnowledge is simple:** It's just a skill check wrapper. The "magic" happens in the LLM interpreting the result and providing appropriate lore.

3. **TreatWoundsImmunity duration:** The engine doesn't track real time. The LLM/session manager is responsible for removing the immunity after 1 hour of game time.

4. **Proficiency requirements:** Treat Wounds at higher DCs (20/30/40) requires Expert/Master/Legendary Medicine. The engine doesn't enforce this—the LLM should only offer appropriate DCs.

5. **NPCs and Hero Points:** NPCs typically have 0 Hero Points. The field exists but won't be used for monsters.
