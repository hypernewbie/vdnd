package ability

import "fmt"

type ProficiencyRank int

const (
	Untrained ProficiencyRank = iota
	Trained
	Expert
	Master
	Legendary
)

func (r ProficiencyRank) String() string {
	switch r {
	case Untrained:
		return "Untrained"
	case Trained:
		return "Trained"
	case Expert:
		return "Expert"
	case Master:
		return "Master"
	case Legendary:
		return "Legendary"
	default:
		return fmt.Sprintf("Unknown(%d)", r)
	}
}

// Bonus calculates the proficiency bonus for a given rank and level.
// Untrained: +0
// Trained: level + 2
// Expert: level + 4
// Master: level + 6
// Legendary: level + 8
func (r ProficiencyRank) Bonus(level int) int {
	if r == Untrained {
		return 0
	}
	bonuses := []int{0, 2, 4, 6, 8} // Untrained is handled above, so index 1-4
	return level + bonuses[int(r)]
}
