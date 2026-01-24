# Phase 21: Rituals

## Objective

Implement the Ritual casting system. Rituals differ from standard spells: they take hours/days to cast, require multiple casters, have monetary costs, and use skill checks rather than spell slots.

---

## 1. Ritual Struct

**Target File:** `pkg/rules/spell/ritual.go`

(Standard Ritual, CastingDuration, SecondaryCheckRequirement structs)

---

## 2. Ritual Effect & Outcome

**Target File:** `pkg/rules/spell/ritual.go`

```go
// RitualOutcome represents the result of casting a ritual
type RitualOutcome struct {
    Success         bool
    RefundMaterials bool      // If true, MaterialsConsumed is set to 0
    Description     string
    Backlash        string    // Effect on critical failure
    TargetEffect    string    // What happens to the target
}

// GenericRitualEffect provides a simple effect implementation
type GenericRitualEffect struct {
    SuccessDesc     string
    CritSuccessDesc string
    FailureDesc     string
    CritFailureDesc string
    BacklashDesc    string
    RefundOnFailure bool      // If true, failures refund materials
}

func (e *GenericRitualEffect) Apply(attempt *RitualCastAttempt, caster *entity.Entity, targets []*entity.Entity) RitualOutcome {
    outcome := RitualOutcome{}

    switch attempt.FinalDegree {
    case check.CriticalSuccess:
        outcome.Success = true
        outcome.Description = e.CritSuccessDesc
    case check.Success:
        outcome.Success = true
        outcome.Description = e.SuccessDesc
    case check.Failure:
        outcome.Success = false
        outcome.Description = e.FailureDesc
        if e.RefundOnFailure {
            outcome.RefundMaterials = true
        }
    case check.CriticalFailure:
        outcome.Success = false
        outcome.Description = e.CritFailureDesc
        outcome.Backlash = e.BacklashDesc
    }

    return outcome
}
```

---

## 3. Ritual Casting Logic

**Target File:** `pkg/rules/spell/ritual_casting.go`

```go
// CastRitual performs all the checks and determines outcome
func CastRitual(attempt *RitualCastAttempt, primaryRoll int, secondaryRolls []int) RitualOutcome {
    // ... (checks performed) ...

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

    // Handle material refund
    if outcome.RefundMaterials {
        attempt.MaterialsConsumed = 0
    }

    return outcome
}
```

---

## 4. Standard Rituals Registry

**Target File:** `pkg/rules/spell/ritual_registry.go`

Updated with `RefundOnFailure`:
- **Resurrect:** `RefundOnFailure: true`
- **Commune:** `RefundOnFailure: true`
- **Atone:** `RefundOnFailure: true`
- **Create Undead:** `RefundOnFailure: true`
- **Plane Shift:** `RefundOnFailure: false`

---

## 5. Tests

Added `TestRitualMaterialRefund` to verify that `RefundOnFailure: true` rituals don't consume materials on a standard failure, but DO consume them on a critical failure.