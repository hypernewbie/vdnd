# Phase 18: Shield Mechanics

## Objective

Implement full shield support for PF2E: the Shield item type, Raise a Shield action, Shield Block reaction, and integration with the damage pipeline for reaction hooks.

---

## 1. Shield Struct

**Target File:** `pkg/rules/item/shield.go`

Shields are distinct from armour. They provide an AC bonus only when raised, can absorb damage via Shield Block, and have their own HP/Hardness.

```go
package item

import "uaa/vdnd/pkg/rules/trait"

type Shield struct {
    ID              string
    Name            string
    ACBonus         int           // Circumstance bonus when raised (+1 or +2)
    SpeedPenalty    int           // Negative number, 0 for most shields
    Hardness        int           // Damage absorbed before shield takes damage
    MaxHP           int           // Total HP
    CurrentHP       int           // Runtime HP tracking
    BT              int           // Broken Threshold (typically MaxHP/2)
    Bulk            int
    Traits          trait.TraitSet

    // Runtime state
    IsRaised bool
}

func NewShield(id, name string, acBonus, hardness, maxHP, bulk int, traits ...trait.TraitID) *Shield {
    return &Shield{
        ID:        id,
        Name:      name,
        ACBonus:   acBonus,
        Hardness:  hardness,
        MaxHP:     maxHP,
        CurrentHP: maxHP,
        BT:        maxHP / 2,
        Bulk:      bulk,
        Traits:    traits,
    }
}

// IsBroken returns true if shield HP is at or below Broken Threshold
func (s *Shield) IsBroken() bool {
    return s.CurrentHP <= s.BT
}

// IsDestroyed returns true if shield HP is 0 or less
func (s *Shield) IsDestroyed() bool {
    return s.CurrentHP <= 0
}

// TakeDamage applies damage to the shield, returns actual damage taken
func (s *Shield) TakeDamage(amount int) int {
    if s.IsDestroyed() {
        return 0
    }
    actual := amount
    if actual > s.CurrentHP {
        actual = s.CurrentHP
    }
    s.CurrentHP -= actual
    return actual
}

// Repair restores HP up to max
func (s *Shield) Repair(amount int) {
    s.CurrentHP += amount
    if s.CurrentHP > s.MaxHP {
        s.CurrentHP = s.MaxHP
    }
}

// Reset clears runtime state (for new encounter)
func (s *Shield) Reset() {
    s.IsRaised = false
}
```

### Standard Shields Registry

**Target File:** `pkg/rules/item/registry.go` (add to existing)

```go
var StandardShields = map[string]*Shield{
    "buckler": NewShield("buckler", "Buckler", 1, 3, 6, 1),
    "wooden_shield": NewShield("wooden_shield", "Wooden Shield", 2, 3, 12, 1),
    "steel_shield": NewShield("steel_shield", "Steel Shield", 2, 5, 20, 1),
    "tower_shield": NewShield("tower_shield", "Tower Shield", 2, 5, 20, 4),
}

func init() {
    // Tower shield has special trait
    StandardShields["tower_shield"].SpeedPenalty = -5
}
```

---

## 2. Entity Integration

**Target File:** `pkg/rules/entity/entity.go`

### 2.1 Add WornShield Field

Add to `Entity` struct:

```go
WornShield *item.Shield
```

Update `NewEntity()` to initialise as nil (already the zero value).

Update `Clone()` to handle shield:

```go
if e.WornShield != nil {
    shieldCopy := *e.WornShield
    clone.WornShield = &shieldCopy
}
```

### 2.2 Modify GetAC()

**Target File:** `pkg/rules/entity/combat.go`

The existing `GetAC()` needs to check for raised shield bonus:

