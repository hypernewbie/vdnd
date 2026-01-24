# Phase 23: Complex Hazard Integration

## Objective

Integrate Complex Hazards into the encounter system. Complex hazards have initiative, take turns, and perform routines. This phase connects the existing `hazard` package with the `encounter` system.

---

## 1. Hazard Participant Type

**Target File:** `pkg/rules/encounter/participant.go`

Extend the participant system to handle both entities and hazards.

(Unified Participant struct handles both Entity and Hazard types)

---

## 2. Hazard Routine System

**Target File:** `pkg/rules/hazard/routine.go`

Complex hazards have defined actions they take each turn.

```go
// RoutineAction represents one action in a hazard's routine
type RoutineAction struct {
    Name        string
    Type        RoutineActionType
    ActionCost  int             // 1, 2, or 3 actions
    TargetCount int            // How many targets it can affect (1 for most attacks, 0/all for AoE)
    
    // For attacks
    AttackBonus int
    DamageDice  dice.DieRoll
    DamageType  item.DamageType
    
    // ... (other fields) ...
}

// AddAttack adds an attack action to the routine
func (r *HazardRoutine) AddAttack(name string, cost int, attackBonus int, damage dice.DieRoll, damageType item.DamageType, targetCount int) *HazardRoutine {
    r.Actions = append(r.Actions, RoutineAction{
        Name:        name,
        Type:        RoutineAttack,
        ActionCost:  cost,
        AttackBonus: attackBonus,
        DamageDice:  damage,
        DamageType:  damageType,
        TargetCount: targetCount,
    })
    return r
}
```

---

## 3. Hazard Turn Execution

**Target File:** `pkg/rules/hazard/turn.go`

Execute a hazard's routine on its turn.

```go
func (h *Hazard) executeAttack(action RoutineAction, targets []*entity.Entity) ActionResult {
    result := ActionResult{
        ActionName: action.Name,
        ActionType: RoutineAttack,
        Targets:    make([]TargetResult, 0),
    }

    if len(targets) == 0 {
        return result
    }

    // Select subset of targets if TargetCount > 0
    affectedTargets := targets
    if action.TargetCount > 0 && len(targets) > action.TargetCount {
        // Randomly select TargetCount entities from the available targets
        r := rand.New(rand.NewSource(time.Now().UnixNano()))
        shuffled := make([]*entity.Entity, len(targets))
        copy(shuffled, targets)
        r.Shuffle(len(shuffled), func(i, j int) {
            shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
        })
        affectedTargets = shuffled[:action.TargetCount]
    }

    for _, target := range affectedTargets {
        // ... (perform attack) ...
    }

    return result
}
```

---

## 4. Complex Hazard Registry

Standard hazards like **Spinning Blade Pillar** now use `TargetCount: 1` for their blade slash actions, ensuring they only attack one creature per strike.
