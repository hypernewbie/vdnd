# Phase 23: Complex Hazard Integration

## Objective

Integrate Complex Hazards into the encounter system. Complex hazards have initiative, take turns, and perform routines. This phase connects the existing `hazard` package with the `encounter` system.

---

## 1. Hazard Participant Type

**Target File:** `pkg/rules/encounter/participant.go`

Extend the participant system to handle both entities and hazards.

```go
package encounter

import (
    "uaa/vdnd/pkg/rules/combat"
    "uaa/vdnd/pkg/rules/entity"
    "uaa/vdnd/pkg/rules/hazard"
)

// ParticipantType distinguishes entities from hazards
type ParticipantType int

const (
    ParticipantEntity ParticipantType = iota
    ParticipantHazard
)

// Participant represents anyone/anything in initiative order
type Participant struct {
    Type       ParticipantType
    Entity     *entity.Entity   // If Type == ParticipantEntity
    Hazard     *hazard.Hazard   // If Type == ParticipantHazard
    
    // Common fields
    Initiative int
    HasActed   bool  // This round
    IsDelaying bool
    TurnState  *combat.TurnState // Only for entities
}

// GetID returns the identifier for this participant
func (p *Participant) GetID() string {
    if p.Type == ParticipantEntity && p.Entity != nil {
        return p.Entity.ID
    }
    if p.Type == ParticipantHazard && p.Hazard != nil {
        return p.Hazard.ID
    }
    return ""
}

// GetName returns display name
func (p *Participant) GetName() string {
    if p.Type == ParticipantEntity && p.Entity != nil {
        return p.Entity.Name
    }
    if p.Type == ParticipantHazard && p.Hazard != nil {
        return p.Hazard.Name
    }
    return "Unknown"
}

// IsActive returns true if participant can still act
func (p *Participant) IsActive() bool {
    if p.Type == ParticipantEntity && p.Entity != nil {
        return p.Entity.CurrentHP > 0
    }
    if p.Type == ParticipantHazard && p.Hazard != nil {
        return !p.Hazard.IsDisabled
    }
    return false
}

// NewEntityParticipant creates a participant from an entity
func NewEntityParticipant(e *entity.Entity) *Participant {
    return &Participant{
        Type:      ParticipantEntity,
        Entity:    e,
        TurnState: combat.NewTurnState(e),
    }
}

// NewHazardParticipant creates a participant from a hazard
func NewHazardParticipant(h *hazard.Hazard) *Participant {
    return &Participant{
        Type:   ParticipantHazard,
        Hazard: h,
    }
}
```

---

## 2. Hazard Routine System

**Target File:** `pkg/rules/hazard/routine.go`

Complex hazards have defined actions they take each turn.

```go
package hazard

import (
    "uaa/vdnd/pkg/rules/ability"
    "uaa/vdnd/pkg/rules/check"
    "uaa/vdnd/pkg/rules/dice"
    "uaa/vdnd/pkg/rules/entity"
)

// RoutineActionType defines what the hazard does
type RoutineActionType int

const (
    RoutineAttack RoutineActionType = iota
    RoutineSaveEffect
    RoutineAreaEffect
    RoutineReset
    RoutineSpecial
)

// RoutineAction represents one action in a hazard's routine
type RoutineAction struct {
    Name        string
    Type        RoutineActionType
    ActionCost  int // 1, 2, or 3 actions
    
    // For attacks
    AttackBonus int
    DamageDice  dice.DieRoll
    DamageType  string
    
    // For saves
    SaveType    ability.SaveType
    SaveDC      int
    SuccessEffect   string
    FailureEffect   string
    CritFailEffect  string
    
    // For area effects
    AffectsPosition string // Which position(s) affected
    
    // Description for special actions
    Description string
    
    // Custom effect function
    CustomEffect func(h *Hazard, targets []*entity.Entity) []HazardResult
}

// HazardRoutine defines all actions a complex hazard takes per turn
type HazardRoutine struct {
    Actions     []RoutineAction
    TotalActions int // How many actions the hazard has (usually 3)
}

// NewRoutine creates an empty routine
func NewRoutine(totalActions int) *HazardRoutine {
    return &HazardRoutine{
        Actions:      make([]RoutineAction, 0),
        TotalActions: totalActions,
    }
}

// AddAttack adds an attack action to the routine
func (r *HazardRoutine) AddAttack(name string, cost int, attackBonus int, damage dice.DieRoll, damageType string) *HazardRoutine {
    r.Actions = append(r.Actions, RoutineAction{
        Name:        name,
        Type:        RoutineAttack,
        ActionCost:  cost,
        AttackBonus: attackBonus,
        DamageDice:  damage,
        DamageType:  damageType,
    })
    return r
}

// AddSaveEffect adds a saving throw effect
func (r *HazardRoutine) AddSaveEffect(name string, cost int, saveType ability.SaveType, dc int, success, failure, critFail string) *HazardRoutine {
    r.Actions = append(r.Actions, RoutineAction{
        Name:           name,
        Type:           RoutineSaveEffect,
        ActionCost:     cost,
        SaveType:       saveType,
        SaveDC:         dc,
        SuccessEffect:  success,
        FailureEffect:  failure,
        CritFailEffect: critFail,
    })
    return r
}

// AddReset adds a reset action
func (r *HazardRoutine) AddReset(name string) *HazardRoutine {
    r.Actions = append(r.Actions, RoutineAction{
        Name:       name,
        Type:       RoutineReset,
        ActionCost: 1,
    })
    return r
}
```

