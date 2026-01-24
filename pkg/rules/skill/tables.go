package skill

import "uaa/vdnd/pkg/rules/ability"

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
