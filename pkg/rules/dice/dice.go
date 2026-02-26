package dice

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DiceGroup represents a collection of dice of the same side.
type DiceGroup struct {
	Count int
	Sides int
}

// DieRoll represents multiple dice groups and a flat modifier.
type DieRoll struct {
	Groups   []DiceGroup
	Modifier int
}

// Roller abstracts dice rolling for testability.
type Roller interface {
	// Roll returns `count` individual die results, each 1 to `sides` inclusive.
	Roll(count, sides int) []int
}

// SimpleRoller uses math/rand for rolling.
type SimpleRoller struct{}

func (r *SimpleRoller) Roll(count, sides int) []int {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	results := make([]int, count)
	for i := range results {
		results[i] = rng.Intn(sides) + 1
	}
	return results
}

// Roll evaluates the dice expression using the default random source.
func (d DieRoll) Roll() int {
	return d.RollWithRNG(rand.New(rand.NewSource(time.Now().UnixNano())))
}

// RollWithRNG allows injecting a random source for testing.
func (d DieRoll) RollWithRNG(rng *rand.Rand) int {
	total := 0
	for _, g := range d.Groups {
		if g.Sides <= 0 {
			continue
		}
		for i := 0; i < g.Count; i++ {
			total += rng.Intn(g.Sides) + 1
		}
	}
	return total + d.Modifier
}

var dicePartRegex = regexp.MustCompile(`^([+-])?(\d+)?d(\d+)$`)
var modPartRegex = regexp.MustCompile(`^([+-])?(\d+)$`)

// Parse converts a string like "2d6+1d4+4", "d20", or "+5" into a DieRoll.
func Parse(expr string) (DieRoll, error) {
	// Normalize: lowercase and remove all whitespace
	clean := strings.ToLower(strings.ReplaceAll(expr, " ", ""))
	if clean == "" {
		return DieRoll{}, fmt.Errorf("empty dice expression")
	}

	// Split by + or -, but keep the operators
	var parts []string
	var current strings.Builder
	for i, r := range clean {
		if (r == '+' || r == '-') && i > 0 {
			parts = append(parts, current.String())
			current.Reset()
		}
		current.WriteRune(r)
	}
	parts = append(parts, current.String())

	dr := DieRoll{}
	for _, part := range parts {
		// Try dice group
		if strings.Contains(part, "d") {
			matches := dicePartRegex.FindStringSubmatch(part)
			if matches == nil {
				return DieRoll{}, fmt.Errorf("invalid dice segment: %s", part)
			}
			
			sign := 1
			if matches[1] == "-" {
				sign = -1
			}

			count := 1
			if matches[2] != "" {
				count, _ = strconv.Atoi(matches[2])
			}

			sides, _ := strconv.Atoi(matches[3])
			dr.Groups = append(dr.Groups, DiceGroup{
				Count: count * sign,
				Sides: sides,
			})
		} else {
			// Try modifier
			matches := modPartRegex.FindStringSubmatch(part)
			if matches == nil {
				return DieRoll{}, fmt.Errorf("invalid modifier segment: %s", part)
			}
			val, _ := strconv.Atoi(matches[0])
			dr.Modifier += val
		}
	}

	return dr, nil
}
