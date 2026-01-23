# Phase 9: Spells

## Agent Prompt

You are implementing the spell system for a Pathfinder 2E rules engine in Go. Spells are magical effects that can deal damage, apply conditions, heal, or create other effects. This phase focuses on the spell casting framework and common spell patterns.

**Your task:** Implement the `pkg/rules/spell` package with full test coverage.

**Prerequisites:** Phases 1-8, 12 should be complete (check, conditions, damage, encounter).

---

## Context

### Source References
- Spell basics: `rules/rules/core-rulebook/chapter-9-playing-the-game.md:245`
- Spell components: `rules/rules/core-rulebook/chapter-7-spells.md`
- Spell lists: `rules/compendium/spells/`

### Spell Components
| Component | Trait | What It Means |
|-----------|-------|---------------|
| Somatic | manipulate | Hand gestures, can trigger AoO |
| Verbal | concentrate, auditory | Speaking, disrupted if silenced |
| Material | manipulate | Using material component |
| Focus | manipulate | Using a focus item |

### Spell Basics
- **Rank (Level):** 1-10, determines power and spell slot used
- **Traditions:** Arcane, Divine, Occult, Primal
- **Action Cost:** Usually 2 actions, some 1 or 3, reactions, or free
- **Range:** Self, touch, or distance in feet
- **Area:** Burst, cone, line, emanation
- **Targets:** Number of creatures or objects
- **Duration:** Instantaneous, sustained, or timed
- **Save:** Fortitude, Reflex, or Will (basic or not)

### Spell Attack vs Saving Throw
- **Spell Attack:** Roll d20 + spellcasting mod + proficiency vs AC
- **Saving Throw:** Target rolls save vs your Spell DC (10 + spellcasting mod + proficiency)

### Basic Saves
When a spell says "basic Reflex save" (or Fort/Will):
- **Critical Success:** No damage
- **Success:** Half damage
- **Failure:** Full damage
- **Critical Failure:** Double damage

---

## File Structure

```
pkg/
└── rules/
    └── spell/
        ├── spell.go        # Spell struct, SpellRank, components
        ├── casting.go      # Cast action, spell attacks, DCs
        ├── effects.go      # Common effect patterns
        ├── registry.go     # Pre-built spells
        └── spell_test.go
```

---

## Implementation Plan

### 1. `pkg/rules/spell/spell.go`

```go
type SpellRank int  // 1-10 (using "rank" as PF2E remaster terminology)

type SpellTradition int
const (
    TraditionArcane SpellTradition = iota
    TraditionDivine
    TraditionOccult
    TraditionPrimal
)

type SpellComponent int
const (
    ComponentSomatic SpellComponent = iota
    ComponentVerbal
    ComponentMaterial
    ComponentFocus
)

type SpellAreaType int
const (
    AreaNone SpellAreaType = iota
    AreaBurst
    AreaCone
    AreaLine
    AreaEmanation
)

type SaveType int
const (
    SaveNone SaveType = iota
    SaveFortitude
    SaveReflex
    SaveWill
)

type Spell struct {
    ID           string
    Name         string
    Rank         SpellRank
    Traditions   []SpellTradition
    ActionCost   combat.ActionCost
    Components   []SpellComponent
    Range        int      // Feet, 0 = touch, -1 = self
    Area         SpellAreaType
    AreaSize     int      // Radius for burst, length for line/cone
    Targets      int      // 0 = area effect, 1+ = targeted
    Duration     int      // Rounds, -1 = instantaneous, -2 = sustained
    Save         SaveType
    IsBasicSave  bool     // True for "basic X save"
    
    // Effect is implemented per-spell
    Effect       SpellEffect
}

type SpellEffect interface {
    Apply(caster, target *entity.Entity, degree check.DegreeOfSuccess) EffectResult
}

type EffectResult struct {
    Damage      int
    DamageType  item.DamageType
    Conditions  []condition.ConditionInstance
    Healed      int
    Description string
}
```

### 2. `pkg/rules/spell/casting.go`

