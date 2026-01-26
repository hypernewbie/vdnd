package skill

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
	14, 15, 16, 18, 19, 20, 22, 23, 24, 26, // 0-9
	27, 28, 30, 31, 32, 34, 35, 36, 38, 39, // 10-19
	40, 42, 44, 46, 48, 50, // 20-25
}

func LevelBasedDC(level int) int {
	if level < 0 {
		level = 0
	}
	if level >= len(levelDCs) {
		level = len(levelDCs) - 1
	}
	return levelDCs[level]
}

type DCAdjustment int

const (
	AdjustIncrediblyEasy DCAdjustment = -10
	AdjustVeryEasy       DCAdjustment = -5
	AdjustEasy           DCAdjustment = -2
	AdjustNormal         DCAdjustment = 0
	AdjustHard           DCAdjustment = +2
	AdjustVeryHard       DCAdjustment = +5
	AdjustIncrediblyHard DCAdjustment = +10
)

func AdjustedDC(baseDC int, adj DCAdjustment) int {
	return baseDC + int(adj)
}
