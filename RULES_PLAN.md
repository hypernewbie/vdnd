# Pathfinder 2E Rules Engine - Implementation Plan

This document outlines a step-by-step plan to implement the mechanical aspects of the Pathfinder 2E ruleset as a Go project. The focus is on the hard game rules—dice rolling, stat calculations, combat resolution, conditions, and effects—not on narrative or GM-discretion elements.

---

## Architecture Overview

### LLM-as-Orchestrator Design

The engine is designed to be driven by an LLM acting as the Game Master. The LLM makes all decisions and calls the CLI to execute game mechanics.

```
┌─────────────────────────────────────────────────────────────┐
│                    LLM (GM/Orchestrator)                    │
│                                                             │
│  Reads CLI output, makes decisions, issues next command     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                         vd CLI                              │
│                                                             │
│  Stateless commands that manipulate global game state       │
│  Returns structured markdown for LLM to parse               │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              Global Game State (CWD-based)                  │
│                                                             │
│  - Structured markdown files for entities/items             │
│  - Rulebook data hardcoded in Go                            │
│  - Runtime state in state.json                              │
└─────────────────────────────────────────────────────────────┘
```

### Key Design Decisions

| Topic | Decision |
|-------|----------|
| **Character depth** | Ancestry/class/background as string fields, not full progression systems |
| **Spell effects** | Hardcoded in Go, with code helpers for common patterns |
| **Reactions** | Event-driven system with CLI commands for mid-action decisions |
| **Persistence** | CWD-based sessions with structured markdown files |
| **UI** | CLI only, markdown output |
| **Testing** | Unit tests with scenario data files |
| **Positioning** | Abstract zone-based system (not grid coordinates) |

---

## CLI Interface

### Output Format

All output is structured markdown that an LLM can easily parse:

```markdown
# ACTION: Strike
**Actor:** Sir Roland (paladin)
**Target:** Goblin Warrior (goblin_1)
**Weapon:** Longsword

## Attack Roll
- Natural: 14
- Modifiers: +8 (STR + proficiency), +1 (weapon potency)
- Total: 23 vs AC 16
- **Result:** Success

## Damage
- Dice: 1d8+4
- Rolled: 7
- Total: 11 slashing

## Result
- Target HP: 15 → 4
- Conditions: none

**Actions Remaining:** 2
```

### Core Commands

```bash
# === Scene Management ===
vd scene new "Blacksmith's Shop"
vd scene load tavern_brawl
vd scene save

# === Entity Management ===
vd entity add paladin --file characters/sir_roland.md
vd entity add shopkeeper --file npcs/grumpy_merchant.md
vd entity spawn "Goblin Warrior" --count 3 --prefix goblin_

vd entity get paladin                    # Full stats
vd entity get paladin --field hp         # Just HP
vd entity set goblin_1 hp 0              # Direct manipulation

# === Roleplay / Social ===
vd roll paladin intimidation --dc 25
vd roll shopkeeper will --vs paladin.intimidation

# === Combat ===
vd combat start
vd combat initiative                     # Rolls for all, returns order
vd combat next                           # Advance to next turn

vd action strike paladin goblin_1
vd action stride paladin --to altar
vd action raise_shield paladin
vd action cast paladin fireball --target main_hall --heighten 5

# === Reaction Handling ===
vd pending                               # List pending events
vd reactions                             # List available reactions for current event
vd react goblin_2 attack_of_opportunity  # Use a reaction
vd react skip                            # No reactions, continue
vd react skip_all                        # Finish resolving event

# === Conditions ===
vd condition add paladin frightened 2 --source "Dragon's Presence"
vd condition remove paladin frightened
vd condition reduce paladin frightened 1

# === Damage & Healing ===
vd damage goblin_1 15 slashing --from paladin
vd heal paladin 20
vd temp_hp paladin 10

# === Queries ===
vd status                                # Scene overview
vd status paladin                        # Detailed entity status
vd query targets paladin melee           # Who can paladin melee?
vd query flanking paladin goblin_1       # Flanking check
vd check paladin athletics --dc 20       # Generic skill check
```

