# Phase 6: Entities

## Agent Prompt

You are implementing the entity system for a Pathfinder 2E rules engine in Go. Entities represent anything that can act in the game—player characters, NPCs, and monsters. The Entity struct ties together ability scores, conditions, equipment, and combat state.

**Your task:** Implement the `pkg/rules/entity` package with full test coverage.

**Prerequisites:** Phases 1-5 should be complete (ability, check, condition, trait, item).

---

## Context

### Source References
- Creature stats: `rules/fantasy-bestiary/`
- Character creation: `rules/rules/core-rulebook/chapter-1-introduction.md`
- Saving throws: `rules/rules/core-rulebook/chapter-9-playing-the-game.md:288`
- Immunities/Resistances: `rules/rules/core-rulebook/chapter-9-playing-the-game.md:499`

### Entity Components
An entity combines:
- **Identity:** Name, level, ancestry/class (flavour)
- **Abilities:** The six ability scores
- **Defenses:** AC, saving throws, HP, immunities, resistances, weaknesses
- **Offenses:** Wielded weapons, attack proficiencies
- **Conditions:** Active conditions affecting the entity
- **Position:** Where they are in the encounter (zone-based)

### Size Categories
| Size | Space | Reach (tall) | Reach (long) |
|------|-------|--------------|--------------|
| Tiny | 2.5ft | 0ft | 0ft |
| Small | 5ft | 5ft | 5ft |
| Medium | 5ft | 5ft | 5ft |
| Large | 10ft | 10ft | 5ft |
| Huge | 15ft | 15ft | 10ft |
| Gargantuan | 20ft+ | 20ft | 15ft |

### Calculating AC
```
AC = 10 + DEX modifier (up to armor's Dex Cap) + proficiency bonus + armor item bonus + other bonuses
```

### Calculating Saves
```
Fortitude = 10 + CON modifier + proficiency bonus + item bonus + other bonuses
Reflex    = 10 + DEX modifier + proficiency bonus + item bonus + other bonuses  
Will      = 10 + WIS modifier + proficiency bonus + item bonus + other bonuses
```

---

## File Structure

```
pkg/
└── rules/
    └── entity/
        ├── entity.go       # Entity struct, core methods
        ├── size.go         # Size enum
        ├── combat.go       # AC, saves, attacks calculations
        ├── damage.go       # HP, damage, healing, dying
        └── entity_test.go
```

---

## Implementation Plan

### 1. `pkg/rules/entity/size.go`

```go
type Size int
const (
    Tiny Size = iota
    Small
    Medium
    Large
    Huge
    Gargantuan
)

// Space returns the space occupied in feet
func (s Size) Space() int

// Reach returns base reach for a tall creature
func (s Size) Reach() int

// String returns the size name
func (s Size) String() string
```

### 2. `pkg/rules/entity/entity.go`

```go
import (
    "vdnd/pkg/rules/ability"
    "vdnd/pkg/rules/condition"
    "vdnd/pkg/rules/item"
    "vdnd/pkg/rules/trait"
)

type Entity struct {
    // Identity
    ID         string
    Name       string
    Level      int
    Size       Size
    
    // Flavour (for PCs primarily)
    Ancestry   string
    Class      string
    Background string
    
    // Core Stats
    Abilities  ability.AbilityScores
    
    // Hit Points
    MaxHP      int
    CurrentHP  int
    TempHP     int
    
    // Proficiencies
    Perception    ability.ProficiencyRank
    Fortitude     ability.ProficiencyRank
    Reflex        ability.ProficiencyRank
    Will          ability.ProficiencyRank
    UnarmoredDefense ability.ProficiencyRank
    
    // Equipment
    WornArmor      *item.Armor
    WieldedWeapons []*item.Weapon  // Up to 2 (or more for multi-limbed)
    
    // Runtime State
    Conditions     *condition.ConditionTracker
    
    // Position (zone-based)
    Position       string    // Zone ID
    EngagedWith    []string  // Entity IDs currently in melee with
    
    // Defenses
    Immunities     []string            // Trait/damage type IDs
    Resistances    map[string]int      // type -> amount
    Weaknesses     map[string]int      // type -> amount
    
    // Creature traits (for monsters)
    Traits         trait.TraitSet
}

// NewEntity creates a basic entity
func NewEntity(id, name string, level int) *Entity

// NewPC creates a player character entity
func NewPC(id, name string, level int, ancestry, class, background string) *Entity

// Clone creates a deep copy of the entity
func (e *Entity) Clone() *Entity
```

### 3. `pkg/rules/entity/combat.go`

AC and saving throw calculations.

