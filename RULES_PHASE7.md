# Phase 7: Actions & Combat

## Agent Prompt

You are implementing the action and combat system for a Pathfinder 2E rules engine in Go. This is the core of gameplay—the Strike action, movement, skill actions, and the action economy that governs turns.

**Your task:** Implement the `pkg/rules/combat` package with full test coverage.

**Prerequisites:** Phases 1-6 should be complete (especially entity, item, check, condition).

---

## Context

### Source References
- Actions: `rules/rules/actions/` (individual markdown files)
- Action economy: `rules/rules/core-rulebook/chapter-9-playing-the-game.md:15`
- Multiple Attack Penalty: `rules/rules/core-rulebook/chapter-9-playing-the-game.md:184`
- Strike: `rules/rules/core-rulebook/chapter-9-playing-the-game.md:237`

### Action Economy (src: chapter-1:153)
Each turn, a creature gets:
- **3 actions** (can be used for single actions or activities)
- **1 reaction** (used on triggers, refreshes at start of turn)
- **Unlimited free actions** (but only one per trigger)

### Action Costs
| Cost | Symbol | Examples |
|------|--------|----------|
| Free | ◇ | Drop item, Release |
| Reaction | ⟳ | Attack of Opportunity, Shield Block |
| 1 action | ◆ | Strike, Stride, Step, Raise Shield |
| 2 actions | ◆◆ | Most spells, Sudden Charge |
| 3 actions | ◆◆◆ | Some spells, complex activities |

### Multiple Attack Penalty (MAP)
| Attack # | Normal Penalty | Agile Weapon |
|----------|----------------|--------------|
| 1st | 0 | 0 |
| 2nd | -5 | -4 |
| 3rd+ | -10 | -8 |

MAP resets at the end of your turn. Reactions don't incur MAP.

### Core Actions to Implement

**Combat Actions:**
| Action | Cost | Effect |
|--------|------|--------|
| **Strike** | 1 | Make a melee or ranged attack |
| **Stride** | 1 | Move up to your Speed |
| **Step** | 1 | Move 5ft without triggering reactions |
| **Raise Shield** | 1 | +Circumstance bonus to AC until next turn |

**Skill Actions (Attack trait):**
| Action | Skill | Effect |
|--------|-------|--------|
| **Grapple** | Athletics | Grab target, make them grabbed |
| **Shove** | Athletics | Push target 5ft, possibly prone |
| **Trip** | Athletics | Knock target prone |
| **Disarm** | Athletics | Force target to drop item |

**Skill Actions (No attack trait):**
| Action | Skill | Effect |
|--------|-------|--------|
| **Demoralize** | Intimidation | Apply frightened condition |
| **Feint** | Deception | Make target flat-footed to you |
| **Hide** | Stealth | Become hidden |
| **Seek** | Perception | Find hidden creatures |

---

## File Structure

```
pkg/
└── rules/
    └── combat/
        ├── action.go       # ActionCost, Action interface
        ├── turn.go         # Turn state, action/reaction tracking
        ├── strike.go       # Strike action implementation
        ├── movement.go     # Stride, Step
        ├── skill_actions.go # Grapple, Shove, Trip, Demoralize, etc.
        ├── map.go          # Multiple Attack Penalty tracking
        └── combat_test.go
```

---

## Implementation Plan

### 1. `pkg/rules/combat/action.go`

```go
type ActionCost int
const (
    CostFree ActionCost = iota
    CostReaction
    CostOne
    CostTwo
    CostThree
)

// ActionResult represents the outcome of performing an action
type ActionResult struct {
    Success     bool
    Degree      check.DegreeOfSuccess  // For checks
    Description string                  // Human-readable outcome
    Damage      int                     // If damage was dealt
    Conditions  []condition.ConditionInstance  // Conditions applied
}

// Action interface for all executable actions
type Action interface {
    Name() string
    Cost() ActionCost
    HasTrait(trait.TraitID) bool
    // Validate checks if action can be performed
    Validate(actor *entity.Entity, target *entity.Entity) error
    // Execute performs the action and returns result
    Execute(actor *entity.Entity, target *entity.Entity, turn *TurnState) ActionResult
}
```