---

## Session & Persistence Model

Each game session is a directory. The `vd` CLI operates on the current working directory.

```
my_campaign/
├── scene.md              # Current scene description & positions
├── state.json            # Runtime state (HP, conditions, initiative)
├── history.log           # Command log for context
│
├── characters/           # PC markdown files
│   ├── sir_roland.md
│   └── zara_the_cunning.md
│
├── npcs/                 # NPC markdown files
│   ├── shopkeeper.md
│   └── tavern_keeper.md
│
├── bestiary/             # Monster templates
│   └── goblin_warrior.md
│
└── items/                # Custom items
    └── magic_sword.md
```

Running `vd` from a different directory = different session. A higher-level "DM" project can manage multiple sessions by changing CWD.

---

## Abstract Positioning System

Based on how real DMs handle space (zones + engagement tracking), not grid coordinates.

### How Real DMs Organise Spatial Info

| Style | How It Works | When Used |
|-------|--------------|-----------|
| **Theater of Mind** | Purely descriptive, DM adjudicates distances on the fly | Narrative-heavy games |
| **Zones** | Map divided into named areas, move between zones costs actions | Modern narrative games (Fate, Blades in the Dark) |
| **Rough Sketch** | Quick drawing, not to scale | Most home games |
| **Grid** | 5ft squares, miniatures, exact measurement | Tactical PF2E RAW |

The zone approach is what experienced DMs do mentally, just formalised. That's what we're using.

### Core Concepts

**Positions** are named locations with relationships:
```markdown
## Positions

### entrance
- Size: small (fits ~4 entities)
- Adjacent to: main_hall
- Entities: goblin_archer

### main_hall
- Size: large (fits ~12 entities)
- Adjacent to: entrance, altar, side_passage
- Near: balcony
- Entities: paladin, fighter, goblin_1, goblin_2

### altar
- Size: medium (fits ~6 entities)
- Adjacent to: main_hall
- Notes: Elevated, provides cover
- Entities: cult_leader

### balcony
- Size: small
- Near: main_hall (overlooking)
- Far: entrance
- Cover: standard (vs main_hall)
- Elevated: yes
- Entities: goblin_shaman
```

**Engagements** track who's in melee:
```markdown
## Engagements
- paladin ↔ goblin_1, goblin_2
- fighter ↔ goblin_3
```

### Distance Categories

| Distance | Meaning | PF2E Equivalent |
|----------|---------|-----------------|
| **Engaged** | In melee, can Strike | 0-5ft (or reach) |
| **Same Position** | Can engage with 1 action | 5-25ft |
| **Adjacent** | One move action between positions | 25-50ft |
| **Near** | Two move actions | 50-100ft |
| **Far** | Three+ moves, or ranged only | 100-200ft |
| **Distant** | Long-range/extreme range | 200ft+ |
<!-- src: rules/rules/core-rulebook/chapter-9-playing-the-game.md:665 -->

### How It Works

**Movement:** Moving between positions, not squares.
```bash
$ vd action stride paladin --to altar
```
If engaged, this may trigger reactions (Attack of Opportunity).

**Flanking:** Based on engagements, not grid position.
```bash
$ vd query flanking paladin goblin_1
```
```markdown
# QUERY: Flanking
**Attacker:** paladin
**Target:** goblin_1

**Result:** Yes, flanked
**Ally providing flank:** fighter
**Bonus:** goblin_1 is flat-footed to paladin (-2 AC)
```

**Area Effects:** Target positions.
```bash
$ vd action cast wizard fireball --target main_hall
```
Hits everyone at that position. Adjacent positions might be partially affected (LLM/GM discretion).