```go
import "vdnd/pkg/rules/check"

// GetAC calculates current Armor Class
func (e *Entity) GetAC() int

// GetACModifiers returns all modifiers contributing to AC
func (e *Entity) GetACModifiers() []check.Modifier

// GetFortitude calculates Fortitude save modifier
func (e *Entity) GetFortitude() int

// GetReflex calculates Reflex save modifier
func (e *Entity) GetReflex() int

// GetWill calculates Will save modifier
func (e *Entity) GetWill() int

// GetPerception calculates Perception modifier
func (e *Entity) GetPerception() int

// GetSaveModifier returns the modifier for a given save type
func (e *Entity) GetSaveModifier(save SaveType) int

// GetSaveDC returns 10 + save modifier (for opposed checks)
func (e *Entity) GetSaveDC(save SaveType) int

// IsImmuneTo checks if entity is immune to a damage type or trait
func (e *Entity) IsImmuneTo(id string) bool

// GetResistance returns resistance amount for a damage type (0 if none)
func (e *Entity) GetResistance(damageType string) int

// GetWeakness returns weakness amount for a damage type (0 if none)
func (e *Entity) GetWeakness(damageType string) int
```

**GetAC Pseudocode:**
```
func (e *Entity) GetAC() int:
    base := 10
    
    # DEX modifier, capped by armor
    dexMod := e.Abilities.Modifier(Dexterity)
    if e.WornArmor != nil:
        dexMod = e.WornArmor.AppliedDexBonus(dexMod)
    
    # Proficiency bonus
    armorProf := e.getArmorProficiency()
    profBonus := armorProf.Bonus(e.Level)
    
    # Armor item bonus
    armorBonus := 0
    if e.WornArmor != nil:
        armorBonus = e.WornArmor.ACBonus
    
    # Condition modifiers (flat-footed, clumsy, etc.)
    conditionMods := e.Conditions.GetACModifiers()
    conditionTotal := check.CalculateTotal(conditionMods)
    
    return base + dexMod + profBonus + armorBonus + conditionTotal
```

**GetFortitude Pseudocode:**
```
func (e *Entity) GetFortitude() int:
    conMod := e.Abilities.Modifier(Constitution)
    profBonus := e.Fortitude.Bonus(e.Level)
    
    # Universal modifiers from conditions (frightened, sickened, etc.)
    conditionMods := e.Conditions.GetModifiers()
    
    # Fortitude-specific modifiers
    saveMods := e.Conditions.GetSaveModifiers()
    
    allMods := append(conditionMods, saveMods...)
    modTotal := check.CalculateTotal(allMods)
    
    return conMod + profBonus + modTotal
```

### 4. `pkg/rules/entity/damage.go`

HP management, damage, healing, dying.

```go
// TakeDamage applies damage after immunities/resistances/weaknesses
// Returns actual damage taken
func (e *Entity) TakeDamage(amount int, damageType string) int

// Heal restores HP (cannot exceed MaxHP)
func (e *Entity) Heal(amount int)

// AddTempHP adds temporary HP (takes higher if already has temp HP)
func (e *Entity) AddTempHP(amount int)

// IsDead returns true if entity has died
func (e *Entity) IsDead() bool

// IsDying returns true if entity has dying condition
func (e *Entity) IsDying() bool

// IsUnconscious returns true if HP <= 0 or has unconscious condition
func (e *Entity) IsUnconscious() bool

// CheckDying should be called after taking damage at 0 HP
// Applies dying condition or increases dying value
func (e *Entity) CheckDying(wasCritical bool)

// RecoveryCheck makes a dying recovery check
// Returns true if stabilized
func (e *Entity) RecoveryCheck() bool
```

**TakeDamage Pseudocode:**
```
func (e *Entity) TakeDamage(amount int, damageType string) int:
    # Check immunity first
    if e.IsImmuneTo(damageType):
        return 0
    
    # Apply weakness
    weakness := e.GetWeakness(damageType)
    amount += weakness
    
    # Apply resistance
    resistance := e.GetResistance(damageType)
    amount -= resistance
    if amount < 0:
        amount = 0
    
    # Temp HP absorbs first
    if e.TempHP > 0:
        absorbed := min(e.TempHP, amount)
        e.TempHP -= absorbed
        amount -= absorbed
    
    # Apply to current HP
    e.CurrentHP -= amount
    if e.CurrentHP < 0:
        e.CurrentHP = 0
    
    return amount  # Return actual damage taken
```

**CheckDying Pseudocode:**
```
func (e *Entity) CheckDying(wasCritical bool):
    if e.CurrentHP > 0:
        return  # Not at 0 HP, nothing to do
    
    if e.Conditions.Has(Dying):
        # Already dying, increase value
        increase := 1
        if wasCritical:
            increase = 2
        current := e.Conditions.Value(Dying)
        e.Conditions.Apply(NewValuedCondition(Dying, current + increase, "damage"))
    else:
        # New dying
        dyingValue := 1
        if wasCritical:
            dyingValue = 2
        # Add wounded value
        dyingValue += e.Conditions.Value(Wounded)
        e.Conditions.Apply(NewValuedCondition(Dying, dyingValue, "reduced to 0 HP"))
        e.Conditions.Apply(NewCondition(Unconscious, "dying"))
    
    # Check for death
    maxDying := 4 - e.Conditions.Value(Doomed)
    if e.Conditions.Value(Dying) >= maxDying:
        e.die()
```

---

## Test Plan

### Size Tests

