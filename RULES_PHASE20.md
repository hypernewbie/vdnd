# Phase 20: Crafting Skill Actions

## Objective

Implement the Crafting skill actions: Craft (item creation), Repair, and Earn Income. These are primarily downtime activities but have mechanical resolution that the engine should handle.

---

## 1. Earn Income Table

**Target File:** `pkg/rules/skill/tables.go`

The Earn Income table defines how much currency is earned per day based on task level and proficiency.

```go
package skill

// EarnIncomeEntry represents daily earnings for a task level
type EarnIncomeEntry struct {
    Level       int
    TrainedCP   int // Copper per day at Trained
    ExpertCP    int
    MasterCP    int
    LegendaryCP int
}

// EarnIncomeTable - earnings in copper pieces per day
// src: rules/rules/tables/earn-income.md
var EarnIncomeTable = []EarnIncomeEntry{
    {Level: 0, TrainedCP: 10, ExpertCP: 10, MasterCP: 10, LegendaryCP: 10},         // 1 sp
    {Level: 1, TrainedCP: 20, ExpertCP: 20, MasterCP: 20, LegendaryCP: 20},         // 2 sp
    {Level: 2, TrainedCP: 30, ExpertCP: 30, MasterCP: 30, LegendaryCP: 30},         // 3 sp
    {Level: 3, TrainedCP: 50, ExpertCP: 50, MasterCP: 50, LegendaryCP: 50},         // 5 sp
    {Level: 4, TrainedCP: 70, ExpertCP: 80, MasterCP: 80, LegendaryCP: 80},         // 7 sp / 8 sp
    {Level: 5, TrainedCP: 90, ExpertCP: 100, MasterCP: 100, LegendaryCP: 100},      // 9 sp / 1 gp
    {Level: 6, TrainedCP: 150, ExpertCP: 200, MasterCP: 200, LegendaryCP: 200},     // 1.5 gp / 2 gp
    {Level: 7, TrainedCP: 200, ExpertCP: 250, MasterCP: 250, LegendaryCP: 250},     // 2 gp / 2.5 gp
    {Level: 8, TrainedCP: 250, ExpertCP: 300, MasterCP: 300, LegendaryCP: 300},     // 2.5 gp / 3 gp
    {Level: 9, TrainedCP: 300, ExpertCP: 400, MasterCP: 400, LegendaryCP: 400},     // 3 gp / 4 gp
    {Level: 10, TrainedCP: 400, ExpertCP: 500, MasterCP: 600, LegendaryCP: 600},    // 4 gp / 5 gp / 6 gp
    {Level: 11, TrainedCP: 500, ExpertCP: 600, MasterCP: 800, LegendaryCP: 800},    // 5 gp / 6 gp / 8 gp
    {Level: 12, TrainedCP: 600, ExpertCP: 800, MasterCP: 1000, LegendaryCP: 1000},  // 6 gp / 8 gp / 10 gp
    {Level: 13, TrainedCP: 700, ExpertCP: 1000, MasterCP: 1500, LegendaryCP: 1500}, // 7 gp / 10 gp / 15 gp
    {Level: 14, TrainedCP: 800, ExpertCP: 1500, MasterCP: 2000, LegendaryCP: 2000}, // 8 gp / 15 gp / 20 gp
    {Level: 15, TrainedCP: 1000, ExpertCP: 2000, MasterCP: 2800, LegendaryCP: 2800},
    {Level: 16, TrainedCP: 1300, ExpertCP: 2500, MasterCP: 3600, LegendaryCP: 4000},
    {Level: 17, TrainedCP: 1500, ExpertCP: 3000, MasterCP: 4500, LegendaryCP: 5500},
    {Level: 18, TrainedCP: 2000, ExpertCP: 4500, MasterCP: 7000, LegendaryCP: 9000},
    {Level: 19, TrainedCP: 3000, ExpertCP: 6000, MasterCP: 10000, LegendaryCP: 13000},
    {Level: 20, TrainedCP: 4000, ExpertCP: 7500, MasterCP: 15000, LegendaryCP: 20000},
}

// GetEarnIncomeAmount returns copper per day for a given level and proficiency
func GetEarnIncomeAmount(level int, prof ability.ProficiencyRank) int {
    if level < 0 {
        level = 0
    }
    if level >= len(EarnIncomeTable) {
        level = len(EarnIncomeTable) - 1
    }

    entry := EarnIncomeTable[level]
    switch prof {
    case ability.Legendary:
        return entry.LegendaryCP
    case ability.Master:
        return entry.MasterCP
    case ability.Expert:
        return entry.ExpertCP
    default:
        return entry.TrainedCP
    }
}
```

