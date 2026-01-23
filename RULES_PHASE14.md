# Phase 14: Hazards

## Agent Prompt

You are implementing the hazard system for a Pathfinder 2E rules engine in Go. Hazards are traps and environmental dangers that can be detected, disarmed, and triggered during encounters.

**Your task:** Implement the `pkg/rules/hazard` package with full test coverage.

**Prerequisites:** Phases 1-8, 12 (check, damage, encounter, entity).

---

## Context

### Source References
- Hazards: `rules/rules/core-rulebook/chapter-10-game-mastering.md` (Hazards section)
- Example hazards: `rules/compendium/hazards/`

### Hazard Types
| Type | Description | Examples |
|------|-------------|----------|
| Trap | Mechanical or magical device | Pit trap, Poison dart |
| Haunt | Undead spiritual manifestation | Ghostly scream |
| Environmental | Natural danger | Quicksand, Lava |

### Hazard Properties
- **Level:** Determines XP and difficulty
- **Complexity:** Simple (one-shot) or Complex (acts in initiative)
- **Stealth DC:** DC to notice before triggering
- **Disable DC:** DC to disarm (Thievery, Arcana, etc.)
- **AC/Saves/HP:** For hazards that can be attacked
- **Trigger:** What sets it off
- **Effect:** What happens when triggered

### Simple vs Complex Hazards
| Aspect | Simple | Complex |
|--------|--------|---------|
| Actions | One-time effect | Rolls initiative, takes actions |
| Disable | One check | May require multiple |
| Duration | Instant | Multiple rounds |

---

## File Structure

```
pkg/
└── rules/
    └── hazard/
        ├── hazard.go       # Hazard struct, types
        ├── trigger.go      # Trigger conditions
        ├── effect.go       # Effect implementations
        ├── registry.go     # Pre-built hazards
        └── hazard_test.go
```

---

## Implementation Plan

### 1. `pkg/rules/hazard/hazard.go`

```go
type HazardType int
const (
    HazardTrap HazardType = iota
    HazardHaunt
    HazardEnvironmental
)

type Complexity int
const (
    ComplexitySimple Complexity = iota
    ComplexityComplex
)

type DisableOption struct {
    Skill SkillID
    DC    int
    Description string
}

type Hazard struct {
    ID          string
    Name        string
    Level       int
    Type        HazardType
    Complexity  Complexity
    Traits      []trait.TraitID
    
    // Detection
    StealthDC   int  // DC to notice with Perception
    
    // Defenses (for hazards that can be attacked)
    AC          int
    Fortitude   int
    Reflex      int
    Will        int
    HP          int
    Hardness    int  // Damage reduction
    Immunities  []string
    
    // Disabling
    DisableOptions []DisableOption
    
    // Trigger
    Trigger     TriggerCondition
    
    // Effects
    Effect      HazardEffect
    
    // Complex hazard initiative
    Initiative  int  // Modifier for complex hazards
    
    // State
    IsTriggered bool
    IsDisabled  bool
    CurrentHP   int
}

func NewHazard(id, name string, level int) *Hazard

// Detect attempts to notice the hazard with Perception
func (h *Hazard) Detect(observer *entity.Entity) bool

// CanDisable checks if entity can attempt to disable
func (h *Hazard) CanDisable(actor *entity.Entity, method DisableOption) bool

// AttemptDisable tries to disable the hazard
func (h *Hazard) AttemptDisable(actor *entity.Entity, method DisableOption) check.CheckResult

// CheckTrigger determines if a hazard triggers
func (h *Hazard) CheckTrigger(event Event) bool

// Activate fires the hazard effect
func (h *Hazard) Activate(targets []*entity.Entity) []HazardResult

// TakeDamage for attackable hazards
func (h *Hazard) TakeDamage(amount int, damageType string) int
```

### 2. `pkg/rules/hazard/trigger.go`

```go
type TriggerType int
const (
    TriggerEnter TriggerType = iota  // Creature enters area
    TriggerTouch                      // Creature touches object
    TriggerOpen                       // Container/door opened
    TriggerProximity                  // Within X feet
    TriggerPressure                   // Weight on plate
    TriggerTimeBased                  // After X rounds/time
)

type TriggerCondition struct {
    Type        TriggerType
    Area        string       // Zone or position
    Radius      int          // For proximity
    MinWeight   int          // For pressure plates
    Delay       int          // Rounds before activation
}

// Matches checks if an event matches this trigger
func (t TriggerCondition) Matches(event Event, hazardPosition string) bool
```

**Matches Pseudocode:**
```
func (t TriggerCondition) Matches(event Event, hazardPos string) bool:
    switch t.Type:
    case TriggerEnter:
        return event.Type == EventMove && event.Position == t.Area
    case TriggerProximity:
        return isWithinRange(event.Actor.Position, hazardPos, t.Radius)
    case TriggerTouch:
        return event.Type == EventManipulate && event.Target == hazardPos
    // etc.
```

### 3. `pkg/rules/hazard/effect.go`