```go
func (e *Entity) GetAC() int {
    // Base 10
    ac := 10

    // DEX modifier (capped by armour)
    dexMod := e.Abilities.Modifier(ability.Dexterity)
    if e.WornArmor != nil {
        dexMod = e.WornArmor.AppliedDexBonus(dexMod)
        ac += e.WornArmor.ACBonus // Item bonus from armour
    }
    ac += dexMod

    // Proficiency bonus
    prof := e.GetArmorProficiency()
    ac += prof.Bonus(e.Level)

    // Circumstance bonus from raised shield
    if e.WornShield != nil && e.WornShield.IsRaised && !e.WornShield.IsBroken() {
        ac += e.WornShield.ACBonus
    }

    // Condition penalties (flat-footed, clumsy, etc.)
    ac += e.Conditions.GetACModifier()

    return ac
}
```

---

## 3. Raise a Shield Action

**Target File:** `pkg/rules/combat/shield_actions.go`

```go
package combat

import (
    "uaa/vdnd/pkg/rules/ability"
    "uaa/vdnd/pkg/rules/entity"
    "uaa/vdnd/pkg/rules/trait"
)

// RaiseShieldAction - 1 action, grants shield's AC bonus until start of next turn
// src: rules/rules/actions/raise-a-shield.md
type RaiseShieldAction struct{}

func (r *RaiseShieldAction) Name() string             { return "Raise a Shield" }
func (r *RaiseShieldAction) Cost() ability.ActionCost { return ability.CostOne }

func (r *RaiseShieldAction) HasTrait(id trait.TraitID) bool {
    return false // No special traits
}

func (r *RaiseShieldAction) Validate(actor, target *entity.Entity, turn *TurnState) error {
    if actor.WornShield == nil {
        return fmt.Errorf("no shield equipped")
    }
    if actor.WornShield.IsBroken() {
        return fmt.Errorf("shield is broken")
    }
    if actor.WornShield.IsRaised {
        return fmt.Errorf("shield already raised")
    }
    return nil
}

func (r *RaiseShieldAction) Execute(actor, _ *entity.Entity, turn *TurnState) ability.ActionResult {
    if err := turn.SpendActions(r.Cost()); err != nil {
        return ability.ActionResult{Success: false, Description: err.Error()}
    }

    if err := r.Validate(actor, nil, turn); err != nil {
        return ability.ActionResult{Success: false, Description: err.Error()}
    }

    actor.WornShield.IsRaised = true

    desc := fmt.Sprintf("Shield raised (+%d AC)", actor.WornShield.ACBonus)
    return ability.ActionResult{Success: true, Description: desc}
}
```

### 3.1 Turn End Reset

**Target File:** `pkg/rules/combat/turn.go`

Add to `TurnState.EndTurn()` or create hook:

```go
// ResetShield should be called at the start of the entity's next turn
func ResetShieldState(actor *entity.Entity) {
    if actor.WornShield != nil {
        actor.WornShield.IsRaised = false
    }
}
```

In encounter turn advancement:

```go
func (e *Encounter) AdvanceTurn() {
    current := e.GetCurrentParticipant()
    if current != nil && current.Entity != nil {
        // Reset shield raised from previous turn
        ResetShieldState(current.Entity)
    }
    // ... advance to next participant
}
```

---

## 4. Shield Block Reaction

**Target File:** `pkg/rules/combat/shield_actions.go`

Shield Block is a **reaction** with trigger: "You take damage from a physical attack while you have your shield raised."

