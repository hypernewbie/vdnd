# Phase 5: Items - Weapons & Armour

## Agent Prompt

You are implementing the weapons and armour system for a Pathfinder 2E rules engine in Go. These items are central to combat—weapons determine attack/damage, armour determines AC and mobility penalties.

**Your task:** Implement the `pkg/rules/item` package with full test coverage.

**Prerequisites:** Phases 1-4 should be complete (especially `pkg/rules/trait` for weapon traits and `pkg/rules/dice` for damage dice).

---

## Context

### Source References
- Weapons: `rules/rules/core-rulebook/chapter-6-equipment.md`
- Weapon traits: `rules/rules/traits/`
- Armour: `rules/rules/core-rulebook/chapter-6-equipment.md`

### Weapon Categories (src: chapter-6)
| Category | Description | Proficiency Source |
|----------|-------------|-------------------|
| Unarmed | Fists, claws | Usually trained by default |
| Simple | Basic weapons anyone can use | Most classes trained |
| Martial | Military weapons requiring training | Fighters, Rangers, etc. |
| Advanced | Exotic weapons, hard to master | Specific feats/classes |

### Weapon Groups (src: chapter-6)
Groups determine critical specialisation effects:
- **Sword**, **Axe**, **Hammer**, **Pick**, **Polearm**, **Spear**, **Knife**, **Flail**, **Brawling**, **Club**, **Shield**
- **Bow**, **Crossbow**, **Dart**, **Sling** (ranged)

### Key Weapon Traits
| Trait | Effect |
|-------|--------|
| `agile` | MAP is -4/-8 instead of -5/-10 |
| `finesse` | Can use DEX instead of STR for attack |
| `reach` | Melee reach 10ft instead of 5ft |
| `thrown X` | Can throw with range increment X |
| `versatile X` | Can deal alternate damage type X |
| `deadly dX` | On crit, add X extra damage dice |
| `fatal dX` | On crit, damage die becomes dX and add one dX |
| `two-hand dX` | Damage die is dX when wielded two-handed |
| `propulsive` | Add half STR to damage |
| `ranged` | Uses DEX for attack, no STR to damage |
| `backstabber` | +2 precision damage vs flat-footed |
| `forceful` | +1/+2 damage on 2nd/3rd+ strikes this turn |
| `sweep` | +1 circumstance to hit if previous strike hit different target |

### Armour Categories
| Category | Proficiency | Typical AC Bonus | Dex Cap |
|----------|-------------|------------------|---------|
| Unarmored | All trained | +0 | No cap |
| Light | Trained+ | +1 to +2 | +4 to +5 |
| Medium | Trained+ | +3 to +4 | +2 to +3 |
| Heavy | Trained+ | +5 to +6 | +0 to +1 |

### Armour Penalties
- **Check Penalty:** Applies to STR/DEX skill checks if not proficient
- **Speed Penalty:** Reduces Speed (usually -5ft or -10ft for heavy)

---

## File Structure

```
pkg/
└── rules/
    └── item/
        ├── weapon.go       # Weapon struct, WeaponCategory, WeaponGroup
        ├── armor.go        # Armor struct, ArmorCategory
        ├── damage.go       # DamageType enum
        ├── registry.go     # Pre-built core weapons/armor
        └── item_test.go
```

---

## Implementation Plan

### 1. `pkg/rules/item/damage.go`

```go
// DamageType represents the type of damage dealt
type DamageType string

const (
    Bludgeoning DamageType = "bludgeoning"
    Piercing    DamageType = "piercing"
    Slashing    DamageType = "slashing"
    Fire        DamageType = "fire"
    Cold        DamageType = "cold"
    Electricity DamageType = "electricity"
    Acid        DamageType = "acid"
    Sonic       DamageType = "sonic"
    Force       DamageType = "force"
    Mental      DamageType = "mental"
    Poison      DamageType = "poison"
    Positive    DamageType = "positive"
    Negative    DamageType = "negative"
    Bleed       DamageType = "bleed"
    Precision   DamageType = "precision"
)

// IsPhysical returns true for bludgeoning, piercing, slashing
func (d DamageType) IsPhysical() bool

// IsEnergy returns true for fire, cold, electricity, acid, sonic
func (d DamageType) IsEnergy() bool
```

### 2. `pkg/rules/item/weapon.go`