```go
type HazardResult struct {
    Target      *entity.Entity
    Damage      int
    DamageType  item.DamageType
    Conditions  []condition.ConditionInstance
    Description string
}

type HazardEffect interface {
    Apply(hazard *Hazard, targets []*entity.Entity) []HazardResult
}

// DamageEffect - deals damage, optionally with save
type DamageEffect struct {
    Damage     dice.DieRoll
    DamageType item.DamageType
    SaveType   SaveType
    SaveDC     int
    IsBasicSave bool
}

func (d *DamageEffect) Apply(hazard *Hazard, targets []*entity.Entity) []HazardResult

// ConditionEffect - applies conditions
type ConditionEffect struct {
    Condition condition.ConditionID
    Value     int
    Duration  int
    SaveType  SaveType
    SaveDC    int
}

func (c *ConditionEffect) Apply(hazard *Hazard, targets []*entity.Entity) []HazardResult

// MultiEffect - combines multiple effects
type MultiEffect struct {
    Effects []HazardEffect
}

func (m *MultiEffect) Apply(hazard *Hazard, targets []*entity.Entity) []HazardResult
```

### 4. `pkg/rules/hazard/registry.go`

```go
var (
    // Pit Trap - Simple, fall damage
    PitTrap = Hazard{
        ID:         "pit-trap",
        Name:       "Pit Trap",
        Level:      1,
        Type:       HazardTrap,
        Complexity: ComplexitySimple,
        Traits:     []trait.TraitID{trait.TraitMechanical, trait.TraitTrap},
        StealthDC:  18,
        DisableOptions: []DisableOption{
            {SkillThievery, 15, "Jam the trapdoor"},
        },
        Trigger: TriggerCondition{
            Type: TriggerPressure,
            MinWeight: 50,
        },
        Effect: &DamageEffect{
            Damage:     dice.DieRoll{2, 6, 0},  // 2d6 fall damage
            DamageType: item.Bludgeoning,
            SaveType:   SaveReflex,
            SaveDC:     17,
            IsBasicSave: false,  // Reflex to grab edge
        },
    }
    
    // Poison Dart Trap - Simple, ranged attack + poison
    PoisonDartTrap = Hazard{
        ID:         "poison-dart-trap",
        Name:       "Poison Dart Trap",
        Level:      2,
        Type:       HazardTrap,
        Complexity: ComplexitySimple,
        StealthDC:  20,
        AC:         18,
        HP:         30,
        Hardness:   8,
        DisableOptions: []DisableOption{
            {SkillThievery, 18, "Disable firing mechanism"},
        },
        Trigger: TriggerCondition{Type: TriggerTouch},
        Effect: &MultiEffect{
            Effects: []HazardEffect{
                &AttackEffect{
                    AttackBonus: 12,
                    Damage:      dice.DieRoll{1, 6, 0},
                    DamageType:  item.Piercing,
                },
                &AfflictionEffect{
                    Affliction: affliction.GiantCentipedeVenom,
                    OnHit:      true,
                },
            },
        },
    }
    
    // Blade Barrier - Complex, acts each round
    BladeBarrier = Hazard{
        ID:         "blade-barrier",
        Name:       "Spinning Blade Trap",
        Level:      5,
        Type:       HazardTrap,
        Complexity: ComplexityComplex,
        StealthDC:  24,
        AC:         22,
        HP:         60,
        Hardness:   12,
        Initiative: 8,  // Rolls initiative
        DisableOptions: []DisableOption{
            {SkillThievery, 22, "Jam the mechanism"},
            {SkillAthletics, 24, "Force blades apart"},
        },
        Trigger: TriggerCondition{Type: TriggerEnter},
        Effect: &DamageEffect{
            Damage:      dice.DieRoll{3, 8, 0},  // 3d8 per round
            DamageType:  item.Slashing,
            SaveType:    SaveReflex,
            SaveDC:      22,
            IsBasicSave: true,
        },
    }
)

func GetHazard(id string) (*Hazard, bool)
```

---

## Test Plan

### Detection Tests
| Observer Perception | Stealth DC | Expected |
|---------------------|------------|----------|
| +8, rolls 15 (23) | 20 | Detected |
| +5, rolls 10 (15) | 20 | Not detected |
| +5, rolls 20 (25) | 20 | Detected |

### Disable Tests
| Skill Mod | DC | Roll | Expected |
|-----------|-----|------|----------|
| Thievery +10 | 18 | 12 | Success (22) |
| Thievery +5 | 20 | 10 | Failure (15) |
| Thievery +8 | 15 | 20 | Crit Success |

### Trigger Tests
| Trigger Type | Event | Expected |
|--------------|-------|----------|
| Enter zone A | Move to A | Triggers |
| Enter zone A | Move to B | No trigger |
| Pressure 50lb | 80lb creature enters | Triggers |
| Pressure 50lb | 30lb creature enters | No trigger |

### Effect Tests
| Hazard | Save Result | Expected |
|--------|-------------|----------|
| Pit Trap | Reflex fail | 2d6 bludgeoning |
| Pit Trap | Reflex success | Catch edge, no damage |
| Dart Trap | Attack hits | 1d6 + poison |
| Blade Barrier (basic) | Success | Half 3d8 |
| Blade Barrier (basic) | Crit success | No damage |

### Complex Hazard Turn Tests
| Test | Expected |
|------|----------|
| Blade Barrier rolls initiative | Joins encounter order |
| On hazard's turn | Deals damage to creatures in zone |
| When destroyed | Removed from initiative |

---

## Validation Checklist
- [ ] Hazards can be detected with Perception
- [ ] Disable options use correct skills/DCs
- [ ] Triggers match correct events
- [ ] Effects apply damage/conditions correctly
- [ ] Basic saves scale damage by degree
- [ ] Attackable hazards take damage, have hardness
- [ ] Complex hazards participate in initiative
