# Phase 4: Conditions System

## Agent Prompt

You are implementing the conditions system for a Pathfinder 2E rules engine in Go. Conditions are persistent status effects that modify a creature's capabilities - from being frightened to dying.

**Your task:** Implement the `pkg/condition` package with full test coverage.

**Prerequisites:** Phases 1-3 should be complete (especially `pkg/check` for modifiers).

---

## Context

Conditions are ongoing effects that change how a creature acts or what bonuses/penalties apply to them. Many conditions have a **value** (e.g., Frightened 2, Sickened 3) that determines their severity.

### Source Reference
- Full condition rules: `rules/rules/conditions.md`

### Condition Categories

**Valued Conditions** (have a numeric value):
| Condition | Effect |
|-----------|--------|
| `frightened X` | -X status penalty to ALL checks and DCs. Reduces by 1 at end of turn. |
| `sickened X` | -X status penalty to ALL checks and DCs. Can't ingest. |
| `clumsy X` | -X status penalty to DEX-based checks and DCs (AC, Reflex, ranged attacks). |
| `enfeebled X` | -X status penalty to STR-based checks (melee attacks, Athletics). |
| `stupefied X` | -X status penalty to INT/WIS/CHA checks; flat check to cast spells. |
| `drained X` | -X status penalty to CON checks; lose X×level max HP. |
| `dying X` | Unconscious, must make recovery checks. Die at dying 4. |
| `wounded X` | When you gain dying, add wounded value to dying value. |
| `doomed X` | Reduces the dying value at which you die. |
| `slowed X` | Lose X actions at start of turn. |
| `stunned X` | Lose X total actions (can span turns). |

**Binary Conditions** (on or off):
| Condition | Effect |
|-----------|--------|
| `flat-footed` | -2 circumstance penalty to AC. |
| `prone` | Flat-footed, -2 circumstance to attacks, must Crawl or Stand. |
| `grabbed` | Flat-footed, immobilized, DC 5 flat check for manipulate actions. |
| `restrained` | Flat-footed, immobilized, can't use attack/manipulate except Escape. |
| `immobilized` | Can't use move actions. |
| `blinded` | Can't see, -4 status to Perception if vision is only precise sense. |
| `deafened` | Can't hear, -2 status to Perception for initiative. |
| `invisible` | Can't be seen, undetected. |
| `hidden` | DC 11 flat check to target. |
| `paralyzed` | Flat-footed, can't act except Recall Knowledge. |
| `unconscious` | Can't act, flat-footed, -4 status to AC/Perception/Reflex. |
| `fatigued` | -1 status penalty to AC and saves. |
| `fascinated` | -2 status to Perception and skill checks. |
| `quickened` | Gain 1 extra action (restricted type). |
| `fleeing` | Must spend actions running away. |
| `confused` | Attack randomly, flat-footed. |

**Special Conditions:**
| Condition | Effect |
|-----------|--------|
| `persistent damage X type` | Take X damage of type at end of turn; DC 15 flat check to end. |

---

## File Structure

```
pkg/
└── rules/
    └── condition/
        ├── condition.go      # ConditionID, Condition definition
        ├── instance.go       # ConditionInstance (applied to entity)
        ├── tracker.go        # ConditionTracker (manages conditions on entity)
        ├── effects.go        # Extract modifiers from conditions
        ├── registry.go       # Known condition definitions
        └── condition_test.go
```

---

## Implementation Plan

### 1. `pkg/rules/condition/condition.go`

```go
type ConditionID string

// Common condition IDs
const (
    Frightened    ConditionID = "frightened"
    Sickened      ConditionID = "sickened"
    Clumsy        ConditionID = "clumsy"
    Enfeebled     ConditionID = "enfeebled"
    Stupefied     ConditionID = "stupefied"
    Drained       ConditionID = "drained"
    FlatFooted    ConditionID = "flat-footed"
    Prone         ConditionID = "prone"
    Grabbed       ConditionID = "grabbed"
    Restrained    ConditionID = "restrained"
    Immobilized   ConditionID = "immobilized"
    Blinded       ConditionID = "blinded"
    Deafened      ConditionID = "deafened"
    Invisible     ConditionID = "invisible"
    Hidden        ConditionID = "hidden"
    Paralyzed     ConditionID = "paralyzed"
    Unconscious   ConditionID = "unconscious"
    Dying         ConditionID = "dying"
    Wounded       ConditionID = "wounded"
    Doomed        ConditionID = "doomed"
    Slowed        ConditionID = "slowed"
    Stunned       ConditionID = "stunned"
    Quickened     ConditionID = "quickened"
    Fatigued      ConditionID = "fatigued"
    Fascinated    ConditionID = "fascinated"
    Fleeing       ConditionID = "fleeing"
    Confused      ConditionID = "confused"
    PersistentDamage ConditionID = "persistent-damage"
)

type Condition struct {
    ID          ConditionID
    Name        string
    HasValue    bool        // true for valued conditions like Frightened X
    Description string
}
```

### 2. `pkg/rules/condition/instance.go`

An instance of a condition applied to a specific entity.

