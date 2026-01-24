# Pathfinder 2E Rules Engine - Implementation Plan (Part 2)

This document covers mechanical gaps identified after a review of the existing codebase against the PF2E rulebook. These features were either partially implemented or entirely missing from the original RULES_PLAN.md phases.

---

## Architecture Notes

The existing codebase follows these patterns which new implementation should respect:

| Pattern | How It Works |
|---------|--------------|
| **Action Interface** | Actions implement `Name()`, `Cost()`, `HasTrait()`, `Validate()`, `Execute()` |
| **Skill Functions** | Standalone functions in `pkg/rules/skill/actions.go` returning `check.CheckResult` or `(value, check.CheckResult)` |
| **Effect Application** | Side effects (conditions, damage) applied inside the function, not returned for caller to apply |
| **Combat Actions** | Wrappers in `pkg/rules/combat/` that handle turn state, MAP, and action cost |
| **State Tracking** | `ConditionTracker`, `AfflictionTracker`, `FeatTracker` on Entity struct |

---

## Phase 18: Shield Mechanics

**Priority:** High — Shields are fundamental defensive equipment.

### 18.1 Shield Struct
**Target File:** `pkg/rules/item/shield.go`

Shields are distinct from armour in PF2E. They provide AC only when raised, can block damage, and break.

```go
type Shield struct {
    ID           string
    Name         string
    ACBonus      int           // +1 or +2 when raised
    Hardness     int           // Damage reduction when blocking
    MaxHP        int           // Break threshold
    CurrentHP    int           // Current durability
    BrokenThreshold int        // Typically half MaxHP
    Bulk         int
    Traits       trait.TraitSet
    
    // Runtime state
    IsRaised     bool          // Set by Raise a Shield action
    BrokenValue  int           // Once CurrentHP <= this, shield is broken
}
```

**Key Methods:**
- `NewShield(id, name string, ac, hardness, hp, bulk int) *Shield`
- `(s *Shield) Block(damage int) int` — Absorbs damage up to Hardness, takes remaining
- `(s *Shield) IsBroken() bool`
- `(s *Shield) IsDestroyed() bool` — When HP reaches 0

### 18.2 Entity Integration
**Target File:** `pkg/rules/entity/entity.go`

Add field:
```go
WornShield *item.Shield
```

Modify `GetAC()`:
```go
// If shield is raised, add its ACBonus as a circumstance bonus
if e.WornShield != nil && e.WornShield.IsRaised && !e.WornShield.IsBroken() {
    circBonus = max(circBonus, e.WornShield.ACBonus)
}
```

### 18.3 Raise a Shield Action
**Target File:** `pkg/rules/combat/shield_actions.go`

```go
type RaiseShieldAction struct{}

func (r *RaiseShieldAction) Name() string            { return "Raise a Shield" }
func (r *RaiseShieldAction) Cost() ability.ActionCost { return ability.CostOne }

func (r *RaiseShieldAction) Execute(actor, _ *entity.Entity, turn *TurnState) ability.ActionResult {
    if actor.WornShield == nil {
        return ability.ActionResult{Success: false, Description: "No shield equipped"}
    }
    if actor.WornShield.IsBroken() {
        return ability.ActionResult{Success: false, Description: "Shield is broken"}
    }
    actor.WornShield.IsRaised = true
    return ability.ActionResult{Success: true, Description: "Shield raised (+2 AC)"}
}
```

**Turn End Reset:**
In `combat.TurnState.EndTurn()`, reset shield raised state:
```go
if actor.WornShield != nil {
    actor.WornShield.IsRaised = false
}
```

### 18.4 Shield Block Reaction
**Target File:** `pkg/rules/combat/shield_actions.go`

Shield Block is a *reaction* triggered when the entity takes damage while shield is raised.

```go
type ShieldBlockReaction struct{}

func (s *ShieldBlockReaction) Name() string { return "Shield Block" }
func (s *ShieldBlockReaction) TriggerCondition() string { return "You take damage while shield raised" }

func (s *ShieldBlockReaction) Execute(actor *entity.Entity, incomingDamage int) (int, int) {
    shield := actor.WornShield
    if shield == nil || !shield.IsRaised || shield.IsBroken() {
        return incomingDamage, 0
    }
    
    reducedDamage := max(0, incomingDamage - shield.Hardness)
    shieldDamage := min(shield.Hardness, incomingDamage)
    
    shield.CurrentHP -= shieldDamage
    return reducedDamage, shieldDamage
}
```

