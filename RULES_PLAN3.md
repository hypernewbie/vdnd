# Pathfinder 2E Rules Engine - Implementation Plan (Part 3)

This document covers the "Glue Mechanics" identified in the gap analysis. These are small but important features that fill holes between the tactical engine and the LLM orchestrator.

---

## Phase 25: Glue Actions

**Priority:** Medium — These are simple but frequently used.

### 25.1 Interact Action (Stub)
**Target File:** `pkg/rules/combat/glue_actions.go`

Interact is a catch-all 1-action activity for manipulating objects. The engine doesn't need to know *what* you're interacting with—just that you spent an action.

```go
type InteractAction struct {
    ObjectDescription string // For output purposes only
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
    desc := "Interacted"
    if i.ObjectDescription != "" {
        desc = fmt.Sprintf("Interacted with %s", i.ObjectDescription)
    }
    return ability.ActionResult{Success: true, Description: desc}
}
```

### 25.2 Drop Prone Action
**Target File:** `pkg/rules/combat/glue_actions.go`

Dropping prone is a free action. It applies the Prone condition immediately.

```go
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
        Description: "Dropped prone. Flat-footed, take -2 to attack rolls, gain +1 AC vs ranged.",
    }
}
```

### 25.3 Stand Action
**Target File:** `pkg/rules/combat/glue_actions.go`

Standing from prone costs 1 action and has the Move trait (can trigger reactions).

```go
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
```

### 25.4 Take Cover Action
**Target File:** `pkg/rules/combat/glue_actions.go`

Taking cover grants a +4 circumstance bonus to AC and Reflex saves against area effects (or +2 if already behind lesser cover). This is tracked as a condition that expires on movement.

```go
// First, add the TakingCover condition to condition/conditions.go
const TakingCover ConditionID = "taking_cover"

type TakeCoverAction struct{}

func (t *TakeCoverAction) Name() string             { return "Take Cover" }
func (t *TakeCoverAction) Cost() ability.ActionCost { return ability.CostOne }
func (t *TakeCoverAction) HasTrait(_ trait.TraitID) bool { return false }

func (t *TakeCoverAction) Validate(actor, _ *entity.Entity, turn *TurnState) error {
    if turn.ActionsRemaining < 1 {
        return errors.New("no actions remaining")
    }
    // Note: In reality you need cover to take cover behind.
    // The LLM determines if cover is available; we just apply the bonus.
    return nil
}

func (t *TakeCoverAction) Execute(actor, _ *entity.Entity, turn *TurnState) ability.ActionResult {
    turn.ActionsRemaining--
    
    // Apply taking cover condition with +4 circumstance bonus
    actor.Conditions.Apply(condition.NewConditionWithValue(
        condition.TakingCover, 
        4, // +4 circumstance bonus
        "Taking Cover",
    ))
    
    return ability.ActionResult{
        Success:     true,
        Description: "Taking cover (+4 circumstance bonus to AC and Reflex vs area effects).",
    }
}
```

**Integration Note:** The Stride action should remove the TakingCover condition when the entity moves.

---

## Phase 26: Movement Modes

**Priority:** Low — Fly/Swim/Climb are less common than walking.

### 26.1 Entity Movement Fields
**Target File:** `pkg/rules/entity/entity.go`

Add alternative speed fields and current movement mode to Entity:

```go
type MoveMode int

const (
    MoveModeGround MoveMode = iota
    MoveModeFly
    MoveModeSwim
    MoveModeClimb
    MoveModeBurrow
)

// Add to Entity struct:
type Entity struct {
    // ... existing fields ...
    
    // Movement
    Speed      int      // Ground speed in feet
    FlySpeed   int      // 0 if can't fly
    SwimSpeed  int      // 0 if can't swim
    ClimbSpeed int      // 0 if can't climb
    BurrowSpeed int     // 0 if can't burrow
    
    CurrentMoveMode MoveMode // What mode of movement is active
}

// Helper to get effective speed for current mode
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
```

### 26.2 Stride Action Update
**Target File:** `pkg/rules/combat/movement.go`

Update Stride to use the effective speed based on current movement mode:

```go
func (s *StrideAction) Execute(actor, _ *entity.Entity, turn *TurnState) ability.ActionResult {
    turn.ActionsRemaining--
    
    speed := actor.EffectiveSpeed()
    if speed == 0 {
        return ability.ActionResult{
            Success:     false,
            Description: fmt.Sprintf("Cannot move in %s mode (speed 0)", actor.CurrentMoveMode),
        }
    }
    
    // Remove taking cover on movement
    actor.Conditions.Remove(condition.TakingCover)
    
    // ... rest of existing stride logic ...
}
```

---

## Phase 27: Counteract System

**Priority:** Medium — Used for Dispel Magic, curing afflictions, etc.

### 27.1 Counteract Check
**Target File:** `pkg/rules/check/counteract.go`

Counteracting uses a special level-comparison check. The counteract level is typically half the caster's level (rounded up) for spells.

```go
// CounterActResult represents the outcome of a counteract attempt
type CounteractResult struct {
    CheckResult
    CanCounteract    bool
    MaxLevelAffected int  // Based on degree of success
}

// CounterActCheck performs a counteract check
// src: rules/rules/core-rulebook/chapter-9-playing-the-game.md (Counteracting)
func CounteractCheck(
    counteractLevel int,    // Your counteract level (usually spell rank or half level)
    counteractMod int,      // Your counteract modifier (usually spell attack or tradition DC - 10)
    targetLevel int,        // Level of the effect being counteracted
    targetDC int,           // DC of the effect (often caster's DC)
) CounteractResult {
    result := PerformCheck(counteractMod, nil, targetDC)
    
    var maxLevel int
    switch result.Degree {
    case CriticalSuccess:
        maxLevel = counteractLevel + 3
    case Success:
        maxLevel = counteractLevel + 1
    case Failure:
        maxLevel = counteractLevel - 1
    case CriticalFailure:
        maxLevel = counteractLevel - 3
    }
    
    return CounteractResult{
        CheckResult:      result,
        CanCounteract:    targetLevel <= maxLevel,
        MaxLevelAffected: maxLevel,
    }
}
```

### 27.2 Example Usage

```go
// Dispel Magic vs a 5th rank spell
result := check.CounteractCheck(
    5,  // Counteract level (your spell rank)
    16, // Your spell attack modifier
    5,  // Target spell's rank
    28, // Target caster's DC
)

if result.CanCounteract {
    // Remove the spell effect
}
```

---

## Phase 28: Odds & Ends

**Priority:** Low — Cleanup to ensure completeness.

### 28.1 Hero Points
**Target File:** `pkg/rules/entity/hero_points.go`

Hero Points are a meta-currency that allow rerolls or prevent death. Each PC typically starts a session with 1 and can earn more.

```go
// Add to Entity struct:
type Entity struct {
    // ... existing fields ...
    HeroPoints int // Typically 0-3
}

// SpendHeroPoint uses a hero point for a reroll or to stabilise when dying
type HeroPointUse int

const (
    HeroPointReroll HeroPointUse = iota
    HeroPointStabilise
)

func (e *Entity) SpendHeroPoint(use HeroPointUse) error {
    if e.HeroPoints <= 0 {
        return errors.New("no hero points available")
    }
    e.HeroPoints--
    return nil
}

func (e *Entity) GainHeroPoint() {
    e.HeroPoints++
    if e.HeroPoints > 3 {
        e.HeroPoints = 3 // Cap at 3
    }
}
```

**Reroll Logic:**
When a hero point is spent for a reroll, the check is re-performed and the player *must* use the new result.

```go
// In pkg/rules/check/check.go
func PerformCheckWithHeroPoint(baseModifier int, modifiers []Modifier, dc int, roller Roller) (original, reroll CheckResult) {
    original = PerformCheck(baseModifier, modifiers, dc)
    reroll = PerformCheck(baseModifier, modifiers, dc)
    return original, reroll
}
```

**Stabilise Logic:**
When dying, spending all remaining hero points immediately stabilises at 0 HP.

```go
func (e *Entity) HeroPointStabilise() bool {
    if e.HeroPoints == 0 {
        return false
    }
    e.HeroPoints = 0
    e.Conditions.Remove(condition.Dying)
    e.HP = 0
    return true
}
```

### 28.2 Recall Knowledge Action
**Target File:** `pkg/rules/skill/actions.go`

Recall Knowledge is a 1-action skill check to identify a creature, object, or concept. The skill used depends on what's being identified.

