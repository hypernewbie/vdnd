package skill

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/entity"
)

// CraftingProject tracks ongoing item creation
type CraftingProject struct {
	ItemID         string
	ItemName       string
	ItemLevel      int
	TargetPriceCP  int // Total cost in copper
	MaterialsSpent int // Raw materials consumed (typically half of TargetPrice)
	SetupDays      int // Days spent in initial setup (min 4 days)
	ProgressCP     int // Value of work completed
	IsComplete     bool
	IsFailed       bool
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

// GetRemainingCost returns the remaining cost to finish the item
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
