# Phase 11: Afflictions

## Agent Prompt

You are implementing afflictions (poisons, diseases, curses) for a Pathfinder 2E rules engine in Go.

**Your task:** Implement the `pkg/rules/affliction` package with full test coverage.

**Prerequisites:** Phases 1-4 (check, condition, entity).

---

## Context

### Source Reference
- Afflictions: `rules/rules/core-rulebook/chapter-9-playing-the-game.md:730`

### Affliction Structure
Afflictions have **stages** that progress over time:
- Make save at regular intervals (rounds, minutes, days)
- **Success:** Reduce stage by 1 (or 2 on crit)
- **Failure:** Increase stage by 1 (or 2 on crit fail)
- **Stage 0:** Cured
- **Max stage:** Worst effect

### Example: Giant Centipede Venom
| Stage | Effect | Duration |
|-------|--------|----------|
| 1 | 1d6 poison, flat-footed | 1 round |
| 2 | 1d8 poison, flat-footed, clumsy 1 | 1 round |
| 3 | 1d12 poison, flat-footed, clumsy 2 | 1 round |

---

## File Structure

```
pkg/rules/affliction/
├── affliction.go   # Affliction, Stage structs
├── tracker.go      # AfflictionTracker on entity
└── affliction_test.go
```

---

## Implementation Plan

### 1. `affliction.go`

```go
type AfflictionType int
const (
    TypePoison AfflictionType = iota
    TypeDisease
    TypeCurse
)

type Stage struct {
    Number      int
    Damage      dice.DieRoll
    DamageType  item.DamageType
    Conditions  []condition.ConditionID
    Duration    int  // In whatever unit
}

type Affliction struct {
    ID          string
    Name        string
    Type        AfflictionType
    DC          int
    Save        SaveType  // Usually Fortitude
    OnsetDelay  int       // Rounds/minutes before first save
    MaxStage    int
    Stages      []Stage
    Interval    int       // Time between saves
}
```

### 2. `tracker.go`

```go
type AfflictionInstance struct {
    Affliction  *Affliction
    CurrentStage int
    TimeToNextSave int
}

type AfflictionTracker struct {
    afflictions []*AfflictionInstance
}

func (t *AfflictionTracker) Apply(aff *Affliction, initialStage int)
func (t *AfflictionTracker) ProcessSave(instance *AfflictionInstance, saveResult check.DegreeOfSuccess)
func (t *AfflictionTracker) Tick(entity *entity.Entity)  // Called each interval
```

**ProcessSave Pseudocode:**
```
func ProcessSave(inst *AfflictionInstance, degree DegreeOfSuccess):
    switch degree:
    case CriticalSuccess:
        inst.CurrentStage -= 2
    case Success:
        inst.CurrentStage -= 1
    case Failure:
        inst.CurrentStage += 1
    case CriticalFailure:
        inst.CurrentStage += 2
    
    # Clamp
    if inst.CurrentStage < 0:
        inst.CurrentStage = 0  # Cured!
    if inst.CurrentStage > inst.Affliction.MaxStage:
        inst.CurrentStage = inst.Affliction.MaxStage
```

---

## Test Plan

| Test | Result | Expected |
|------|--------|----------|
| Stage 2, Success | Stage 1 |
| Stage 2, Crit Success | Stage 0 (cured) |
| Stage 1, Failure | Stage 2 |
| Stage 2, Crit Failure | Stage 3 (or max) |
| At max stage, failure | Stays at max |
| At stage 0 | Removed from tracker |

---

## Validation Checklist

- [ ] Stages progress correctly on save results
- [ ] Can't go below 0 or above max
- [ ] Stage 0 = affliction cured
- [ ] Stage effects (damage, conditions) apply correctly