```go
// ShieldBlockReaction - Reaction to reduce incoming physical damage
// src: rules/rules/actions/shield-block.md
type ShieldBlockReaction struct{}

func (s *ShieldBlockReaction) Name() string { return "Shield Block" }

func (s *ShieldBlockReaction) TriggerType() ReactionTrigger {
    return TriggerOnDamageTaken
}

func (s *ShieldBlockReaction) CanUse(actor *entity.Entity, event ReactionEvent) bool {
    // Must have shield raised
    if actor.WornShield == nil || !actor.WornShield.IsRaised {
        return false
    }
    // Shield must not be broken
    if actor.WornShield.IsBroken() {
        return false
    }
    // Damage must be physical (slashing, piercing, bludgeoning)
    if !isPhysicalDamage(event.DamageType) {
        return false
    }
    return true
}

// Execute reduces damage by Hardness, deals remainder to both target and shield
func (s *ShieldBlockReaction) Execute(actor *entity.Entity, event *ReactionEvent) ShieldBlockResult {
    shield := actor.WornShield
    hardness := shield.Hardness

    // Reduce damage by hardness
    reducedDamage := event.Damage - hardness
    if reducedDamage < 0 {
        reducedDamage = 0
    }

    // Shield takes damage equal to the damage dealt (after hardness, but shield takes from original)
    // PF2E: "The shield prevents you from taking an amount of damage up to the shield's Hardness.
    //        You and the shield each take any remaining damage."
    shieldDamage := event.Damage - hardness
    if shieldDamage < 0 {
        shieldDamage = 0
    }
    shield.TakeDamage(shieldDamage)

    return ShieldBlockResult{
        DamageToActor:  reducedDamage,
        DamageToShield: shieldDamage,
        ShieldBroken:   shield.IsBroken(),
        ShieldDestroyed: shield.IsDestroyed(),
    }
}

type ShieldBlockResult struct {
    DamageToActor   int
    DamageToShield  int
    ShieldBroken    bool
    ShieldDestroyed bool
}

func isPhysicalDamage(dt item.DamageType) bool {
    return dt == item.DamageSlashing || dt == item.DamageSlashing || dt == item.DamageBludgeoning
}
```

---

## 5. Reaction System Integration

**Target File:** `pkg/rules/combat/reaction.go`

The damage pipeline needs to pause for reactions. Define the reaction interface:

```go
package combat

import "uaa/vdnd/pkg/rules/entity"

type ReactionTrigger int

const (
    TriggerOnDamageTaken ReactionTrigger = iota
    TriggerOnMovementInReach
    TriggerOnManipulateInReach
    TriggerOnAllyDamaged
)

type ReactionEvent struct {
    Trigger     ReactionTrigger
    Source      *entity.Entity // Who caused the trigger
    Target      *entity.Entity // Who is affected
    Damage      int            // For damage triggers
    DamageType  item.DamageType
    Position    string         // For movement triggers
}

type Reaction interface {
    Name() string
    TriggerType() ReactionTrigger
    CanUse(actor *entity.Entity, event ReactionEvent) bool
}

// ReactionQueue holds pending reactions for an event
type ReactionQueue struct {
    Event      ReactionEvent
    Available  []AvailableReaction
    Resolved   bool
}

type AvailableReaction struct {
    Actor    *entity.Entity
    Reaction Reaction
}
```

### 5.1 Damage Pipeline Hook

**Target File:** `pkg/rules/damage/pipeline.go`

Modify `ProcessDamage` to return a pending state when reactions are available:

```go
type DamageResult struct {
    FinalDamage      int
    DamageBlocked    int
    WasLethal        bool
    PendingReactions *combat.ReactionQueue // nil if no reactions available
}

func ProcessDamageWithReactions(target *entity.Entity, dmg DamageInstance, isCrit bool, encounter *encounter.Encounter) DamageResult {
    // Step 1: Calculate raw damage (resistances, weaknesses, immunities)
    rawDamage := calculateRawDamage(target, dmg)

    // Step 2: Check for available reactions (Shield Block, etc.)
    event := combat.ReactionEvent{
        Trigger:    combat.TriggerOnDamageTaken,
        Target:     target,
        Damage:     rawDamage,
        DamageType: dmg.Type,
    }

    available := findAvailableReactions(target, event, encounter)
    if len(available) > 0 {
        // Return without applying damage - caller must resolve reactions first
        return DamageResult{
            FinalDamage:      rawDamage,
            PendingReactions: &combat.ReactionQueue{Event: event, Available: available},
        }
    }

    // Step 3: Apply damage directly if no reactions
    target.ApplyDamage(rawDamage)
    return DamageResult{FinalDamage: rawDamage, WasLethal: target.CurrentHP <= 0}
}

func findAvailableReactions(target *entity.Entity, event combat.ReactionEvent, enc *encounter.Encounter) []combat.AvailableReaction {
    available := make([]combat.AvailableReaction, 0)

    // Check target's own reactions (Shield Block)
    shieldBlock := &combat.ShieldBlockReaction{}
    if shieldBlock.CanUse(target, event) {
        available = append(available, combat.AvailableReaction{Actor: target, Reaction: shieldBlock})
    }

    // TODO: Check allies for reactions like Liberating Step

    return available
}
```

