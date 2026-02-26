package cli

import (
	"fmt"
	"strconv"
	"strings"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/dice"
)

func cmdRoll(args []string, deps Deps) (string, error) {
	if len(args) < 1 {
		return "", NewUsageError("missing dice expression", "vd roll <expression> (e.g., 2d6+4, d20, 1d8+1d6)")
	}
	expr := strings.Join(args, "")

	d, err := dice.Parse(expr)
	if err != nil {
		return "", NewRuleError(fmt.Sprintf("invalid dice expression: %s", expr), "Use standard notation like '2d6+4' or shorthand like 'd20'.")
	}

	var allResults []string
	total := 0
	
	for _, g := range d.Groups {
		count := g.Count
		if count < 0 { count = -count }
		
		results := deps.Roller.Roll(count, g.Sides)
		groupTotal := 0
		for _, r := range results {
			groupTotal += r
		}
		
		if g.Count < 0 {
			total -= groupTotal
			allResults = append(allResults, fmt.Sprintf("-[%v]", formatResults(results)))
		} else {
			total += groupTotal
			allResults = append(allResults, fmt.Sprintf("[%v]", formatResults(results)))
		}
	}
	
	total += d.Modifier
	if d.Modifier != 0 || len(allResults) == 0 {
		allResults = append(allResults, fmt.Sprintf("%+d", d.Modifier))
	}

	return fmt.Sprintf("Rolled **%s**: %s = **%d**\n", expr, strings.Join(allResults, " "), total), nil
}

func formatResults(res []int) string {
	str := make([]string, len(res))
	for i, v := range res {
		str[i] = strconv.Itoa(v)
	}
	return strings.Join(str, ", ")
}

func cmdCheck(args []string, deps Deps) (string, error) {
	positional, flags := ParseFlags(args)
	if len(positional) < 2 {
		return "", NewUsageError("missing entity or skill", "vd check <entity_id> <skill> [--dc <N>]")
	}
	entityID := positional[0]
	skill := positional[1]

	dc := 0
	if val, ok := flags["dc"]; ok {
		dc, _ = strconv.Atoi(val)
	}

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", WrapSystemError(err, "failed to load state")
	}

	entity, ok := gameState.Entities[entityID]
	if !ok {
		return "", NewNotFoundError("Entity", entityID, "")
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
