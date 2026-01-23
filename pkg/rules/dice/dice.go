package dice

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DieRoll represents a collection of dice of the same side, plus a flat modifier.
type DieRoll struct {
	Count    int // Number of dice (e.g., 2 in "2d6")
	Sides    int // Die type (e.g., 6 in "2d6")
	Modifier int // Flat bonus (e.g., 4 in "2d6+4")
}

// Roll evaluates the dice expression using the default random source.
func (d DieRoll) Roll() int {
	return d.RollWithRNG(rand.New(rand.NewSource(time.Now().UnixNano())))
}

// RollWithRNG allows injecting a random source for testing.
func (d DieRoll) RollWithRNG(rng *rand.Rand) int {
	if d.Sides <= 0 {
		return d.Modifier
	}
	total := 0
	for i := 0; i < d.Count; i++ {
		total += rng.Intn(d.Sides) + 1
	}
	return total + d.Modifier
}

var diceRegex = regexp.MustCompile(`^(\d+)d(\d+)([+-]\d+)?$`)

// Parse converts a string like "2d6+4" or "1d20" into a DieRoll.
func Parse(expr string) (DieRoll, error) {
	expr = strings.ReplaceAll(expr, " ", "")
	matches := diceRegex.FindStringSubmatch(expr)
	if matches == nil {
		return DieRoll{}, fmt.Errorf("invalid dice expression: %s", expr)
	}

	count, _ := strconv.Atoi(matches[1])
	sides, _ := strconv.Atoi(matches[2])
	modifier := 0
	if matches[3] != "" {
		modifier, _ = strconv.Atoi(matches[3])
	}

	return DieRoll{
		Count:    count,
		Sides:    sides,
		Modifier: modifier,
	}, nil
}