---

## 3. Hazard Turn Execution

**Target File:** `pkg/rules/hazard/turn.go`

Execute a hazard's routine on its turn.

```go
package hazard

import (
    "uaa/vdnd/pkg/rules/check"
    "uaa/vdnd/pkg/rules/damage"
    "uaa/vdnd/pkg/rules/entity"
)

// TurnResult contains all results from a hazard's turn
type TurnResult struct {
    HazardID     string
    HazardName   string
    ActionResults []ActionResult
    TotalDamage  int
    WasReset     bool
}

// ActionResult contains the result of a single routine action
type ActionResult struct {
    ActionName  string
    ActionType  RoutineActionType
    Targets     []TargetResult
    Description string
}

// TargetResult contains what happened to a specific target
type TargetResult struct {
    EntityID    string
    EntityName  string
    Hit         bool
    SaveResult  check.DegreeOfSuccess
    Damage      int
    Effect      string
}

// TakeTurn executes the hazard's routine
func (h *Hazard) TakeTurn(targets []*entity.Entity) TurnResult {
    result := TurnResult{
        HazardID:      h.ID,
        HazardName:    h.Name,
        ActionResults: make([]ActionResult, 0),
    }
    
    if h.IsDisabled || h.Routine == nil {
        return result
    }
    
    actionsRemaining := h.Routine.TotalActions
    
    for _, action := range h.Routine.Actions {
        if actionsRemaining < action.ActionCost {
            continue
        }
        actionsRemaining -= action.ActionCost
        
        actionResult := h.executeAction(action, targets)
        result.ActionResults = append(result.ActionResults, actionResult)
        
        // Tally damage
        for _, tr := range actionResult.Targets {
            result.TotalDamage += tr.Damage
        }
        
        if action.Type == RoutineReset {
            result.WasReset = true
        }
    }
    
    return result
}

// executeAction runs a single routine action
func (h *Hazard) executeAction(action RoutineAction, targets []*entity.Entity) ActionResult {
    result := ActionResult{
        ActionName: action.Name,
        ActionType: action.Type,
        Targets:    make([]TargetResult, 0),
    }
    
    // Filter targets by position if needed
    affectedTargets := h.filterTargetsByPosition(targets, action.AffectsPosition)
    
    switch action.Type {
    case RoutineAttack:
        result = h.executeAttack(action, affectedTargets)
    case RoutineSaveEffect:
        result = h.executeSaveEffect(action, affectedTargets)
    case RoutineAreaEffect:
        result = h.executeAreaEffect(action, affectedTargets)
    case RoutineReset:
        h.Reset()
        result.Description = "Hazard resets for another activation"
    case RoutineSpecial:
        if action.CustomEffect != nil {
            hazardResults := action.CustomEffect(h, affectedTargets)
            for _, hr := range hazardResults {
                result.Targets = append(result.Targets, TargetResult{
                    EntityID:   hr.EntityID,
                    EntityName: hr.EntityName,
                    Damage:     hr.Damage,
                    Effect:     hr.Description,
                })
            }
        }
    }
    
    return result
}

func (h *Hazard) executeAttack(action RoutineAction, targets []*entity.Entity) ActionResult {
    result := ActionResult{
        ActionName: action.Name,
        ActionType: RoutineAttack,
        Targets:    make([]TargetResult, 0),
    }
    
    for _, target := range targets {
        tr := TargetResult{
            EntityID:   target.ID,
            EntityName: target.Name,
        }
        
        // Roll attack
        attackRoll := check.PerformCheck(action.AttackBonus, nil, target.GetAC())
        
        if attackRoll.Degree >= check.Success {
            tr.Hit = true
            dmgRoll := action.DamageDice.Roll()
            if attackRoll.Degree == check.CriticalSuccess {
                dmgRoll *= 2
            }
            tr.Damage = dmgRoll
            
            // Apply damage
            dmgInstance := damage.DamageInstance{
                Amount: dmgRoll,
                Type:   action.DamageType,
                Source: h.Name,
            }
            damage.ProcessDamage(target, dmgInstance, attackRoll.Degree == check.CriticalSuccess)
            
            tr.Effect = fmt.Sprintf("%d %s damage", tr.Damage, action.DamageType)
        } else {
            tr.Effect = "Missed"
        }
        
        result.Targets = append(result.Targets, tr)
    }
    
    return result
}

func (h *Hazard) executeSaveEffect(action RoutineAction, targets []*entity.Entity) ActionResult {
    result := ActionResult{
        ActionName: action.Name,
        ActionType: RoutineSaveEffect,
        Targets:    make([]TargetResult, 0),
    }
    
    for _, target := range targets {
        tr := TargetResult{
            EntityID:   target.ID,
            EntityName: target.Name,
        }
        
        // Roll save
        saveMod := target.GetSaveModifier(action.SaveType)
        saveResult := check.PerformCheck(saveMod, nil, action.SaveDC)
        tr.SaveResult = saveResult.Degree
        
        switch saveResult.Degree {
        case check.CriticalSuccess:
            tr.Effect = "Unaffected"
        case check.Success:
            tr.Effect = action.SuccessEffect
        case check.Failure:
            tr.Effect = action.FailureEffect
        case check.CriticalFailure:
            tr.Effect = action.CritFailEffect
        }
        
        result.Targets = append(result.Targets, tr)
    }
    
    return result
}

func (h *Hazard) executeAreaEffect(action RoutineAction, targets []*entity.Entity) ActionResult {
    // Similar to save effect but always affects all targets at position
    return h.executeSaveEffect(action, targets)
}

func (h *Hazard) filterTargetsByPosition(targets []*entity.Entity, position string) []*entity.Entity {
    if position == "" {
        // Affects all targets (usually those at hazard's position)
        filtered := make([]*entity.Entity, 0)
        for _, t := range targets {
            if t.Position == h.Position {
                filtered = append(filtered, t)
            }
        }
        return filtered
    }
    
    // Filter to specific position
    filtered := make([]*entity.Entity, 0)
    for _, t := range targets {
        if t.Position == position {
            filtered = append(filtered, t)
        }
    }
    return filtered
}

// Reset prepares the hazard to trigger again
func (h *Hazard) Reset() {
    h.IsTriggered = false
}
```

