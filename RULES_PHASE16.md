# Phase 16: Exhaustive Skill, Condition, and Combat Logic

## Objective
Achieve 100% mechanical coverage of the Skill Actions, Condition Logic, and Advanced Combat Traits identified in previous surveys. This phase is not about "adding placeholders" but about **verifying exact mechanical outcomes** via strict test matrices.

## 1. Skill Action Implementation & Testing
**File**: `pkg/rules/skill/actions.go` | **Tests**: `pkg/rules/skill/skill_test.go`

For EACH action below, you must implement the logic and write a test case that covers **Critical Success**, **Success**, **Failure**, and **Critical Failure**.

### Acrobatics
*   [ ] **Balance**: 
    *   *Crit Fail*: Triggers `Prone` condition on actor.
*   [ ] **Tumble Through**:
    *   *Success*: Move through enemy space.
    *   *Failure*: Movement ends.
*   [ ] **Maneuver in Flight**
*   [ ] **Squeeze**

### Athletics
*   [ ] **Climb**:
    *   *Crit Success*: 8ft/turn speed.
    *   *Crit Fail*: Fall prone + Damage.
*   [ ] **Swim**
*   [ ] **High Jump / Long Jump**: Implement distance formula (e.g., Leap + Check Result feet).
*   [ ] **Disarm**:
    *   *Success*: +2 circumstance bonus to further attempts.
    *   *Crit Fail*: Actor becomes flat-footed.
*   [ ] **Force Open**

### Deception
*   [ ] **Create a Diversion**:
    *   *Success*: Become `Hidden` (relational).
*   [ ] **Feint**:
    *   *Crit Success*: Target flat-footed to *all* melee attacks (next turn).
    *   *Success*: Target flat-footed to *next* melee attack.
    *   *Crit Fail*: Actor flat-footed to target.
*   [ ] **Impersonate**
*   [ ] **Lie**

### Diplomacy
*   [ ] **Gather Information**
*   [ ] **Make an Impression**
*   [ ] **Request**

### Intimidation
*   [ ] **Coerce**
*   [ ] **Demoralize** (Verify immunity for 10 mins after attempt).

### Medicine
*   [ ] **treat Wounds**:
    *   *Crit Success*: 4d8 healing.
    *   *Crit Fail*: Dealing 1d8 damage.
*   [ ] **Administer First Aid**: Stabilize dying creature.
*   [ ] **Treat Poison/Disease**: Grant bonus to next save.

### Stealth
*   [ ] **Sneak**: Update `Hidden` -> `Undetected` or fail to `Observed`.
*   [ ] **Conceal Object**

### Thievery
*   [ ] **Pick Lock**: Track successes needed (complex lock).
*   [ ] **Disable Device**
*   [ ] **Palm Object**
*   [ ] **Steal**

## 2. Condition Logic Deep-Dive
**File**: `pkg/rules/condition/tracker.go` | **Tests**: `pkg/rules/condition/condition_test.go`

*   [ ] **Persistent Damage Stacking**:
    *   Verify `Fire 10` replaces `Fire 5`.
    *   Verify `Acid 5` coexists with `Fire 5`.
    *   Verify `EndTurn` applies damage -> rolls DC 15 check -> removes on success.
*   [ ] **Valued Condition Decay**:
    *   `Frightened 2` -> `Frightened 1` -> Removed at end of turns.
    *   `Drained` (Permanent until rest).

## 3. Advanced Combat Traits
**File**: `pkg/rules/combat/strike.go` | **Tests**: `pkg/rules/combat/combat_test.go`

*   [ ] **Sweep**: 
    *   Refactor `TurnState` to track `TargetID` of all attacks.
    *   Test: Attack A (hit/miss) -> Attack B (+1 bonus).
    *   Test: Attack A -> Attack A (no bonus).
*   [ ] **Forceful**:
    *   Test: Attack A (base) -> Attack A (+damage) -> Attack A (+double damage).
*   [ ] **Agile**:
    *   Test: MAP is -4/-8 instead of -5/-10.
*   [ ] **Backswing**: Implement "Miss" sets "Next Attack Bonus".

## 4. Afflictions (Poisons/Diseases)
**File**: `pkg/rules/affliction/*`

*   [ ] **Stage State Machine**:
    *   Implement `Tick(roll int)`:
    *   Success: Stage -1.
    *   Crit Success: Stage -2.
    *   Failure: Stage +1.
    *   Crit Failure: Stage +2.
    *   End Condition: If Stage 0 or > MaxStage.

## Deliverables
1.  **Code**: Complete implementation of all above functions.
2.  **Tests**: `go test -v ./pkg/rules/...` must pass with high coverage.
3.  **Verification**: Manual walk-through of a "Combat Simulation" in a test file (Rogue Feints, Sneaks, Strikes with Agile weapon).
