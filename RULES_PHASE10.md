# Phase 10: Skills

## Agent Prompt

You are implementing the skill system for a Pathfinder 2E rules engine in Go. Skills represent trained abilities for non-combat tasks—climbing, persuading, recalling knowledge, treating wounds.

**Your task:** Implement the `pkg/rules/skill` package with full test coverage.

**Prerequisites:** Phases 1-4 (check, ability, condition, trait).

---

## Context

### Source References
- Skills: `rules/compendium/skills.md`
- Skill actions: `rules/rules/core-rulebook/chapter-4-skills.md`

### The 17 Core Skills

| Skill | Key Ability | Notable Actions |
|-------|-------------|-----------------|
| Acrobatics | DEX | Balance, Tumble Through |
| Arcana | INT | Recall Knowledge, Identify Magic |
| Athletics | STR | Climb, Grapple, Shove, Trip |
| Crafting | INT | Craft, Repair |
| Deception | CHA | Lie, Feint |
| Diplomacy | CHA | Make Impression, Request |
| Intimidation | CHA | Coerce, Demoralize |
| Lore (varies) | INT | Recall Knowledge |
| Medicine | WIS | Treat Wounds, First Aid |
| Nature | WIS | Command Animal |
| Occultism | INT | Recall Knowledge |
| Performance | CHA | Perform |
| Religion | WIS | Recall Knowledge |
| Society | INT | Recall Knowledge |
| Stealth | DEX | Hide, Sneak |
| Survival | WIS | Track, Subsist |
| Thievery | DEX | Pick Lock, Disable Device |

### Skill Check Formula
```
d20 + ability modifier + proficiency bonus + item bonus + condition modifiers
```

### Standard DCs by Difficulty
| Difficulty | DC |
|------------|-----|
| Untrained | 10 |
| Trained | 15 |
| Expert | 20 |
| Master | 30 |
| Legendary | 40 |

### Level-Based DCs (CRB p.503)
| Level | DC | Level | DC |
|-------|-----|-------|-----|
| 0 | 14 | 10 | 27 |
| 1 | 15 | 11 | 28 |
| 2 | 16 | 12 | 30 |
| 3 | 18 | 13 | 31 |
| 4 | 19 | 14 | 32 |
| 5 | 20 | 15 | 34 |
| 6 | 22 | 16 | 35 |
| 7 | 23 | 17 | 36 |
| 8 | 24 | 18 | 38 |
| 9 | 26 | 19 | 39 |
|   |    | 20 | 40 |

---

## File Structure

```
pkg/
└── rules/
    └── skill/
        ├── skill.go        # SkillID, key ability mapping
        ├── dc.go           # DC calculations
        ├── actions.go      # Skill action implementations
        └── skill_test.go
```

---

## Implementation Plan

### 1. `pkg/rules/skill/skill.go`

```go
type SkillID string

const (
    SkillAcrobatics   SkillID = "acrobatics"
    SkillArcana       SkillID = "arcana"
    SkillAthletics    SkillID = "athletics"
    SkillCrafting     SkillID = "crafting"
    SkillDeception    SkillID = "deception"
    SkillDiplomacy    SkillID = "diplomacy"
    SkillIntimidation SkillID = "intimidation"
    SkillMedicine     SkillID = "medicine"
    SkillNature       SkillID = "nature"
    SkillOccultism    SkillID = "occultism"
    SkillPerformance  SkillID = "performance"
    SkillReligion     SkillID = "religion"
    SkillSociety      SkillID = "society"
    SkillStealth      SkillID = "stealth"
    SkillSurvival     SkillID = "survival"
    SkillThievery     SkillID = "thievery"
)

type Skill struct {
    ID          SkillID
    Name        string
    KeyAbility  ability.Ability
}

var skillAbilities = map[SkillID]ability.Ability{
    SkillAcrobatics:   ability.Dexterity,
    SkillArcana:       ability.Intelligence,
    SkillAthletics:    ability.Strength,
    SkillCrafting:     ability.Intelligence,
    SkillDeception:    ability.Charisma,
    SkillDiplomacy:    ability.Charisma,
    SkillIntimidation: ability.Charisma,
    SkillMedicine:     ability.Wisdom,
    SkillNature:       ability.Wisdom,
    SkillOccultism:    ability.Intelligence,
    SkillPerformance:  ability.Charisma,
    SkillReligion:     ability.Wisdom,
    SkillSociety:      ability.Intelligence,
    SkillStealth:      ability.Dexterity,
    SkillSurvival:     ability.Wisdom,
    SkillThievery:     ability.Dexterity,
}

func GetKeyAbility(skill SkillID) ability.Ability
func AllSkills() []Skill
```

### 2. Entity Integration

```go
// Add to entity.Entity
type Entity struct {
    // ...
    Skills map[SkillID]ability.ProficiencyRank
}

func (e *Entity) GetSkillModifier(skill SkillID) int {
    keyAbility := GetKeyAbility(skill)
    abilityMod := e.Abilities.Modifier(keyAbility)
    
    rank := e.Skills[skill]
    profBonus := rank.Bonus(e.Level)
    
    // Condition modifiers
    condMods := e.getSkillConditionModifiers(skill)
    
    return abilityMod + profBonus + check.CalculateTotal(condMods)
}

func (e *Entity) GetSkillDC(skill SkillID) int {
    return 10 + e.GetSkillModifier(skill)
}
```

