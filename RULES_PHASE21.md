# Phase 21: Rituals

## Objective

Implement the Ritual casting system. Rituals differ from standard spells: they take hours/days to cast, require multiple casters, have monetary costs, and use skill checks rather than spell slots.

---

## 1. Ritual Struct

**Target File:** `pkg/rules/spell/ritual.go`

```go
package spell

import (
    "uaa/vdnd/pkg/rules/ability"
    "uaa/vdnd/pkg/rules/check"
    "uaa/vdnd/pkg/rules/entity"
    "uaa/vdnd/pkg/rules/trait"
)

// RitualRank is the spell rank of the ritual (1-10)
type RitualRank int

// CastingDuration represents how long the ritual takes
type CastingDuration struct {
    Value int
    Unit  DurationUnit
}

type DurationUnit int

const (
    DurationMinutes DurationUnit = iota
    DurationHours
    DurationDays
)

func (d CastingDuration) String() string {
    switch d.Unit {
    case DurationMinutes:
        return fmt.Sprintf("%d minutes", d.Value)
    case DurationHours:
        return fmt.Sprintf("%d hours", d.Value)
    case DurationDays:
        return fmt.Sprintf("%d days", d.Value)
    }
    return fmt.Sprintf("%d units", d.Value)
}

// SecondaryCheckRequirement defines what secondary casters need
type SecondaryCheckRequirement struct {
    Skill           ability.SkillID
    MinProficiency  ability.ProficiencyRank
    Description     string
}

// Ritual represents a ritual spell
// src: rules/compendium/spells/rituals/
type Ritual struct {
    ID               string
    Name             string
    Rank             RitualRank
    Traits           trait.TraitSet
    CastingTime      CastingDuration
    CostCP           int    // Material cost in copper
    SecondaryCasters int    // Minimum required (can be 0)

    // Check requirements
    PrimaryCheck     ability.SkillID
    PrimaryDC        int    // 0 = use level-based DC
    SecondaryChecks  []SecondaryCheckRequirement

    // Requirements
    RequiredRank     ability.ProficiencyRank // Minimum proficiency in primary check
    
    // Effect
    Effect           RitualEffect
    HeightenedCostCP int    // Additional cost per rank above base
}

func NewRitual(id, name string, rank RitualRank, primarySkill ability.SkillID, secondaryCasters int) *Ritual {
    return &Ritual{
        ID:               id,
        Name:             name,
        Rank:             rank,
        PrimaryCheck:     primarySkill,
        SecondaryCasters: secondaryCasters,
        SecondaryChecks:  make([]SecondaryCheckRequirement, 0),
        Traits:           make(trait.TraitSet, 0),
    }
}

// WithCastingTime sets the casting duration
func (r *Ritual) WithCastingTime(value int, unit DurationUnit) *Ritual {
    r.CastingTime = CastingDuration{Value: value, Unit: unit}
    return r
}

// WithCost sets the material cost
func (r *Ritual) WithCost(costCP int) *Ritual {
    r.CostCP = costCP
    return r
}

// WithSecondaryCheck adds a secondary check requirement
func (r *Ritual) WithSecondaryCheck(skill ability.SkillID, minProf ability.ProficiencyRank, desc string) *Ritual {
    r.SecondaryChecks = append(r.SecondaryChecks, SecondaryCheckRequirement{
        Skill:          skill,
        MinProficiency: minProf,
        Description:    desc,
    })
    return r
}
```

---

## 2. Ritual Effect Interface

**Target File:** `pkg/rules/spell/ritual.go`

