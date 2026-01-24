# Phase 24: Familiars & Animal Companions

## Objective

Implement a **Minion** framework to support Familiars and Animal Companions. Minions are creatures that share a connection with a Master, derive some statistics from them, and rely on the "Command" action to act in combat.

---

## 1. Minion Definition

**Target File:** `pkg/rules/entity/minion.go`

We define concepts to distinguish minions from standard creatures.

```go
package entity

import "uaa/vdnd/pkg/rules/ability"

type MinionType int

const (
    MinionFamiliar MinionType = iota
    MinionAnimalCompanion
    MinionSummon
)

// MinionSettings defines the configurables for a minion
type MinionSettings struct {
    Type           MinionType
    MasterID       string
    IsCommanded    bool // Reset every round
    ActionsPerTurn int  // Usually 2 if commanded
}

// MinionAbilities tracks specific minion powers (e.g. "Fly Speed", "Manual Dexterity")
type MinionAbility string

const (
    MinionAbilityFly      MinionAbility = "fly"
    MinionAbilityDarkvision MinionAbility = "darkvision"
    MinionAbilitySpeech   MinionAbility = "speech"
    MinionAbilityTouch    MinionAbility = "touch_telepathy"
)
```

---

## 2. Entity Integration

**Target File:** `pkg/rules/entity/entity.go`

Add minion data to the main `Entity` struct.

```go
type Entity struct {
    // ... existing fields ...

    // Minion Data (nil if not a minion)
    Minion *MinionSettings
    
    // Master Data (ids of owned minions)
    MinionIDs []string
}
```

**Target File:** `pkg/rules/entity/minion_logic.go` (new file)

Implement the stat derivation logic. This must be called whenever the Master levels up or daily preparations occur.

```go
package entity

import "fmt"

// DeriveFamiliarStats updates a familiar's stats based on the master
// src: rules/rulebook/chapter-3-classes.md (Familiars)
func (e *Entity) DeriveFamiliarStats(master *Entity) error {
    if e.Minion == nil || e.Minion.Type != MinionFamiliar {
        return fmt.Errorf("entity is not a familiar")
    }

    // 1. Level = Master Level
    e.Level = master.Level

    // 2. AC = Master AC (before bonuses) + 2 (size/dex) ?
    // Actually raw rule: "Your familiar's save modifiers and AC are equal to yours..." 
    // but typically they don't get item bonuses. 
    // Implementation: Copy proficiency + DEX + Level. 
    // Simplified: Copy Master's AC.
    e.AC = master.GetAC() // Simplification for now

    // 3. HP = 5 * Level
    e.MaxHP = 5 * e.Level
    // Reset HP if needed, or keep damage? Usually generic update involves full heal or ratio
    if e.CurrentHP > e.MaxHP {
        e.CurrentHP = e.MaxHP
    }

    // 4. Saves = Master's Saves
    e.Fortitude = master.Fortitude
    e.Reflex = master.Reflex
    e.Will = master.Will
    
    // 5. Perception
    e.Perception = master.Perception

    // 6. Skills
    // "It can't use any actions that require a skill check unless it has the skill."
    // Typically uses master's level + spellcasting attribute or similar. 
    // For MVP: Copy master's Acrobatics/Stealth if present.
    
    return nil
}

// DeriveCompanionStats updates an animal companion
// src: rules/rulebook/chapter-3-classes.md (Animal Companions)
func (e *Entity) DeriveCompanionStats(master *Entity) error {
    if e.Minion == nil || e.Minion.Type != MinionAnimalCompanion {
        return fmt.Errorf("entity is not an animal companion")
    }
    
    // Companions scale differently (Mature, Nimble, Savage, etc.)
    // MVP: Just ensure Level matches Master for effect scaling
    e.Level = master.Level
    
    return nil
}
```

---

## 3. Command Action

**Target File:** `pkg/rules/combat/minion_actions.go`

One of the most distinct mechanical interactions. The Master spends 1 action to give the Minion 2 actions.

```go
package combat

import (
    "fmt"
    "uaa/vdnd/pkg/rules/ability"
    "uaa/vdnd/pkg/rules/entity"
    "uaa/vdnd/pkg/rules/trait"
)

// CommandMinionAction
// src: rules/actions/command-an-animal.md (adapted for minions)
type CommandMinionAction struct {
    TargetMinionID string
}

func (c *CommandMinionAction) Name() string             { return "Command Minion" }
func (c *CommandMinionAction) Cost() ability.ActionCost { return ability.CostOne }
func (c *CommandMinionAction) HasTrait(id trait.TraitID) bool {
    return id == trait.TraitAuditory || id == trait.TraitConcentrate
}

func (c *CommandMinionAction) Validate(actor, target *entity.Entity, turn *TurnState) error {
    if c.TargetMinionID == "" {
        return fmt.Errorf("no minion specified")
    }
    
    // Verify ownership
    owns := false
    for _, id := range actor.MinionIDs {
        if id == c.TargetMinionID {
            owns = true
            break
        }
    }
    
    if !owns {
        return fmt.Errorf("you do not own this minion")
    }
    
    return nil
}

func (c *CommandMinionAction) Execute(actor, _ *entity.Entity, turn *TurnState) ability.ActionResult {
    if err := turn.SpendActions(c.Cost()); err != nil {
        return ability.ActionResult{Success: false, Description: err.Error()}
    }

    // The caller (Encounter) needs to handle the logic of *finding* the minion and granting actions.
    // The Execute signature only gives us actor/target (which is nil here usually).
    // The caller (Encounter) needs to handle the logic of *finding* the minion and granting actions.
    // However, we can return a specific Result metadata that the Encounter interprets.
    // ARCHITECTURAL NOTE: Using Meta map to communicate side-effects decouples Action logic from Encounter state.
    
    return ability.ActionResult{
        Success: true, 
        Description: "Minion commanded",
        // The encounter loop must inspect this and grant actions
        Meta: map[string]interface{}{
            "GrantActions": 2,
            "TargetID": c.TargetMinionID,
        },
    }
}
```