**Integration Point:** The damage pipeline (`pkg/rules/damage/pipeline.go`) needs a hook for reactions. Currently it applies damage directly. We need:
1. Pause before HP reduction
2. Check for Shield Block availability
3. Allow reaction execution
4. Continue with reduced damage

---

## Phase 19: Inventory & Bulk System

**Priority:** Medium — Enables proper equipment management.

### 19.1 Inventory Struct
**Target File:** `pkg/rules/entity/inventory.go`

```go
type InventorySlot struct {
    ItemID   string
    Quantity int
    Bulk     int // Pre-calculated for stacking
}

type Inventory struct {
    Slots      []InventorySlot
    CoinsGold   int
    CoinsSilver int
    CoinsCopper int
}

func (inv *Inventory) TotalBulk() int {
    total := 0
    for _, slot := range inv.Slots {
        total += slot.Bulk
    }
    // 10 coins = 1 Light bulk, 10 Light = 1 Bulk
    coinBulk := (inv.CoinsGold + inv.CoinsSilver + inv.CoinsCopper) / 1000
    return total + coinBulk
}
```

### 19.2 Weapon Bulk
**Target File:** `pkg/rules/item/weapon.go`

Add field to `Weapon`:
```go
Bulk int
```

### 19.3 Encumbrance Calculation
**Target File:** `pkg/rules/entity/entity.go`

```go
func (e *Entity) BulkLimit() int {
    return 5 + e.Abilities.Modifier(ability.Strength)
}

func (e *Entity) EncumberedLimit() int {
    return e.BulkLimit() + 5
}

func (e *Entity) CurrentBulk() int {
    total := e.Inventory.TotalBulk()
    if e.WornArmor != nil {
        total += e.WornArmor.Bulk
    }
    if e.WornShield != nil {
        total += e.WornShield.Bulk
    }
    for _, w := range e.WieldedWeapons {
        total += w.Bulk
    }
    return total
}

func (e *Entity) UpdateEncumbranceCondition() {
    bulk := e.CurrentBulk()
    if bulk > e.EncumberedLimit() {
        // Cannot move
        e.Conditions.Apply(condition.NewCondition(condition.Immobilized, "Over-encumbered"))
    } else if bulk > e.BulkLimit() {
        e.Conditions.Apply(condition.NewCondition(condition.Encumbered, "Encumbered"))
        e.Conditions.Remove(condition.Immobilized)
    } else {
        e.Conditions.Remove(condition.Encumbered)
        e.Conditions.Remove(condition.Immobilized)
    }
}
```

### 19.4 Encumbered Condition Effects
**Target File:** `pkg/rules/condition/effects.go`

Encumbered applies:
- Clumsy 1
- -10 ft penalty to all speeds

```go
func (c ConditionID) SpeedPenalty() int {
    if c == Encumbered {
        return -10
    }
    return 0
}
```

---

## Phase 20: Crafting Skill Actions

**Priority:** Low — Important for downtime but not combat.

### 20.1 Craft Action
**Target File:** `pkg/rules/skill/actions.go`

Crafting uses a 4-day cycle with daily progress checks.

```go
type CraftProgress struct {
    ItemID          string
    TargetCost      int // In copper
    SpentCost       int // Materials consumed
    DaysWorked      int
    SuccessProgress int // In copper worth of work done
}

// Craft performs a daily crafting check
// src: rules/compendium/skills.md "Craft"
func Craft(actor *entity.Entity, progress *CraftProgress, dc int, naturalRoll int) (int, check.CheckResult) {
    if actor.SkillProficiencies[ability.SkillCrafting] < ability.Trained {
        return 0, check.CheckResult{Degree: check.Failure}
    }
    
    var res check.CheckResult
    if naturalRoll > 0 {
        res = PerformSkillCheckWithRoll(actor, ability.SkillCrafting, dc, naturalRoll)
    } else {
        res = PerformSkillCheck(actor, ability.SkillCrafting, dc)
    }
    
    earnedProgress := 0
    switch res.Degree {
    case check.CriticalSuccess:
        earnedProgress = EarnIncomeAmount(actor.Level) * 2
    case check.Success:
        earnedProgress = EarnIncomeAmount(actor.Level)
    case check.CriticalFailure:
        // Lose 10% of materials
        progress.SpentCost = int(float64(progress.SpentCost) * 0.9)
    }
    
    progress.SuccessProgress += earnedProgress
    progress.DaysWorked++
    return earnedProgress, res
}
```