**Ranged Attacks:** Distance between positions determines range penalties.
```bash
$ vd query targets wizard --range 120
```
```markdown
# QUERY: Ranged Targets (120ft)
**Entity:** wizard

## In Range
| Entity | Position | Distance | Notes |
|--------|----------|----------|-------|
| goblin_1 | main_hall | Same | - |
| cult_leader | altar | Adjacent | - |
| goblin_shaman | balcony | Near | Has cover |
```

### Advantages for LLM DM

- Natural language friendly ("the wizard is at the balcony, overlooking the main hall")
- Easy reasoning about engagements and threats
- No coordinate math
- Area effects just target positions
- Flank detection from engagement graph

---

## Reaction System

Reactions pause action resolution until the LLM decides whether to use them.

### Flow Example

```bash
# LLM initiates movement
$ vd action stride goblin_1 --to entrance
```

```markdown
# ACTION: Stride (PENDING)
**Actor:** Goblin Warrior (goblin_1)
**From:** main_hall → **To:** entrance

## Reactions Available
Goblin is leaving the following entities' reach:

| Entity | Reaction | Description |
|--------|----------|-------------|
| paladin | Attack of Opportunity | Make a melee Strike |

**Status:** PENDING_REACTION
**Event ID:** evt_001
```

```bash
# LLM decides to use the reaction
$ vd react paladin attack_of_opportunity
```

```markdown
# REACTION: Attack of Opportunity
**Reactor:** Sir Roland (paladin)
**Trigger:** Goblin Warrior leaving reach

## Attack Roll
- Natural: 17
- Total: 25 vs AC 16
- **Result:** Critical Success

## Damage
- Dice: 1d8+4 (doubled)
- Total: 18 slashing

## Effect
- Goblin HP: 6 → 0
- Goblin is now **dying 1**
- Movement **interrupted** (critical hit)

**Event Status:** INTERRUPTED
```

```bash
# Check what's pending now
$ vd pending
```

```markdown
# PENDING EVENTS
None. Combat continues.

**Current Turn:** paladin
**Actions Remaining:** 3
```

---

## Phase 1: Core Dice & Check System

**Goal:** Implement the fundamental d20 check resolution engine.

### Structs
```go
// DieRoll represents a dice expression like "2d6+4"
// src: rules/rules/core-rulebook/chapter-9-playing-the-game.md:382
type DieRoll struct {
    Count    int
    Sides    int
    Modifier int
}

// BonusType determines stacking behaviour
// src: rules/rules/core-rulebook/chapter-9-playing-the-game.md:112
type BonusType int
const (
    BonusUntyped BonusType = iota
    BonusCircumstance
    BonusItem
    BonusStatus
)

// Modifier represents a single bonus or penalty
// src: rules/rules/core-rulebook/chapter-9-playing-the-game.md:91
type Modifier struct {
    Value  int
    Type   BonusType
    Source string
}

// CheckResult represents the outcome of a d20 check
// src: rules/rules/core-rulebook/chapter-9-playing-the-game.md:39
type CheckResult struct {
    NaturalRoll int
    Total       int
    DC          int
    Degree      DegreeOfSuccess
}

// src: rules/rules/core-rulebook/chapter-9-playing-the-game.md:145
type DegreeOfSuccess int
const (
    CriticalFailure DegreeOfSuccess = iota
    Failure
    Success
    CriticalSuccess
)
```

### Functions
- `RollDice(expr DieRoll) int`
- `CalculateTotalModifier(modifiers []Modifier) int` - Apply stacking rules
- `PerformCheck(baseModifier int, modifiers []Modifier, dc int) CheckResult`
- `DetermineDegree(roll, total, dc int) DegreeOfSuccess` - Handles nat 1/20

---

## Phase 2: Ability Scores & Proficiency

### Structs
```go
// src: rules/rules/core-rulebook/chapter-9-playing-the-game.md:98
type AbilityScores struct {
    Strength     int
    Dexterity    int
    Constitution int
    Intelligence int
    Wisdom       int
    Charisma     int
}

// src: rules/rules/core-rulebook/chapter-9-playing-the-game.md:104
type ProficiencyRank int
const (
    Untrained ProficiencyRank = iota  // +0
    Trained                            // level + 2
    Expert                             // level + 4
    Master                             // level + 6
    Legendary                          // level + 8
)
```

