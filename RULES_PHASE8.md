# Phase 8: Damage Pipeline

## Agent Prompt

You are implementing the damage pipeline for a Pathfinder 2E rules engine in Go. This is the system that takes raw damage from an attack or spell, applies all modifications (criticals, immunities, resistances, weaknesses), and reduces HP.

**Your task:** Implement the `pkg/rules/damage` package with full test coverage.

**Prerequisites:** Phases 1-6 should be complete (especially entity with HP/immunities/resistances).

---

## Context

### Source References
- Damage: `rules/rules/core-rulebook/chapter-9-playing-the-game.md:377`
- Critical hits: `rules/rules/core-rulebook/chapter-9-playing-the-game.md:430`
- Immunities: `rules/rules/core-rulebook/chapter-9-playing-the-game.md:504`
- Weaknesses: `rules/rules/core-rulebook/chapter-9-playing-the-game.md:520`
- Resistances: `rules/rules/core-rulebook/chapter-9-playing-the-game.md:527`

### The Damage Pipeline (src: chapter-9)

```
1. Determine base damage (roll dice + modifiers)
2. If critical hit: double the damage
3. Check immunities (immune = 0 damage of that type)
4. Apply weaknesses (add weakness value to damage)
5. Apply resistances (subtract resistance value, minimum 0)
6. Apply damage to HP (temp HP first, then current HP)
7. Check for dying/death
```

### Damage Types
**Physical:** Bludgeoning, Piercing, Slashing
**Energy:** Fire, Cold, Electricity, Acid, Sonic
**Alignment:** Good, Evil, Lawful, Chaotic
**Special:** Force, Mental, Poison, Positive, Negative, Bleed, Precision

### Critical Hit Special Rules
- Damage is **doubled** (roll once, multiply)
- **Deadly trait:** Add extra dice on crit (e.g., deadly d10 adds 1d10)
- **Fatal trait:** Damage die becomes larger AND adds extra die
- Critical specialization effects (weapon group specific)

### Resistance/Weakness Interactions
- Multiple resistances: Use highest that applies
- Multiple weaknesses: Use highest that applies
- Resistance + Weakness to same type: Both apply (weakness adds, resistance subtracts)
- Resistance "except X": Resistance doesn't apply if X condition met (e.g., "resistance 5 to physical except silver")

### Precision Damage
- Extra damage (like Sneak Attack) is the same type as the base damage
- Creatures immune to precision damage ignore it entirely
- Not doubled on crit (unless specifically stated)

---

## File Structure

```
pkg/
└── rules/
    └── damage/
        ├── damage.go       # DamageInstance, DamageType
        ├── pipeline.go     # The damage pipeline processor
        ├── critical.go     # Critical hit handling
        ├── modifiers.go    # Resistance, weakness, immunity checks
        └── damage_test.go
```

---

## Implementation Plan

### 1. `pkg/rules/damage/damage.go`

```go
// DamageInstance represents a single source of damage
type DamageInstance struct {
    Amount      int
    Type        item.DamageType
    Source      string            // "Longsword", "Fireball", etc.
    IsPrecision bool              // Sneak Attack, etc.
    Traits      []trait.TraitID   // For things like "magical" damage
}

// DamageRoll represents the components of a damage roll before resolution
type DamageRoll struct {
    BaseDice    dice.DieRoll      // e.g., 2d6
    Modifier    int               // Usually STR mod
    BonusDice   []dice.DieRoll    // Extra dice (deadly, striking rune, etc.)
    DamageType  item.DamageType
    Source      string
    IsPrecision bool
}

// Roll evaluates the damage roll and returns a DamageInstance
func (d DamageRoll) Roll() DamageInstance

// RollCritical evaluates as a critical hit (double damage, extra deadly dice)
func (d DamageRoll) RollCritical(deadlyDie dice.DieRoll, fatalDie dice.DieRoll) DamageInstance
```

**Roll Pseudocode:**
```
func (d DamageRoll) Roll() DamageInstance:
    total := d.BaseDice.Roll() + d.Modifier
    for _, bonus := range d.BonusDice:
        total += bonus.Roll()
    return DamageInstance{Amount: total, Type: d.DamageType, Source: d.Source}
```

**RollCritical Pseudocode:**
```
func (d DamageRoll) RollCritical(deadlyDie, fatalDie dice.DieRoll) DamageInstance:
    # If fatal: damage die changes AND adds extra die
    baseDice := d.BaseDice
    if fatalDie.Sides > 0:
        baseDice = dice.DieRoll{baseDice.Count, fatalDie.Sides, baseDice.Modifier}
    
    # Roll base + modifiers, then double
    baseTotal := baseDice.Roll() + d.Modifier
    for _, bonus := range d.BonusDice:
        baseTotal += bonus.Roll()
    
    doubledTotal := baseTotal * 2
    
    # Add deadly dice (NOT doubled)
    if deadlyDie.Sides > 0:
        deadlyDice := deadlyDie.Count
        if deadlyDice == 0: deadlyDice = 1
        doubledTotal += dice.DieRoll{deadlyDice, deadlyDie.Sides, 0}.Roll()
    
    # Add fatal extra die (NOT doubled)
    if fatalDie.Sides > 0:
        doubledTotal += dice.DieRoll{1, fatalDie.Sides, 0}.Roll()
    
    return DamageInstance{Amount: doubledTotal, Type: d.DamageType, Source: d.Source}
```

