# Phase 17d: Engine Logic Implementation

## Context
Phase 17c passed verification, but revealed that Scenarios 20, 21, 28, and 29 relied on manual calculations instead of exercising the Engine's actual logic. The user has requested we implement this logic immediately.

## Objectives

1.  **Implement Undetected Logic**:
    - Modify `pkg/rules/combat/strike.go`: In `ExecuteWithRoll`, check if `target` is `Hidden` or `Undetected` relative to `actor`.
    - If so, perform a DC 11 Flat Check (use `check.PerformFlatCheck` or simulate with random if needed).
    - If check fails: Return `Miss` immediately without rolling attack.

2.  **Refactor "Ghostly Duel" (Scenario 28)**:
    - Update `pkg/rules/combat/scenarios_test.go`.
    - Instead of checking `if roll >= 11`, call `strike.ExecuteWithRoll(actor, target, turn, 15)`.
    - **CRITICAL**: The `Execute` method might need a way to inject the Flat Check roll for deterministic testing.
    - **Strategy**: Add `FlatCheckRoll` (int) field to `StrikeAction` struct (default 0 = random) OR pass it in `ExecuteWithRoll`?
    - **Better Strategy**: Since `ExecuteWithRoll` is for testing, maybe add a variadic `extraRolls ...int`? Or simpler: Just rely on likelihood in the test? No, tests must be deterministic.
    - **Selected Approach**: Add `FlatCheckResult` int to `StrikeAction` (optional, for testing). If set > 0, use it.

3.  **Refactor "Action Tax" (Scenario 21)**:
    - Update `pkg/rules/combat/scenarios_test.go`.
    - Call `turn := combat.NewTurn(wizard)`.
    - Assert `turn.ActionsRemaining == 1`. (This verifies `NewTurn` logic works).

4.  **Refactor "The Bunker" (Scenario 29)**:
    - If feasible, create `entity.GetReflex` helper in test file if strictly needed, but prioritize the above two as they are core combat mechanics.

## Implementation Plan

1.  **Modify `pkg/rules/combat/strike.go`**:
    - Add `TestFlatCheckRoll int` to `StrikeAction` struct (or similar mechanism).
    - Implement `check.FlatCheck` call using this value.
    - Logic: `if target.Conditions.HasRelative(Hidden, actor.ID) || ...HasRelative(Undetected...) { result := Check(DC 11); if !result { return Miss } }`.

2.  **Update `scenarios_test.go`**:
    - Update Scenario 21 and 28 as described.

3.  **Run Tests**: Verify all 30 scenarios pass with the real engine code.