### 2. `pkg/rules/combat/turn.go`

Track action economy for a turn.

```go
type TurnState struct {
    Entity          *entity.Entity
    ActionsRemaining int  // Starts at 3 (modified by slowed/quickened)
    ReactionUsed    bool
    AttacksMade     int  // For MAP calculation
    
    // Track what happened this turn for sweep, forceful, etc.
    StrikesMade     []StrikeRecord
}

type StrikeRecord struct {
    TargetID    string
    Hit         bool
    WeaponID    string
}

func NewTurn(e *entity.Entity) *TurnState

// SpendActions deducts actions, returns error if not enough
func (t *TurnState) SpendActions(cost ActionCost) error

// SpendReaction marks reaction as used
func (t *TurnState) SpendReaction() error

// GetMAP returns current Multiple Attack Penalty
func (t *TurnState) GetMAP(isAgile bool) int

// RecordAttack increments attack counter for MAP
func (t *TurnState) RecordAttack()

// RecordStrike records a strike for sweep/forceful traits
func (t *TurnState) RecordStrike(record StrikeRecord)

// CanAct returns true if entity can take actions (not stunned, etc.)
func (t *TurnState) CanAct() bool
```

**NewTurn Pseudocode:**
```
func NewTurn(e *entity.Entity) *TurnState:
    actions := 3
    
    # Quickened grants +1 action
    if e.Conditions.Has(Quickened):
        actions += 1
    
    # Slowed reduces actions
    actions -= e.Conditions.Value(Slowed)
    
    # Stunned is handled differently (consumed as actions are taken)
    
    if actions < 0:
        actions = 0
    
    return &TurnState{
        Entity: e,
        ActionsRemaining: actions,
        ReactionUsed: false,
        AttacksMade: 0,
    }
```

**GetMAP Pseudocode:**
```
func (t *TurnState) GetMAP(isAgile bool) int:
    if t.AttacksMade == 0:
        return 0
    elif t.AttacksMade == 1:
        return -4 if isAgile else -5
    else:
        return -8 if isAgile else -10
```

### 3. `pkg/rules/combat/strike.go`

The Strike action—make an attack roll, deal damage on hit.

```go
type StrikeAction struct {
    Weapon *item.Weapon
}

func NewStrike(weapon *item.Weapon) *StrikeAction

func (s *StrikeAction) Name() string { return "Strike" }
func (s *StrikeAction) Cost() ActionCost { return CostOne }
func (s *StrikeAction) HasTrait(id trait.TraitID) bool {
    // Strike always has attack trait
    if id == trait.TraitAttack { return true }
    return s.Weapon.HasTrait(id)
}

func (s *StrikeAction) Validate(actor, target *entity.Entity) error {
    // Check range/reach
    // Check weapon is wielded
    // Check actor can act
}

func (s *StrikeAction) Execute(actor, target *entity.Entity, turn *TurnState) ActionResult
```

**Strike Execute Pseudocode:**
```
func (s *StrikeAction) Execute(actor, target *entity.Entity, turn *TurnState) ActionResult:
    # Calculate attack modifier
    attackMod := s.calculateAttackModifier(actor)
    
    # Apply MAP
    map := turn.GetMAP(s.Weapon.IsAgile())
    
    # Get all bonuses/penalties
    modifiers := []check.Modifier{
        {map, Untyped, "MAP"},
    }
    modifiers = append(modifiers, actor.Conditions.GetAttackModifiers()...)
    
    # Target's AC
    targetAC := target.GetAC()
    
    # Perform the check
    result := check.PerformCheck(attackMod, modifiers, targetAC)
    
    # Record attack for MAP
    turn.RecordAttack()
    
    # Process result
    switch result.Degree:
    case CriticalSuccess:
        damage := s.rollDamage(actor, true)
        target.TakeDamage(damage, s.Weapon.DamageType)
        turn.RecordStrike(StrikeRecord{target.ID, true, s.Weapon.ID})
        return ActionResult{Success: true, Degree: CriticalSuccess, Damage: damage * 2}
    
    case Success:
        damage := s.rollDamage(actor, false)
        target.TakeDamage(damage, s.Weapon.DamageType)
        turn.RecordStrike(StrikeRecord{target.ID, true, s.Weapon.ID})
        return ActionResult{Success: true, Degree: Success, Damage: damage}
    
    case Failure, CriticalFailure:
        turn.RecordStrike(StrikeRecord{target.ID, false, s.Weapon.ID})
        return ActionResult{Success: false, Degree: result.Degree}
```

