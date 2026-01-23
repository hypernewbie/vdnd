# Phase 13: Feats

## Agent Prompt

You are implementing the feat system for a Pathfinder 2E rules engine in Go. Feats are special abilities that grant new actions, reactions, passive bonuses, or modify existing abilities.

**Your task:** Implement the `pkg/rules/feat` package with full test coverage.

**Prerequisites:** Phases 1-8, 12 (check, combat, entity, encounter).

---

## Context

### Source References
- General feats: `rules/rules/core-rulebook/chapter-5-feats.md`
- Class feats: `rules/rules/core-rulebook/chapter-3-classes.md`
- Ancestry feats: `rules/rules/core-rulebook/chapter-2-ancestries.md`

### Feat Categories
| Type | Source | Examples |
|------|--------|----------|
| Ancestry | Heritage | Darkvision, Nimble Elf |
| Class | Class levels | Power Attack, Sneak Attack |
| Skill | Skill training | Battle Medicine, Titan Wrestler |
| General | General training | Toughness, Fleet |
| Archetype | Multiclass | Basic Spellcasting |

### Feat Effects
Feats can:
1. **Grant actions:** New 1/2/3-action abilities (Power Attack)
2. **Grant reactions:** Reactive abilities with triggers (Attack of Opportunity)
3. **Grant passives:** Always-on modifiers (Toughness: +HP per level)
4. **Modify actions:** Change how existing actions work (Quick Draw)
5. **Grant proficiencies:** Weapon/armor/skill training
6. **Have prerequisites:** Level, stat, proficiency requirements

---

## File Structure

```
pkg/
└── rules/
    └── feat/
        ├── feat.go         # Feat struct, FeatType
        ├── effect.go       # Effect types (action, reaction, passive)
        ├── registry.go     # Pre-built feats
        ├── tracker.go      # Feats on an entity
        └── feat_test.go
```

---

## Implementation Plan

### 1. `pkg/rules/feat/feat.go`

```go
type FeatType int
const (
    FeatTypeAncestry FeatType = iota
    FeatTypeClass
    FeatTypeSkill
    FeatTypeGeneral
    FeatTypeArchetype
)

type Prerequisite struct {
    MinLevel        int
    MinAbilityScore map[ability.Ability]int
    RequiredFeat    string
    RequiredSkill   SkillRequirement
    RequiredTrait   trait.TraitID
}

type SkillRequirement struct {
    Skill SkillID
    Rank  ability.ProficiencyRank
}

type Feat struct {
    ID            string
    Name          string
    Type          FeatType
    Level         int  // Minimum level to take
    Prerequisites []Prerequisite
    Traits        []trait.TraitID
    Description   string
    
    // Effects
    GrantsAction   *ActionGrant
    GrantsReaction *ReactionGrant
    Passives       []PassiveEffect
}

// MeetsPrerequisites checks if an entity qualifies
func (f *Feat) MeetsPrerequisites(e *entity.Entity) bool
```

### 2. `pkg/rules/feat/effect.go`

```go
// ActionGrant represents an action granted by a feat
type ActionGrant struct {
    Name        string
    Cost        combat.ActionCost
    Traits      []trait.TraitID
    Execute     ActionFunc
}

type ActionFunc func(actor *entity.Entity, target *entity.Entity, turn *combat.TurnState) combat.ActionResult

// ReactionGrant represents a reaction granted by a feat
type ReactionGrant struct {
    Name        string
    Trigger     EventType
    Condition   ReactionCondition
    Execute     ReactionFunc
}

type ReactionCondition func(event Event, reactor *entity.Entity) bool
type ReactionFunc func(event Event, reactor *entity.Entity, encounter *encounter.Encounter) combat.ActionResult

// PassiveEffect represents an always-on bonus
type PassiveEffect struct {
    Type        PassiveType
    Value       int
    Condition   string  // Optional condition description
    Apply       PassiveFunc
}

type PassiveType int
const (
    PassiveHP PassiveType = iota
    PassiveAC
    PassiveSpeed
    PassiveSave
    PassiveSkill
    PassiveProficiency
)

type PassiveFunc func(e *entity.Entity)
```

### 3. `pkg/rules/feat/registry.go`