```go
import (
    "vdnd/pkg/rules/dice"
    "vdnd/pkg/rules/trait"
)

type WeaponCategory int
const (
    CategoryUnarmed WeaponCategory = iota
    CategorySimple
    CategoryMartial
    CategoryAdvanced
)

type WeaponGroup string
const (
    GroupSword   WeaponGroup = "sword"
    GroupAxe     WeaponGroup = "axe"
    GroupHammer  WeaponGroup = "hammer"
    GroupPolearm WeaponGroup = "polearm"
    GroupSpear   WeaponGroup = "spear"
    GroupKnife   WeaponGroup = "knife"
    GroupFlail   WeaponGroup = "flail"
    GroupBrawling WeaponGroup = "brawling"
    GroupClub    WeaponGroup = "club"
    GroupBow     WeaponGroup = "bow"
    GroupCrossbow WeaponGroup = "crossbow"
    GroupDart    WeaponGroup = "dart"
    GroupSling   WeaponGroup = "sling"
    // etc.
)

type Weapon struct {
    ID             string
    Name           string
    Category       WeaponCategory
    Group          WeaponGroup
    Damage         dice.DieRoll   // e.g., {1, 8, 0} for 1d8
    DamageType     DamageType
    Hands          int            // 1 or 2
    Traits         trait.TraitSet
    RangeIncrement int            // 0 for melee, feet for ranged/thrown
    
    // Parameterised Trait Fields (MVP approach)
    ThrownRange    int            // From "thrown X"
    VersatileType  DamageType     // From "versatile X"
    DeadlyDie      dice.DieRoll   // From "deadly dX"
    FatalDie       dice.DieRoll   // From "fatal dX"
    TwoHandDie     dice.DieRoll   // From "two-hand dX"

    // Derived from traits, cached for convenience
    IsRanged       bool
    IsMelee        bool
}

// NewWeapon creates a weapon with basic stats. 
// rangeIncrement: 0 for melee, value for ranged/thrown
func NewWeapon(id, name string, cat WeaponCategory, group WeaponGroup, 
               damage dice.DieRoll, damageType DamageType, hands int, 
               rangeIncrement int, traits ...trait.TraitID) Weapon

// HasTrait checks if weapon has a specific trait
func (w Weapon) HasTrait(id trait.TraitID) bool

// IsAgile returns true if weapon has agile trait
func (w Weapon) IsAgile() bool

// IsFinesse returns true if weapon has finesse trait
func (w Weapon) IsFinesse() bool

// GetReach returns reach in feet (5 normally, 10 with reach trait)
func (w Weapon) GetReach() int

// GetRangeIncrement returns range increment (0 for pure melee)
func (w Weapon) GetRangeIncrement() int
```

### 3. `pkg/rules/item/armor.go`

```go
type ArmorCategory int
const (
    Unarmored ArmorCategory = iota
    LightArmor
    MediumArmor
    HeavyArmor
)

type Armor struct {
    ID           string
    Name         string
    Category     ArmorCategory
    ACBonus      int            // Item bonus to AC
    DexCap       int            // Max DEX to AC (-1 = no cap)
    CheckPenalty int            // Penalty to STR/DEX checks (negative number)
    SpeedPenalty int            // Penalty to Speed (negative number)
    Strength     int            // Min STR to ignore penalties
    Bulk         int
    Traits       trait.TraitSet
}

// NewArmor creates armour with given stats
func NewArmor(id, name string, cat ArmorCategory, acBonus, dexCap, checkPen, speedPen, strength int) Armor

// EffectiveCheckPenalty returns 0 if character meets STR requirement
func (a Armor) EffectiveCheckPenalty(strength int) int

// EffectiveSpeedPenalty returns 0 if character meets STR requirement  
func (a Armor) EffectiveSpeedPenalty(strength int) int

// AppliedDexBonus caps DEX bonus at DexCap
func (a Armor) AppliedDexBonus(dexMod int) int
```

### 4. `pkg/rules/item/registry.go`

Pre-built weapons and armour from the core rules.

```go
// Core Melee Weapons
var (
    Fist       = NewWeapon("fist", "Fist", Unarmed, Brawling, 
                           dice.DieRoll{1, 4, 0}, Bludgeoning, 1, 0,
                           trait.TraitAgile, trait.TraitFinesse, trait.TraitNonlethal)
    
    Dagger    = NewWeapon("dagger", "Dagger", Simple, Knife,
                           dice.DieRoll{1, 4, 0}, Piercing, 1, 10,
                           trait.TraitAgile, trait.TraitFinesse, trait.TraitThrown, trait.TraitVersatile)
    // Note: Dagger should set VersatileType=Slashing manually in a builder block or helper

    Longsword = NewWeapon("longsword", "Longsword", Martial, Sword,
                           dice.DieRoll{1, 8, 0}, Slashing, 1, 0,
                           trait.TraitVersatile)
    
    Greatsword = NewWeapon("greatsword", "Greatsword", Martial, Sword,
                            dice.DieRoll{1, 12, 0}, Slashing, 2, 0)
    
    Rapier    = NewWeapon("rapier", "Rapier", Martial, Sword,
                           dice.DieRoll{1, 6, 0}, Piercing, 1, 0,
                           trait.TraitDeadly, trait.TraitDisarm, trait.TraitFinesse)
)

// Core Ranged Weapons
var (
    Shortbow  = NewWeapon("shortbow", "Shortbow", Martial, Bow,
                           dice.DieRoll{1, 6, 0}, Piercing, 2, 60,
                           trait.TraitDeadly)
    
    Crossbow  = NewWeapon("crossbow", "Crossbow", Simple, Crossbow,
                           dice.DieRoll{1, 8, 0}, Piercing, 2, 120)
)

// Core Armor
var (
    NoArmor       = NewArmor("unarmored", "Unarmored", Unarmored, 0, -1, 0, 0, 0)
    LeatherArmor  = NewArmor("leather", "Leather Armor", LightArmor, 1, 4, -1, 0, 10)
    ChainShirt    = NewArmor("chain-shirt", "Chain Shirt", LightArmor, 2, 3, -1, 0, 12)
    ChainMail     = NewArmor("chain-mail", "Chain Mail", MediumArmor, 4, 1, -2, -5, 16)
    PlateArmor    = NewArmor("plate", "Full Plate", HeavyArmor, 6, 0, -3, -10, 18)
)

// GetWeapon returns a weapon by ID
func GetWeapon(id string) (Weapon, bool)

// GetArmor returns armor by ID
func GetArmor(id string) (Armor, bool)
```