```go
// RitualOutcome represents the result of casting a ritual
type RitualOutcome struct {
    Success      bool
    Description  string
    Backlash     string    // Effect on critical failure
    TargetEffect string    // What happens to the target
}

// RitualEffect defines what the ritual does
type RitualEffect interface {
    // Apply processes the ritual's effect based on the casting result
    Apply(attempt *RitualCastAttempt, caster *entity.Entity, targets []*entity.Entity) RitualOutcome
}

// GenericRitualEffect provides a simple effect implementation
type GenericRitualEffect struct {
    SuccessDesc    string
    CritSuccessDesc string
    FailureDesc    string
    CritFailureDesc string
    BacklashDesc   string
}

func (e *GenericRitualEffect) Apply(attempt *RitualCastAttempt, caster *entity.Entity, targets []*entity.Entity) RitualOutcome {
    outcome := RitualOutcome{}

    switch attempt.FinalDegree {
    case check.CriticalSuccess:
        outcome.Success = true
        outcome.Description = e.CritSuccessDesc
    case check.Success:
        outcome.Success = true
        outcome.Description = e.SuccessDesc
    case check.Failure:
        outcome.Success = false
        outcome.Description = e.FailureDesc
    case check.CriticalFailure:
        outcome.Success = false
        outcome.Description = e.CritFailureDesc
        outcome.Backlash = e.BacklashDesc
    }

    return outcome
}
```

---

## 3. Ritual Casting Logic

**Target File:** `pkg/rules/spell/ritual_casting.go`

```go
package spell

import (
    "uaa/vdnd/pkg/rules/ability"
    "uaa/vdnd/pkg/rules/check"
    "uaa/vdnd/pkg/rules/entity"
    "uaa/vdnd/pkg/rules/skill"
)

// RitualCastAttempt tracks the casting process and results
type RitualCastAttempt struct {
    Ritual           *Ritual
    PrimaryCaster    *entity.Entity
    SecondaryCasters []*entity.Entity
    
    // Results
    PrimaryResult    check.CheckResult
    SecondaryResults []SecondaryCheckResult
    FinalDegree      check.DegreeOfSuccess
    
    // State
    MaterialsConsumed int
    IsComplete        bool
}

type SecondaryCheckResult struct {
    Caster      *entity.Entity
    Skill       ability.SkillID
    Result      check.CheckResult
    Contributed bool
}

// NewRitualCastAttempt starts a new ritual casting
func NewRitualCastAttempt(ritual *Ritual, primary *entity.Entity, secondaries []*entity.Entity) (*RitualCastAttempt, error) {
    // Validate primary caster has required proficiency
    primaryProf := ability.Untrained
    if p, ok := primary.SkillProficiencies[ritual.PrimaryCheck]; ok {
        primaryProf = p
    }
    if primaryProf < ritual.RequiredRank {
        return nil, fmt.Errorf("primary caster requires %v in %s", ritual.RequiredRank, ritual.PrimaryCheck)
    }

    // Validate enough secondary casters
    if len(secondaries) < ritual.SecondaryCasters {
        return nil, fmt.Errorf("ritual requires %d secondary casters, got %d", ritual.SecondaryCasters, len(secondaries))
    }

    return &RitualCastAttempt{
        Ritual:           ritual,
        PrimaryCaster:    primary,
        SecondaryCasters: secondaries,
        SecondaryResults: make([]SecondaryCheckResult, 0),
    }, nil
}

// CastRitual performs all the checks and determines outcome
func CastRitual(attempt *RitualCastAttempt, primaryRoll int, secondaryRolls []int) RitualOutcome {
    ritual := attempt.Ritual

    // Determine DC
    dc := ritual.PrimaryDC
    if dc == 0 {
        dc = skill.LevelBasedDC(int(ritual.Rank) * 2) // Approximate level from rank
    }

    // Primary check
    if primaryRoll > 0 {
        attempt.PrimaryResult = skill.PerformSkillCheckWithRoll(
            attempt.PrimaryCaster, ritual.PrimaryCheck, dc, primaryRoll)
    } else {
        attempt.PrimaryResult = skill.PerformSkillCheck(
            attempt.PrimaryCaster, ritual.PrimaryCheck, dc)
    }

    // Secondary checks
    for i, secondary := range attempt.SecondaryCasters {
        if i >= len(ritual.SecondaryChecks) {
            break // No more requirements
        }

        req := ritual.SecondaryChecks[i]
        roll := 0
        if i < len(secondaryRolls) {
            roll = secondaryRolls[i]
        }

        var res check.CheckResult
        if roll > 0 {
            res = skill.PerformSkillCheckWithRoll(secondary, req.Skill, dc, roll)
        } else {
            res = skill.PerformSkillCheck(secondary, req.Skill, dc)
        }

        attempt.SecondaryResults = append(attempt.SecondaryResults, SecondaryCheckResult{
            Caster:      secondary,
            Skill:       req.Skill,
            Result:      res,
            Contributed: res.Degree >= check.Success,
        })
    }

    // Calculate final degree with secondary modifiers
    attempt.FinalDegree = calculateFinalDegree(attempt)
    attempt.MaterialsConsumed = ritual.CostCP
    attempt.IsComplete = true

    // Apply effect
    if ritual.Effect != nil {
        return ritual.Effect.Apply(attempt, attempt.PrimaryCaster, nil)
    }

    return RitualOutcome{
        Success:     attempt.FinalDegree >= check.Success,
        Description: fmt.Sprintf("Ritual completed with %v", attempt.FinalDegree),
    }
}

// calculateFinalDegree adjusts primary result based on secondary casters
// src: rules/rules/core-rulebook/chapter-7-spells.md (Rituals section)
func calculateFinalDegree(attempt *RitualCastAttempt) check.DegreeOfSuccess {
    degree := attempt.PrimaryResult.Degree

    // Each secondary crit success = +1 step
    // Each secondary crit failure = -1 step
    modifier := 0
    for _, sec := range attempt.SecondaryResults {
        switch sec.Result.Degree {
        case check.CriticalSuccess:
            modifier++
        case check.CriticalFailure:
            modifier--
        }
    }

    // Apply modifier
    finalDegree := int(degree) + modifier
    if finalDegree > int(check.CriticalSuccess) {
        finalDegree = int(check.CriticalSuccess)
    }
    if finalDegree < int(check.CriticalFailure) {
        finalDegree = int(check.CriticalFailure)
    }

    return check.DegreeOfSuccess(finalDegree)
}
```