```go
var (
    // Toughness - General feat, +HP
    Toughness = Feat{
        ID:    "toughness",
        Name:  "Toughness",
        Type:  FeatTypeGeneral,
        Level: 1,
        Passives: []PassiveEffect{{
            Type:  PassiveHP,
            Value: 0,  // +Level HP, calculated dynamically
            Apply: func(e *entity.Entity) {
                e.MaxHP += e.Level
            },
        }},
    }
    
    // Fleet - General feat, +5 Speed
    Fleet = Feat{
        ID:    "fleet",
        Name:  "Fleet",
        Type:  FeatTypeGeneral,
        Level: 1,
        Passives: []PassiveEffect{{
            Type:  PassiveSpeed,
            Value: 5,
            Apply: func(e *entity.Entity) {
                e.Speed += 5
            },
        }},
    }
    
    // Attack of Opportunity - Fighter feat, reaction
    AttackOfOpportunity = Feat{
        ID:    "attack-of-opportunity",
        Name:  "Attack of Opportunity",
        Type:  FeatTypeClass,
        Level: 1,
        GrantsReaction: &ReactionGrant{
            Name:    "Attack of Opportunity",
            Trigger: EventManipulate,  // Also EventMove for some
            Condition: func(event Event, reactor *entity.Entity) bool {
                return reactor.IsEngagedWith(event.Actor.ID)
            },
            Execute: func(event Event, reactor *entity.Entity, enc *encounter.Encounter) combat.ActionResult {
                // Make Strike against triggering creature
                weapon := reactor.GetPrimaryWeapon()
                strike := combat.NewStrike(weapon)
                return strike.Execute(reactor, event.Actor, nil)  // No turn state for reactions
            },
        },
    }
    
    // Power Attack - Fighter feat, 2-action attack
    PowerAttack = Feat{
        ID:    "power-attack",
        Name:  "Power Attack",
        Type:  FeatTypeClass,
        Level: 1,
        Prerequisites: []Prerequisite{{
            RequiredTrait: trait.TraitFighter,
        }},
        GrantsAction: &ActionGrant{
            Name: "Power Attack",
            Cost: combat.CostTwo,
            Traits: []trait.TraitID{trait.TraitAttack, trait.TraitFighter},
            Execute: func(actor, target *entity.Entity, turn *combat.TurnState) combat.ActionResult {
                // Strike with extra damage die
                // ... implementation
            },
        },
    }
    
    // Battle Medicine - Skill feat, use Medicine in combat
    BattleMedicine = Feat{
        ID:    "battle-medicine",
        Name:  "Battle Medicine",
        Type:  FeatTypeSkill,
        Level: 1,
        Prerequisites: []Prerequisite{{
            RequiredSkill: SkillRequirement{SkillMedicine, ability.Trained},
        }},
        GrantsAction: &ActionGrant{
            Name: "Battle Medicine",
            Cost: combat.CostOne,
            Traits: []trait.TraitID{trait.TraitHealing, trait.TraitManipulate},
            Execute: func(actor, target *entity.Entity, turn *combat.TurnState) combat.ActionResult {
                // Treat Wounds as 1 action in combat
                // ... implementation
            },
        },
    }
)

func GetFeat(id string) (*Feat, bool)
func AllFeats() []Feat
```

### 4. `pkg/rules/feat/tracker.go`

```go
// Entity integration
type FeatTracker struct {
    feats map[string]*Feat
}

func NewFeatTracker() *FeatTracker

func (t *FeatTracker) Add(feat *Feat)
func (t *FeatTracker) Has(featID string) bool
func (t *FeatTracker) Get(featID string) *Feat
func (t *FeatTracker) All() []*Feat

// GetGrantedActions returns all actions from feats
func (t *FeatTracker) GetGrantedActions() []*ActionGrant

// GetGrantedReactions returns all reactions from feats
func (t *FeatTracker) GetGrantedReactions() []*ReactionGrant

// ApplyPassives applies all passive effects to entity
func (t *FeatTracker) ApplyPassives(e *entity.Entity)
```

---

## Test Plan

### Prerequisite Tests
| Feat | Entity | Expected |
|------|--------|----------|
| Power Attack | Fighter L1 | Meets prereqs |
| Power Attack | Wizard L1 | Fails (not fighter) |
| Battle Medicine | Trained Medicine | Meets prereqs |
| Battle Medicine | Untrained | Fails |
| L4 feat | L3 character | Fails (too low) |

### Passive Effect Tests
| Feat | Entity | Expected |
|------|--------|----------|
| Toughness | Level 5 | MaxHP +5 |
| Fleet | Speed 25 | Speed 30 |

### Action Grant Tests
| Feat | Action | Cost | Expected |
|------|--------|------|----------|
| Power Attack | Power Attack | 2 | Extra damage die |
| Battle Medicine | Battle Med | 1 | Heal in combat |

### Reaction Tests
| Feat | Trigger | Condition | Expected |
|------|---------|-----------|----------|
| AoO | Manipulate | Engaged | Strike triggered |
| AoO | Move | Not engaged | No trigger |

---

## Validation Checklist
- [ ] Prerequisites checked correctly
- [ ] Passives apply to entity stats
- [ ] Granted actions executable
- [ ] Reactions trigger on correct events
- [ ] Feat tracker stores/retrieves correctly