---

## Phase 3: Traits System

```go
// src: rules/rules/traits/
type TraitID string

// src: rules/rules/traits/
type Trait struct {
    ID          TraitID
    Name        string
    Description string
    Category    TraitCategory
}
```

Key traits: `attack`, `agile`, `finesse`, `versatile`, `deadly`, `fatal`, `reach`, `thrown`, plus all damage types.

---

## Phase 4: Conditions System

```go
// src: rules/rules/conditions.md
type ConditionInstance struct {
    ID        ConditionID
    Value     int           // For valued conditions (Frightened 2)
    Remaining int           // Duration (-1 = until removed)
    Source    string
}

// src: rules/rules/conditions.md
type ConditionTracker struct {
    conditions map[ConditionID]*ConditionInstance
}
```

Priority conditions: `flat-footed`, `frightened`, `sickened`, `clumsy`, `enfeebled`, `stupefied`, `drained`, `slowed`, `quickened`, `stunned`, `dying`, `wounded`, `doomed`, `prone`, `grabbed`, `restrained`, `hidden`, `invisible`, `blinded`, `persistent damage`.

---

## Phase 5: Items - Weapons & Armour

```go
// src: rules/rules/core-rulebook/chapter-6-equipment.md:278
type Weapon struct {
    ID             string
    Name           string
    Category       WeaponCategory
    Group          WeaponGroup
    Damage         DieRoll
    DamageType     DamageType
    Hands          int
    Traits         []TraitID
    RangeIncrement int
}

// src: rules/rules/core-rulebook/chapter-6-equipment.md:274
type Armor struct {
    ID           string
    Name         string
    Category     ArmorCategory
    ACBonus      int
    DexCap       int
    CheckPenalty int
    SpeedPenalty int
    Traits       []TraitID
}
```

---

## Phase 6: Entities

```go
// src: rules/fantasy-bestiary/
type Entity struct {
    ID         string
    Name       string
    Level      int
    Size       Size
    
    // Flavour strings
    Ancestry   string
    Class      string
    Background string
    
    // Stats
    Abilities  AbilityScores
    HP, MaxHP  int
    AC         int
    Speed      int
    
    // Saves
    // src: rules/rules/core-rulebook/chapter-9-playing-the-game.md:288
    Fortitude, Reflex, Will Proficiency
    
    // Equipment
    WieldedWeapons []*Weapon
    WornArmor      *Armor
    
    // State
    Conditions *ConditionTracker
    Position   string            // Zone name
    EngagedWith []string         // Entity IDs
    
    // Defences
    // src: rules/rules/core-rulebook/chapter-9-playing-the-game.md:499
    Immunities  []string
    Weaknesses  map[string]int
    Resistances map[string]int
}
```

---

## Phase 7: Actions & Combat

Core actions to implement (src: rules/rules/actions/):
- **Strike** (handles MAP, src: rules/rules/core-rulebook/chapter-9-playing-the-game.md:184)
- **Stride** (move between positions, src: rules/rules/core-rulebook/chapter-9-playing-the-game.md:233)
- **Step** (disengage without triggering, src: rules/rules/core-rulebook/chapter-9-playing-the-game.md:235)
- **Raise Shield** (src: rules/rules/actions/raise-a-shield.md)
- **Seek**, **Hide** (src: rules/rules/actions/seek.md, rules/rules/actions/hide.md)
- **Grapple**, **Shove**, **Trip**, **Disarm** (src: rules/rules/actions/)
- **Demoralize**, **Feint** (src: rules/rules/actions/)

### Action Economy
// src: rules/rules/core-rulebook/chapter-1-introduction.md:582
const (
    ActionFree = iota
    ActionReaction
    ActionOne
    ActionTwo
    ActionThree
)

---