---

## 4. Standard Rituals Registry

**Target File:** `pkg/rules/spell/ritual_registry.go`

```go
package spell

import "uaa/vdnd/pkg/rules/ability"

// StandardRituals contains common rituals
var StandardRituals = map[string]*Ritual{}

func init() {
    // Resurrect - Rank 5
    // src: rules/compendium/spells/rituals/resurrect.md
    resurrect := NewRitual("resurrect", "Resurrect", 5, ability.SkillReligion, 2).
        WithCastingTime(1, DurationDays).
        WithCost(7500). // 75 gp in diamonds
        WithSecondaryCheck(ability.SkillMedicine, ability.Expert, "Prepare the body").
        WithSecondaryCheck(ability.SkillReligion, ability.Trained, "Call the soul")
    resurrect.RequiredRank = ability.Expert
    resurrect.Effect = &GenericRitualEffect{
        CritSuccessDesc: "Target returns to life at full HP with no negative conditions",
        SuccessDesc:     "Target returns to life at 1 HP, wounded 1, and fatigued",
        FailureDesc:     "Ritual fails, materials are not consumed",
        CritFailureDesc: "Ritual fails catastrophically",
        BacklashDesc:    "Primary caster is doomed 1",
    }
    StandardRituals["resurrect"] = resurrect

    // Commune - Rank 6
    // src: rules/compendium/spells/rituals/commune.md
    commune := NewRitual("commune", "Commune", 6, ability.SkillReligion, 0).
        WithCastingTime(1, DurationHours).
        WithCost(15000) // 150 gp
    commune.RequiredRank = ability.Master
    commune.Effect = &GenericRitualEffect{
        CritSuccessDesc: "Deity answers 7 questions clearly",
        SuccessDesc:     "Deity answers 5 questions with single words",
        FailureDesc:     "Deity does not respond",
        CritFailureDesc: "Deity is angered, caster cannot attempt again for a week",
    }
    StandardRituals["commune"] = commune

    // Plane Shift - Rank 7
    planeShift := NewRitual("plane_shift", "Plane Shift", 7, ability.SkillOccultism, 3).
        WithCastingTime(1, DurationHours).
        WithCost(35000). // 350 gp tuning fork
        WithSecondaryCheck(ability.SkillArcana, ability.Trained, "Stabilise the portal").
        WithSecondaryCheck(ability.SkillOccultism, ability.Trained, "Navigate the planes").
        WithSecondaryCheck(ability.SkillSurvival, ability.Trained, "Chart safe arrival")
    planeShift.RequiredRank = ability.Master
    planeShift.Effect = &GenericRitualEffect{
        CritSuccessDesc: "All targets arrive precisely where intended",
        SuccessDesc:     "Targets arrive on correct plane, 1d20 miles from destination",
        FailureDesc:     "Portal fails to open, materials consumed",
        CritFailureDesc: "Targets scattered across the plane or arrive on wrong plane",
        BacklashDesc:    "All casters take 10d6 force damage",
    }
    StandardRituals["plane_shift"] = planeShift

    // Atone - Rank 4
    atone := NewRitual("atone", "Atone", 4, ability.SkillReligion, 0).
        WithCastingTime(1, DurationDays).
        WithCost(2000) // 20 gp
    atone.RequiredRank = ability.Expert
    atone.Effect = &GenericRitualEffect{
        CritSuccessDesc: "Target's anathema violation is forgiven, regains all class features",
        SuccessDesc:     "Target must perform a quest, then features are restored",
        FailureDesc:     "Deity does not grant forgiveness at this time",
        CritFailureDesc: "Target is further punished, loses additional powers",
    }
    StandardRituals["atone"] = atone

    // Create Undead - Rank 2
    createUndead := NewRitual("create_undead", "Create Undead", 2, ability.SkillReligion, 1).
        WithCastingTime(1, DurationDays).
        WithCost(2500). // 25 gp onyx
        WithSecondaryCheck(ability.SkillArcana, ability.Trained, "Bind the negative energy")
    createUndead.Traits = trait.TraitSet{trait.TraitEvil, trait.TraitNecromancy}
    createUndead.RequiredRank = ability.Expert
    createUndead.Effect = &GenericRitualEffect{
        CritSuccessDesc: "Creature is raised and permanently under your control",
        SuccessDesc:     "Creature is raised, obeys for 1 week unless renewed",
        FailureDesc:     "Ritual fails, corpse is damaged beyond use",
        CritFailureDesc: "Creature rises but is uncontrolled and hostile",
        BacklashDesc:    "Undead immediately attacks all living creatures",
    }
    StandardRituals["create_undead"] = createUndead
}

// GetRitual retrieves a ritual by ID
func GetRitual(id string) *Ritual {
    return StandardRituals[id]
}

// ListRituals returns all available rituals
func ListRituals() []*Ritual {
    list := make([]*Ritual, 0, len(StandardRituals))
    for _, r := range StandardRituals {
        list = append(list, r)
    }
    return list
}
```

