# Phase 17: Rules Engine Logic Polish & Verification

## Objective
Finalize the mechanical implementation of Skill Actions and Afflictions. A recent review identified precise gaps between the "Plan" and the "Code". Most logic exists, but side effects (movement, lock picking progress) are not yet returned or tested effectively.

## 1. Skill Action Signatures & Logic
**Target File**: `pkg/rules/skill/actions.go`

Refactor the following functions to return mechanical values useful for the engine, rather than just the Check Result.

*   [ ] **Disarm**:
    *   *Current*: Adds `disarm-bonus` as a TemporaryImmunity.
    *   *Required*: Apply a `Condition` named `DisarmWeakness` (or similar) to the target. This ensures the engine can easily query `target.Conditions.Has("DisarmWeakness")` to apply penalties/bonuses later.
*   [ ] **Squeeze**:
    *   *Required*: Return `(int, check.CheckResult)`.
    *   *Logic*: Critical Success = 10ft, Success = 5ft.
*   [ ] **PickLock**:
    *   *Required*: Return `(int, check.CheckResult)`.
    *   *Logic*: Critical Success = 2 successes, Success = 1 success. This is crucial for Complex Locks.
*   [ ] **ManeuverInFlight**:
    *   *Required*: Ensure it returns meaningful state or just confirm the distance logic implies "movement allowed". (If just a check, ensure the test verifies the check matches the DC).

## 2. Affliction Interface
**Target File**: `pkg/rules/affliction/instance.go` or `tracker.go`

The test suite (`affliction_test.go`) expects a method `TickWithRoll` which combines the time-passing tick and the saving throw. The current implementation splits them (correctly) but the test helper is missing.

*   [ ] **Add `TickWithRoll` Method**:
    *   Implement a helper method on `AfflictionInstance` (or generic wrapper) that runs the logic: `Roll Save -> Apply Stage Change -> Update Timers`.
    *   Ensure strict adherence to the Stage State Machine:
        *   Crit Success: Stage -2
        *   Success: Stage -1
        *   Failure: Stage +1
        *   Crit Failure: Stage +2
    *   *Note*: The separate `Tick()` (time passes) and `ProcessSave()` methods are good, but `TickWithRoll` is needed to satisfy the "Exhaustive Test" requirement without writing boilerplate in every test.

## 3. Test Assertion hardening
**Target File**: `pkg/rules/skill/skill_test.go`

The current tests loop through degrees of success but often fail to assert the *side effects*.

*   [ ] **Add State Assertions**:
    *   **Balance**: Assert `actor.Conditions.Has(condition.Prone)` on Critical Failure.
    *   **Climb**: Assert returned speed is `8` on Crit Success.
    *   **Demoralize**: Assert `target.Conditions.Value(condition.Frightened) == 2` on Crit Success.
    *   **Feint**: Assert `target.Conditions.HasRelative(condition.FlatFooted, actor.ID)` on Success.
    *   **Disarm**: Assert the new `DisarmWeakness` condition exists on Success.

## 4. Execution
1.  Modify `pkg/rules/skill/actions.go` with the signature changes.
2.  Modify/Add `TickWithRoll` in `pkg/rules/affliction`.
3.  Update `pkg/rules/skill/skill_test.go` to match new signatures and **fail** if side effects are missing.
4.  Run `go test -v ./pkg/rules/...` and ensure 100% pass rate.
