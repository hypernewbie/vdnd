# Phase 25: Glue Actions

## Agent Prompt

You are implementing "glue" actions for a Pathfinder 2E rules engine in Go. These are simple actions that primarily exist for action economy tracking and condition management. They don't involve complex rolls—they just spend actions and flip state.

**Your task:** Implement `Interact`, `DropProne`, `Stand`, and `TakeCover` actions in `pkg/rules/combat/glue_actions.go` with full test coverage.

---

## Context

PF2E has several "simple" actions that are mechanically trivial but important for tracking:

- **Interact** (1 action, Manipulate trait): Generic object manipulation. The engine just tracks the action spent; the narrative outcome is LLM-determined.
- **Drop Prone** (Free action): Immediately gain Prone condition. Tactical choice for ranged cover.
- **Stand** (1 action, Move trait): Remove Prone condition. Can trigger reactions like Attack of Opportunity.
- **Take Cover** (1 action): Gain +4 circumstance bonus to AC and Reflex saves against area effects. Lost when you move.

Key rules (src: `rules/rules/core-rulebook/chapter-9-playing-the-game.md`):
- **Prone effects:** Flat-footed, −2 to attack rolls, +1 circumstance bonus to AC vs ranged attacks, must Stand before Striding.
- **Cover bonuses:** Standard cover grants +2, greater cover +4. Taking Cover improves cover by one step (or grants +2 if no cover).
- **Move trait:** Actions with the Move trait can trigger reactions like Attack of Opportunity.

---

## File Structure

```
pkg/
└── rules/
    ├── combat/
    │   ├── glue_actions.go      # Interact, DropProne, Stand, TakeCover
    │   └── glue_actions_test.go
    └── condition/
        └── conditions.go        # Add TakingCover condition
```

---

## Implementation Plan

### 1. Add TakingCover Condition

**Target:** `pkg/rules/condition/conditions.go`

```go
const TakingCover ConditionID = "taking_cover"

// In the condition effects logic:
func (c ConditionID) ACBonus() int {
    if c == TakingCover {
        return 4 // +4 circumstance bonus
    }
    // ... other conditions
    return 0
}
```

### 2. `pkg/rules/combat/glue_actions.go`

```go
package combat

import (
    "errors"
    "fmt"

    "vdnd/pkg/rules/ability"
    "vdnd/pkg/rules/condition"
    "vdnd/pkg/rules/entity"
    "vdnd/pkg/rules/trait"
)

// InteractAction is a generic 1-action object manipulation.
// The outcome is determined by the LLM; this just tracks the action cost.
type InteractAction struct {
    ObjectDescription string // Optional, for output
}

func (i *InteractAction) Name() string             { return "Interact" }
func (i *InteractAction) Cost() ability.ActionCost { return ability.CostOne }
func (i *InteractAction) HasTrait(t trait.TraitID) bool {
    return t == trait.Manipulate
}

func (i *InteractAction) Validate(actor, _ *entity.Entity, turn *TurnState) error {
    if turn.ActionsRemaining < 1 {
        return errors.New("no actions remaining")
    }
    return nil
}

func (i *InteractAction) Execute(actor, _ *entity.Entity, turn *TurnState) ability.ActionResult {
    turn.ActionsRemaining--
    desc := "Interacted with an object"
    if i.ObjectDescription != "" {
        desc = fmt.Sprintf("Interacted with %s", i.ObjectDescription)
    }
    return ability.ActionResult{Success: true, Description: desc}
}

// DropProneAction is a free action to fall prone.
type DropProneAction struct{}

func (d *DropProneAction) Name() string             { return "Drop Prone" }
func (d *DropProneAction) Cost() ability.ActionCost { return ability.CostFree }
func (d *DropProneAction) HasTrait(_ trait.TraitID) bool { return false }

func (d *DropProneAction) Validate(actor, _ *entity.Entity, _ *TurnState) error {
    if actor.Conditions.Has(condition.Prone) {
        return errors.New("already prone")
    }
    return nil
}

func (d *DropProneAction) Execute(actor, _ *entity.Entity, _ *TurnState) ability.ActionResult {
    actor.Conditions.Apply(condition.NewCondition(condition.Prone, "Dropped prone"))
    return ability.ActionResult{
        Success:     true,
        Description: "Dropped prone. You are flat-footed (-2 AC), take -2 to attack rolls, and gain +1 AC vs ranged.",
    }
}

// StandAction costs 1 action and removes Prone. Has Move trait.
type StandAction struct{}

func (s *StandAction) Name() string             { return "Stand" }
func (s *StandAction) Cost() ability.ActionCost { return ability.CostOne }
func (s *StandAction) HasTrait(t trait.TraitID) bool {
    return t == trait.Move
}

func (s *StandAction) Validate(actor, _ *entity.Entity, turn *TurnState) error {
    if !actor.Conditions.Has(condition.Prone) {
        return errors.New("not prone")
    }
    if turn.ActionsRemaining < 1 {
        return errors.New("no actions remaining")
    }
    return nil
}

func (s *StandAction) Execute(actor, _ *entity.Entity, turn *TurnState) ability.ActionResult {
    turn.ActionsRemaining--
    actor.Conditions.Remove(condition.Prone)
    return ability.ActionResult{
        Success:     true,
        Description: "Stood up from prone.",
    }
}

// TakeCoverAction grants +4 circumstance bonus to AC and Reflex.
// This bonus is lost when the entity moves.
type TakeCoverAction struct{}

func (t *TakeCoverAction) Name() string             { return "Take Cover" }
func (t *TakeCoverAction) Cost() ability.ActionCost { return ability.CostOne }
func (t *TakeCoverAction) HasTrait(_ trait.TraitID) bool { return false }

func (t *TakeCoverAction) Validate(actor, _ *entity.Entity, turn *TurnState) error {
    if turn.ActionsRemaining < 1 {
        return errors.New("no actions remaining")
    }
    // GM/LLM determines if cover is available; we just apply the bonus
    return nil
}

func (t *TakeCoverAction) Execute(actor, _ *entity.Entity, turn *TurnState) ability.ActionResult {
    turn.ActionsRemaining--
    actor.Conditions.Apply(condition.NewConditionWithValue(
        condition.TakingCover,
        4,
        "Taking Cover",
    ))
    return ability.ActionResult{
        Success:     true,
        Description: "Taking cover (+4 circumstance bonus to AC and Reflex vs area effects). Lost on movement.",
    }
}
```

