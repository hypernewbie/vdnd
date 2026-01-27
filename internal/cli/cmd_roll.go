package cli

import (
	"fmt"
	"strconv"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/dice"
)

func cmdRoll(args []string, deps Deps) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("usage: vd roll <expression>")
	}
	expr := args[0]

	d, err := dice.Parse(expr)
	if err != nil {
		return "", err
	}

	results := deps.Roller.Roll(d.Count, d.Sides)
	total := 0
	for _, r := range results {
		total += r
	}
	total += d.Modifier

	return fmt.Sprintf("Rolled **%s**: %v + %d = **%d**\n", expr, results, d.Modifier, total), nil
}

func cmdCheck(args []string, deps Deps) (string, error) {
	positional, flags := ParseFlags(args)
	if len(positional) < 2 {
		return "", fmt.Errorf("usage: vd check <entity_id> <skill> [--dc <N>]")
	}
	entityID := positional[0]
	skill := positional[1]

	dc := 0
	if val, ok := flags["dc"]; ok {
		dc, _ = strconv.Atoi(val)
	}

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	entity, ok := gameState.Entities[entityID]
	if !ok {
		return "", fmt.Errorf("entity not found: %s", entityID)
	}

	mod := entity.GetSkillModifier(skill)
	naturalRoll := deps.Roller.Roll(1, 20)[0]

	var res check.CheckResult
	if dc > 0 {
		res = check.PerformCheckWithRoll(naturalRoll, mod, nil, dc)
		return fmt.Sprintf("**%s** Check for **%s**: %d + %d = **%d** vs DC %d. **%s**\n",
			skill, entity.Name, res.NaturalRoll, res.Modifiers, res.Total, res.DC, res.Degree.String()), nil
	} else {
		total := naturalRoll + mod
		return fmt.Sprintf("**%s** Check for **%s**: %d + %d = **%d**\n",
			skill, entity.Name, naturalRoll, mod, total), nil
	}
}