---

## 4. Encounter Integration

**Target File:** `pkg/rules/encounter/hazard_integration.go`

Add hazards to encounters and handle their turns.

```go
package encounter

import (
    "sort"
    "uaa/vdnd/pkg/rules/check"
    "uaa/vdnd/pkg/rules/dice"
    "uaa/vdnd/pkg/rules/hazard"
)

// AddHazard adds a hazard to the encounter
func (e *Encounter) AddHazard(h *hazard.Hazard) {
    if h.Complexity != hazard.ComplexityComplex {
        // Simple hazards don't join initiative
        return
    }
    
    e.Participants = append(e.Participants, NewHazardParticipant(h))
}

// RollHazardInitiative rolls initiative for all hazard participants
func (e *Encounter) RollHazardInitiative() {
    for i := range e.Participants {
        if e.Participants[i].Type == ParticipantHazard && e.Participants[i].Hazard != nil {
            h := e.Participants[i].Hazard
            // Hazards use their Initiative modifier
            roll := dice.DieRoll{Count: 1, Sides: 20}.Roll()
            e.Participants[i].Initiative = roll + h.Initiative
        }
    }
}

// ExecuteHazardTurn runs a hazard's routine
func (e *Encounter) ExecuteHazardTurn(hazardID string) hazard.TurnResult {
    p := e.GetParticipantByID(hazardID)
    if p == nil || p.Type != ParticipantHazard {
        return hazard.TurnResult{}
    }
    
    // Gather potential targets (all entities at hazard's position)
    targets := e.GetEntitiesAtPosition(p.Hazard.Position)
    
    // Execute turn
    result := p.Hazard.TakeTurn(targets)
    p.HasActed = true
    
    return result
}

// GetEntitiesAtPosition returns all entities at a given position
func (e *Encounter) GetEntitiesAtPosition(position string) []*entity.Entity {
    entities := make([]*entity.Entity, 0)
    for _, p := range e.Participants {
        if p.Type == ParticipantEntity && p.Entity != nil {
            if p.Entity.Position == position {
                entities = append(entities, p.Entity)
            }
        }
    }
    return entities
}

// GetParticipantByID finds a participant by their ID
func (e *Encounter) GetParticipantByID(id string) *Participant {
    for i := range e.Participants {
        if e.Participants[i].GetID() == id {
            return &e.Participants[i]
        }
    }
    return nil
}

// ProcessHazardTriggers checks if any hazards should trigger
func (e *Encounter) ProcessHazardTriggers(event ability.Event) []hazard.TriggerResult {
    results := make([]hazard.TriggerResult, 0)
    
    for _, p := range e.Participants {
        if p.Type != ParticipantHazard || p.Hazard == nil {
            continue
        }
        
        h := p.Hazard
        if h.IsDisabled {
            continue
        }
        
        if h.CheckTrigger(event) {
            // Hazard triggered, execute immediate effect
            targets := e.GetEntitiesAtPosition(h.Position)
            hazardResults := h.Activate(targets)
            
            results = append(results, hazard.TriggerResult{
                HazardID: h.ID,
                HazardName: h.Name,
                Triggered: true,
                Results: hazardResults,
            })
        }
    }
    
    return results
}
```

