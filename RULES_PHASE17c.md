# Phase 17c: Code Review & Hardening Proposal

## Review Summary
**Status**: :white_check_mark: **PASSED** (Compiles, Runs, All 30 Scenarios present).
**Quality**: :warning: **MIXED**.
- **Strong Tests**: Scenarios 1-17, 30 use real Engine objects (`NewStrike`, `NewTurn`, `skill.Trip`, `Conditions`).
- **Weak Tests**: Scenarios 20, 21, 22, 28, 29, 25 use **hardcoded math** inside the test function rather than calling Engine methods. They test the *rules*, not the *code*.

## Proposed Weakness Fixes

### 1. Harden "Action Tax" (Scenario 21)
- **Current**: Defines `actions := 3` and does manual subtraction.
- **Problem**: Does not verify if `NewTurn` actually respects the `Slowed` condition.
- **Proposed Fix**: Use `turn := combat.NewTurn(wizard)` and assert `turn.ActionsRemaining == 1`.
    - *Note*: `NewTurn` in `turn.go` already implements Slowed logic! The test ignores it.

### 2. Harden "Reaction Economy" (Scenario 20)
- **Current**: Manually checking `SpendReaction` on a fresh turn.
- **Problem**: Good, but isolated.
- **Proposed Fix**: Keep as is, this uses the real API.

### 3. Harden "Heightened Spell" (Scenario 22)
- **Current**: `totalDice := baseDamage + heightenedDamage`. Pure math.
- **Problem**: If `spell.Cast` is ever implemented, this test won't catch bugs in it.
- **Proposed Fix**: Create a mock `Spell` object (if exists) and call `Resize/Heighten`. If no Spell object exists in engine yet, mark as "Placeholder".

### 4. Harden "Ghostly Duel" (Scenario 28)
- **Current**: `if roll >= dc`.
- **Problem**: The `StrikeAction.Execute` method (in `strike.go`) does **NOT** perform this Flat Check. The test passes, but the Engine is missing the feature.
- **Proposed Fix**:
    - **Engine Change**: implementations of `StrikeAction.Execute` should call `check.PerformFlatCheck(11)` if target is Undetected.
    - **Test Change**: Call `strike.Execute` and verify it aborts/misses on flat check failure.

### 5. Harden "The Bunker" (Scenario 29)
- **Current**: Manually creates a modifier `Type: BonusCircumstance, Value: 2`.
- **Problem**: Verifies `CalculateTotal` works (basic math), but doesn't test if "Cover" actually *generates* that modifier on a Reflex save.
- **Proposed Fix**: Use `entity.GetReflex(modifiers...)` context if available, or accept as unit test for Modifier math.

## Conclusion
The file allows us to move forward, but Scenarios 21 and 28 specifically expose that **Engine Logic is either ignored (21) or missing (28)**.

**Recommendation**:
1.  **Accept** the current file as "Phase 1 completion" (Math Validation).
2.  **Create Phase 19 task**: "Engine Hardening" - Update `turn.go` and `strike.go` to match the logic validated in these tests.