### 3. `pkg/rules/skill/dc.go`

```go
type Difficulty int
const (
    DifficultyUntrained Difficulty = iota
    DifficultyTrained
    DifficultyExpert
    DifficultyMaster
    DifficultyLegendary
)

func DifficultyDC(diff Difficulty) int {
    return []int{10, 15, 20, 30, 40}[diff]
}

var levelDCs = []int{
    14, 15, 16, 18, 19, 20, 22, 23, 24, 26,  // 0-9
    27, 28, 30, 31, 32, 34, 35, 36, 38, 39,  // 10-19
    40, 42, 44, 46, 48, 50,                   // 20-25
}

func LevelBasedDC(level int) int {
    if level < 0 { level = 0 }
    if level >= len(levelDCs) { level = len(levelDCs) - 1 }
    return levelDCs[level]
}

type DCAdjustment int
const (
    AdjustIncrediblyEasy DCAdjustment = -10
    AdjustVeryEasy       DCAdjustment = -5
    AdjustEasy           DCAdjustment = -2
    AdjustHard           DCAdjustment = +2
    AdjustVeryHard       DCAdjustment = +5
    AdjustIncrediblyHard DCAdjustment = +10
)

func AdjustedDC(baseDC int, adj DCAdjustment) int {
    return baseDC + int(adj)
}
```

### 4. `pkg/rules/skill/actions.go`

```go
// PerformSkillCheck makes a skill check
func PerformSkillCheck(actor *entity.Entity, skill SkillID, dc int) check.CheckResult {
    mod := actor.GetSkillModifier(skill)
    return check.PerformCheck(mod, nil, dc)
}

// Demoralize - Intimidation vs Will DC
func Demoralize(actor, target *entity.Entity) check.CheckResult {
    dc := target.GetSaveDC(SaveWill)
    result := PerformSkillCheck(actor, SkillIntimidation, dc)
    
    switch result.Degree {
    case check.CriticalSuccess:
        target.Conditions.Apply(condition.NewValuedCondition(
            condition.Frightened, 2, "Demoralize"))
    case check.Success:
        target.Conditions.Apply(condition.NewValuedCondition(
            condition.Frightened, 1, "Demoralize"))
    }
    return result
}

// TreatWounds - Medicine check, requires trained
func TreatWounds(healer, patient *entity.Entity, dc int) (healing int, result check.CheckResult) {
    if healer.Skills[SkillMedicine] < ability.Trained {
        return 0, check.CheckResult{Degree: check.Failure}
    }
    
    result = PerformSkillCheck(healer, SkillMedicine, dc)
    
    switch result.Degree {
    case check.CriticalSuccess:
        healing = dice.DieRoll{4, 8, 0}.Roll()  // 4d8
    case check.Success:
        healing = dice.DieRoll{2, 8, 0}.Roll()  // 2d8
    case check.CriticalFailure:
        patient.TakeDamage(dice.DieRoll{1, 8, 0}.Roll(), "slashing")
    }
    
    if healing > 0 {
        patient.Heal(healing)
    }
    return
}

// RecallKnowledge - various skills vs topic DC
func RecallKnowledge(actor *entity.Entity, skill SkillID, dc int) (info string, result check.CheckResult) {
    result = PerformSkillCheck(actor, skill, dc)
    
    switch result.Degree {
    case check.CriticalSuccess:
        info = "Accurate info + extra fact"
    case check.Success:
        info = "Accurate info"
    case check.Failure:
        info = "No useful info"
    case check.CriticalFailure:
        info = "FALSE info"  // GM provides misleading info
    }
    return
}
```

---

## Test Plan

### Key Ability Tests
| Skill | Expected |
|-------|----------|
| Athletics | Strength |
| Acrobatics | Dexterity |
| Arcana | Intelligence |
| Medicine | Wisdom |
| Diplomacy | Charisma |

### Skill Modifier Tests
| Setup | Skill | Expected |
|-------|-------|----------|
| DEX 16, Untrained | Acrobatics | +3 |
| DEX 16, Trained L1 | Acrobatics | +6 |
| STR 18, Expert L5 | Athletics | +13 |
| INT 10, Legendary L20 | Arcana | +28 |

### DC Calculation Tests
| Input | Expected |
|-------|----------|
| DifficultyTrained | 15 |
| DifficultyMaster | 30 |
| LevelBasedDC(1) | 15 |
| LevelBasedDC(5) | 20 |
| LevelBasedDC(10) | 27 |
| LevelBasedDC(20) | 40 |
| AdjustedDC(20, Hard) | 22 |

### Skill Action Tests
| Action | Degree | Expected |
|--------|--------|----------|
| Demoralize | CritSuccess | Frightened 2 |
| Demoralize | Success | Frightened 1 |
| Demoralize | Failure | Nothing |
| TreatWounds | CritSuccess | 4d8 healing |
| TreatWounds | Success | 2d8 healing |
| TreatWounds | CritFail | 1d8 damage |
| RecallKnowledge | CritFail | False info |

---

## Validation Checklist

- [ ] All 17 skills mapped to correct key ability
- [ ] Skill modifiers calculate correctly  
- [ ] Level-based DCs match table
- [ ] Demoralize applies Frightened correctly
- [ ] Treat Wounds requires Trained
- [ ] Condition modifiers affect skill checks