### 20.2 Repair Action
**Target File:** `pkg/rules/skill/actions.go`

```go
// Repair fixes a damaged item (10 minute activity)
// src: rules/compendium/skills.md "Repair"
func Repair(actor *entity.Entity, itemLevel int, naturalRoll int) (bool, check.CheckResult) {
    dc := LevelBasedDC(itemLevel)
    
    var res check.CheckResult
    if naturalRoll > 0 {
        res = PerformSkillCheckWithRoll(actor, ability.SkillCrafting, dc, naturalRoll)
    } else {
        res = PerformSkillCheck(actor, ability.SkillCrafting, dc)
    }
    
    repaired := res.Degree >= check.Success
    return repaired, res
}
```

### 20.3 Earn Income (Downtime)
**Target File:** `pkg/rules/skill/actions.go`

```go
// EarnIncomeAmount returns copper per day based on level
// src: rules/rules/tables/dc-by-level.md (Earn Income column)
func EarnIncomeAmount(level int) int {
    table := []int{
        10,   // Level 0: 1 sp
        20,   // Level 1: 2 sp
        30,   // Level 2: 3 sp
        50,   // Level 3: 5 sp
        70,   // Level 4: 7 sp
        // ... continues
    }
    if level < 0 { level = 0 }
    if level >= len(table) { level = len(table) - 1 }
    return table[level]
}

func EarnIncome(actor *entity.Entity, skillID ability.SkillID, taskLevel int, naturalRoll int) (int, check.CheckResult) {
    dc := LevelBasedDC(taskLevel)
    
    var res check.CheckResult
    if naturalRoll > 0 {
        res = PerformSkillCheckWithRoll(actor, skillID, dc, naturalRoll)
    } else {
        res = PerformSkillCheck(actor, skillID, dc)
    }
    
    earned := 0
    switch res.Degree {
    case check.CriticalSuccess:
        earned = EarnIncomeAmount(taskLevel + 1)
    case check.Success:
        earned = EarnIncomeAmount(taskLevel)
    case check.Failure:
        earned = EarnIncomeAmount(taskLevel) / 2
    }
    
    return earned, res
}
```

---

## Phase 21: Rituals

**Priority:** Low — Rarely used in combat scenarios.

### 21.1 Ritual Struct
**Target File:** `pkg/rules/spell/ritual.go`

Rituals differ from spells: they take hours, require secondary casters, and have monetary costs.

```go
type Ritual struct {
    ID               string
    Name             string
    Rank             SpellRank
    Traits           trait.TraitSet
    CastingTime      string         // "1 hour", "1 day", etc.
    Cost             int            // In gold
    SecondaryCasters int            // Minimum required
    PrimaryCheck     ability.SkillID
    SecondaryChecks  []ability.SkillID
    HeightenedCost   int            // Additional cost per rank above base
    
    Effect RitualEffect
}

type RitualCastAttempt struct {
    PrimaryResult     check.CheckResult
    SecondaryResults  []check.CheckResult
    TotalDegree       check.DegreeOfSuccess
}

type RitualEffect interface {
    Apply(attempt RitualCastAttempt, caster *entity.Entity, targets []*entity.Entity) RitualOutcome
}

type RitualOutcome struct {
    Success     bool
    Description string
    Backlash    string // On critical failure
}
```

### 21.2 Ritual Casting Logic
**Target File:** `pkg/rules/spell/ritual_casting.go`

```go
func CastRitual(ritual *Ritual, primary *entity.Entity, secondaries []*entity.Entity, dc int) RitualCastAttempt {
    attempt := RitualCastAttempt{}
    
    // Primary check
    attempt.PrimaryResult = skill.PerformSkillCheck(primary, ritual.PrimaryCheck, dc)
    
    // Secondary checks (each caster picks from available checks)
    for i, sec := range secondaries {
        checkSkill := ritual.SecondaryChecks[i % len(ritual.SecondaryChecks)]
        attempt.SecondaryResults = append(attempt.SecondaryResults, 
            skill.PerformSkillCheck(sec, checkSkill, dc))
    }
    
    // Determine overall degree
    // Each secondary crit failure = -2 to primary, each crit success = +2
    modifier := 0
    for _, r := range attempt.SecondaryResults {
        switch r.Degree {
        case check.CriticalSuccess:
            modifier += 2
        case check.CriticalFailure:
            modifier -= 2
        }
    }
    
    attempt.TotalDegree = adjustDegree(attempt.PrimaryResult.Degree, modifier)
    return attempt
}
```