---

## 4. Encounter Turn Logic for Minions

**Target File:** `pkg/rules/encounter/turn_logic.go` (Modification)

Minions participate in combat but function differently.

1.  **Initiative**: They (usually) share the Master's initiative.
2.  **Actions**: They start their turn with **0 actions** (unlike the standard 3). They only gain actions if Commanded (or if they are independent, which is rare/high level).
3.  **Reaction**: Minions typically do *not* get reactions unless specified.

**Implementation Strategy:**

In `StartTurn(participant)`:

```go
func (e *Encounter) StartTurn(p *Participant) {
    if p.Entity.Minion != nil {
        // Minions start with 0 actions by default
        // (Unless they have a specific ability like "Independent")
        p.TurnState.ActionsRemaining = 0
        p.TurnState.Reset()
    } else {
        // Normal entities get 3
        p.TurnState.ActionsRemaining = 3
    }
    // ...
}
```

In `ResolveAction(result)`:

```go
func (e *Encounter) HandleActionResult(result ability.ActionResult) {
    if grants, ok := result.Meta["GrantActions"]; ok {
        actions := grants.(int)
        targetID := result.Meta["TargetID"].(string)
        
        // Find minion participant
        minionP := e.GetParticipant(targetID)
        if minionP != nil {
            minionP.TurnState.ActionsRemaining += actions
            // Mark as commanded for the round
            if minionP.Entity.Minion != nil {
                minionP.Entity.Minion.IsCommanded = true
            }
        }
    }
}
```

---

## 5. Tests

**Target File:** `pkg/rules/entity/minion_test.go`

```go
package entity_test

import (
    "testing"
    "uaa/vdnd/pkg/rules/ability"
    "uaa/vdnd/pkg/rules/entity"
)

func TestDeriveFamiliarStats(t *testing.T) {
    master := entity.NewPC("wiz", "Wizard", 5, "Elf", "Wizard", "Scholar")
    master.Abilities.Set(ability.Constitution, 14) 
    
    fam := entity.NewEntity("cat", "Cat", 1)
    fam.Minion = &entity.MinionSettings{
        Type: entity.MinionFamiliar,
        MasterID: master.ID,
    }
    
    // Master HP: say 40
    master.MaxHP = 40
    master.CurrentHP = 40
    
    // Derive
    err := fam.DeriveFamiliarStats(master)
    if err != nil {
        t.Fatalf("Derive failed: %v", err)
    }
    
    // Check Level
    if fam.Level != 5 {
        t.Errorf("Familiar should take master level 5, got %d", fam.Level)
    }
    
    // Check HP (5 * Level = 25)
    if fam.MaxHP != 25 {
        t.Errorf("Familiar HP should be 25, got %d", fam.MaxHP)
    }
}
```

**Target File:** `pkg/rules/combat/minion_test.go`

```go
package combat_test

import (
    "testing"
    "uaa/vdnd/pkg/rules/combat"
    "uaa/vdnd/pkg/rules/entity"
)

func TestCommandMinion(t *testing.T) {
    master := entity.NewEntity("master", "Master", 1)
    master.MinionIDs = []string{"minion1"}
    
    turn := combat.NewTurnState(master)
    
    cmd := &combat.CommandMinionAction{TargetMinionID: "minion1"}
    
    // 1. Validation
    err := cmd.Validate(master, nil, turn)
    if err != nil {
        t.Errorf("Validation failed: %v", err)
    }
    
    // 2. Execution
    res := cmd.Execute(master, nil, turn)
    if !res.Success {
        t.Error("Command should succeed")
    }
    
    if res.Meta["GrantActions"] != 2 {
        t.Error("Should grant 2 actions")
    }
    
    // 3. Cost
    if turn.ActionsSpent != 1 {
        t.Error("Should cost 1 action")
    }
}
```

---

## 6. Execution Checklist

- [ ] Modify `Entity` struct in `pkg/rules/entity/entity.go` to include `Minion` (*MinionSettings) and `MinionIDs`.
- [ ] Create `pkg/rules/entity/minion.go` with enums and struct definitions.
- [ ] Implement `DeriveFamiliarStats` in `pkg/rules/entity/minion_logic.go`.
- [ ] Create `CommandMinionAction` in `pkg/rules/combat/minion_actions.go`.
- [ ] Update Encounter logic to handle `ActionGrant` metadata (or simpler direct modification if architecture checks out).
- [ ] Add tests for derivation and command.
- [ ] Run `go test -v ./pkg/rules/...`.

---

## 7. CLI Commands

```bash
# Add a familiar
vd entity spawn "Cat" --id fam1 --type familiar --master paladin

# Sync stats (e.g. after level up)
vd minion sync fam1

# In combat: Command
vd action command paladin --minion fam1
# Output:
# **Command Minion**
# paladin commands Cat (fam1).
# Cat gains 2 actions.

# Minion takes turn (immediately or after master)
vd action stride fam1 --to "goblins"
```
