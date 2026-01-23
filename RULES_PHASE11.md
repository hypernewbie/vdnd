# Phase 11: Afflictions

## Agent Prompt

You are implementing afflictions (poisons, diseases, curses) for a Pathfinder 2E rules engine in Go. Afflictions are ongoing harmful effects that progress through stages based on saving throws.

**Your task:** Implement the `pkg/rules/affliction` package with full test coverage.

**Prerequisites:** Phases 1-4 (check, condition, entity, dice).

---

## Context

### Source Reference
- Afflictions: `rules/rules/core-rulebook/chapter-9-playing-the-game.md:730`

### Affliction Types
| Type | Save | Interval | Notes |
|------|------|----------|-------|
| Poison | Fortitude | Rounds | Often from attacks |
| Disease | Fortitude | Days | Long-term, onset delay |
| Curse | Will | Variable | Usually magical |

### Affliction Mechanics

1. **Exposure:** Entity fails save or is hit by affliction source
2. **Onset:** Some afflictions have delay before first effect
3. **Stages:** Each stage applies specific effects (damage, conditions)
4. **Progression:** Make save at each interval:
   - **Critical Success:** Reduce stage by 2
   - **Success:** Reduce stage by 1  
   - **Failure:** Increase stage by 1
   - **Critical Failure:** Increase stage by 2
5. **Resolution:** Stage 0 = cured; max stage = worst effect

### Example: Giant Centipede Venom (Poison)
- **DC:** 17 Fortitude
- **Max Stage:** 3

| Stage | Effect |
|-------|--------|
| 1 | 1d6 poison damage, flat-footed |
| 2 | 1d8 poison damage, flat-footed, clumsy 1 |
| 3 | 1d12 poison damage, flat-footed, clumsy 2 |

### Example: Zombie Rot (Disease)
- **DC:** 14 Fortitude
- **Onset:** 1 day
- **Interval:** 1 day

| Stage | Effect |
|-------|--------|
| 1 | Carrier (no symptoms) |
| 2 | 1d6 negative damage, slowed 1 |
| 3 | 1d6 negative damage, slowed 2 |
| 4 | Dead, rises as zombie |

---

## File Structure

```
pkg/
└── rules/
    └── affliction/
        ├── affliction.go   # Affliction, Stage structs
        ├── instance.go     # AfflictionInstance (applied)
        ├── tracker.go      # AfflictionTracker on entity
        ├── registry.go     # Pre-built afflictions
        └── affliction_test.go
```

---

## Implementation Plan

### 1. `pkg/rules/affliction/affliction.go`

```go
type AfflictionType int
const (
    TypePoison AfflictionType = iota
    TypeDisease
    TypeCurse
)

type SaveType int
const (
    SaveFortitude SaveType = iota
    SaveReflex
    SaveWill
)

type IntervalUnit int
const (
    IntervalRounds IntervalUnit = iota
    IntervalMinutes
    IntervalHours
    IntervalDays
)

type Stage struct {
    Number      int
    Damage      dice.DieRoll
    DamageType  item.DamageType
    Conditions  []ConditionEffect
}

type ConditionEffect struct {
    ID    condition.ConditionID
    Value int  // For valued conditions
}

type Affliction struct {
    ID           string
    Name         string
    Type         AfflictionType
    DC           int
    Save         SaveType
    OnsetDelay   int           // 0 = immediate
    OnsetUnit    IntervalUnit
    MaxStage     int
    Stages       []Stage       // Indexed by stage number
    Interval     int
    IntervalUnit IntervalUnit
}

func (a *Affliction) GetStage(num int) *Stage
```

### 2. `pkg/rules/affliction/instance.go`

