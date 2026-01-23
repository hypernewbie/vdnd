# VD Phase 2: State Management

> **Status:** ✅ Implemented (by Gemini 3 Pro, who jumped the gun)

This phase establishes game state persistence and the core data structures.

---

## What Was Built

### File Structure

```
internal/state/
├── state.go    # GameState and all related structs
└── store.go    # Store interface, FileStore, MemoryStore

internal/parser/
└── entity.go   # Markdown entity parser (technically Phase 3, but done early)
```

---

## GameState Structure

The central state object persisted as `state.json`:

```go
type GameState struct {
    // Scene
    SceneName string
    Positions map[string]*Zone

    // Entities
    Entities map[string]*EntityState

    // Combat
    InCombat         bool
    Round            int
    InitiativeOrder  []string
    CurrentTurn      string
    TurnIndex        int
    ActionsRemaining int
    ReactionsUsed    map[string]bool
    AttacksMade      int  // For MAP calculation

    // Pending reactions
    PendingEvents []PendingEvent
}
```

### Zone (Positioning)

Abstract zone-based positioning (not grid):

```go
type Zone struct {
    Name     string
    Size     string   // small, medium, large
    Adjacent []string // One move action away
    Near     []string // Two moves away
    Far      []string // Three+ moves
    Cover    string   // none, lesser, standard, greater
    Elevated bool
    Notes    string
}
```

### EntityState

Full entity representation for persistence:

```go
type EntityState struct {
    // Identity
    ID, Name string
    Level    int

    // Flavour (strings, not full systems)
    Ancestry, Class, Background string

    // Combat stats
    HP, MaxHP, TempHP int
    AC, Speed         int

    // Saves (total bonuses)
    Fortitude, Reflex, Will int

    // Abilities (uses pkg/rules/ability.AbilityScores)
    Abilities ability.AbilityScores

    // Position
    Position    string
    EngagedWith []string

    // Conditions
    Conditions []ConditionInstance

    // Equipment (simplified)
    WieldedWeapons []WeaponState
    WornArmor      *ArmorState
    RaisedShield   bool

    // Defences
    Immunities  []string
    Weaknesses  map[string]int
    Resistances map[string]int

    // Available reactions
    Reactions []string
}
```

### Supporting Types

```go
type ConditionInstance struct {
    ID       string
    Value    int    // For valued conditions (Frightened 2)
    Duration int    // -1 = until removed
    Source   string
}

type WeaponState struct {
    ID, Damage, DamageType string
}

type ArmorState struct {
    ID      string
    ACBonus int
}

type PendingEvent struct {
    ID, Type, Actor, Description string
    Reactors  []string
    Reactions []AvailableReaction
}

type AvailableReaction struct {
    Entity, Reaction, Trigger string
}
```

---

## Store Interface

```go
type Store interface {
    Load() (*GameState, error)
    Save(state *GameState) error
    Exists() bool
}
```

### FileStore (Production)

- Persists to `state.json` in CWD
- Pretty-printed JSON with indentation
- Used by `DefaultDeps()`

### MemoryStore (Testing)

- Holds state in memory
- No file I/O
- Inject into `Deps` for tests

---

## Scene Commands

### `vd scene new <name>`

- Creates new GameState with given name
- Initialises empty Positions and Entities maps
- Saves to state.json
- **Error** if session already exists

### `vd scene save`

- Explicit checkpoint (mostly no-op in stateless CLI)
- **Error** if no active session

### `vd scene load`

- Stub - not fully implemented yet

---

## Entity Parser

Parses markdown format into `EntityState`:

```markdown
# Sir Roland
- Level: 5
- HP: 60/60
- AC: 22
- Speed: 25ft
- Strength: 18
- Dexterity: 12
- Fortitude: +12
- Reflex: +8
- Will: +10
- Ancestry: Human
- Class: Paladin
```

Supports:
- `#` for name
- Full or abbreviated ability names (str/strength)
- HP as `current/max` or just `current`
- Modifiers with or without `+` prefix
- Speed with or without `ft` suffix

---

## Tests

### Scene Command Tests

- `TestSceneNew` - Creates scene, verifies state saved
- `TestSceneNew_Exists` - Errors when session already exists

---

## JSON State File Example

```json
{
  "sceneName": "The Dungeon",
  "positions": {},
  "entities": {
    "paladin": {
      "id": "paladin",
      "name": "Sir Roland",
      "level": 5,
      "hp": 60,
      "maxHp": 60,
      "ac": 22,
      "abilities": {
        "strength": 18,
        "dexterity": 12
      },
      "position": "entrance"
    }
  },
  "inCombat": false,
  "round": 0
}
```

---

## What's NOT Done Yet

- `scene load` from template directories
- Entity add/spawn commands (Phase 3)
- Position/zone management commands
- State validation

---

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| JSON persistence | Human-readable, easy to debug, LLM-friendly |
| Single `state.json` | Simple, atomic saves, no partial state |
| `EntityState` separate from `pkg/rules/entity` | State is persistence format, rules engine uses richer types |
| String-based conditions/reactions | Flexible, no enum coupling, LLM can reason about them |
| `AbilityScores` from rules package | Reuse existing type, consistent modifier calculations |