---

## 2. Earn Income Action

**Target File:** `pkg/rules/skill/actions.go`

```go
// EarnIncomeResult contains the outcome of an Earn Income check
type EarnIncomeResult struct {
    EarnedCP    int  // Copper pieces earned
    TaskLevel   int  // Level of task attempted
    DaysWorked  int  // Always 1 per check
}

// EarnIncome performs a skill check to earn money during downtime
// src: rules/compendium/skills.md "Earn Income"
// Can use: Crafting, Lore, Performance
func EarnIncome(actor *entity.Entity, skillID ability.SkillID, taskLevel int, naturalRoll int) (EarnIncomeResult, check.CheckResult) {
    result := EarnIncomeResult{TaskLevel: taskLevel, DaysWorked: 1}

    // Must be trained in the skill
    prof := ability.Untrained
    if p, ok := actor.SkillProficiencies[skillID]; ok {
        prof = p
    }
    if prof < ability.Trained {
        return result, check.CheckResult{Degree: check.Failure}
    }

    // Task level cannot exceed your level
    if taskLevel > actor.Level {
        taskLevel = actor.Level
    }

    dc := LevelBasedDC(taskLevel)
    var res check.CheckResult
    if naturalRoll > 0 {
        res = PerformSkillCheckWithRoll(actor, skillID, dc, naturalRoll)
    } else {
        res = PerformSkillCheck(actor, skillID, dc)
    }

    switch res.Degree {
    case check.CriticalSuccess:
        // Earn as if task level was 1 higher
        result.EarnedCP = GetEarnIncomeAmount(taskLevel+1, prof)
    case check.Success:
        result.EarnedCP = GetEarnIncomeAmount(taskLevel, prof)
    case check.Failure:
        // Earn for a task 1 level lower (minimum 0)
        fallbackLevel := taskLevel - 1
        if fallbackLevel < 0 {
            fallbackLevel = 0
        }
        result.EarnedCP = GetEarnIncomeAmount(fallbackLevel, prof)
    case check.CriticalFailure:
        // No earnings, may have wasted resources
        result.EarnedCP = 0
    }

    return result, res
}
```

---

## 3. Craft Action

**Target File:** `pkg/rules/skill/crafting.go`

Crafting is a multi-day activity with setup and daily checks.