```go
type AfflictionInstance struct {
    Affliction    *Affliction
    CurrentStage  int
    TimeToOnset   int  // Countdown, -1 = onset passed
    TimeToNextSave int
    Source        string  // "Giant Centipede bite"
}

func NewInstance(aff *Affliction, source string) *AfflictionInstance

// IsCured returns true if stage reached 0
func (i *AfflictionInstance) IsCured() bool

// IsActive returns true if past onset and not cured
func (i *AfflictionInstance) IsActive() bool

// GetCurrentEffects returns damage and conditions for current stage
func (i *AfflictionInstance) GetCurrentEffects() (dice.DieRoll, item.DamageType, []ConditionEffect)
```

### 3. `pkg/rules/affliction/tracker.go`

```go
type AfflictionTracker struct {
    afflictions []*AfflictionInstance
}

func NewTracker() *AfflictionTracker

// Add applies a new affliction, starting at stage 1
func (t *AfflictionTracker) Add(aff *Affliction, source string)

// Has checks if entity has a specific affliction
func (t *AfflictionTracker) Has(afflictionID string) bool

// Get returns an affliction instance by ID
func (t *AfflictionTracker) Get(afflictionID string) *AfflictionInstance

// Remove cures/removes an affliction
func (t *AfflictionTracker) Remove(afflictionID string)

// All returns all active affliction instances
func (t *AfflictionTracker) All() []*AfflictionInstance

// ProcessSave updates stage based on save result
func (t *AfflictionTracker) ProcessSave(afflictionID string, result check.DegreeOfSuccess)

// Tick advances time and processes effects
// Called once per interval (round for poisons, day for diseases)
func (t *AfflictionTracker) Tick(entity *entity.Entity, unit IntervalUnit)
```

**ProcessSave Pseudocode:**
```
func (t *AfflictionTracker) ProcessSave(id string, result check.DegreeOfSuccess):
    inst := t.Get(id)
    if inst == nil: return
    
    switch result:
    case CriticalSuccess:
        inst.CurrentStage -= 2
    case Success:
        inst.CurrentStage -= 1
    case Failure:
        inst.CurrentStage += 1
    case CriticalFailure:
        inst.CurrentStage += 2
    
    # Clamp to valid range
    if inst.CurrentStage < 0:
        inst.CurrentStage = 0
    if inst.CurrentStage > inst.Affliction.MaxStage:
        inst.CurrentStage = inst.Affliction.MaxStage
    
    # Remove if cured
    if inst.CurrentStage == 0:
        t.Remove(id)
```

**Tick Pseudocode:**
```
func (t *AfflictionTracker) Tick(entity *entity.Entity, unit IntervalUnit):
    for _, inst := range t.afflictions:
        if inst.Affliction.IntervalUnit != unit:
            continue
        
        # Check onset
        if inst.TimeToOnset > 0:
            inst.TimeToOnset -= 1
            continue
        
        # Apply current stage effects
        damage, damageType, conditions := inst.GetCurrentEffects()
        if damage.Count > 0:
            dmg := damage.Roll()
            entity.TakeDamage(dmg, string(damageType))
        
        for _, cond := range conditions:
            entity.Conditions.Apply(condition.NewValuedCondition(
                cond.ID, cond.Value, inst.Affliction.Name))
        
        # Prompt for save (handled by caller/encounter system)
        inst.TimeToNextSave -= 1
```

### 4. `pkg/rules/affliction/registry.go`

