# Phase 15: Relational Conditions & Perception Refactor

## Objective
Refactor the Condition system (`pkg/rules/condition`) to support **Relational Conditions**. This is the foundational prerequisite for implementing proper Stealth, Hiding, and Detection rules (Hidden, Undetected, Observed).

Currently, a condition like `Hidden` is binary on the actor suitable for a global state (e.g., "Invisible"). However, in Pathfinder 2e, an entity can be **Hidden to Goblin A** but **Observed by Goblin B**.

## Task Description

### 1. Refactor `ConditionInstance`
Modify `pkg/rules/condition/instance.go`:
- Add a field `RelativeData` (map or struct) or specific fields to track whom the condition applies to.
- *Suggestion*: Add `PerceivedBy []string` (list of Entity IDs) or `SpecificTo []string`.
- If `SpecificTo` is empty, the condition is Global.
- If `SpecificTo` has IDs, the condition *only* applies when interacting with those IDs.

### 2. Update `ConditionTracker`
Modify `pkg/rules/condition/tracker.go`:
- Update `Has(id)` to likely return true if *any* instance exists (global or relative), OR split the API.
- Add `HasRelative(id ConditionID, observerID string) bool`:
    - Returns true if a Global instance exists.
    - Returns true if a Relative instance exists where `SpecificTo` contains `observerID`.
- Add `ApplyRelative(c ConditionInstance, observerID string)` helper.

### 3. Perception Matrix (Optional but Recommended)
Consider if `Entity` needs a helper to manage these quickly.
- `func (e *Entity) IsHiddenFrom(observer *Entity) bool`

### 4. Tests
Create `pkg/rules/condition/relational_test.go`:
- Test Case: "Rogue hides behind a pillar."
    - Rogue gains `Hidden` (Relative: Guard A).
    - Rogue does NOT have `Hidden` (Relative: Guard B).
- Verify `HasRelative(Hidden, "Guard A")` == true.
- Verify `HasRelative(Hidden, "Guard B")` == false.
- Verify Global `Invisible` implies `Hidden` to all.

## Technical Context
- **Files**: `pkg/rules/condition/*`
- **Dependencies**: `pkg/rules/entity` (for IDs, though conditions usually use string IDs to avoid import cycles).

## Deliverables
1. Updated `ConditionInstance` struct.
2. Updated `ConditionTracker` methods.
3. Unit tests proving relational logic works.
