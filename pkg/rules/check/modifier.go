package check

type BonusType int

const (
	BonusUntyped BonusType = iota
	BonusCircumstance
	BonusItem
	BonusStatus
)

type Modifier struct {
	Value  int
	Type   BonusType
	Source string // For display/debugging: "Heroism", "Cover", etc.
}

// CalculateTotal applies PF2E stacking rules to a slice of modifiers.
// Returns the net modifier to add to a d20 roll.
func CalculateTotal(modifiers []Modifier) int {
	// Bonuses: take highest of each type
	bonuses := make(map[BonusType]int)
	// Penalties: take worst (most negative) of each type
	penalties := make(map[BonusType]int)
	untypedPenaltyTotal := 0

	for _, mod := range modifiers {
		if mod.Value > 0 {
			// Bonus
			if mod.Value > bonuses[mod.Type] {
				bonuses[mod.Type] = mod.Value
			}
		} else if mod.Value < 0 {
			// Penalty
			if mod.Type == BonusUntyped {
				// Untyped penalties ALL stack
				untypedPenaltyTotal += mod.Value
			} else {
				// Typed penalties: take the worst (most negative)
				if mod.Value < penalties[mod.Type] {
					penalties[mod.Type] = mod.Value
				}
			}
		}
	}

	total := 0
	for _, v := range bonuses {
		total += v
	}
	for _, v := range penalties {
		total += v
	}
	total += untypedPenaltyTotal

	return total
}