```go
type CastSpellAction struct {
    Spell   *Spell
    Caster  *entity.Entity
    Targets []*entity.Entity  // Or area
}

func NewCastSpell(spell *Spell, caster *entity.Entity) *CastSpellAction

func (c *CastSpellAction) Cost() combat.ActionCost { return c.Spell.ActionCost }

// Execute casts the spell
func (c *CastSpellAction) Execute(turn *combat.TurnState) []EffectResult

// GetSpellAttackModifier returns caster's spell attack bonus
func GetSpellAttackModifier(caster *entity.Entity) int

// GetSpellDC returns caster's spell DC
func GetSpellDC(caster *entity.Entity) int

// RollSpellAttack makes a spell attack roll
func RollSpellAttack(caster, target *entity.Entity, modifiers []check.Modifier) check.CheckResult

// TargetMakesSave has target roll a saving throw
func TargetMakesSave(target *entity.Entity, saveType SaveType, dc int) check.CheckResult

// ApplyBasicSaveDamage calculates damage based on basic save result
func ApplyBasicSaveDamage(baseDamage int, degree check.DegreeOfSuccess) int
```

**Execute Pseudocode:**
```
func (c *CastSpellAction) Execute(turn *combat.TurnState) []EffectResult:
    results := []EffectResult{}
    
    dc := GetSpellDC(c.Caster)
    
    for _, target := range c.Targets:
        var degree check.DegreeOfSuccess
        
        if c.Spell.Save != SaveNone:
            # Target makes save
            saveResult := TargetMakesSave(target, c.Spell.Save, dc)
            degree = saveResult.Degree
        else:
            # Spell attack or auto-hit
            if c.Spell.requiresAttackRoll:
                attackResult := RollSpellAttack(c.Caster, target, nil)
                degree = attackResult.Degree
            else:
                degree = check.Success  # Auto-hit effects
        
        # Apply spell effect
        result := c.Spell.Effect.Apply(c.Caster, target, degree)
        
        # Handle basic save damage adjustment
        if c.Spell.IsBasicSave:
            result.Damage = ApplyBasicSaveDamage(result.Damage, degree)
        
        # Apply damage through pipeline
        if result.Damage > 0:
            damage.ProcessDamage(target, damage.DamageInstance{
                Amount: result.Damage,
                Type: result.DamageType,
                Source: c.Spell.Name,
            }, degree == check.CriticalSuccess)
        
        # Apply conditions
        for _, cond := range result.Conditions:
            target.Conditions.Apply(cond)
        
        results = append(results, result)
    
    return results
```

**ApplyBasicSaveDamage:**
```
func ApplyBasicSaveDamage(baseDamage int, degree check.DegreeOfSuccess) int:
    switch degree:
    case CriticalSuccess:
        return 0
    case Success:
        return baseDamage / 2
    case Failure:
        return baseDamage
    case CriticalFailure:
        return baseDamage * 2
```

### 3. `pkg/rules/spell/effects.go`

Common spell effect patterns.

```go
// DamageEffect is a simple damage-dealing spell
type DamageEffect struct {
    DamageDice dice.DieRoll
    DamageType item.DamageType
}

func (d *DamageEffect) Apply(caster, target *entity.Entity, degree check.DegreeOfSuccess) EffectResult {
    damage := d.DamageDice.Roll()
    return EffectResult{Damage: damage, DamageType: d.DamageType}
}

// ConditionEffect applies a condition
type ConditionEffect struct {
    ConditionID condition.ConditionID
    Value       int  // For valued conditions
    Duration    int  // Rounds
}

func (c *ConditionEffect) Apply(caster, target *entity.Entity, degree check.DegreeOfSuccess) EffectResult

// HealEffect heals the target
type HealEffect struct {
    HealDice dice.DieRoll
}

func (h *HealEffect) Apply(caster, target *entity.Entity, degree check.DegreeOfSuccess) EffectResult

// HeightenedDamage scales damage with spell rank
type HeightenedDamage struct {
    BaseDice      dice.DieRoll
    ExtraDicePerRank int  // e.g., +1d6 per rank above base
    BaseRank      SpellRank
    CastRank      SpellRank
    DamageType    item.DamageType
}
```

### 4. `pkg/rules/spell/registry.go`

Pre-built common spells.