| Size | Space | Reach |
|------|-------|-------|
| Tiny | 2 (or 2.5) | 0 |
| Small | 5 | 5 |
| Medium | 5 | 5 |
| Large | 10 | 10 |
| Huge | 15 | 15 |
| Gargantuan | 20 | 20 |

### Entity Creation Tests

| Test | Expected |
|------|----------|
| NewEntity sets ID, Name, Level | Fields populated |
| NewEntity initializes condition tracker | `Conditions != nil` |
| NewPC sets ancestry/class/background | Fields populated |
| Clone creates independent copy | Modifying clone doesn't affect original |

### AC Calculation Tests

| Test | Setup | Expected AC |
|------|-------|-------------|
| Unarmored, DEX +3, trained | Level 1, DEX 16, no armor | 10 + 3 + 3 = 16 |
| With armor | Leather (+1 AC), DEX +4 | 10 + 4 + 3 + 1 = 18 |
| DEX exceeds cap | DEX +5, leather (cap 4) | Uses +4 not +5 |
| Flat-footed | Add flat-footed condition | AC - 2 |
| Clumsy 2 | Add clumsy 2 | AC - 2 (status) |
| Flat-footed + Clumsy | Both conditions | AC - 2 (circ) - 2 (status) = -4 |

### Save Calculation Tests

| Test | Setup | Expected |
|------|-------|----------|
| Fortitude untrained | CON +2, untrained, level 1 | +2 |
| Fortitude trained | CON +2, trained, level 1 | 2 + 3 = +5 |
| Fortitude with frightened | Frightened 2, trained | +5 - 2 = +3 |
| Reflex expert | DEX +4, expert, level 5 | 4 + 9 = +13 |
| Will master | WIS +3, master, level 10 | 3 + 16 = +19 |

### Damage Tests

| Test | Setup | Damage | Expected |
|------|-------|--------|----------|
| Normal damage | MaxHP 20, CurrentHP 20 | 5 slashing | HP = 15 |
| Immune to damage type | Immune to fire | 10 fire | HP unchanged |
| Resistance | Resist fire 5 | 8 fire | Takes 3 |
| Weakness | Weakness cold 5 | 10 cold | Takes 15 |
| Temp HP absorbs | TempHP 5, CurrentHP 20 | 8 damage | TempHP 0, CurrentHP 17 |
| Temp HP partial | TempHP 10, CurrentHP 20 | 8 damage | TempHP 2, CurrentHP 20 |
| Resistance reduces to 0 | Resist fire 10 | 5 fire | Takes 0 |
| Weakness + Resistance | Weakness fire 5, resist fire 3 | 10 fire | 10 + 5 - 3 = 12 |

### Dying Tests

| Test | Setup | Action | Expected |
|------|-------|--------|----------|
| Reduced to 0 HP | CurrentHP 5 | Take 10 damage | Dying 1, Unconscious |
| Critical at 0 HP | CurrentHP 5 | Take 10 crit damage | Dying 2, Unconscious |
| Already dying, take damage | Dying 1 | Take damage | Dying 2 |
| Already dying, crit | Dying 1 | Take crit damage | Dying 3 |
| Wounded + dying | Wounded 1, CurrentHP 5 | Take 10 damage | Dying 2 (1 + wounded) |
| Doomed affects max | Doomed 1 | Dying reaches 3 | Death (max = 4-1 = 3) |
| Dying 4 = death | Normal | Dying reaches 4 | IsDead() = true |

### Immunity/Resistance Tests

| Test | Setup | Expected |
|------|-------|----------|
| IsImmuneTo("fire") | Immunities = ["fire"] | true |
| IsImmuneTo("cold") | Immunities = ["fire"] | false |
| GetResistance("fire") | Resistances = {fire: 5} | 5 |
| GetResistance("cold") | Resistances = {fire: 5} | 0 |
| GetWeakness works similarly | | |

---

## Validation Checklist

- [ ] All tests pass
- [ ] `go vet` reports no issues
- [ ] `go fmt` applied
- [ ] AC calculation uses armor dex cap correctly
- [ ] Condition modifiers apply to AC/saves
- [ ] Damage immunity/resistance/weakness order is correct
- [ ] Dying+wounded+doomed interactions work
- [ ] Temp HP absorbs before regular HP

---

## Notes for Implementation

1. **Condition tracker must be initialized:** `NewEntity` should create `condition.NewTracker()`.

2. **Armor proficiency lookup:** Entity needs to know its proficiency with its worn armor category. Either store per-category proficiencies or have a method to look it up.

3. **Multiple weapons:** Store as slice. An entity might dual-wield or have natural weapons + held weapon.

4. **Immunities are strings:** Could be damage types ("fire") or traits ("mental", "poison"). Check both when applying damage or effects.

5. **Engaged tracking:** `EngagedWith` is updated by movement actions. Used for flanking, AoO triggers, etc.

6. **Deep clone:** Important for "what if" calculations or saving state. Make sure Conditions tracker is cloned properly.

7. **Integration test idea:** Create entity, apply conditions, take damage, verify AC/saves/HP all update correctly together.