## Phase 8: Damage Pipeline
// src: rules/rules/core-rulebook/chapter-9-playing-the-game.md:384

```
1. Roll damage dice + modifiers
2. Double for critical (src: line 430)
3. Check immunities (src: line 504)
4. Apply weaknesses (+X) (src: line 520)
5. Apply resistances (-X, min 0) (src: line 527)
6. Reduce HP
7. Check dying/death
```

---

## Phase 9: Spells
// src: rules/rules/core-rulebook/chapter-9-playing-the-game.md:245

Hardcoded spell effects with helper functions for common patterns (damage + save, buff duration, area targeting).

---

## Phase 10: Skills
// src: rules/compendium/skills.md

17 core skills, each linked to a key ability. Skill actions (Demoralize, Grapple, etc.) reference the skill system.

---

## Phase 11: Afflictions
// src: rules/rules/core-rulebook/chapter-9-playing-the-game.md:730

Staged progression system for poisons, diseases, curses.

---

## Phase 12: Encounter Management
// src: rules/rules/core-rulebook/chapter-9-playing-the-game.md:15 (Modes of Play), 283 (Initiative)

Initiative, turn order, round structure, action tracking.

---

## Phase 13: Feats

Framework for feats that grant actions, reactions, or passive modifiers.

---

## Phase 14: Hazards

Traps and environmental dangers with Stealth DCs, disable options, and triggered effects.

---

## Phase 15: Data Loading

Parse entity/item markdown files from the session directory. Rulebook data (conditions, traits, base weapons) hardcoded in Go.

---

## File Structure

```
vdnd/
├── cmd/
│   └── vd/
│       └── main.go              # CLI entry point
├── pkg/
│   └── rules/                   # Core PF2E rules engine
│       ├── ability/             # Ability scores and proficiency
│       ├── dice/                # Dice rolling
│       ├── check/               # Check resolution
│       ├── trait/               # Trait registry
│       ├── condition/           # Condition tracker
│       ├── entity/              # Entity management
│       ├── item/                # Weapons, armour
│       ├── combat/              # Actions, encounters
│       ├── position/            # Zone-based positioning
│       ├── spell/               # Spell effects
│       └── skill/               # Skill system
│   ├── session/                 # State persistence (not rules logic)
│   └── output/                  # Markdown output formatting
├── internal/
│   ├── cli/                     # CLI command handlers
│   └── parser/                  # Markdown parsing
├── testdata/                    # Test scenario files
├── rules/                       # Source compendium (existing)
├── go.mod
└── RULES_PLAN.md
```

---

## Testing Strategy

1. **Unit tests** - Test internal functions with hardcoded inputs, expect specific outputs
2. **CLI handler tests** - Call CLI handler functions with parameters, verify markdown output
3. **Scenario tests** - Load test scenario data files, run sequences of commands, verify final state

Example test structure:
```
testdata/
├── combat_basic/
│   ├── setup.md              # Initial scene state
│   ├── commands.txt          # Commands to run
│   └── expected_state.json   # Expected final state
├── reactions/
│   ├── attack_of_opportunity/
│   └── shield_block/
└── conditions/
    ├── frightened_reduction/
    └── dying_recovery/
```

---

## Implementation Milestones

| Milestone | Phases | Deliverable |
|-----------|--------|-------------|
| **M1** | 1, 2 | Dice rolling, checks, degrees of success |
| **M2** | 3, 4 | Traits, conditions with modifier extraction |
| **M3** | 5, 6 | Weapons, armour, entities, AC calculation |
| **M4** | 7, 8 | Strike action, damage pipeline, HP tracking |
| **M5** | 12 | Initiative, turn order, encounter flow |
| **M6** | 9, 10 | Skills, spell casting |
| **M7** | 11, 13, 14 | Afflictions, feats, hazards |
| **M8** | 15 | Data loading from markdown files |

---

## Next Steps

1. Set up folder structure
2. Implement M1 (dice & checks) with tests
3. Build out CLI skeleton
4. Iterate through milestones