```go
package skill

import (
    "uaa/vdnd/pkg/rules/ability"
    "uaa/vdnd/pkg/rules/check"
    "uaa/vdnd/pkg/rules/entity"
)

// CraftingProject tracks ongoing item creation
type CraftingProject struct {
    ItemID          string
    ItemName        string
    ItemLevel       int
    TargetPriceCP   int    // Total cost in copper
    MaterialsSpent  int    // Raw materials consumed (typically half of TargetPrice)
    SetupDays       int    // Days spent in initial setup (min 4 days)
    ProgressCP      int    // Value of work completed
    IsComplete      bool
    IsFailed        bool
}

// NewCraftingProject creates a new crafting project
// src: rules/compendium/skills.md "Craft"
func NewCraftingProject(itemID, itemName string, itemLevel, priceCP int) *CraftingProject {
    return &CraftingProject{
        ItemID:         itemID,
        ItemName:       itemName,
        ItemLevel:      itemLevel,
        TargetPriceCP:  priceCP,
        MaterialsSpent: priceCP / 2, // Raw materials = half price
    }
}

// CraftSetup performs the initial 4 days of crafting setup
// Returns true if setup is complete (after 4 days of calls)
func (p *CraftingProject) CraftSetup() bool {
    p.SetupDays++
    return p.SetupDays >= 4
}

// CraftDailyCheck performs a daily crafting check after setup
// Returns progress made (in CP worth of work)
func CraftDailyCheck(actor *entity.Entity, project *CraftingProject, naturalRoll int) (int, check.CheckResult) {
    // Must be trained in Crafting
    prof := ability.Untrained
    if p, ok := actor.SkillProficiencies[ability.SkillCrafting]; ok {
        prof = p
    }
    if prof < ability.Trained {
        return 0, check.CheckResult{Degree: check.Failure}
    }

    // Must have appropriate proficiency for item level
    // Level 9+ items require Expert, Level 16+ require Master, Level 17+ require Legendary
    requiredProf := ability.Trained
    if project.ItemLevel >= 17 {
        requiredProf = ability.Legendary
    } else if project.ItemLevel >= 16 {
        requiredProf = ability.Master
    } else if project.ItemLevel >= 9 {
        requiredProf = ability.Expert
    }
    if prof < requiredProf {
        return 0, check.CheckResult{Degree: check.Failure}
    }

    dc := LevelBasedDC(project.ItemLevel)
    var res check.CheckResult
    if naturalRoll > 0 {
        res = PerformSkillCheckWithRoll(actor, ability.SkillCrafting, dc, naturalRoll)
    } else {
        res = PerformSkillCheck(actor, ability.SkillCrafting, dc)
    }

    progress := 0
    switch res.Degree {
    case check.CriticalSuccess:
        // Earn income at item level + 1
        progress = GetEarnIncomeAmount(project.ItemLevel+1, prof)
    case check.Success:
        // Earn income at item level
        progress = GetEarnIncomeAmount(project.ItemLevel, prof)
    case check.Failure:
        // No progress but no penalty
        progress = 0
    case check.CriticalFailure:
        // Lose 10% of materials
        lost := project.MaterialsSpent / 10
        project.MaterialsSpent -= lost
        progress = 0
    }

    project.ProgressCP += progress

    // Check if complete
    // Remaining cost = TargetPrice - MaterialsSpent - Progress
    remainingCost := project.TargetPriceCP - project.MaterialsSpent - project.ProgressCP
    if remainingCost <= 0 {
        project.IsComplete = true
    }

    return progress, res
}

// CraftPayRemaining allows spending money to finish immediately
func (p *CraftingProject) GetRemainingCost() int {
    remaining := p.TargetPriceCP - p.MaterialsSpent - p.ProgressCP
    if remaining < 0 {
        return 0
    }
    return remaining
}

// FinishWithPayment completes the item by paying the remaining cost
func (p *CraftingProject) FinishWithPayment(paidCP int) bool {
    if paidCP >= p.GetRemainingCost() {
        p.IsComplete = true
        return true
    }
    return false
}
```

---

## 4. Repair Action

**Target File:** `pkg/rules/skill/actions.go`

