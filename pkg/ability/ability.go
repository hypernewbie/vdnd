package ability

// Ability represents one of the six ability scores
type Ability int

const (
	Strength Ability = iota
	Dexterity
	Constitution
	Intelligence
	Wisdom
	Charisma
)

func (a Ability) String() string {
	return [...]string{"Strength", "Dexterity", "Constitution", "Intelligence", "Wisdom", "Charisma"}[a]
}

// AbilityScores holds all six scores for an entity
type AbilityScores struct {
	Strength     int
	Dexterity    int
	Constitution int
	Intelligence int
	Wisdom       int
	Charisma     int
}

// Get returns the score for a given ability
func (a AbilityScores) Get(ability Ability) int {
	switch ability {
	case Strength:
		return a.Strength
	case Dexterity:
		return a.Dexterity
	case Constitution:
		return a.Constitution
	case Intelligence:
		return a.Intelligence
	case Wisdom:
		return a.Wisdom
	case Charisma:
		return a.Charisma
	default:
		return 0
	}
}

// Modifier returns the modifier for a given ability
func (a AbilityScores) Modifier(ability Ability) int {
	return ModifierFromScore(a.Get(ability))
}

// ModifierFromScore calculates modifier from a raw score.
// Formula: (score - 10) / 2, rounding down (floor).
func ModifierFromScore(score int) int {
	diff := score - 10
	if diff >= 0 {
		return diff / 2
	}
	// Go's integer division truncates toward zero.
	// For negative differences, we need to shift by 1 to simulate a floor.
	return (diff - 1) / 2
}

// CalculateModifier returns the total modifier for a check
// given an ability score, proficiency rank, and character level.
func CalculateModifier(abilityScore int, rank ProficiencyRank, level int) int {
	return ModifierFromScore(abilityScore) + rank.Bonus(level)
}

// CalculateDC returns 10 + the modifier
func CalculateDC(modifier int) int {
	return 10 + modifier
}
