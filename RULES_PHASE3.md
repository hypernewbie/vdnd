# Phase 3: Traits System

## Agent Prompt

You are implementing the traits system for a Pathfinder 2E rules engine in Go. Traits are keywords that tag game elements (weapons, spells, actions, creatures) and modify how rules interact with them.

**Your task:** Implement the `pkg/trait` package with full test coverage.

**Prerequisites:** Phase 1 and 2 should be complete.

---

## Context

Traits are ubiquitous in PF2E. Every weapon, spell, action, condition, and creature has traits. They serve multiple purposes:
- **Categorisation:** Identifying what something is (e.g., `fire`, `arcane`, `humanoid`)
- **Rule modification:** Changing how mechanics work (e.g., `agile` reduces MAP, `finesse` lets you use DEX)
- **Interaction triggers:** Determining what affects what (e.g., `mental` damage doesn't affect mindless creatures)

### Source References
- Trait definitions: `rules/rules/traits/` (individual markdown files per trait)
- Weapon traits: `rules/rules/core-rulebook/chapter-6-equipment.md`
- Action traits: `rules/rules/core-rulebook/chapter-9-playing-the-game.md`

### Key Traits to Implement

**Weapon Traits:**
| Trait | Effect |
|-------|--------|
| `agile` | MAP is -4/-8 instead of -5/-10 |
| `finesse` | Can use DEX instead of STR for melee attacks |
| `reach` | Melee reach is 10ft instead of 5ft |
| `thrown <range>` | Can be thrown with specified range increment |
| `versatile <type>` | Can deal alternate damage type |
| `deadly <die>` | Extra damage die on critical hit |
| `fatal <die>` | Damage die increases and adds extra die on crit |
| `two-hand <die>` | Different damage die when wielded in two hands |
| `backstabber` | +2 precision damage vs flat-footed |
| `forceful` | +1 damage per previous Strike this turn |
| `sweep` | +1 circumstance to attack if previous Strike hit different target |

**Damage Type Traits:**
`bludgeoning`, `piercing`, `slashing`, `fire`, `cold`, `electricity`, `acid`, `sonic`, `force`, `mental`, `poison`, `positive`, `negative`, `chaotic`, `evil`, `good`, `lawful`, `bleed`, `precision`

**Action Traits:**
| Trait | Meaning |
|-------|---------|
| `attack` | Counts toward MAP, can trigger reactions |
| `move` | Movement action, can trigger reactions |
| `manipulate` | Uses hands, can trigger reactions like AoO |
| `concentrate` | Requires focus, disrupted by certain effects |
| `auditory` | Requires hearing to perceive |
| `visual` | Requires sight to perceive |
| `linguistic` | Requires shared language |
| `mental` | Affects the mind, ignored by mindless creatures |
| `emotion` | Affects emotions, immunity exists |
| `fear` | Subset of emotion, causes frightened |

**Creature Traits:**
`humanoid`, `beast`, `construct`, `undead`, `fiend`, `celestial`, `dragon`, `elemental`, `fey`, `giant`, `ooze`, `plant`, `mindless`, `incorporeal`

---

## File Structure

```
pkg/
└── rules/
    └── trait/
        ├── trait.go        # TraitID, Trait struct, TraitCategory
        ├── registry.go     # Global trait registry with all known traits
        ├── hastraits.go    # HasTraits interface
        └── trait_test.go
```

---

## Implementation Plan

### 1. `pkg/rules/trait/trait.go`

```go
// TraitID is a unique identifier for a trait (lowercase, hyphenated)
type TraitID string

// Common trait IDs as constants for type safety
const (
    TraitAgile     TraitID = "agile"
    TraitFinesse   TraitID = "finesse"
    TraitReach     TraitID = "reach"
    TraitAttack    TraitID = "attack"
    TraitMove      TraitID = "move"
    TraitFire      TraitID = "fire"
    TraitMental    TraitID = "mental"
    // ... etc
)

type TraitCategory int
const (
    CategoryWeapon TraitCategory = iota
    CategoryArmor
    CategoryAction
    CategorySpell
    CategoryDamage
    CategoryCreature
    CategoryCondition
    CategoryGeneral
)

type Trait struct {
    ID          TraitID
    Name        string        // Display name: "Agile"
    Description string        // Full rules text
    Category    TraitCategory
    // Optional: parameters for traits like "deadly d10" or "thrown 20"
    Parameter   string        // "d10", "20", "slashing", etc.
}

// NewTrait creates a trait with basic info
func NewTrait(id TraitID, name string, category TraitCategory) Trait

// NewParameterizedTrait creates a trait with a parameter
func NewParameterizedTrait(id TraitID, name string, category TraitCategory, param string) Trait
```

### 2. `pkg/rules/trait/registry.go`

A global registry of all known traits. This allows lookup and validation.

```go
// Registry holds all known traits
type Registry struct {
    traits map[TraitID]Trait
}

// DefaultRegistry returns a registry pre-populated with core PF2E traits
func DefaultRegistry() *Registry

// Get retrieves a trait by ID, returns (trait, found)
func (r *Registry) Get(id TraitID) (Trait, bool)

// Register adds a new trait to the registry
func (r *Registry) Register(t Trait)

// Has checks if a trait ID is known
func (r *Registry) Has(id TraitID) bool

// AllInCategory returns all traits of a given category
func (r *Registry) AllInCategory(cat TraitCategory) []Trait
```

**Pseudocode for DefaultRegistry:**
```
func DefaultRegistry() *Registry:
    r := new Registry
    
    // Weapon traits
    r.Register(Trait{ID: "agile", Name: "Agile", Category: Weapon})
    r.Register(Trait{ID: "finesse", Name: "Finesse", Category: Weapon})
    r.Register(Trait{ID: "reach", Name: "Reach", Category: Weapon})
    r.Register(Trait{ID: "deadly", Name: "Deadly", Category: Weapon})
    r.Register(Trait{ID: "fatal", Name: "Fatal", Category: Weapon})
    // ...
    
    // Damage types
    for _, dmg := range ["fire", "cold", "electricity", "acid", "sonic", "force", 
                         "mental", "poison", "positive", "negative",
                         "bludgeoning", "piercing", "slashing"]:
        r.Register(Trait{ID: dmg, Name: Title(dmg), Category: Damage})
    
    // Action traits
    r.Register(Trait{ID: "attack", Name: "Attack", Category: Action})
    r.Register(Trait{ID: "move", Name: "Move", Category: Action})
    r.Register(Trait{ID: "manipulate", Name: "Manipulate", Category: Action})
    // ...
    
    return r
```

### 3. `pkg/rules/trait/hastraits.go`

Interface for anything that has traits (weapons, spells, creatures, etc.)

```go
// HasTraits is implemented by anything with traits
type HasTraits interface {
    Traits() []TraitID
    HasTrait(id TraitID) bool
}

// TraitSet is a helper type for storing traits on a struct
type TraitSet []TraitID

func (ts TraitSet) Traits() []TraitID {
    return ts
}

func (ts TraitSet) HasTrait(id TraitID) bool {
    for _, t := range ts {
        if t == id {
            return true
        }
    }
    return false
}

// HasAnyTrait checks if the thing has any of the given traits
func HasAnyTrait(h HasTraits, ids ...TraitID) bool

// HasAllTraits checks if the thing has all of the given traits
func HasAllTraits(h HasTraits, ids ...TraitID) bool
```

---

## Test Plan

### Trait Creation Tests

| Test Case | Input | Expected |
|-----------|-------|----------|
| Create basic trait | `NewTrait("agile", "Agile", Weapon)` | Trait with correct fields |
| Create parameterized trait | `NewParameterizedTrait("deadly", "Deadly", Weapon, "d10")` | Trait with param="d10" |

### Registry Tests

| Test Case | Expected |
|-----------|----------|
| DefaultRegistry contains "agile" | `Get("agile")` returns trait, true |
| DefaultRegistry contains "fire" | `Get("fire")` returns trait, true |
| Unknown trait returns false | `Get("nonexistent")` returns zero, false |
| Register new trait | After `Register()`, `Get()` finds it |
| AllInCategory returns correct traits | `AllInCategory(Damage)` includes fire, cold, etc. |

### TraitSet Tests

| Test Case | TraitSet | Check | Expected |
|-----------|----------|-------|----------|
| HasTrait finds match | `["agile", "finesse"]` | `HasTrait("finesse")` | true |
| HasTrait no match | `["agile", "finesse"]` | `HasTrait("reach")` | false |
| Empty set | `[]` | `HasTrait("agile")` | false |
| Traits returns slice | `["a", "b", "c"]` | `Traits()` | `["a", "b", "c"]` |

### HasAnyTrait / HasAllTraits Tests

| Test Case | Traits | Query | Expected |
|-----------|--------|-------|----------|
| HasAnyTrait - one match | `["fire", "cold"]` | `HasAnyTrait(fire, acid)` | true |
| HasAnyTrait - no match | `["fire", "cold"]` | `HasAnyTrait(acid, sonic)` | false |
| HasAllTraits - all present | `["fire", "cold", "spell"]` | `HasAllTraits(fire, cold)` | true |
| HasAllTraits - missing one | `["fire", "spell"]` | `HasAllTraits(fire, cold)` | false |

### Edge Cases

| Test Case | Expected |
|-----------|----------|
| Empty TraitSet.HasTrait | false |
| HasAnyTrait with empty query | false (or true? - define behaviour) |
| HasAllTraits with empty query | true (vacuously true) |
| Case sensitivity | TraitID should be lowercase, "Fire" != "fire" |

---

## Validation Checklist

- [ ] All tests pass
- [ ] `go vet` reports no issues
- [ ] `go fmt` applied
- [ ] DefaultRegistry has all common weapon/damage/action traits
- [ ] TraitID constants match string values
- [ ] HasTraits interface can be implemented by external types

---

## Notes for Implementation

1. **TraitID as string:** Using `type TraitID string` gives type safety while allowing easy creation from strings (for parsing).

2. **Parameterized traits:** Some traits have parameters (e.g., "deadly d10", "thrown 20"). Store the parameter separately rather than encoding it in the ID.

3. **Registry is read-mostly:** In practice, the registry is populated at startup and then only read. Consider making DefaultRegistry return a singleton.

4. **Future integration:**
   - Weapons will use `TraitSet` to store their traits
   - Combat code will check `HasTrait(TraitAgile)` to calculate MAP
   - Damage code will check `HasTrait(TraitFire)` for weaknesses/resistances

5. **Don't over-engineer:** The registry is a simple lookup. Traits don't have behaviour attached - that lives in the combat/damage code that checks for traits.