```go
// Example spells
var (
    // Electric Arc - 1-action, 2 targets, basic Reflex
    ElectricArc = Spell{
        ID: "electric-arc",
        Name: "Electric Arc",
        Rank: 1,
        Traditions: []SpellTradition{TraditionArcane, TraditionPrimal},
        ActionCost: combat.CostTwo,
        Components: []SpellComponent{ComponentSomatic, ComponentVerbal},
        Range: 30,
        Targets: 2,
        Save: SaveReflex,
        IsBasicSave: true,
        Effect: &DamageEffect{
            DamageDice: dice.DieRoll{1, 4, 0},  // 1d4+spellcasting mod
            DamageType: item.Electricity,
        },
    }
    
    // Fireball - 2-action, 20ft burst, basic Reflex
    Fireball = Spell{
        ID: "fireball",
        Name: "Fireball",
        Rank: 3,
        Traditions: []SpellTradition{TraditionArcane, TraditionPrimal},
        ActionCost: combat.CostTwo,
        Components: []SpellComponent{ComponentSomatic, ComponentVerbal},
        Range: 500,
        Area: AreaBurst,
        AreaSize: 20,
        Save: SaveReflex,
        IsBasicSave: true,
        Effect: &DamageEffect{
            DamageDice: dice.DieRoll{6, 6, 0},  // 6d6
            DamageType: item.Fire,
        },
    }
    
    // Fear - 2-action, single target, Will save
    Fear = Spell{
        ID: "fear",
        Name: "Fear",
        Rank: 1,
        Traditions: []SpellTradition{TraditionArcane, TraditionDivine, TraditionOccult, TraditionPrimal},
        ActionCost: combat.CostTwo,
        Range: 30,
        Targets: 1,
        Save: SaveWill,
        IsBasicSave: false,
        Effect: &FearEffect{},
    }
    
    // Heal - 1-3 actions, variable effect
    // (Complex - 1 action = touch 1d8, 2 actions = ranged 1d8, 3 actions = 30ft burst 1d8)
)

func GetSpell(id string) (*Spell, bool)
```

---

## Test Plan

### Spell Attack Tests

| Test | Caster Mod | Target AC | Roll | Expected |
|------|-----------|-----------|------|----------|
| Hit | +8 | 15 | 10 | Success |
| Crit | +8 | 15 | 20 | Critical Success |
| Miss | +5 | 20 | 10 | Failure |
| Nat 20 upgrades | +5 | 30 | 20 | Success (not crit, too far) |

### Saving Throw Tests

| Test | Save Mod | DC | Roll | Expected |
|------|----------|-----|------|----------|
| Make save | +8 | 18 | 12 | Success |
| Fail save | +5 | 20 | 10 | Failure |
| Crit success | +8 | 15 | 20 | Critical Success |
| Crit fail | +3 | 25 | 1 | Critical Failure |

### Basic Save Damage Tests

| Degree | Base Damage | Expected |
|--------|-------------|----------|
| Critical Success | 20 | 0 |
| Success | 20 | 10 |
| Failure | 20 | 20 |
| Critical Failure | 20 | 40 |

### Spell Effect Tests

| Spell | Target | Save Result | Expected |
|-------|--------|-------------|----------|
| Fireball (6d6) | Target A | Failure | Full 6d6 fire damage |
| Fireball | Target A | Success | Half damage |
| Electric Arc | Target A | Crit Fail | Double damage |
| Fear | Target A | Failure | Frightened 2 |
| Fear | Target A | Success | Frightened 1 |
| Fear | Target A | Crit Success | No effect |

### Heightening Tests

| Spell | Base Rank | Cast Rank | Expected |
|-------|-----------|-----------|----------|
| Fireball | 3 | 3 | 6d6 |
| Fireball | 3 | 5 | 10d6 (+2d6 per rank) |
| Heal | 1 | 1 | 1d8 |
| Heal | 1 | 3 | 3d8 |

---

## Validation Checklist

- [ ] All tests pass
- [ ] `go vet` reports no issues
- [ ] `go fmt` applied
- [ ] Spell attacks use correct modifiers
- [ ] Saving throws calculate correctly
- [ ] Basic save damage scales with degree
- [ ] Conditions from spell effects apply
- [ ] Heightening increases damage appropriately

---

## Notes for Implementation

1. **Spellcasting ability:** Different classes use different abilities (INT for Wizard, WIS for Cleric, CHA for Sorcerer). Store on entity or pass in.

2. **Spell slots:** Not implementing slot tracking in this phase. Just the casting mechanics.

3. **Components and reactions:** Somatic has manipulate trait, verbal has concentrate. These can trigger AoO. Emit events in the encounter system.

4. **Heightening:** Many spells improve when cast at higher ranks. Store base effect and heightening rules.

5. **Area effects:** For now, assume LLM/GM determines who is in area. Geometry is abstracted.

6. **Focus spells and cantrips:** Cantrips auto-heighten to half caster level. Focus spells use focus points. Keep these in mind but can simplify.
