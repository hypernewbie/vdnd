package cli

import (
	"fmt"
	"strconv"
	"strings"
	"uaa/vdnd/internal/state"
	"uaa/vdnd/pkg/rules/check"
)

func cmdActionSkill(args []string, deps Deps) (string, error) {
	// Dispatcher for skill actions
	if len(args) < 1 {
		return "", fmt.Errorf("usage: vd action <grapple|trip|shove|demoralize|hide|seek> ...")
	}
	action := strings.ToLower(args[0])
	subArgs := args[1:]

	switch action {
	case "grapple":
		return cmdActionGrapple(subArgs, deps)
	case "trip":
		return cmdActionTrip(subArgs, deps)
	case "shove":
		return cmdActionShove(subArgs, deps)
	case "demoralize":
		return cmdActionDemoralize(subArgs, deps)
	case "hide":
		return cmdActionHide(subArgs, deps)
	case "seek":
		return cmdActionSeek(subArgs, deps)
	default:
		return "", fmt.Errorf("unknown skill action: %s", action)
	}
}

func cmdActionGrapple(args []string, deps Deps) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("usage: vd action grapple <actor_id> <target_id>")
	}
	actorID := args[0]
	targetID := args[1]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	actor, ok := gameState.Entities[actorID]
	if !ok { return "", fmt.Errorf("actor not found: %s", actorID) }
	target, ok := gameState.Entities[targetID]
	if !ok { return "", fmt.Errorf("target not found: %s", targetID) }

	// Athletics vs Fortitude DC
	mod := actor.GetSkillModifier("athletics")
	dc := target.Fortitude + 10
	naturalRoll := deps.Roller.Roll(1, 20)[0]
	res := check.PerformCheckWithRoll(naturalRoll, mod, nil, dc)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s** attempts to **Grapple** **%s**!\n", actor.Name, target.Name))
	sb.WriteString(fmt.Sprintf("- **Athletics Check:** %d + %d = **%d** vs Fortitude DC %d\n", res.NaturalRoll, res.Modifiers, res.Total, res.DC))
	sb.WriteString(fmt.Sprintf("- **Result:** %s\n", res.Degree.String()))

	if res.Degree == check.CriticalSuccess {
		addCondition(target, "restrained", 0, 1)
		sb.WriteString("- **%s** is **Restrained**!\n")
	} else if res.Degree == check.Success {
		addCondition(target, "grabbed", 0, 1)
		sb.WriteString("- **%s** is **Grabbed**!\n")
	}

	if err := deps.Store.Save(gameState); err != nil {
		return "", err
	}
	return fmt.Sprintf(sb.String(), target.Name), nil
}

func cmdActionTrip(args []string, deps Deps) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("usage: vd action trip <actor_id> <target_id>")
	}
	actorID := args[0]
	targetID := args[1]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	actor, ok := gameState.Entities[actorID]
	if !ok { return "", fmt.Errorf("actor not found: %s", actorID) }
	target, ok := gameState.Entities[targetID]
	if !ok { return "", fmt.Errorf("target not found: %s", targetID) }

	// Athletics vs Reflex DC
	mod := actor.GetSkillModifier("athletics")
	dc := target.Reflex + 10
	naturalRoll := deps.Roller.Roll(1, 20)[0]
	res := check.PerformCheckWithRoll(naturalRoll, mod, nil, dc)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s** attempts to **Trip** **%s**!\n", actor.Name, target.Name))
	sb.WriteString(fmt.Sprintf("- **Athletics Check:** %d + %d = **%d** vs Reflex DC %d\n", res.NaturalRoll, res.Modifiers, res.Total, res.DC))
	sb.WriteString(fmt.Sprintf("- **Result:** %s\n", res.Degree.String()))

	if res.Degree == check.CriticalSuccess {
		addCondition(target, "prone", 0, -1)
		sb.WriteString("- **%s** falls **Prone** and takes 1d6 damage!\n")
		// Apply damage manually for now
		dmg := deps.Roller.Roll(1, 6)[0]
		target.HP -= dmg
		if target.HP < 0 { target.HP = 0 }
	} else if res.Degree == check.Success {
		addCondition(target, "prone", 0, -1)
		sb.WriteString("- **%s** falls **Prone**!\n")
	} else if res.Degree == check.CriticalFailure {
		addCondition(actor, "prone", 0, -1)
		sb.WriteString("- **%s** (the actor) falls **Prone**!\n")
	}

	if err := deps.Store.Save(gameState); err != nil {
		return "", err
	}
	return fmt.Sprintf(sb.String(), target.Name, actor.Name), nil
}

func cmdActionShove(args []string, deps Deps) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("usage: vd action shove <actor_id> <target_id>")
	}
	actorID := args[0]
	targetID := args[1]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	actor, ok := gameState.Entities[actorID]
	if !ok { return "", fmt.Errorf("actor not found: %s", actorID) }
	target, ok := gameState.Entities[targetID]
	if !ok { return "", fmt.Errorf("target not found: %s", targetID) }

	// Athletics vs Fortitude DC
	mod := actor.GetSkillModifier("athletics")
	dc := target.Fortitude + 10
	naturalRoll := deps.Roller.Roll(1, 20)[0]
	res := check.PerformCheckWithRoll(naturalRoll, mod, nil, dc)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s** attempts to **Shove** **%s**!\n", actor.Name, target.Name))
	sb.WriteString(fmt.Sprintf("- **Athletics Check:** %d + %d = **%d** vs Fortitude DC %d\n", res.NaturalRoll, res.Modifiers, res.Total, res.DC))
	sb.WriteString(fmt.Sprintf("- **Result:** %s\n", res.Degree.String()))

	if res.Degree == check.CriticalSuccess {
		sb.WriteString("- **%s** is pushed 10ft back!\n")
	} else if res.Degree == check.Success {
		sb.WriteString("- **%s** is pushed 5ft back!\n")
	}

	if err := deps.Store.Save(gameState); err != nil {
		return "", err
	}
	return fmt.Sprintf(sb.String(), target.Name), nil
}