### 5.2 CLI Reaction Flow

When reactions are pending, the CLI must pause and query:

```bash
$ vd action strike goblin paladin
# DAMAGE PENDING
**Target:** Paladin
**Incoming Damage:** 15 slashing

## Available Reactions
| Entity | Reaction | Description |
|--------|----------|-------------|
| paladin | Shield Block | Reduce damage by 5 (Hardness), shield takes remaining |

**Status:** PENDING_REACTION
**Event ID:** evt_dmg_001

$ vd react paladin shield_block

# REACTION: Shield Block
**Actor:** Paladin
**Shield:** Steel Shield (Hardness 5, HP 18/20)

**Damage Blocked:** 5
**Damage to Paladin:** 10
**Damage to Shield:** 10
**Shield HP:** 20 → 10

**Paladin HP:** 45 → 35
```

---

## 6. Tests

**Target File:** `pkg/rules/item/shield_test.go`

```go
package item_test

import (
    "testing"
    "uaa/vdnd/pkg/rules/item"
)

func TestShieldBasics(t *testing.T) {
    shield := item.NewShield("steel_shield", "Steel Shield", 2, 5, 20, 1)

    if shield.IsBroken() {
        t.Error("New shield should not be broken")
    }
    if shield.IsDestroyed() {
        t.Error("New shield should not be destroyed")
    }

    // Take damage below hardness - actually no, hardness is for blocking, shield takes raw damage
    shield.TakeDamage(8)
    if shield.CurrentHP != 12 {
        t.Errorf("Expected HP 12, got %d", shield.CurrentHP)
    }
    if shield.IsBroken() {
        t.Error("Shield should not be broken at 12 HP (BT=10)")
    }

    // Take more damage to break
    shield.TakeDamage(3)
    if shield.CurrentHP != 9 {
        t.Errorf("Expected HP 9, got %d", shield.CurrentHP)
    }
    if !shield.IsBroken() {
        t.Error("Shield should be broken at 9 HP (BT=10)")
    }
}

func TestShieldACBonus(t *testing.T) {
    shield := item.NewShield("steel_shield", "Steel Shield", 2, 5, 20, 1)

    if shield.ACBonus != 2 {
        t.Errorf("Expected AC bonus 2, got %d", shield.ACBonus)
    }

    // Raised state
    shield.IsRaised = true
    if !shield.IsRaised {
        t.Error("Shield should be raised")
    }

    // Break the shield
    shield.CurrentHP = 5
    if !shield.IsBroken() {
        t.Error("Shield should be broken")
    }
    // Broken shields can still be raised but shouldn't grant AC (handled in Entity.GetAC)
}
```

**Target File:** `pkg/rules/combat/shield_test.go`

