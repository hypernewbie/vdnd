# Phase 26: Movement Modes

## Agent Prompt

You are implementing alternative movement modes for a Pathfinder 2E rules engine in Go. Creatures can fly, swim, climb, and burrow in addition to walking. The engine needs to track which mode is active and use the appropriate speed.

**Your task:** Add movement mode fields to `Entity` and update `Stride` to use the effective speed based on current mode.

---

## Context

PF2E creatures can have multiple movement speeds:
- **Speed (Ground):** Default walking speed, typically 25-30 ft for humanoids.
- **Fly Speed:** Movement through the air. Requires the creature to have wings, magic, or similar.
- **Swim Speed:** Movement through water. Without a swim speed, creatures move at half speed and must use Athletics.
- **Climb Speed:** Movement up vertical surfaces. Without a climb speed, requires Athletics checks.
- **Burrow Speed:** Movement through earth or loose material.

Key rules (src: `rules/rules/core-rulebook/chapter-9-playing-the-game.md`):
- A creature uses its Speed for the appropriate movement mode
- If a creature lacks a special speed, it may be unable to move in that mode or require skill checks
- Some effects grant temporary movement modes (e.g., *fly* spell)

The engine tracks the *current* movement mode. The LLM is responsible for setting it appropriately based on the narrative ("you dive into the water" → set mode to Swim).

---

## File Structure

```
pkg/
└── rules/
    └── entity/
        ├── entity.go       # Add speed fields and MoveMode
        └── entity_test.go  # Test EffectiveSpeed()
```

---

## Implementation Plan

### 1. Define MoveMode Enum

**Target:** `pkg/rules/entity/entity.go`

```go
type MoveMode int

const (
    MoveModeGround MoveMode = iota
    MoveModeFly
    MoveModeSwim
    MoveModeClimb
    MoveModeBurrow
)

func (m MoveMode) String() string {
    switch m {
    case MoveModeFly:
        return "fly"
    case MoveModeSwim:
        return "swim"
    case MoveModeClimb:
        return "climb"
    case MoveModeBurrow:
        return "burrow"
    default:
        return "ground"
    }
}
```

### 2. Add Speed Fields to Entity

**Target:** `pkg/rules/entity/entity.go`

```go
type Entity struct {
    // ... existing fields ...

    // Movement speeds (in feet)
    Speed       int // Ground speed, the default
    FlySpeed    int // 0 if creature cannot fly
    SwimSpeed   int // 0 if creature cannot swim naturally
    ClimbSpeed  int // 0 if creature cannot climb naturally
    BurrowSpeed int // 0 if creature cannot burrow

    // Current movement mode
    CurrentMoveMode MoveMode
}
```

### 3. EffectiveSpeed Helper

**Target:** `pkg/rules/entity/entity.go`

```go
// EffectiveSpeed returns the speed for the current movement mode.
// Returns 0 if the creature cannot move in the current mode.
func (e *Entity) EffectiveSpeed() int {
    switch e.CurrentMoveMode {
    case MoveModeFly:
        return e.FlySpeed
    case MoveModeSwim:
        return e.SwimSpeed
    case MoveModeClimb:
        return e.ClimbSpeed
    case MoveModeBurrow:
        return e.BurrowSpeed
    default:
        return e.Speed
    }
}

// SetMoveMode changes the current movement mode.
// Returns an error if the creature has 0 speed for that mode.
func (e *Entity) SetMoveMode(mode MoveMode) error {
    // Check if the creature can use this mode
    var speed int
    switch mode {
    case MoveModeFly:
        speed = e.FlySpeed
    case MoveModeSwim:
        speed = e.SwimSpeed
    case MoveModeClimb:
        speed = e.ClimbSpeed
    case MoveModeBurrow:
        speed = e.BurrowSpeed
    default:
        speed = e.Speed
    }

    if speed == 0 && mode != MoveModeGround {
        return fmt.Errorf("cannot use %s mode: no %s speed", mode, mode)
    }

    e.CurrentMoveMode = mode
    return nil
}

// AllSpeeds returns a map of all non-zero speeds.
// Useful for status display.
func (e *Entity) AllSpeeds() map[string]int {
    speeds := make(map[string]int)
    if e.Speed > 0 {
        speeds["ground"] = e.Speed
    }
    if e.FlySpeed > 0 {
        speeds["fly"] = e.FlySpeed
    }
    if e.SwimSpeed > 0 {
        speeds["swim"] = e.SwimSpeed
    }
    if e.ClimbSpeed > 0 {
        speeds["climb"] = e.ClimbSpeed
    }
    if e.BurrowSpeed > 0 {
        speeds["burrow"] = e.BurrowSpeed
    }
    return speeds
}
```