### 2. `pkg/rules/damage/pipeline.go`

The main damage processor.

```go
// PipelineResult contains the outcome of processing damage
type PipelineResult struct {
    OriginalDamage   int
    FinalDamage      int
    WasImmune        bool
    WeaknessApplied  int
    ResistanceApplied int
    PreviousHP       int
    CurrentHP        int
    BecameDying      bool
    Died             bool
}

// ProcessDamage applies damage to an entity through the full pipeline
func ProcessDamage(target *entity.Entity, damage DamageInstance, isCritical bool) PipelineResult

// ProcessMultipleDamage handles multiple damage types at once (e.g., sword + flaming)
func ProcessMultipleDamage(target *entity.Entity, damages []DamageInstance, isCritical bool) PipelineResult
```

**ProcessDamage Pseudocode:**
```
func ProcessDamage(target *entity.Entity, damage DamageInstance, isCritical bool) PipelineResult:
    result := PipelineResult{
        OriginalDamage: damage.Amount,
        PreviousHP: target.CurrentHP,
    }
    
    # Step 1: Already rolled (damage.Amount is the total)
    amount := damage.Amount
    
    # Step 2: Critical doubling already applied in RollCritical
    # (or apply here if raw damage passed in)
    if isCritical and !damage.alreadyDoubled:
        amount *= 2
    
    # Step 3: Check immunity
    if target.IsImmuneTo(string(damage.Type)):
        result.WasImmune = true
        result.FinalDamage = 0
        return result
    
    # Precision immunity check
    if damage.IsPrecision and target.IsImmuneTo("precision"):
        result.WasImmune = true
        result.FinalDamage = 0
        return result
    
    # Step 4: Apply weakness
    weakness := target.GetWeakness(string(damage.Type))
    amount += weakness
    result.WeaknessApplied = weakness
    
    # Step 5: Apply resistance
    resistance := target.GetResistance(string(damage.Type))
    amount -= resistance
    if amount < 0:
        amount = 0
    result.ResistanceApplied = resistance
    
    result.FinalDamage = amount
    
    # Step 6: Apply to HP
    target.TakeDamage(amount, string(damage.Type))  # Handles temp HP internally
    result.CurrentHP = target.CurrentHP
    
    # Step 7: Check dying/death
    if target.CurrentHP <= 0 and !target.Conditions.Has(Dying):
        target.CheckDying(isCritical)
        result.BecameDying = true
    
    if target.IsDead():
        result.Died = true
    
    return result
```

### 3. `pkg/rules/damage/critical.go`

Critical hit handling and weapon trait processing.

```go
// CriticalEffects determines what extra effects happen on a crit
type CriticalEffects struct {
    DeadlyDie   dice.DieRoll  // From deadly trait
    FatalDie    dice.DieRoll  // From fatal trait
    ExtraEffects []string     // Critical specialization, etc.
}

// GetCriticalEffects inspects a weapon for crit-related traits
func GetCriticalEffects(weapon *item.Weapon) CriticalEffects

// ApplyCriticalSpecialization applies weapon group crit effects
func ApplyCriticalSpecialization(target *entity.Entity, group item.WeaponGroup) []condition.ConditionInstance
```

**Critical Specialization Effects (by weapon group):**
| Group | Effect |
|-------|--------|
| Sword | Target takes 1d6 persistent bleed |
| Axe | Target takes 1d6 persistent bleed (larger axe = 2d6) |
| Hammer | Target knocked prone |
| Pick | +2 damage per weapon die |
| Polearm | Target pushed 5ft |
| Spear | Target takes 1d6 persistent bleed |
| Knife | Target flat-footed until end of your next turn |
| Flail | Target knocked prone |
| Brawling | Target slowed 1 until end of your next turn |
| Club | Target knocked prone |
| Bow | Target takes 1d10 persistent bleed |

### 4. `pkg/rules/damage/modifiers.go`

Helper functions for resistance/weakness/immunity checks.

```go
// CheckImmunity returns true if target is immune to damage type or traits
func CheckImmunity(target *entity.Entity, damageType string, traits []trait.TraitID) bool

// CalculateWeakness returns total weakness value for damage type
func CalculateWeakness(target *entity.Entity, damageType string, traits []trait.TraitID) int

// CalculateResistance returns total resistance value for damage type
// Handles "except" conditions
func CalculateResistance(target *entity.Entity, damageType string, traits []trait.TraitID) int
```