```go
// RepairResult contains the outcome of a Repair check
type RepairResult struct {
    Repaired     bool
    HPRestored   int  // For items with HP (shields, etc.)
    MaterialCost int  // Copper spent on materials (if any)
}

// Repair fixes a damaged item (10 minute activity)
// src: rules/compendium/skills.md "Repair"
func Repair(actor *entity.Entity, itemLevel int, naturalRoll int) (RepairResult, check.CheckResult) {
    result := RepairResult{}

    // Must be trained in Crafting
    prof := ability.Untrained
    if p, ok := actor.SkillProficiencies[ability.SkillCrafting]; ok {
        prof = p
    }
    if prof < ability.Trained {
        return result, check.CheckResult{Degree: check.Failure}
    }

    dc := LevelBasedDC(itemLevel)
    var res check.CheckResult
    if naturalRoll > 0 {
        res = PerformSkillCheckWithRoll(actor, ability.SkillCrafting, dc, naturalRoll)
    } else {
        res = PerformSkillCheck(actor, ability.SkillCrafting, dc)
    }

    switch res.Degree {
    case check.CriticalSuccess:
        // Fully repaired, restore all HP
        result.Repaired = true
        result.HPRestored = -1 // -1 means "full"
    case check.Success:
        // Repaired to just above Broken Threshold
        result.Repaired = true
        result.HPRestored = 0 // Enough to not be broken
    case check.Failure:
        // No repair
        result.Repaired = false
    case check.CriticalFailure:
        // Item takes 2d6 damage (handled by caller if item has HP)
        result.Repaired = false
    }

    return result, res
}

// RepairShield specifically repairs a shield
func RepairShield(actor *entity.Entity, shield *item.Shield, naturalRoll int) (RepairResult, check.CheckResult) {
    if shield == nil {
        return RepairResult{}, check.CheckResult{Degree: check.Failure}
    }

    // Use item level 0 for basic shields (could be parameterised)
    itemLevel := 0 // Or derive from shield type

    result, res := Repair(actor, itemLevel, naturalRoll)

    if result.Repaired {
        if result.HPRestored == -1 {
            // Critical success: full HP
            shield.CurrentHP = shield.MaxHP
        } else {
            // Success: restore to just above BT
            if shield.CurrentHP <= shield.BT {
                shield.CurrentHP = shield.BT + 1
            }
        }
    } else if res.Degree == check.CriticalFailure {
        // Deal 2d6 damage to shield
        damage := dice.DieRoll{Count: 2, Sides: 6}.Roll()
        shield.TakeDamage(damage)
    }

    return result, res
}
```

---

## 5. Tests

**Target File:** `pkg/rules/skill/crafting_test.go`

```go
package skill_test

import (
    "testing"
    "uaa/vdnd/pkg/rules/ability"
    "uaa/vdnd/pkg/rules/check"
    "uaa/vdnd/pkg/rules/entity"
    "uaa/vdnd/pkg/rules/skill"
)

func TestEarnIncomeTable(t *testing.T) {
    // Level 5, Trained = 9 sp = 90 cp
    earned := skill.GetEarnIncomeAmount(5, ability.Trained)
    if earned != 90 {
        t.Errorf("Expected 90 cp at level 5 Trained, got %d", earned)
    }

    // Level 5, Expert = 1 gp = 100 cp
    earned = skill.GetEarnIncomeAmount(5, ability.Expert)
    if earned != 100 {
        t.Errorf("Expected 100 cp at level 5 Expert, got %d", earned)
    }
}

func TestEarnIncome(t *testing.T) {
    actor := entity.NewEntity("crafter", "Crafter", 5)
    actor.SkillProficiencies[ability.SkillCrafting] = ability.Trained

    // Force a success roll
    result, res := skill.EarnIncome(actor, ability.SkillCrafting, 3, 15) // DC 18 for level 3

    if result.DaysWorked != 1 {
        t.Error("Should always be 1 day worked per check")
    }

    // With natural 15 + mods, check the degree
    t.Logf("Earn Income result: %d cp, degree: %v", result.EarnedCP, res.Degree)
}

func TestEarnIncomeRequiresTraining(t *testing.T) {
    actor := entity.NewEntity("novice", "Novice", 5)
    // No skill proficiencies set = Untrained

    result, res := skill.EarnIncome(actor, ability.SkillCrafting, 1, 20)

    if res.Degree != check.Failure {
        t.Error("Untrained should automatically fail Earn Income")
    }
    if result.EarnedCP != 0 {
        t.Errorf("Untrained should earn 0, got %d", result.EarnedCP)
    }
}

func TestCraftingProject(t *testing.T) {
    // Create a 10 gp item (1000 cp)
    project := skill.NewCraftingProject("longsword", "Longsword", 0, 1000)

    if project.MaterialsSpent != 500 {
        t.Errorf("Materials should be half price (500), got %d", project.MaterialsSpent)
    }

    // Setup phase (4 days)
    for i := 0; i < 3; i++ {
        if project.CraftSetup() {
            t.Error("Setup should not complete before 4 days")
        }
    }
    if !project.CraftSetup() {
        t.Error("Setup should complete on day 4")
    }
}

func TestCraftDailyProgress(t *testing.T) {
    actor := entity.NewEntity("crafter", "Crafter", 5)
    actor.SkillProficiencies[ability.SkillCrafting] = ability.Trained

    project := skill.NewCraftingProject("dagger", "Dagger", 0, 200) // 2 gp dagger

    // Force critical success
    progress, res := skill.CraftDailyCheck(actor, project, 20)

    if res.Degree != check.CriticalSuccess {
        t.Logf("Degree was %v, not crit success", res.Degree)
    }

    t.Logf("Daily progress: %d cp", progress)
    if project.ProgressCP != progress {
        t.Error("Project progress should match returned progress")
    }
}

func TestRepair(t *testing.T) {
    actor := entity.NewEntity("crafter", "Crafter", 5)
    actor.SkillProficiencies[ability.SkillCrafting] = ability.Trained

    // Force success
    result, res := skill.Repair(actor, 0, 15)

    if res.Degree >= check.Success && !result.Repaired {
        t.Error("Success should repair item")
    }
}

func TestRepairShield(t *testing.T) {
    actor := entity.NewEntity("crafter", "Crafter", 5)
    actor.SkillProficiencies[ability.SkillCrafting] = ability.Trained

    shield := item.NewShield("steel_shield", "Steel Shield", 2, 5, 20, 1)
    shield.CurrentHP = 5 // Broken (BT = 10)

    // Force critical success
    _, res := skill.RepairShield(actor, shield, 20)

    if res.Degree == check.CriticalSuccess {
        if shield.CurrentHP != shield.MaxHP {
            t.Errorf("Crit success should fully repair, got HP %d/%d", shield.CurrentHP, shield.MaxHP)
        }
    }
}
```