```go
type ConditionInstance struct {
    ID        ConditionID
    Value     int         // For valued conditions; 0 for binary conditions
    Duration  int         // Rounds remaining; -1 = until removed
    Source    string      // What caused this: "Demoralize", "Dragon Breath", etc.
    
    // For persistent damage
    DamageType string     // "fire", "bleed", etc. (only for persistent damage)
}

// NewCondition creates a binary condition instance
func NewCondition(id ConditionID, source string) ConditionInstance

// NewValuedCondition creates a valued condition instance
func NewValuedCondition(id ConditionID, value int, source string) ConditionInstance

// NewPersistentDamage creates a persistent damage condition
func NewPersistentDamage(amount int, damageType, source string) ConditionInstance
```

### 3. `pkg/rules/condition/tracker.go`

Manages conditions on a single entity.

```go
type ConditionTracker struct {
    conditions map[ConditionID]*ConditionInstance
    // Persistent damage is special - can have multiple of different types
    persistentDamage []ConditionInstance
}

func NewTracker() *ConditionTracker

// Apply adds or updates a condition
// For valued conditions: takes the HIGHER value if already present
func (t *ConditionTracker) Apply(c ConditionInstance)

// Remove completely removes a condition
func (t *ConditionTracker) Remove(id ConditionID)

// Reduce decreases a valued condition's value; removes if reduced to 0
func (t *ConditionTracker) Reduce(id ConditionID, amount int)

// Has checks if the entity has a specific condition
func (t *ConditionTracker) Has(id ConditionID) bool

// Get returns the condition instance, or nil if not present
func (t *ConditionTracker) Get(id ConditionID) *ConditionInstance

// Value returns the value of a condition (0 if not present or binary)
func (t *ConditionTracker) Value(id ConditionID) int

// All returns all active conditions
func (t *ConditionTracker) All() []ConditionInstance

// EndTurn processes end-of-turn effects (reduce frightened, etc.)
func (t *ConditionTracker) EndTurn()

// StartTurn processes start-of-turn effects
func (t *ConditionTracker) StartTurn()
```

**Apply Pseudocode:**
```
func (t *ConditionTracker) Apply(c ConditionInstance):
    if c.ID == PersistentDamage:
        # Persistent damage: check if same type exists
        for existing in t.persistentDamage:
            if existing.DamageType == c.DamageType:
                # Take higher value
                existing.Value = max(existing.Value, c.Value)
                return
        # New damage type
        t.persistentDamage = append(t.persistentDamage, c)
        return
    
    existing := t.conditions[c.ID]
    if existing == nil:
        t.conditions[c.ID] = &c
    else:
        # For valued conditions, take higher value
        existing.Value = max(existing.Value, c.Value)
        # Source might update to new source, or keep original - your choice
```

**EndTurn Pseudocode:**
```
func (t *ConditionTracker) EndTurn():
    # Frightened reduces by 1 at end of each turn
    if t.Has(Frightened):
        t.Reduce(Frightened, 1)
    
    # Duration-based conditions tick down
    for id, cond in t.conditions:
        if cond.Duration > 0:
            cond.Duration--
            if cond.Duration == 0:
                t.Remove(id)
```

### 4. `pkg/rules/condition/effects.go`

Extracts game effects from conditions. Integrates with `pkg/check`.

```go
import "vdnd/pkg/rules/check"

// GetModifiers returns all modifiers from active conditions
// This is used when calculating checks
func (t *ConditionTracker) GetModifiers() []check.Modifier

// GetACModifiers returns modifiers that apply to AC
func (t *ConditionTracker) GetACModifiers() []check.Modifier

// GetAttackModifiers returns modifiers that apply to attack rolls
func (t *ConditionTracker) GetAttackModifiers() []check.Modifier

// GetSaveModifiers returns modifiers that apply to saving throws
func (t *ConditionTracker) GetSaveModifiers() []check.Modifier

// GetActionsLost returns how many actions are lost (from slowed/stunned)
func (t *ConditionTracker) GetActionsLost() int
```

**GetModifiers Pseudocode:**
```
func (t *ConditionTracker) GetModifiers() []check.Modifier:
    mods := []
    
    if t.Has(Frightened):
        mods = append(mods, Modifier{-t.Value(Frightened), Status, "Frightened"})
    
    if t.Has(Sickened):
        mods = append(mods, Modifier{-t.Value(Sickened), Status, "Sickened"})
    
    if t.Has(Fatigued):
        mods = append(mods, Modifier{-1, Status, "Fatigued"})
    
    if t.Has(Fascinated):
        mods = append(mods, Modifier{-2, Status, "Fascinated"})
    
    # Note: flat-footed, clumsy, enfeebled, etc. only apply to specific checks
    # Those are returned by the more specific methods
    
    return mods
```