**calculateAttackModifier Pseudocode:**
```
func (s *StrikeAction) calculateAttackModifier(actor *entity.Entity) int:
    # Determine ability modifier
    if s.Weapon.IsMelee:
        if s.Weapon.IsFinesse():
            abilityMod = max(actor.Abilities.Modifier(STR), actor.Abilities.Modifier(DEX))
        else:
            abilityMod = actor.Abilities.Modifier(STR)
    else:  # Ranged
        abilityMod = actor.Abilities.Modifier(DEX)
    
    # Proficiency bonus (simplified - would need weapon proficiency tracking)
    profBonus = actor.getWeaponProficiency(s.Weapon).Bonus(actor.Level)
    
    return abilityMod + profBonus
```

### 4. `pkg/rules/combat/movement.go`

```go
type StrideAction struct{}

func (s *StrideAction) Name() string { return "Stride" }
func (s *StrideAction) Cost() ActionCost { return CostOne }
func (s *StrideAction) HasTrait(id trait.TraitID) bool {
    return id == trait.TraitMove
}

func (s *StrideAction) Execute(actor *entity.Entity, destination string, turn *TurnState) ActionResult {
    // Update actor.Position
    // Check for reactions (AoO) - handled by reaction system
    // Disengage from current engagements if leaving
}

type StepAction struct{}

func (s *StepAction) Name() string { return "Step" }
func (s *StepAction) Cost() ActionCost { return CostOne }
func (s *StepAction) HasTrait(id trait.TraitID) bool {
    return id == trait.TraitMove
}

// Step: Move 5ft, doesn't trigger reactions
func (s *StepAction) Execute(actor *entity.Entity, direction string, turn *TurnState) ActionResult
```

### 5. `pkg/rules/combat/skill_actions.go`

```go
// Demoralize - Intimidation vs Will DC, applies Frightened
type DemoralizeAction struct{}

func (d *DemoralizeAction) Execute(actor, target *entity.Entity, turn *TurnState) ActionResult {
    // Actor's Intimidation check vs target's Will DC
    intimidation := actor.GetSkillModifier(SkillIntimidation)
    willDC := target.GetSaveDC(Will)
    
    result := check.PerformCheck(intimidation, nil, willDC)
    
    switch result.Degree:
    case CriticalSuccess:
        target.Conditions.Apply(NewValuedCondition(Frightened, 2, "Demoralize"))
        // Target immune to your Demoralize for 10 minutes
    case Success:
        target.Conditions.Apply(NewValuedCondition(Frightened, 1, "Demoralize"))
    case Failure:
        // Nothing happens
    case CriticalFailure:
        // Target immune for 10 minutes
    
    return ActionResult{Degree: result.Degree}
}

// Grapple - Athletics vs Fortitude DC, applies Grabbed
type GrappleAction struct{}

func (g *GrappleAction) Cost() ActionCost { return CostOne }
func (g *GrappleAction) HasTrait(id trait.TraitID) bool {
    return id == trait.TraitAttack  // Grapple has attack trait!
}

// Shove, Trip, Disarm similar structure
```

### 6. `pkg/rules/combat/map.go`

Isolated MAP logic for clarity.

```go
// CalculateMAP returns the penalty for the Nth attack
func CalculateMAP(attackNumber int, isAgile bool) int {
    if attackNumber <= 1 {
        return 0
    }
    if isAgile {
        if attackNumber == 2 { return -4 }
        return -8
    }
    if attackNumber == 2 { return -5 }
    return -10
}
```

---

## Test Plan

### Action Cost Tests

| Test | Expected |
|------|----------|
| Strike cost | CostOne |
| Stride cost | CostOne |
| Step cost | CostOne |
| Demoralize cost | CostOne |