func cmdActionDemoralize(args []string, deps Deps) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("usage: vd action demoralize <actor_id> <target_id>")
	}
	actorID := args[0]
	targetID := args[1]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	actor, ok := gameState.Entities[actorID]
	if !ok { return "", fmt.Errorf("actor not found: %s", actorID) }
	target, ok := gameState.Entities[targetID]
	if !ok { return "", fmt.Errorf("target not found: %s", targetID) }

	// Intimidation vs Will DC
	mod := actor.GetSkillModifier("intimidation")
	dc := target.Will + 10
	naturalRoll := deps.Roller.Roll(1, 20)[0]
	res := check.PerformCheckWithRoll(naturalRoll, mod, nil, dc)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s** attempts to **Demoralize** **%s**!\n", actor.Name, target.Name))
	sb.WriteString(fmt.Sprintf("- **Intimidation Check:** %d + %d = **%d** vs Will DC %d\n", res.NaturalRoll, res.Modifiers, res.Total, res.DC))
	sb.WriteString(fmt.Sprintf("- **Result:** %s\n", res.Degree.String()))

	if res.Degree == check.CriticalSuccess {
		addCondition(target, "frightened", 2, -1)
		sb.WriteString("- **%s** is **Frightened 2**!\n")
	} else if res.Degree == check.Success {
		addCondition(target, "frightened", 1, -1)
		sb.WriteString("- **%s** is **Frightened 1**!\n")
	}

	if err := deps.Store.Save(gameState); err != nil {
		return "", err
	}
	return fmt.Sprintf(sb.String(), target.Name), nil
}

func cmdActionHide(args []string, deps Deps) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("usage: vd action hide <actor_id> [--dc <N>]")
	}
	actorID := args[0]
	_, flags := ParseFlags(args)
	dc := 15 // Default DC if not specified
	if val, ok := flags["dc"]; ok {
		dc, _ = strconv.Atoi(val)
	}

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	actor, ok := gameState.Entities[actorID]
	if !ok { return "", fmt.Errorf("actor not found: %s", actorID) }

	mod := actor.GetSkillModifier("stealth")
	naturalRoll := deps.Roller.Roll(1, 20)[0]
	res := check.PerformCheckWithRoll(naturalRoll, mod, nil, dc)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s** attempts to **Hide**!\n", actor.Name))
	sb.WriteString(fmt.Sprintf("- **Stealth Check:** %d + %d = **%d** vs DC %d\n", res.NaturalRoll, res.Modifiers, res.Total, res.DC))
	sb.WriteString(fmt.Sprintf("- **Result:** %s\n", res.Degree.String()))

	if res.Degree == check.Success || res.Degree == check.CriticalSuccess {
		addCondition(actor, "hidden", 0, -1)
		sb.WriteString("- **%s** is now **Hidden**!\n")
	}

	if err := deps.Store.Save(gameState); err != nil {
		return "", err
	}
	return fmt.Sprintf(sb.String(), actor.Name), nil
}

func cmdActionSeek(args []string, deps Deps) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("usage: vd action seek <actor_id> <target_id>")
	}
	actorID := args[0]
	targetID := args[1]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	actor, ok := gameState.Entities[actorID]
	if !ok { return "", fmt.Errorf("actor not found: %s", actorID) }
	target, ok := gameState.Entities[targetID]
	if !ok { return "", fmt.Errorf("target not found: %s", targetID) }

	// Perception vs Stealth DC (or default 15)
	mod := actor.GetSkillModifier("perception")
	dc := target.GetSkillModifier("stealth") + 10
	if dc < 10 { dc = 15 } // Fallback

	naturalRoll := deps.Roller.Roll(1, 20)[0]
	res := check.PerformCheckWithRoll(naturalRoll, mod, nil, dc)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s** attempts to **Seek** **%s**!\n", actor.Name, target.Name))
	sb.WriteString(fmt.Sprintf("- **Perception Check:** %d + %d = **%d** vs Stealth DC %d\n", res.NaturalRoll, res.Modifiers, res.Total, res.DC))
	sb.WriteString(fmt.Sprintf("- **Result:** %s\n", res.Degree.String()))

	if res.Degree == check.Success || res.Degree == check.CriticalSuccess {
		// Remove hidden
		removed := false
		newConditions := make([]state.ConditionInstance, 0, len(target.Conditions))
		for _, c := range target.Conditions {
			if c.ID == "hidden" {
				removed = true
				continue
			}
			newConditions = append(newConditions, c)
		}
		target.Conditions = newConditions
		if removed {
			sb.WriteString("- **%s** is no longer **Hidden** to you!\n")
		} else {
			sb.WriteString("- Found nothing new.\n")
		}
	}

	if err := deps.Store.Save(gameState); err != nil {
		return "", err
	}
	return fmt.Sprintf(sb.String(), target.Name), nil
}

// Internal helper
func addCondition(e *state.EntityState, id string, val int, dur int) {
	for i, c := range e.Conditions {
		if c.ID == id {
			if val > c.Value {
				e.Conditions[i].Value = val
			}
			return
		}
	}
	e.Conditions = append(e.Conditions, state.ConditionInstance{
		ID:       id,
		Value:    val,
		Duration: dur,
		Source:   "Action",
	})
}