---

## 5. Complex Hazard Registry

**Target File:** `pkg/rules/hazard/registry.go`

Add some standard complex hazards.

```go
package hazard

import (
    "uaa/vdnd/pkg/rules/ability"
    "uaa/vdnd/pkg/rules/dice"
)

// StandardComplexHazards contains predefined complex hazards
var StandardComplexHazards = map[string]func() *Hazard{}

func init() {
    // Spinning Blade Pillar - Level 4 Complex Trap
    // src: rules/rules/gm/hazards/spinning-blade-pillar.md
    StandardComplexHazards["spinning_blade_pillar"] = func() *Hazard {
        h := NewHazard("spinning_blade_pillar", "Spinning Blade Pillar", 4)
        h.Type = HazardTrap
        h.Complexity = ComplexityComplex
        h.StealthDC = 23
        h.AC = 21
        h.Fortitude = 14
        h.Reflex = 10
        h.HP = 50
        h.Hardness = 10
        h.Initiative = 8
        
        h.DisableOptions = []DisableOption{
            {Skill: ability.SkillThievery, DC: 21, Description: "Jam the mechanism"},
        }
        
        h.Routine = NewRoutine(2).
            AddAttack("Blade Slash", 1, 15, dice.DieRoll{Count: 2, Sides: 8, Modifier: 5}, "slashing").
            AddAttack("Blade Slash", 1, 15, dice.DieRoll{Count: 2, Sides: 8, Modifier: 5}, "slashing")
        
        return h
    }
    
    // Poisoned Dart Gallery - Level 6 Complex Trap
    StandardComplexHazards["poisoned_dart_gallery"] = func() *Hazard {
        h := NewHazard("poisoned_dart_gallery", "Poisoned Dart Gallery", 6)
        h.Type = HazardTrap
        h.Complexity = ComplexityComplex
        h.StealthDC = 26
        h.AC = 24
        h.Fortitude = 15
        h.Reflex = 12
        h.HP = 60
        h.Hardness = 12
        h.Initiative = 10
        
        h.DisableOptions = []DisableOption{
            {Skill: ability.SkillThievery, DC: 24, Description: "Block the dart holes"},
            {Skill: ability.SkillAthletics, DC: 26, Description: "Smash the pressure plates"},
        }
        
        h.Routine = NewRoutine(3).
            AddAttack("Poison Dart", 1, 17, dice.DieRoll{Count: 1, Sides: 6, Modifier: 3}, "piercing").
            AddSaveEffect("Poison", 0, ability.SaveFortitude, 22,
                "No effect",
                "Sickened 1 and 1d6 poison damage",
                "Sickened 2 and 2d6 poison damage").
            AddAttack("Poison Dart", 1, 17, dice.DieRoll{Count: 1, Sides: 6, Modifier: 3}, "piercing")
        
        return h
    }
    
    // Flooding Room - Level 8 Complex Trap
    StandardComplexHazards["flooding_room"] = func() *Hazard {
        h := NewHazard("flooding_room", "Flooding Room", 8)
        h.Type = HazardTrap
        h.Complexity = ComplexityComplex
        h.StealthDC = 28
        h.AC = 26
        h.Fortitude = 18
        h.Reflex = 14
        h.HP = 80
        h.Hardness = 15
        h.Initiative = 12
        
        h.DisableOptions = []DisableOption{
            {Skill: ability.SkillThievery, DC: 28, Description: "Open the drainage grate"},
            {Skill: ability.SkillAthletics, DC: 30, Description: "Force open the door"},
        }
        
        h.Routine = NewRoutine(1).
            AddSaveEffect("Rising Waters", 1, ability.SaveReflex, 26,
                "Avoid worst of current, take half damage",
                "Swept off feet, 2d6 bludgeoning and prone",
                "Pulled under, 4d6 bludgeoning, prone, and grabbed by water")
        
        return h
    }
    
    // Haunted Stage - Level 5 Complex Haunt
    StandardComplexHazards["haunted_stage"] = func() *Hazard {
        h := NewHazard("haunted_stage", "Haunted Stage", 5)
        h.Type = HazardHaunt
        h.Complexity = ComplexityComplex
        h.StealthDC = 24
        h.Initiative = 9
        
        h.DisableOptions = []DisableOption{
            {Skill: ability.SkillReligion, DC: 22, Description: "Perform last rites"},
            {Skill: ability.SkillPerformance, DC: 24, Description: "Complete the unfinished play"},
        }
        
        h.Routine = NewRoutine(2).
            AddSaveEffect("Terrifying Visage", 1, ability.SaveWill, 22,
                "Unaffected",
                "Frightened 1",
                "Frightened 2 and fleeing").
            AddSaveEffect("Ghostly Props", 1, ability.SaveReflex, 20,
                "Dodge the flying objects",
                "2d6 bludgeoning from hurled props",
                "4d6 bludgeoning and knocked prone")
        
        return h
    }
}

// GetComplexHazard retrieves a hazard template by ID
func GetComplexHazard(id string) *Hazard {
    if factory, ok := StandardComplexHazards[id]; ok {
        return factory()
    }
    return nil
}

// ListComplexHazards returns all available complex hazard IDs
func ListComplexHazards() []string {
    ids := make([]string, 0, len(StandardComplexHazards))
    for id := range StandardComplexHazards {
        ids = append(ids, id)
    }
    return ids
}
```