**Resistance "except" handling:**
Some creatures have "Resistance 5 to physical damage (except silver)" or "(except magical)". We need to check if damage has the excepting trait.

```go
type ResistanceEntry struct {
    Amount    int
    Except    []string  // Traits/materials that bypass this resistance
}

// Entity stores:
// Resistances map[string]ResistanceEntry
```

---

## Test Plan

### DamageRoll Tests

| Test | Roll | Expected |
|------|------|----------|
| Simple 1d8+4 | {1,8,0} mod 4, RNG=5 | 9 |
| 2d6+3 | {2,6,0} mod 3, RNG=3,4 | 10 |
| With bonus dice | Base 1d8, bonus 1d6, RNG=5,3 | 8 |

### Critical Roll Tests

| Test | Weapon | Expected |
|------|--------|----------|
| Normal crit (2x) | 1d8+4 rolls 6+4=10 | 20 |
| Deadly d10 crit | 1d8+4, deadly d10, base=10, deadly rolls 7 | 20 + 7 = 27 |
| Fatal d12 crit | 1d6+4 fatal d12, base=5+4=9 | (uses d12) 2*(1d12+4) + 1d12 |

### Immunity Tests

| Test | Target | Damage | Expected |
|------|--------|--------|----------|
| Immune to fire | Immunities: [fire] | 10 fire | 0 damage |
| Not immune | Immunities: [fire] | 10 cold | 10 damage |
| Immune to precision | Immunities: [precision] | 5 precision | 0 damage |

### Weakness Tests

| Test | Weakness | Damage | Expected |
|------|----------|--------|----------|
| Fire weakness 5 | Weaknesses: {fire: 5} | 10 fire | 15 damage |
| No weakness | Weaknesses: {} | 10 fire | 10 damage |
| Multiple types, only one applies | Weaknesses: {fire: 5} | 10 cold | 10 damage |

### Resistance Tests

| Test | Resistance | Damage | Expected |
|------|------------|--------|----------|
| Fire resist 5 | Resistances: {fire: 5} | 10 fire | 5 damage |
| Resist reduces to 0 | Resistances: {fire: 15} | 10 fire | 0 damage |
| Physical except silver | Resist physical 5 except silver | 10 slashing | 5 damage |
| Physical except silver (has silver) | Resist physical 5 except silver | 10 slashing (silver) | 10 damage |

### Combined Tests

| Test | Setup | Damage | Expected |
|------|-------|--------|----------|
| Weakness + Resistance | Weakness fire 5, Resist fire 3 | 10 fire | 10 + 5 - 3 = 12 |
| Immunity overrides all | Immune fire, Weakness cold 5 | 10 fire | 0 (immune) |
| Order matters | Weakness 5, Resist 3 | 1 fire | 1 + 5 - 3 = 3 |

### Pipeline Integration Tests

| Test | Initial HP | Damage | Expected HP | Other |
|------|------------|--------|-------------|-------|
| Normal damage | 20 | 5 slashing | 15 | - |
| Damage with temp HP | HP 20, TempHP 5 | 8 slashing | 17, TempHP 0 | Temp absorbed 5 |
| Reduced to 0 | 10 | 15 slashing | 0 | BecameDying = true |
| Crit to 0 | 10 | 15 crit slashing | 0 | Dying 2 (crit) |
| Already dying | HP 0, Dying 1 | 5 slashing | 0 | Dying 2 |
| Death | HP 0, Dying 3 | 5 slashing | 0 | Died = true |

### Critical Specialization Tests

| Group | Effect |
|-------|--------|
| Sword | Target gains persistent bleed 1d6 |
| Hammer | Target gains prone |
| Knife | Target gains flat-footed |
| Brawling | Target gains slowed 1 |

---

## Validation Checklist

- [ ] All tests pass
- [ ] `go vet` reports no issues
- [ ] `go fmt` applied
- [ ] Damage pipeline order: immunity → weakness → resistance
- [ ] Critical hits double damage correctly
- [ ] Deadly/fatal traits add dice (not doubled)
- [ ] Resistance can't reduce below 0
- [ ] Precision immunity works
- [ ] "Except" resistances handled

---

## Notes for Implementation

1. **Order is crucial:** Immunity → Weakness → Resistance. This is explicitly stated in rules.

2. **Deadly vs Fatal:**
   - Deadly dX: On crit, add X extra dice of that size
   - Fatal dX: On crit, damage die BECOMES dX AND add 1 dX
   - These are NOT doubled (added after doubling)

3. **Precision damage:** Uses the base damage type. A rogue's Sneak Attack with a dagger deals extra piercing, not "precision" as a type.

4. **Multiple damage types:** An attack might deal "2d6 slashing + 1d6 fire" (flaming sword). Process each separately, then sum.

5. **Critical specialization:** Only applies if the attacker has the feature (usually from Fighter level 7+). For now, can be a flag/option.

6. **Integration with Phase 7:** Strike.Execute should call this pipeline when dealing damage.
