package parser

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"uaa/vdnd/internal/state"
)

// ParseEntity parses a simple markdown format into an EntityState.
// Format expected:
// # Name
// - Level: 5
// - HP: 60/60
// ...
func ParseEntity(r io.Reader) (*state.EntityState, error) {
	s := bufio.NewScanner(r)
	e := &state.EntityState{}

	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "# ") {
			e.Name = strings.TrimPrefix(line, "# ")
			continue
		}

		// Strip list marker
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			line = line[2:]
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])

		switch key {
		case "level":
			e.Level, _ = strconv.Atoi(val)
		case "hp":
			hpParts := strings.Split(val, "/")
			e.HP, _ = strconv.Atoi(strings.TrimSpace(hpParts[0]))
			if len(hpParts) > 1 {
				e.MaxHP, _ = strconv.Atoi(strings.TrimSpace(hpParts[1]))
			} else {
				e.MaxHP = e.HP
			}
		case "ac":
			e.AC, _ = strconv.Atoi(val)
		case "speed":
			e.Speed, _ = strconv.Atoi(strings.TrimSuffix(val, "ft"))
		case "strength", "str":
			e.Abilities.Strength, _ = strconv.Atoi(val)
		case "dexterity", "dex":
			e.Abilities.Dexterity, _ = strconv.Atoi(val)
		case "constitution", "con":
			e.Abilities.Constitution, _ = strconv.Atoi(val)
		case "intelligence", "int":
			e.Abilities.Intelligence, _ = strconv.Atoi(val)
		case "wisdom", "wis":
			e.Abilities.Wisdom, _ = strconv.Atoi(val)
		case "charisma", "cha":
			e.Abilities.Charisma, _ = strconv.Atoi(val)
		case "fortitude", "fort":
			e.Fortitude = parseModifier(val)
		case "reflex", "ref":
			e.Reflex = parseModifier(val)
		case "will":
			e.Will = parseModifier(val)
		case "perception":
			e.Perception = parseModifier(val)
		case "ancestry":
			e.Ancestry = val
		case "class":
			e.Class = val
		case "background":
			e.Background = val
		default:
			// Check if it's a known skill or just any other skill
			if e.Skills == nil {
				e.Skills = make(map[string]int)
			}
			e.Skills[key] = parseModifier(val)
		}
	}

	return e, nil
}

func parseModifier(s string) int {
	// Handle "+5", "5", "-2"
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "+")
	i, _ := strconv.Atoi(s)
	return i
}