**GetACModifiers Pseudocode:**
```
func (t *ConditionTracker) GetACModifiers() []check.Modifier:
    mods := t.GetModifiers()  # Start with universal modifiers
    
    if t.Has(FlatFooted):
        mods = append(mods, Modifier{-2, Circumstance, "Flat-footed"})
    
    if t.Has(Clumsy):
        mods = append(mods, Modifier{-t.Value(Clumsy), Status, "Clumsy"})
    
    if t.Has(Prone):
        # Prone already makes you flat-footed, but explicitly adds -2 to attacks
        # AC effect is via flat-footed
        pass
    
    if t.Has(Unconscious):
        mods = append(mods, Modifier{-4, Status, "Unconscious"})
    
    return mods
```

---

## Test Plan

### ConditionInstance Creation Tests

| Test | Input | Expected |
|------|-------|----------|
| Binary condition | `NewCondition(FlatFooted, "Flanked")` | Value=0, Source="Flanked" |
| Valued condition | `NewValuedCondition(Frightened, 2, "Dragon")` | Value=2, Source="Dragon" |
| Persistent damage | `NewPersistentDamage(5, "fire", "Torch")` | ID=PersistentDamage, Value=5, DamageType="fire" |

### ConditionTracker Tests

| Test | Actions | Expected |
|------|---------|----------|
| Apply new condition | Apply Frightened 2 | Has(Frightened)=true, Value=2 |
| Apply higher value | Apply Frightened 2, then Frightened 3 | Value=3 |
| Apply lower value (no change) | Apply Frightened 3, then Frightened 1 | Value=3 |
| Remove condition | Apply Frightened 2, Remove | Has(Frightened)=false |
| Reduce valued condition | Apply Frightened 3, Reduce by 1 | Value=2 |
| Reduce to zero removes | Apply Frightened 1, Reduce by 1 | Has(Frightened)=false |
| Has returns false if absent | Fresh tracker | Has(Blinded)=false |
| Value returns 0 if absent | Fresh tracker | Value(Frightened)=0 |
| Binary condition Value | Apply FlatFooted | Value=0 (binary) |

### EndTurn Tests

| Test | Initial State | After EndTurn | Expected |
|------|---------------|---------------|----------|
| Frightened reduces | Frightened 3 | EndTurn | Frightened 2 |
| Frightened 1 removes | Frightened 1 | EndTurn | No frightened |
| Sickened doesn't auto-reduce | Sickened 2 | EndTurn | Sickened 2 |
| Duration expires | Condition with Duration=1 | EndTurn | Removed |
| Duration -1 persists | Condition with Duration=-1 | EndTurn | Still present |

### Modifier Extraction Tests

| Conditions | GetModifiers() | Expected |
|------------|---------------|----------|
| Frightened 2 | All modifiers | Contains `{-2, Status, "Frightened"}` |
| Frightened 2 + Sickened 1 | All modifiers | Two separate status penalties |
| Flat-footed | GetACModifiers | Contains `{-2, Circumstance, "Flat-footed"}` |
| Clumsy 2 | GetACModifiers | Contains `{-2, Status, "Clumsy"}` |
| Enfeebled 3 | GetAttackModifiers (melee) | Contains `{-3, Status, "Enfeebled"}` |
| Fatigued | GetSaveModifiers | Contains `{-1, Status, "Fatigued"}` |
| Unconscious | GetACModifiers | Contains `{-4, Status, "Unconscious"}` |

### Persistent Damage Tests

| Test | Actions | Expected |
|------|---------|----------|
| Add persistent fire | `NewPersistentDamage(5, "fire", src)` | HasPersistentDamage("fire")=true |
| Add different type | Add fire 5, then bleed 3 | Both present |
| Same type takes higher | Add fire 5, then fire 8 | fire value = 8 |
| Same type ignores lower | Add fire 8, then fire 3 | fire value = 8 |

### Edge Cases

| Test | Expected |
|------|----------|
| Reduce by more than value | Removes condition (can't go negative) |
| Apply condition with value 0 | For valued condition, this is weird - handle gracefully |
| Remove non-existent condition | No error, no-op |

---

## Validation Checklist

- [ ] All tests pass
- [ ] `go vet` reports no issues
- [ ] `go fmt` applied
- [ ] Frightened auto-reduces at end of turn
- [ ] Valued conditions take higher value when stacked
- [ ] Modifier extraction produces correct types (Status vs Circumstance)
- [ ] Persistent damage handles multiple damage types

---

## Notes for Implementation

1. **Modifier types matter:** Flat-footed is a *circumstance* penalty (can stack with status penalties). Most other conditions give *status* penalties.

2. **Condition interactions:**
   - Prone implies Flat-footed
   - Grabbed implies Flat-footed + Immobilized
   - Restrained implies everything Grabbed does, plus more
   - Consider helper methods: `IsEffectivelyFlatFooted()` that checks for any of these

3. **EndTurn vs StartTurn:**
   - Frightened reduces at END of turn
   - Slowed/Stunned consume actions at START of turn
   - Duration countdown typically happens at start of the condition-giver's turn (complex, simplify for now)

4. **Future: Condition immunity:** Some creatures are immune to certain conditions (e.g., mindless creatures immune to mental effects). This will be handled in the entity package.

5. **Persistent damage edge cases:**
   - Multiple types stack (fire + bleed = both apply)
   - Same type: only highest applies
   - DC 15 flat check at end of turn to end each (handled in combat loop, not here)