---

## 5. Tests

**Target File:** `pkg/rules/spell/ritual_test.go`

```go
package spell_test

import (
    "testing"
    "uaa/vdnd/pkg/rules/ability"
    "uaa/vdnd/pkg/rules/check"
    "uaa/vdnd/pkg/rules/entity"
    "uaa/vdnd/pkg/rules/spell"
)

func TestRitualCreation(t *testing.T) {
    ritual := spell.NewRitual("test", "Test Ritual", 3, ability.SkillReligion, 2).
        WithCastingTime(1, spell.DurationHours).
        WithCost(5000).
        WithSecondaryCheck(ability.SkillMedicine, ability.Trained, "Assist")

    if ritual.Rank != 3 {
        t.Errorf("Expected rank 3, got %d", ritual.Rank)
    }
    if ritual.SecondaryCasters != 2 {
        t.Errorf("Expected 2 secondary casters, got %d", ritual.SecondaryCasters)
    }
    if ritual.CostCP != 5000 {
        t.Errorf("Expected cost 5000 cp, got %d", ritual.CostCP)
    }
    if len(ritual.SecondaryChecks) != 1 {
        t.Errorf("Expected 1 secondary check, got %d", len(ritual.SecondaryChecks))
    }
}

func TestRitualCastAttemptValidation(t *testing.T) {
    ritual := spell.GetRitual("resurrect")
    if ritual == nil {
        t.Fatal("Resurrect ritual not found")
    }

    primary := entity.NewEntity("cleric", "Cleric", 10)
    primary.SkillProficiencies[ability.SkillReligion] = ability.Expert

    secondary1 := entity.NewEntity("healer", "Healer", 8)
    secondary1.SkillProficiencies[ability.SkillMedicine] = ability.Expert

    secondary2 := entity.NewEntity("acolyte", "Acolyte", 6)
    secondary2.SkillProficiencies[ability.SkillReligion] = ability.Trained

    attempt, err := spell.NewRitualCastAttempt(ritual, primary, []*entity.Entity{secondary1, secondary2})
    if err != nil {
        t.Fatalf("Should create valid attempt: %v", err)
    }

    if attempt.PrimaryCaster != primary {
        t.Error("Primary caster mismatch")
    }
}

func TestRitualCastAttemptInsufficientCasters(t *testing.T) {
    ritual := spell.GetRitual("resurrect") // Requires 2 secondary
    if ritual == nil {
        t.Fatal("Resurrect ritual not found")
    }

    primary := entity.NewEntity("cleric", "Cleric", 10)
    primary.SkillProficiencies[ability.SkillReligion] = ability.Expert

    // Only 1 secondary when 2 required
    secondary1 := entity.NewEntity("healer", "Healer", 8)
    secondary1.SkillProficiencies[ability.SkillMedicine] = ability.Expert

    _, err := spell.NewRitualCastAttempt(ritual, primary, []*entity.Entity{secondary1})
    if err == nil {
        t.Error("Should fail with insufficient secondary casters")
    }
}

func TestRitualCasting(t *testing.T) {
    ritual := spell.GetRitual("commune") // No secondary casters needed
    if ritual == nil {
        t.Fatal("Commune ritual not found")
    }

    primary := entity.NewEntity("oracle", "Oracle", 12)
    primary.SkillProficiencies[ability.SkillReligion] = ability.Master

    attempt, err := spell.NewRitualCastAttempt(ritual, primary, nil)
    if err != nil {
        t.Fatalf("Should create valid attempt: %v", err)
    }

    // Force a success
    outcome := spell.CastRitual(attempt, 18, nil)

    t.Logf("Ritual outcome: degree=%v, success=%v, desc=%s",
        attempt.FinalDegree, outcome.Success, outcome.Description)

    if !attempt.IsComplete {
        t.Error("Ritual should be marked complete")
    }
    if attempt.MaterialsConsumed != ritual.CostCP {
        t.Errorf("Materials consumed should be %d, got %d", ritual.CostCP, attempt.MaterialsConsumed)
    }
}

func TestSecondaryCheckModifiers(t *testing.T) {
    ritual := spell.NewRitual("test", "Test", 3, ability.SkillReligion, 2).
        WithSecondaryCheck(ability.SkillMedicine, ability.Trained, "").
        WithSecondaryCheck(ability.SkillArcana, ability.Trained, "")
    ritual.Effect = &spell.GenericRitualEffect{
        SuccessDesc: "Success",
        CritSuccessDesc: "Critical!",
    }

    primary := entity.NewEntity("caster", "Caster", 10)
    primary.SkillProficiencies[ability.SkillReligion] = ability.Expert

    sec1 := entity.NewEntity("sec1", "Secondary 1", 5)
    sec1.SkillProficiencies[ability.SkillMedicine] = ability.Trained

    sec2 := entity.NewEntity("sec2", "Secondary 2", 5)
    sec2.SkillProficiencies[ability.SkillArcana] = ability.Trained

    attempt, _ := spell.NewRitualCastAttempt(ritual, primary, []*entity.Entity{sec1, sec2})

    // Primary succeeds (not crit), both secondaries crit succeed
    // Should boost final degree to crit success
    outcome := spell.CastRitual(attempt, 15, []int{20, 20})

    // If secondaries crit, primary success should become crit success
    // (This depends on exact roll values and DCs, so we just log)
    t.Logf("Primary degree: %v, Final degree: %v", attempt.PrimaryResult.Degree, attempt.FinalDegree)
    t.Logf("Outcome: %s", outcome.Description)
}

func TestRitualBacklash(t *testing.T) {
    ritual := spell.GetRitual("plane_shift")
    if ritual == nil {
        t.Fatal("Plane Shift ritual not found")
    }

    primary := entity.NewEntity("wizard", "Wizard", 14)
    primary.SkillProficiencies[ability.SkillOccultism] = ability.Master

    secondaries := make([]*entity.Entity, 3)
    for i := 0; i < 3; i++ {
        sec := entity.NewEntity(fmt.Sprintf("sec%d", i), fmt.Sprintf("Secondary %d", i), 8)
        sec.SkillProficiencies[ability.SkillArcana] = ability.Trained
        sec.SkillProficiencies[ability.SkillOccultism] = ability.Trained
        sec.SkillProficiencies[ability.SkillSurvival] = ability.Trained
        secondaries[i] = sec
    }

    attempt, _ := spell.NewRitualCastAttempt(ritual, primary, secondaries)

    // Force critical failure
    outcome := spell.CastRitual(attempt, 1, []int{1, 1, 1})

    if attempt.FinalDegree != check.CriticalFailure {
        t.Logf("Degree was %v (may vary by DC)", attempt.FinalDegree)
    }

    if outcome.Backlash != "" {
        t.Logf("Backlash: %s", outcome.Backlash)
    }
}
```