---

## 6. Execution Checklist

- [ ] Create `pkg/rules/skill/tables.go` with EarnIncomeTable
- [ ] Add `GetEarnIncomeAmount()` function
- [ ] Add `EarnIncome()` to `pkg/rules/skill/actions.go`
- [ ] Create `pkg/rules/skill/crafting.go` with CraftingProject
- [ ] Implement `CraftSetup()` and `CraftDailyCheck()`
- [ ] Add `Repair()` and `RepairShield()` functions
- [ ] Create `pkg/rules/skill/crafting_test.go`
- [ ] Run `go test -v ./pkg/rules/...` and ensure 100% pass

---

## 7. CLI Commands

```bash
# Start earning income (returns daily amount)
vd downtime earn_income paladin --skill crafting --level 5
# Output:
# **Earn Income Check**
# Skill: Crafting (Trained)
# Task Level: 5
# DC: 20
# Roll: 14 + 8 = 22
# **Result:** Success
# **Earned:** 9 sp (90 cp)

# Start crafting project
vd craft start paladin longsword
# Output:
# **Crafting Project Started**
# Item: Longsword (Level 0)
# Price: 10 gp
# Materials Required: 5 gp (spent)
# Setup: 0/4 days

# Advance crafting (setup phase)
vd craft advance paladin
# Output (day 1-3):
# **Crafting Setup**
# Day 1/4 complete
# (Repeat until day 4)

# Advance crafting (production phase)
vd craft advance paladin
# Output:
# **Crafting Check**
# DC: 14 (Level 0)
# Roll: 18 + 8 = 26
# **Result:** Critical Success
# **Progress:** 2 sp (20 cp)
# **Remaining Cost:** 480 cp / 500 cp paid

# Finish by paying remainder
vd craft finish paladin --pay
# Output:
# **Crafting Complete**
# Paid: 4 gp 80 cp
# Item Added: Longsword

# Repair broken shield
vd repair paladin shield
# Output:
# **Repair Check**
# Item: Steel Shield (HP 5/20, Broken)
# DC: 14
# Roll: 15 + 8 = 23
# **Result:** Success
# Shield HP: 5 → 11 (no longer broken)
```