---

## Phase 22: Gamemastery Subsystems (Victory Points)

**Priority:** Low — Used for narrative encounters, not tactical combat.

### 22.1 Subsystem Framework
**Target File:** `pkg/rules/subsystem/victory_points.go`

Victory Points (VP) are a generic framework for tracking progress in non-combat challenges.

```go
type Subsystem struct {
    ID              string
    Name            string
    TargetVP        int
    CurrentVP       int
    FailureThreshold int // Negative VP that ends in failure
    RoundsLimit     int  // 0 = no time limit
    CurrentRound    int
    
    // Track who has contributed
    Contributions   map[string]int // EntityID -> VP contributed
}

func NewSubsystem(id, name string, targetVP, failureThreshold, rounds int) *Subsystem {
    return &Subsystem{
        ID:               id,
        Name:             name,
        TargetVP:         targetVP,
        FailureThreshold: failureThreshold,
        RoundsLimit:      rounds,
        Contributions:    make(map[string]int),
    }
}

func (s *Subsystem) Contribute(entityID string, degree check.DegreeOfSuccess, vpOnSuccess, vpOnCrit int) int {
    delta := 0
    switch degree {
    case check.CriticalSuccess:
        delta = vpOnCrit
    case check.Success:
        delta = vpOnSuccess
    case check.CriticalFailure:
        delta = -1
    }
    
    s.CurrentVP += delta
    s.Contributions[entityID] += delta
    return delta
}

func (s *Subsystem) IsComplete() bool { return s.CurrentVP >= s.TargetVP }
func (s *Subsystem) IsFailed() bool   { return s.CurrentVP <= s.FailureThreshold }
```

### 22.2 Chase Subsystem
**Target File:** `pkg/rules/subsystem/chase.go`

A Chase is a VP subsystem with position tracking.

```go
type ChaseParticipant struct {
    EntityID string
    Position int // Higher = further ahead
    HasActed bool
}

type Chase struct {
    Subsystem
    Participants []ChaseParticipant
    Obstacles    []ChaseObstacle
}

type ChaseObstacle struct {
    Position    int
    Skill       ability.SkillID
    DC          int
    Description string
}
```

---

## Phase 23: Complex Hazard Integration

**Priority:** Medium — Hazard struct exists but encounter integration is incomplete.

### 23.1 Hazard as Encounter Participant
**Target File:** `pkg/rules/encounter/encounter.go`

Add a union type for participants:

```go
type ParticipantType int

const (
    ParticipantEntity ParticipantType = iota
    ParticipantHazard
)

type Participant struct {
    Type       ParticipantType
    Entity     *entity.Entity  // If Type == ParticipantEntity
    Hazard     *hazard.Hazard  // If Type == ParticipantHazard
    Initiative int
    HasActed   bool
    IsDelaying bool
    TurnState  *combat.TurnState
}
```

### 23.2 Hazard Turn Actions
**Target File:** `pkg/rules/hazard/actions.go`

Complex hazards get a turn to perform their routine:

```go
type HazardRoutine struct {
    Actions []HazardAction
}

func (h *Hazard) TakeTurn(encounter *encounter.Encounter) []HazardResult {
    if h.Complexity != ComplexityComplex || h.IsDisabled {
        return nil
    }
    
    results := make([]HazardResult, 0)
    for _, action := range h.Routine.Actions {
        result := action.Execute(h, encounter)
        results = append(results, result)
    }
    return results
}
```

### 23.3 Reset Logic
Complex hazards may reset after being disabled:

```go
func (h *Hazard) AttemptReset() bool {
    if h.ResetCondition == "" {
        return false
    }
    h.IsTriggered = false
    return true
}
```

---

## Phase 24: Familiars & Animal Companions

**Priority:** Medium — Common character options.

### 24.1 Minion Framework
**Target File:** `pkg/rules/entity/minion.go`

Minions are entities with reduced agency and derived stats.