---

## Test Plan

### DamageType Tests

| Test | Input | Expected |
|------|-------|----------|
| Bludgeoning is physical | `Bludgeoning.IsPhysical()` | true |
| Fire is not physical | `Fire.IsPhysical()` | false |
| Fire is energy | `Fire.IsEnergy()` | true |
| Slashing is not energy | `Slashing.IsEnergy()` | false |
| Mental is neither | `Mental.IsPhysical()`, `Mental.IsEnergy()` | false, false |

### Weapon Tests

| Test | Weapon | Check | Expected |
|------|--------|-------|----------|
| Dagger has agile | Dagger | `IsAgile()` | true |
| Longsword not agile | Longsword | `IsAgile()` | false |
| Rapier has finesse | Rapier | `IsFinesse()` | true |
| Greatsword needs 2 hands | Greatsword | `Hands` | 2 |
| Longsword reach | Longsword | `GetReach()` | 5 |
| Glaive reach | (with reach trait) | `GetReach()` | 10 |
| HasTrait works | Dagger | `HasTrait(TraitThrown)` | true |
| HasTrait negative | Dagger | `HasTrait(TraitReach)` | false |

### Armor Tests

| Test | Armor | Check | Expected |
|------|-------|-------|----------|
| Leather AC bonus | Leather | `ACBonus` | 1 |
| Plate AC bonus | Plate | `ACBonus` | 6 |
| Leather dex cap | Leather | `DexCap` | 4 |
| Plate dex cap | Plate | `DexCap` | 0 |
| DEX +5 with cap 4 | Leather | `AppliedDexBonus(5)` | 4 |
| DEX +2 with cap 4 | Leather | `AppliedDexBonus(2)` | 2 |
| No armor no cap | NoArmor | `AppliedDexBonus(10)` | 10 |

### Armor Penalty Tests

| Test | Armor | STR | Expected |
|------|-------|-----|----------|
| Chain mail check penalty | ChainMail (STR 14 req) | STR 12 | -2 |
| Chain mail check penalty met | ChainMail | STR 16 | 0 |
| Plate speed penalty | Plate (STR 16 req) | STR 14 | -10 |
| Plate speed penalty met | Plate | STR 18 | 0 |
| Light armor no STR check | Leather | STR 8 | -1 (always applies if not proficient) |

### Registry Tests

| Test | Expected |
|------|----------|
| GetWeapon("longsword") | Returns Longsword, true |
| GetWeapon("nonexistent") | Returns zero, false |
| GetArmor("leather") | Returns LeatherArmor, true |
| All core weapons registered | Check for dagger, longsword, greataxe, etc. |

---

## Validation Checklist

- [ ] All tests pass
- [ ] `go vet` reports no issues
- [ ] `go fmt` applied
- [ ] Weapon traits correctly linked
- [ ] Armor penalties calculate correctly with STR requirements
- [ ] DexCap of -1 means no cap
- [ ] Registry has all common PF2E weapons

---

## Notes for Implementation

1. **DexCap -1 means no cap:** Use a sentinel value. `AppliedDexBonus` should check for this.

2. **Traits are IDs, not full Trait objects:** We store `[]TraitID` and lookup in the trait registry when needed.

3. **Range for ranged weapons:** Set `RangeIncrement` separately or in constructor. 0 = melee only.

4. **Thrown weapons:** Have both `IsMelee = true` and `RangeIncrement > 0`.

5. **Two-hand trait:** Weapons like longsword have `versatile P, two-hand d10`. When wielded two-handed, damage die changes. Store as trait parameter.

6. **Future: Magic weapons:** Will extend Weapon with Potency rune (+1/+2/+3), Striking rune (extra damage dice), property runes. For now, just base weapons.
