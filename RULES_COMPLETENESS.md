# Implementation Plan: Completing the Rules Engine

**Role**: You are a Senior Go Engineer.
**Task**: methodically implement the missing game components identified in the completeness survey.

## Instructions
1.  **Iterative Approach**: Do not try to do everything in one massive file write. Tackle one section at a time.
2.  **Test-Driven**: For *every* new mechanic or logic piece, write a test in the corresponding `_test.go` file.
    - Example: If you add `Tumble Through` action, add a test case in `skill_test.go` checking the DC and Success/Failure results.
3.  **Strict Typing**: Use Enums/Constants where possible (e.g., for Languages, Alignments), not raw strings.

---

## 1. Missing Data Structures (Enums & Registries)
**Location**: `pkg/rules/entity/`, `pkg/rules/trait/`

*   [ ] **Languages**: Create `Language` string enum (Common, Draconic, etc.). Add `Languages []Language` to Entity struct.
*   [ ] **Alignment**: Create `Alignment` enum (LG, NE, etc.). Add `Alignment` field to Entity.
*   [ ] **Rarity**: Add `Rarity` trait category and default traits (Common/Uncommon/Rare/Unique).
*   [ ] **Traditions & Schools**: Add `MagicTradition` (Arcane...) and `MagicSchool` (Evocation...) traits.
*   [ ] **Light Levels**: Define constants for Bright, Dim, Darkness.

## 2. Missing Conditions
**Location**: `pkg/rules/condition/`

Implement the logic for these missing conditions. Ensure they integrate with the new `Relational` system if applicable (indicated by *).

*   [ ] **Object States**: `Broken` (affects items).
*   [ ] **Visibility* (Relational)**: `Concealed`, `Dazzled`, `Observed`, `Undetected`, `Unnoticed`.
*   [ ] **Encumbrance**: `Encumbered` (affects Speed).
*   [ ] **Status**: `Petrified`.
*   [ ] **Attitudes* (Relational)**: `Friendly`, `Helpful`, `Hostile`, `Indifferent`, `Unfriendly`.

## 3. Missing Skill Actions
**Location**: `pkg/rules/skill/actions.go` (or new files like `athletics.go` if it gets too big).

Implement the checks for these actions.
*   **Acrobatics**: `Balance` (Reflex vs DC), `Tumble Through` (Acrobatics vs Reflex DC), `Maneuver in Flight`, `Squeeze`.
*   **Athletics**: `Climb`, `Swim`, `High/Long Jump` (DC calculation logic), `Disarm` (Attack vs Reflex), `Force Open`.
*   **Deception**: `Create a Diversion`, `Feint` (Bluff vs Perception), `Impersonate`, `Lie`.
*   **Diplomacy**: `Gather Information`, `Make an Impression`, `Request`.
*   **Intimidation**: `Coerce`.
*   **Medicine**: `Administer First Aid` (Stabilize/Stop Bleeding), `Treat Poison/Disease`.
*   **Stealth**: `Sneak` (Movement + Stealth vs Perception), `Conceal Object`.
*   **Thievery**: `Pick Lock` (Thievery vs DC), `Disable Device`, `Palm Object`, `Steal`.

## 4. Complex Logic Implementation
These items require more than just data entry.

### A. Advanced Weapon Traits
**Location**: `pkg/rules/combat/`
*   [ ] **Sweep**: +1 bonus if attacking a *different* target this turn.
    *   *Logic*: Check `Turn.StrikesMade`. If `WeaponID` matches AND `TargetID` is different from current target, apply +1.
*   [ ] **Forceful**: Bonus damage on 2nd/3rd attack with same weapon.
    *   *Logic*: Check `Turn.StrikesMade`. Count previous strikes with same `WeaponID`.
*   [ ] **Twin**: Bonus damage if wielding two of same weapon.
*   [ ] **Backswing/Shove/Trip**: Implement weapon property flags.

### B. Persistent Damage
**Location**: `pkg/rules/damage/`, `pkg/rules/condition/tracker.go`
*   [ ] Allow multiple `PersistentDamage` instances of *different* types (Fire + Acid).
*   [ ] If adding same type (Fire 5 + Fire 2), keep only the higher value.
*   [ ] Implement `EndTurn` logic: Apply damage -> Roll DC 15 Flat Check -> Remove if success.

### C. Afflictions (Poisons/Diseases)
**Location**: `pkg/rules/affliction/`
*   [ ] Implement `Stage` logic.
*   [ ] `Tick` function: triggers saving throw.
    *   Success: Stage -1 (or -2 crit).
    *   Failure: Stage +1 (or +2 crit).
    *   End: If Stage 0, removed. If > MaxStage, apply max stage effect + keep duration? (Check rules).

---

## 5. Mandatory Test Coverage

A simple "it runs" is not enough. You must implement the following tests for each new component:

### A. Skill Action Test Matrix (Exhaustive Degrees)
For *every* skill action (e.g., `Tumble Through`, `Trip`, `Feint`), you must test all 4 Degrees of Success:
- [ ] **Critical Success**: Verify double effects/extra bonuses (e.g., Trip deals 1d6 damage, Feint lasts longer).
- [ ] **Success**: Verify standard effect (e.g., target is Prone/Flat-footed).
- [ ] **Failure**: Verify no effect + check for "failure" specific penalties if any.
- [ ] **Critical Failure**: Verify "backfire" effects (e.g., Tripper falls prone, Shove backfires).

### B. Condition Logic & Stacking
- [ ] **Valued Conditions**: Test that `Frightened 2` correctly reduces to `Frightened 1` at end of turn.
- [ ] **Relational States**: Verify observer-specific visibility.
    - Rogue is `Hidden` (Relative: Guard A), `Observed` (Relative: Guard B).
    - Rogue attacks Guard A -> `Hidden` removed relative to A.
- [ ] **Persistent Damage Stacking**:
    - Apply `Fire 5` and `Fire 10` -> Resolve to `Fire 10` only.
    - Apply `Fire 5` and `Acid 5` -> Resolve to BOTH `Fire 5` AND `Acid 5`.
    - Apply `EndTurn` logic -> Verify damage is dealt BEFORE the flat check to remove.

### C. Advanced Combat Logic
- [ ] **MAP Integration**: Verify skill actions with the `Attack` trait correctly increment the character's MAP and *suffer* from current MAP.
- [ ] **Trait History**:
    - **Sweep**: Strike T1, then Strike T2 -> +1 bonus. Strike T1, then Strike T1 again -> 0 bonus.
    - **Forceful**: Strike 1 (base), Strike 2 (+damage), Strike 3 (+double damage).
- [ ] **Reaction Tracking**: Verify reactions are properly exhausted and cannot be used twice in one round.

### D. Affliction Lifecycle
- [ ] **Stage Progression**:
    - Start at Stage 1. Fail save -> Move to Stage 2.
    - Crit Pass save -> Move to Stage 0 (Removed).
- [ ] **Latency/Tick**: Verify intervals (Round vs Day) correctly trigger checks in a mock time system.

## 6. Verification
Run `go test -v ./pkg/rules/...` and ensure 100% of the newly added logic paths are hit. If a branch (like Critical Failure backfire) isn't tested, the implementation is incomplete.