### Turn State Tests

| Test | Setup | Expected |
|------|-------|----------|
| Fresh turn has 3 actions | NewTurn(healthy entity) | ActionsRemaining = 3 |
| Slowed 1 reduces to 2 | Entity with Slowed 1 | ActionsRemaining = 2 |
| Quickened adds 1 | Entity with Quickened | ActionsRemaining = 4 |
| Slowed 2 + Quickened | Both conditions | ActionsRemaining = 2 (3+1-2) |
| Spend 1 action | SpendActions(CostOne) | ActionsRemaining = 2 |
| Spend 2 actions | SpendActions(CostTwo) | ActionsRemaining = 1 |
| Can't overspend | 1 remaining, SpendActions(CostTwo) | Error |

### MAP Tests

| Attack # | Weapon | Expected Penalty |
|----------|--------|------------------|
| 1 | Normal | 0 |
| 1 | Agile | 0 |
| 2 | Normal | -5 |
| 2 | Agile | -4 |
| 3 | Normal | -10 |
| 3 | Agile | -8 |
| 4 | Normal | -10 (same as 3rd) |
| 4 | Agile | -8 |

### Strike Tests

| Test | Setup | Roll | Expected |
|------|-------|------|----------|
| Hit | Attack +8 vs AC 15 | Roll 10 (total 18) | Success, damage dealt |
| Miss | Attack +5 vs AC 20 | Roll 10 (total 15) | Failure |
| Crit by +10 | Attack +10 vs AC 15 | Roll 15 (total 25) | Critical Success, double damage |
| Nat 20 | Attack +5 vs AC 25 | Roll 20 | Success (upgraded from fail) |
| Finesse uses DEX | DEX +4, STR +1, finesse weapon | | Uses +4 |
| Non-finesse uses STR | DEX +4, STR +1, longsword | | Uses +1 |
| MAP applied | 2nd attack | | -5 penalty in roll |

### Skill Action Tests

| Action | Roll | Expected |
|--------|------|----------|
| Demoralize crit success | Intimidation beats Will by 10+ | Frightened 2 |
| Demoralize success | Intimidation beats Will | Frightened 1 |
| Demoralize failure | Intimidation < Will | No effect |
| Grapple success | Athletics beats Fort | Target is Grabbed |
| Trip success | Athletics beats Reflex | Target is Prone |
| Shove success | Athletics beats Fort | Target moves 5ft |

### Integration Tests

| Test | Expected |
|------|----------|
| Full turn: 3 strikes | All 3 execute, MAP applies: 0, -5, -10 |
| Stride + Strike + Strike | 3 actions used, MAP: 0, -5 |
| Can't take 4 actions | 4th action fails with error |

---

## Validation Checklist

- [ ] All tests pass
- [ ] `go vet` reports no issues
- [ ] `go fmt` applied
- [ ] MAP calculates correctly for agile vs non-agile
- [ ] Turn state tracks actions remaining
- [ ] Strike uses correct ability (STR/DEX/finesse)
- [ ] Skill actions apply correct conditions
- [ ] Attack trait actions increment MAP counter

---

## Notes for Implementation

1. **Attack trait matters:** Grapple, Shove, Trip, Disarm all have the attack trait and count toward MAP. Demoralize and Feint do NOT.

2. **Reactions not implemented here:** Attack of Opportunity, Shield Block, etc. are reactions. They'll be handled in a reaction system that triggers on events.

3. **Movement is abstract:** We're using zones, not exact squares. Stride moves between zones, Step is for minor repositioning.

4. **Weapon proficiency:** Entity needs to track proficiency per weapon category. Simplified for now.

5. **Damage calculation:** Strike's `rollDamage` needs to handle:
   - Base weapon dice
   - STR modifier (melee) or half STR (propulsive)
   - Deadly/fatal traits on crit
   - This integrates with Phase 8 (Damage Pipeline)

6. **Recording strikes:** The `StrikesMade` slice enables traits like:
   - **Sweep:** +1 if previous strike hit different target
   - **Forceful:** +1/+2 damage on 2nd/3rd+ strikes