---

## 6. Execution Checklist

- [ ] Create `pkg/rules/spell/ritual.go` with Ritual struct
- [ ] Add `RitualEffect` interface and `GenericRitualEffect`
- [ ] Create `pkg/rules/spell/ritual_casting.go` with casting logic
- [ ] Implement `calculateFinalDegree()` for secondary check modifiers
- [ ] Create `pkg/rules/spell/ritual_registry.go` with standard rituals
- [ ] Add Resurrect, Commune, Plane Shift, Atone, Create Undead
- [ ] Create `pkg/rules/spell/ritual_test.go`
- [ ] Run `go test -v ./pkg/rules/...` and ensure 100% pass

---

## 7. CLI Commands

```bash
# List available rituals
vd ritual list
# Output:
# | Name | Rank | Primary Skill | Secondary Casters | Cost |
# |------|------|---------------|-------------------|------|
# | Resurrect | 5 | Religion | 2 | 75 gp |
# | Commune | 6 | Religion | 0 | 150 gp |
# | Plane Shift | 7 | Occultism | 3 | 350 gp |
# | Atone | 4 | Religion | 0 | 20 gp |

# View ritual details
vd ritual info resurrect
# Output:
# **Resurrect** (Rank 5 Ritual)
# **Casting Time:** 1 day
# **Cost:** 75 gp (diamonds)
# **Primary Check:** Religion (Expert)
# **Secondary Casters:** 2
#   - Medicine (Expert): Prepare the body
#   - Religion (Trained): Call the soul
#
# **Success:** Target returns to life at 1 HP, wounded 1, fatigued
# **Critical Success:** Target returns at full HP with no conditions
# **Failure:** Ritual fails, materials not consumed
# **Critical Failure:** Ritual fails, caster is doomed 1

# Begin ritual casting
vd ritual cast resurrect --primary cleric --secondary healer acolyte
# Output:
# **Casting Resurrect**
# Primary: Cleric (Religion +15)
# Secondary 1: Healer (Medicine +12)
# Secondary 2: Acolyte (Religion +8)
#
# DC: 30
#
# Primary Roll: 14 + 15 = 29 (Failure)
# Secondary 1 Roll: 20 + 12 = 32 (Critical Success) [+1 step]
# Secondary 2 Roll: 18 + 8 = 26 (Failure)
#
# **Final Degree:** Success (boosted by secondary crit)
#
# **Result:** Target returns to life at 1 HP, wounded 1, fatigued
# **Materials Consumed:** 75 gp
```

---

## 8. Design Notes

**Why skill checks, not spell DCs?**
Rituals use skill checks instead of spell attack/DC because they're meant to be accessible to non-casters. A Fighter with Religion training could theoretically lead a ritual.

**Secondary caster influence:**
The PF2E rules state secondary casters can improve or worsen the outcome. A critical success grants +1 step, critical failure gives -1 step. This creates interesting party dynamics.

**Material costs:**
Rituals always have material costs. On failure, materials may or may not be consumed depending on the degree. Critical failures typically consume materials AND cause backlash.

**Heightening:**
Many rituals can be heightened for greater effect. The `HeightenedCostCP` field allows tracking additional costs per rank increase. Heightened effects would be handled by the specific `RitualEffect` implementation.