---

## 6. Tests

**Target File:** `pkg/rules/hazard/complex_test.go`

```go
package hazard_test

import (
    "testing"
    "uaa/vdnd/pkg/rules/ability"
    "uaa/vdnd/pkg/rules/entity"
    "uaa/vdnd/pkg/rules/hazard"
)

func TestComplexHazardCreation(t *testing.T) {
    h := hazard.GetComplexHazard("spinning_blade_pillar")
    if h == nil {
        t.Fatal("Failed to create spinning blade pillar")
    }
    
    if h.Complexity != hazard.ComplexityComplex {
        t.Error("Should be complex hazard")
    }
    
    if h.Routine == nil {
        t.Error("Complex hazard should have routine")
    }
    
    if len(h.Routine.Actions) != 2 {
        t.Errorf("Expected 2 routine actions, got %d", len(h.Routine.Actions))
    }
}

func TestHazardTurn(t *testing.T) {
    h := hazard.GetComplexHazard("spinning_blade_pillar")
    h.Position = "trap_room"
    
    target := entity.NewEntity("victim", "Unfortunate Adventurer", 5)
    target.MaxHP = 40
    target.CurrentHP = 40
    target.Position = "trap_room"
    
    result := h.TakeTurn([]*entity.Entity{target})
    
    if result.HazardID != h.ID {
        t.Error("Result should have hazard ID")
    }
    
    if len(result.ActionResults) == 0 {
        t.Error("Should have action results")
    }
    
    t.Logf("Hazard dealt %d total damage", result.TotalDamage)
}

func TestHazardDisable(t *testing.T) {
    h := hazard.GetComplexHazard("spinning_blade_pillar")
    
    rogue := entity.NewEntity("rogue", "Skilled Rogue", 5)
    rogue.SkillProficiencies[ability.SkillThievery] = ability.Expert
    
    option := h.DisableOptions[0]
    result := h.AttemptDisable(rogue, option)
    
    t.Logf("Disable attempt: %v", result.Degree)
    
    if result.Degree >= check.Success && !h.IsDisabled {
        t.Error("Successful disable should disable hazard")
    }
}

func TestHazardReset(t *testing.T) {
    h := hazard.GetComplexHazard("spinning_blade_pillar")
    h.IsTriggered = true
    
    h.Reset()
    
    if h.IsTriggered {
        t.Error("Reset should clear triggered state")
    }
}

func TestHazardPositionFiltering(t *testing.T) {
    h := hazard.NewHazard("test", "Test Hazard", 1)
    h.Position = "room_a"
    
    inRoom := entity.NewEntity("in", "In Room", 1)
    inRoom.Position = "room_a"
    
    outOfRoom := entity.NewEntity("out", "Out of Room", 1)
    outOfRoom.Position = "room_b"
    
    targets := []*entity.Entity{inRoom, outOfRoom}
    
    // Internal filter method (tested via TakeTurn behavior)
    // Only inRoom should be affected
    
    h.Routine = hazard.NewRoutine(1).
        AddAttack("Test Attack", 1, 10, dice.DieRoll{Count: 1, Sides: 4}, "bludgeoning")
    
    result := h.TakeTurn(targets)
    
    if len(result.ActionResults[0].Targets) != 1 {
        t.Errorf("Should only affect 1 target, got %d", len(result.ActionResults[0].Targets))
    }
    
    if result.ActionResults[0].Targets[0].EntityID != "in" {
        t.Error("Wrong target affected")
    }
}
```