### 3. Update Stride to Remove TakingCover

**Target:** `pkg/rules/combat/movement.go`

In the `Execute` method of `StrideAction`:

```go
func (s *StrideAction) Execute(actor, _ *entity.Entity, turn *TurnState) ability.ActionResult {
    // ... existing validation ...
    
    // Remove TakingCover on any movement
    actor.Conditions.Remove(condition.TakingCover)
    
    // ... rest of stride logic ...
}
```

---

## Test Plan

### `pkg/rules/combat/glue_actions_test.go`

#### Interact Tests

| Test Case | Setup | Action | Expected |
|-----------|-------|--------|----------|
| Basic interact | 3 actions remaining | Interact | Success, 2 actions remaining |
| Interact with description | 3 actions | Interact("door lever") | Output contains "door lever" |
| No actions | 0 actions remaining | Interact | Error: "no actions remaining" |

#### Drop Prone Tests

| Test Case | Setup | Action | Expected |
|-----------|-------|--------|----------|
| Drop prone | Standing entity | DropProne | Prone condition applied, still has all actions (free) |
| Already prone | Entity with Prone | DropProne | Error: "already prone" |

#### Stand Tests

| Test Case | Setup | Action | Expected |
|-----------|-------|--------|----------|
| Stand from prone | Prone, 3 actions | Stand | Prone removed, 2 actions remaining |
| Not prone | Standing, 3 actions | Stand | Error: "not prone" |
| No actions | Prone, 0 actions | Stand | Error: "no actions remaining" |
| Stand has Move trait | - | `StandAction{}.HasTrait(trait.Move)` | true |

#### Take Cover Tests

| Test Case | Setup | Action | Expected |
|-----------|-------|--------|----------|
| Take cover | 3 actions | TakeCover | TakingCover condition applied, 2 actions remaining |
| No actions | 0 actions | TakeCover | Error: "no actions remaining" |
| Cover provides AC | Has TakingCover | GetAC() | +4 circumstance bonus |
| Cover removed on stride | Has TakingCover | Stride | TakingCover removed |

---

## Validation Checklist

- [ ] All tests pass
- [ ] `go vet` reports no issues
- [ ] `go fmt` applied
- [ ] Interact has Manipulate trait
- [ ] Stand has Move trait (can trigger AoO)
- [ ] TakingCover is removed by Stride
- [ ] Drop Prone is free (doesn't consume actions)

---

## Notes for Implementation

1. **Trait checking:** `Stand` having the `Move` trait is important—it means Attack of Opportunity can be triggered when standing. Ensure the reaction system queries `HasTrait(trait.Move)`.

2. **TakingCover as condition:** Using a condition makes it easy to track and remove. The AC bonus should be extracted via `GetACBonuses()` which checks conditions.

3. **Interact is a stub:** It does nothing mechanically except track action economy. The LLM determines the narrative outcome ("you flip the switch", "the door opens").

4. **Order of operations:** When Stride executes, remove TakingCover *before* checking for reactions, so the entity doesn't benefit from cover during the reactive strike.