```go
type MinionType int

const (
    MinionFamiliar MinionType = iota
    MinionAnimalCompanion
    MinionSummoned
)

type Minion struct {
    *Entity
    Type      MinionType
    MasterID  string
    
    // Minion-specific
    ActionsPerTurn int // Familiars get 2 if commanded
}

func (m *Minion) DeriveFamiliarStats(master *Entity) {
    // Familiars use master's level for proficiency
    m.Level = master.Level
    
    // AC = master's AC (before item bonuses) - 2
    // HP = 5 * master level
    m.MaxHP = 5 * master.Level
    m.CurrentHP = m.MaxHP
    
    // Perception, saves use master's modifiers
    m.Perception = master.Perception
    m.Fortitude = master.Fortitude
    m.Reflex = master.Reflex
    m.Will = master.Will
}
```

### 24.2 Command Action
**Target File:** `pkg/rules/combat/minion_actions.go`

```go
type CommandMinionAction struct {
    TargetMinionID string
}

func (c *CommandMinionAction) Name() string            { return "Command" }
func (c *CommandMinionAction) Cost() ability.ActionCost { return ability.CostOne }

func (c *CommandMinionAction) Execute(actor, _ *entity.Entity, turn *TurnState) ability.ActionResult {
    // Minion gains 2 actions this turn
    // Implementation: Set flag on minion's TurnState when their turn comes
    return ability.ActionResult{Success: true, Description: "Minion commanded"}
}
```

---

## Implementation Milestones

| Milestone | Phases | Deliverable |
|-----------|--------|-------------|
| **M9** | 18 | Shield mechanics, Raise Shield, Shield Block |
| **M10** | 19 | Inventory system, Bulk tracking, Encumbrance |
| **M11** | 20, 21 | Crafting actions, Ritual framework |
| **M12** | 22 | Victory Point subsystems (Chase, Research) |
| **M13** | 23 | Complex Hazard turn integration |
| **M14** | 24 | Minion/Familiar/Companion support |

---

## Architectural Insights

### 1. Reaction System Gap

The biggest architectural gap is that the damage pipeline (`pkg/rules/damage/pipeline.go`) and strike execution don't have hooks for reactions. Shield Block, Parry, and similar reactions need:

```
Trigger → Pause → Offer Reactions → Execute Chosen Reaction → Resume
```

This isn't just Shield Block; it's needed for:
- Attack of Opportunity (already identified in RULES_PLAN.md)
- Retributive Strike
- Liberating Step
- Shield Block

**Recommendation:** Add an `EventBus` or `ReactionQueue` to the encounter that intercepts at damage application time.

### 2. Duration Tracking

Many effects (Raise Shield, Feint's flat-footed, buffs) last "until end of next turn" or "until start of your turn". The current `TurnState` resets on turn end, but there's no cross-turn duration tracker.

**Recommendation:** Add duration tracking to `ConditionInstance`:
```go
type DurationTrigger int
const (
    DurationEndOfYourTurn DurationTrigger = iota
    DurationStartOfYourTurn
    DurationEndOfTargetsTurn
    DurationRounds
)
```

### 3. Item State vs Entity State

Shields have mutable state (`IsRaised`, `CurrentHP`). If an entity has multiple shields in inventory, only one is "active". The `WornShield` field handles this, but inventory items need a clear distinction between "in inventory" vs "actively equipped".

**Recommendation:** Use explicit equipment slots:
```go
type EquipmentSlots struct {
    Armor     *item.Armor
    Shield    *item.Shield
    MainHand  *item.Weapon
    OffHand   interface{} // Weapon or Shield
}
```

### 4. LLM Orchestration Implications

These new systems affect CLI output:
- Shield HP should appear in entity status
- Inventory weight/bulk limits
- Minion summaries under master

The `vd entity get` and `vd status` commands need extended output formats.

---

## Testing Strategy for New Phases

```
testdata/
├── shields/
│   ├── raise_shield_ac_bonus/
│   ├── shield_block_damage/
│   └── shield_break_threshold/
├── inventory/
│   ├── bulk_calculation/
│   └── encumbrance_conditions/
├── crafting/
│   └── repair_check/
├── subsystems/
│   └── chase_victory_points/
└── minions/
    ├── familiar_derived_stats/
    └── command_action_granting/
```

Each scenario should have:
1. `setup.md` — Initial state
2. `commands.txt` — CLI commands to run
3. `expected_state.json` — Final state assertions
4. `expected_output.md` — Key output verification