### 4. Update Stride Action

**Target:** `pkg/rules/combat/movement.go`

```go
func (s *StrideAction) Execute(actor, _ *entity.Entity, turn *TurnState) ability.ActionResult {
    turn.ActionsRemaining--

    speed := actor.EffectiveSpeed()
    if speed == 0 {
        return ability.ActionResult{
            Success:     false,
            Description: fmt.Sprintf("Cannot move: no %s speed", actor.CurrentMoveMode),
        }
    }

    // Remove TakingCover on movement (from Phase 25)
    actor.Conditions.Remove(condition.TakingCover)

    // ... rest of existing stride logic (position changes, engagement) ...

    return ability.ActionResult{
        Success: true,
        Description: fmt.Sprintf("Moved up to %d ft (%s)", speed, actor.CurrentMoveMode),
    }
}
```

---

## Test Plan

### `pkg/rules/entity/entity_test.go`

#### EffectiveSpeed Tests

| Test Case | Entity Speeds | CurrentMode | Expected Speed |
|-----------|---------------|-------------|----------------|
| Ground default | Speed: 30 | MoveModeGround | 30 |
| Fly mode | Speed: 30, FlySpeed: 40 | MoveModeFly | 40 |
| Swim mode | Speed: 30, SwimSpeed: 20 | MoveModeSwim | 20 |
| Climb mode | Speed: 30, ClimbSpeed: 15 | MoveModeClimb | 15 |
| Burrow mode | Speed: 30, BurrowSpeed: 10 | MoveModeBurrow | 10 |
| No fly speed | Speed: 30, FlySpeed: 0 | MoveModeFly | 0 |
| No swim speed | Speed: 30, SwimSpeed: 0 | MoveModeSwim | 0 |

#### SetMoveMode Tests

| Test Case | Entity Speeds | SetMoveMode | Expected |
|-----------|---------------|-------------|----------|
| Valid fly | FlySpeed: 40 | MoveModeFly | Success, mode changed |
| Invalid fly | FlySpeed: 0 | MoveModeFly | Error: "no fly speed" |
| Valid swim | SwimSpeed: 20 | MoveModeSwim | Success |
| Ground always works | Speed: 0 | MoveModeGround | Success (even with 0 speed) |
| Invalid burrow | BurrowSpeed: 0 | MoveModeBurrow | Error: "no burrow speed" |

#### AllSpeeds Tests

| Test Case | Entity Speeds | Expected Map |
|-----------|---------------|--------------|
| Ground only | Speed: 30 | `{"ground": 30}` |
| Multiple speeds | Speed: 30, FlySpeed: 40, SwimSpeed: 20 | `{"ground": 30, "fly": 40, "swim": 20}` |
| Zero speeds excluded | Speed: 30, FlySpeed: 0 | `{"ground": 30}` |

### `pkg/rules/combat/movement_test.go`

| Test Case | Setup | Expected |
|-----------|-------|----------|
| Stride ground | Speed: 30, Ground mode | Success, "30 ft (ground)" |
| Stride fly | FlySpeed: 40, Fly mode | Success, "40 ft (fly)" |
| Stride no speed | FlySpeed: 0, Fly mode | Failure, "no fly speed" |
| Stride removes cover | Has TakingCover, Stride | TakingCover removed |

---

## Validation Checklist

- [ ] All tests pass
- [ ] `go vet` reports no issues
- [ ] `go fmt` applied
- [ ] MoveMode has String() for display
- [ ] EffectiveSpeed returns 0 for unavailable modes
- [ ] SetMoveMode rejects modes with 0 speed (except ground)
- [ ] Stride uses EffectiveSpeed()

---

## Notes for Implementation

1. **Ground with 0 speed:** A creature with Speed 0 is typically Immobilized. The condition check happens elsewhere; here we just return the raw value.

2. **LLM responsibility:** The LLM decides when to change modes. "You dive into the water" → `vd entity set rogue movemode swim`. The engine doesn't auto-detect terrain.

3. **Temporary speeds:** Spells like *fly* can grant a FlySpeed. The LLM can set this directly: `vd entity set wizard flyspeed 40`.

4. **Speed penalties:** Conditions like Encumbered reduce speed. These are applied via condition modifiers, not by changing the base Speed field.

5. **Step action:** The Step action should also respect EffectiveSpeed(), though Step is fixed at 5 ft regardless of mode.