**Target File:** `pkg/rules/encounter/hazard_integration_test.go`

```go
package encounter_test

import (
    "testing"
    "uaa/vdnd/pkg/rules/encounter"
    "uaa/vdnd/pkg/rules/entity"
    "uaa/vdnd/pkg/rules/hazard"
)

func TestEncounterWithHazard(t *testing.T) {
    enc := encounter.NewEncounter("trapped_room")
    
    // Add party
    fighter := entity.NewEntity("fighter", "Bold Fighter", 5)
    fighter.Position = "trap_room"
    enc.AddParticipant(fighter)
    
    // Add complex hazard
    trap := hazard.GetComplexHazard("spinning_blade_pillar")
    trap.Position = "trap_room"
    enc.AddHazard(trap)
    
    // Roll initiatives
    enc.RollInitiative()
    enc.RollHazardInitiative()
    
    if err := enc.Start(); err != nil {
        t.Fatalf("Failed to start encounter: %v", err)
    }
    
    // Verify hazard is in initiative
    found := false
    for _, p := range enc.Participants {
        if p.Type == encounter.ParticipantHazard {
            found = true
            t.Logf("Hazard initiative: %d", p.Initiative)
        }
    }
    
    if !found {
        t.Error("Hazard should be in participants")
    }
}

func TestHazardTurnInEncounter(t *testing.T) {
    enc := encounter.NewEncounter("test")
    
    victim := entity.NewEntity("victim", "Victim", 5)
    victim.MaxHP = 50
    victim.CurrentHP = 50
    victim.Position = "danger_zone"
    enc.AddParticipant(victim)
    
    trap := hazard.GetComplexHazard("poisoned_dart_gallery")
    trap.Position = "danger_zone"
    enc.AddHazard(trap)
    
    enc.Start()
    
    // Execute hazard turn
    result := enc.ExecuteHazardTurn(trap.ID)
    
    t.Logf("Hazard turn result: %d actions, %d damage", 
        len(result.ActionResults), result.TotalDamage)
    
    if len(result.ActionResults) == 0 {
        t.Error("Hazard should have taken actions")
    }
}

func TestSimpleHazardNotInInitiative(t *testing.T) {
    enc := encounter.NewEncounter("simple_test")
    
    // Simple hazard (not complex)
    pit := hazard.NewHazard("pit_trap", "Pit Trap", 2)
    pit.Type = hazard.HazardTrap
    pit.Complexity = hazard.ComplexitySimple
    
    enc.AddHazard(pit)
    
    // Simple hazards should not be added
    for _, p := range enc.Participants {
        if p.Type == encounter.ParticipantHazard {
            t.Error("Simple hazards should not be in initiative")
        }
    }
}
```