```go
var (
    GiantCentipedeVenom = Affliction{
        ID:           "giant-centipede-venom",
        Name:         "Giant Centipede Venom",
        Type:         TypePoison,
        DC:           17,
        Save:         SaveFortitude,
        MaxStage:     3,
        Interval:     1,
        IntervalUnit: IntervalRounds,
        Stages: []Stage{
            {1, dice.DieRoll{1, 6, 0}, item.Poison, 
             []ConditionEffect{{condition.FlatFooted, 0}}},
            {2, dice.DieRoll{1, 8, 0}, item.Poison,
             []ConditionEffect{{condition.FlatFooted, 0}, {condition.Clumsy, 1}}},
            {3, dice.DieRoll{1, 12, 0}, item.Poison,
             []ConditionEffect{{condition.FlatFooted, 0}, {condition.Clumsy, 2}}},
        },
    }
    
    ZombieRot = Affliction{
        ID:           "zombie-rot",
        Name:         "Zombie Rot",
        Type:         TypeDisease,
        DC:           14,
        Save:         SaveFortitude,
        OnsetDelay:   1,
        OnsetUnit:    IntervalDays,
        MaxStage:     4,
        Interval:     1,
        IntervalUnit: IntervalDays,
        Stages: []Stage{
            {1, dice.DieRoll{0, 0, 0}, item.DamageType(""), nil}, // Carrier
            {2, dice.DieRoll{1, 6, 0}, item.Negative,
             []ConditionEffect{{condition.Slowed, 1}}},
            {3, dice.DieRoll{1, 6, 0}, item.Negative,
             []ConditionEffect{{condition.Slowed, 2}}},
            {4, dice.DieRoll{0, 0, 0}, item.DamageType(""), nil}, // Death
        },
    }
)

func GetAffliction(id string) (*Affliction, bool)
```

---

## Test Plan

### Stage Progression Tests
| Current Stage | Save Result | Expected Stage |
|---------------|-------------|----------------|
| 1 | Critical Success | 0 (cured) |
| 1 | Success | 0 (cured) |
| 1 | Failure | 2 |
| 1 | Critical Failure | 3 |
| 2 | Success | 1 |
| 2 | Critical Success | 0 (cured) |
| 3 | Failure | 3 (at max) |
| 3 | Critical Failure | 3 (capped) |

### Clamping Tests
| Current | Action | Expected |
|---------|--------|----------|
| 1 | CritSuccess | 0, removed |
| 0 | - | Already cured |
| Max | Fail | Stays at max |
| Max | CritFail | Stays at max |

### Effect Application Tests
| Affliction | Stage | Expected Effects |
|------------|-------|------------------|
| Centipede Venom | 1 | 1d6 poison, flat-footed |
| Centipede Venom | 2 | 1d8 poison, flat-footed, clumsy 1 |
| Centipede Venom | 3 | 1d12 poison, flat-footed, clumsy 2 |
| Zombie Rot | 1 | No damage, no conditions |
| Zombie Rot | 2 | 1d6 negative, slowed 1 |

### Onset Delay Tests
| Affliction | After Tick 0 | After Tick 1 |
|------------|--------------|--------------|
| Centipede Venom | Active (no onset) | Takes damage |
| Zombie Rot | TimeToOnset=1 | Active, takes damage |

### Tracker Tests
| Test | Expected |
|------|----------|
| Add affliction | Has() returns true |
| Remove by ID | Has() returns false |
| ProcessSave to 0 | Auto-removed |
| Tick applies damage | Entity HP reduced |
| Tick applies conditions | Conditions on entity |

---

## Validation Checklist

- [ ] Stage progression follows CritSuccess/Success/Fail/CritFail rules
- [ ] Can't go below 0 or above MaxStage
- [ ] Stage 0 = affliction removed
- [ ] Onset delay works correctly
- [ ] Each stage applies correct damage and conditions
- [ ] Tick only processes matching interval units
- [ ] Registry has common afflictions

---

## Notes for Implementation

1. **Onset vs First Effect:** Some afflictions have onset delay. Don't apply effects until onset passes.

2. **Interval timing:** Poisons tick every round in combat. Diseases tick daily (handled in downtime). The caller (encounter or downtime system) calls Tick with appropriate unit.

3. **Save prompting:** Tick should indicate when a save is needed, but the actual save roll is handled by the encounter/CLI layer.

4. **Stage 0 special handling:** When reaching stage 0, auto-remove the affliction. Don't process effects for stage 0.

5. **Virulence:** Some high-level afflictions require 2+ consecutive successes to reduce stage. Not implementing in base system, but consider for future.

6. **Death at max:** Some afflictions (like Zombie Rot stage 4) cause death. Handle specially, perhaps with a "IsFatal" flag on the stage.