```go
// RecallKnowledge performs a knowledge check
// src: rules/compendium/skills.md "Recall Knowledge"
// The LLM determines which skill is appropriate based on target type:
//   - Arcana: Constructs, Dragons, Magical traditions
//   - Crafting: Alchemical items, Constructs (crafted)
//   - Medicine: Diseases, Poisons (effects)
//   - Nature: Animals, Beasts, Fey, Plants
//   - Occultism: Aberrations, Spirits, Esoterica
//   - Religion: Undead, Divine creatures, Religious lore
//   - Society: Humanoids, History, Culture
func RecallKnowledge(actor *entity.Entity, skillID ability.SkillID, dc int, naturalRoll int) (bool, check.CheckResult) {
    var res check.CheckResult
    if naturalRoll > 0 {
        res = PerformSkillCheckWithRoll(actor, skillID, dc, naturalRoll)
    } else {
        res = PerformSkillCheck(actor, skillID, dc)
    }
    
    // Success = learn one useful fact
    // Critical Success = learn additional/more specific fact
    // Failure = no information
    // Critical Failure = false information (LLM provides misleading info)
    
    learned := res.Degree >= check.Success
    return learned, res
}
```

### 28.3 Treat Wounds
**Target File:** `pkg/rules/skill/actions.go`

Treat Wounds is a 10-minute exploration activity that heals HP using Medicine.

```go
// TreatWounds heals a target using Medicine skill
// src: rules/compendium/skills.md "Treat Wounds"
func TreatWounds(healer, patient *entity.Entity, dc int, naturalRoll int) (int, check.CheckResult) {
    if healer.SkillProficiencies[ability.SkillMedicine] < ability.Trained {
        return 0, check.CheckResult{Degree: check.Failure}
    }
    
    var res check.CheckResult
    if naturalRoll > 0 {
        res = PerformSkillCheckWithRoll(healer, ability.SkillMedicine, dc, naturalRoll)
    } else {
        res = PerformSkillCheck(healer, ability.SkillMedicine, dc)
    }
    
    healed := 0
    switch res.Degree {
    case check.CriticalSuccess:
        // 4d8 healing (or 4d8+10 at DC 30, 4d8+30 at DC 40)
        healed = rollDice(4, 8)
        if dc >= 40 {
            healed += 30
        } else if dc >= 30 {
            healed += 10
        }
    case check.Success:
        // 2d8 healing (or 2d8+10 at DC 30, 2d8+30 at DC 40)
        healed = rollDice(2, 8)
        if dc >= 40 {
            healed += 30
        } else if dc >= 30 {
            healed += 10
        }
    case check.CriticalFailure:
        // Deal 1d8 damage
        damage := rollDice(1, 8)
        patient.TakeDamage(damage, "slashing") // Treated badly
        healed = -damage
    }
    
    if healed > 0 {
        patient.Heal(healed)
    }
    
    // Apply immunity: can't Treat Wounds same patient for 1 hour
    patient.Conditions.Apply(condition.NewConditionWithDuration(
        condition.TreatWoundsImmunity,
        1, // 1 hour (tracked by LLM/session)
        "Treated by " + healer.ID,
    ))
    
    return healed, res
}
```

**Note:** Add `TreatWoundsImmunity` to conditions. This prevents repeated healing cheese.

---

## Implementation Milestones

| Milestone | Phases | Deliverable |
|-----------|--------|-------------|
| **M15** | 25 | Glue Actions: Interact, Drop Prone, Stand, Take Cover |
| **M16** | 26 | Movement Modes: Fly/Swim/Climb speed fields, mode-aware Stride |
| **M17** | 27 | Counteract System: Level-comparison check helper |
| **M18** | 28 | Odds & Ends: Hero Points, Recall Knowledge, Treat Wounds |

---

## Testing Strategy

```
testdata/
├── glue_actions/
│   ├── interact/
│   ├── drop_prone_stand/
│   └── take_cover_movement/
├── movement_modes/
│   ├── fly_speed/
│   └── swim_no_speed/
├── counteract/
│   ├── dispel_success/
│   └── dispel_crit_higher_level/
└── odds_ends/
    ├── hero_point_reroll/
    ├── hero_point_stabilise/
    ├── recall_knowledge/
    └── treat_wounds/
```

Each test should verify:
1. Action economy is correctly tracked
2. Conditions are applied/removed appropriately
3. Speed calculations respect current mode
4. Counteract degree-of-success logic is correct
5. Hero point spending and caps work correctly
6. Treat Wounds immunity prevents re-treatment