---

## 7. Execution Checklist

- [ ] Create `pkg/rules/encounter/participant.go` with ParticipantType union
- [ ] Add `NewEntityParticipant()` and `NewHazardParticipant()`
- [ ] Create `pkg/rules/hazard/routine.go` with RoutineAction types
- [ ] Implement `NewRoutine()` builder pattern
- [ ] Create `pkg/rules/hazard/turn.go` with `TakeTurn()`
- [ ] Implement `executeAttack()`, `executeSaveEffect()`, `executeAreaEffect()`
- [ ] Add `Reset()` method for reset actions
- [ ] Create `pkg/rules/encounter/hazard_integration.go`
- [ ] Implement `AddHazard()`, `RollHazardInitiative()`, `ExecuteHazardTurn()`
- [ ] Add `ProcessHazardTriggers()` for trigger detection
- [ ] Update `pkg/rules/hazard/registry.go` with complex hazard templates
- [ ] Create test files
- [ ] Run `go test -v ./pkg/rules/...` and ensure 100% pass

---

## 8. CLI Commands

```bash
# Add hazard to encounter
vd encounter add_hazard battle_1 spinning_blade_pillar --position trap_corridor

# View encounter with hazards
vd encounter status battle_1
# Output:
# **Encounter: battle_1** (Round 2)
# | # | Name | Type | Init | HP | Status |
# |---|------|------|------|-----|--------|
# | 1 | Fighter | Entity | 18 | 45/45 | Active |
# | 2 | Spinning Blade Pillar | Hazard | 15 | 50/50 | Active |
# | 3 | Rogue | Entity | 12 | 32/32 | Active |

# Execute hazard turn
vd encounter hazard_turn battle_1 spinning_blade_pillar
# Output:
# **Spinning Blade Pillar's Turn**
# 
# **Action 1: Blade Slash**
# - Target: Fighter
# - Attack: 18 vs AC 19 - Miss
# 
# **Action 2: Blade Slash**
# - Target: Fighter
# - Attack: 24 vs AC 19 - Hit!
# - Damage: 14 slashing
# 
# **Fighter HP:** 45 → 31

# Attempt to disable
vd action disable rogue spinning_blade_pillar
# Output:
# **Disable Device**
# Actor: Rogue
# Target: Spinning Blade Pillar
# Method: Jam the mechanism (Thievery DC 21)
# Roll: 15 + 12 = 27
# **Result:** Success
# 
# **Spinning Blade Pillar is disabled!**
```

---

## 9. Design Notes

**Why separate Participant type?**
Entities and hazards share initiative order but have fundamentally different capabilities. Entities have HP, conditions, actions. Hazards have HP, routines, triggers. The union type lets the encounter system treat them uniformly for initiative while preserving their distinct behaviours.

**Hazard Routines vs Entity Turns**
Entity turns are freeform (player/LLM chooses actions). Hazard routines are scripted—they always do the same thing in the same order. This is intentional; complex hazards are meant to be predictable threats that players can learn to work around.

**Position-Based Targeting**
Hazards affect creatures at their position (zone). This meshes with the abstract positioning system from RULES_PLAN.md. A hazard doesn't track individual squares—it affects "everyone in the trap corridor."

**Trigger vs Turn**
Simple hazards only trigger (they react to events). Complex hazards also take turns (they actively attack). The `ProcessHazardTriggers()` method handles the reactive case, while `ExecuteHazardTurn()` handles the active case.