```go
package combat_test

import (
    "testing"
    "uaa/vdnd/pkg/rules/combat"
    "uaa/vdnd/pkg/rules/entity"
    "uaa/vdnd/pkg/rules/item"
)

func TestRaiseShieldAction(t *testing.T) {
    actor := entity.NewEntity("test", "Test", 1)
    actor.WornShield = item.NewShield("steel_shield", "Steel Shield", 2, 5, 20, 1)

    turn := combat.NewTurnState(actor)
    action := &combat.RaiseShieldAction{}

    result := action.Execute(actor, nil, turn)
    if !result.Success {
        t.Errorf("Raise Shield should succeed: %s", result.Description)
    }
    if !actor.WornShield.IsRaised {
        t.Error("Shield should be raised after action")
    }
}

func TestShieldBlockReducesDamage(t *testing.T) {
    actor := entity.NewEntity("test", "Test", 1)
    actor.WornShield = item.NewShield("steel_shield", "Steel Shield", 2, 5, 20, 1)
    actor.WornShield.IsRaised = true

    reaction := &combat.ShieldBlockReaction{}
    event := &combat.ReactionEvent{
        Damage:     12,
        DamageType: item.DamageSlashing,
    }

    if !reaction.CanUse(actor, *event) {
        t.Error("Should be able to use Shield Block with raised shield")
    }

    result := reaction.Execute(actor, event)

    // Damage to actor = 12 - 5 (hardness) = 7
    if result.DamageToActor != 7 {
        t.Errorf("Expected 7 damage to actor, got %d", result.DamageToActor)
    }

    // Shield takes 7 damage
    if result.DamageToShield != 7 {
        t.Errorf("Expected 7 damage to shield, got %d", result.DamageToShield)
    }

    // Shield HP = 20 - 7 = 13
    if actor.WornShield.CurrentHP != 13 {
        t.Errorf("Expected shield HP 13, got %d", actor.WornShield.CurrentHP)
    }
}

func TestShieldBlockRequiresRaised(t *testing.T) {
    actor := entity.NewEntity("test", "Test", 1)
    actor.WornShield = item.NewShield("steel_shield", "Steel Shield", 2, 5, 20, 1)
    // Shield NOT raised

    reaction := &combat.ShieldBlockReaction{}
    event := combat.ReactionEvent{
        Damage:     10,
        DamageType: item.DamageSlashing,
    }

    if reaction.CanUse(actor, event) {
        t.Error("Should NOT be able to use Shield Block without raised shield")
    }
}

func TestBrokenShieldNoACBonus(t *testing.T) {
    actor := entity.NewEntity("test", "Test", 1)
    actor.Abilities.Set(ability.Dexterity, 14) // +2 mod
    actor.WornShield = item.NewShield("steel_shield", "Steel Shield", 2, 5, 20, 1)
    actor.WornShield.IsRaised = true

    acWithShield := actor.GetAC()

    // Break the shield
    actor.WornShield.CurrentHP = 5 // Below BT of 10

    acBroken := actor.GetAC()

    if acBroken >= acWithShield {
        t.Errorf("Broken shield should not grant AC bonus. With: %d, Broken: %d", acWithShield, acBroken)
    }
}
```

---

## 7. Execution Checklist

- [ ] Create `pkg/rules/item/shield.go` with Shield struct
- [ ] Add standard shields to `pkg/rules/item/registry.go`
- [ ] Add `WornShield` field to `Entity` struct
- [ ] Modify `Entity.Clone()` to copy shield
- [ ] Modify `Entity.GetAC()` to include raised shield bonus
- [ ] Create `pkg/rules/combat/shield_actions.go` with RaiseShieldAction
- [ ] Add ShieldBlockReaction to same file
- [ ] Create `pkg/rules/combat/reaction.go` with reaction types
- [ ] Modify turn advancement to reset shield state
- [ ] Create `pkg/rules/item/shield_test.go`
- [ ] Create `pkg/rules/combat/shield_test.go`
- [ ] Run `go test -v ./pkg/rules/...` and ensure 100% pass

---

## 8. CLI Commands

New commands to support:

```bash
# Raise shield
vd action raise_shield paladin

# Use Shield Block reaction
vd react paladin shield_block

# Check shield status
vd entity get paladin --field shield
# Output:
# **Shield:** Steel Shield
# - AC Bonus: +2
# - Hardness: 5
# - HP: 18/20 (BT 10)
# - Status: Raised
```
