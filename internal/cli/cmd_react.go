package cli

import (
	"fmt"
	"strings"
	"uaa/vdnd/internal/state"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/dice"
)

func cmdPending(args []string, deps Deps) (string, error) {
	gameState, err := deps.Store.Load()
	if err != nil {
		return "", WrapSystemError(err, "failed to load state")
	}

	if len(gameState.PendingEvents) == 0 {
		return "No pending events.\n", nil
	}

	event := gameState.PendingEvents[len(gameState.PendingEvents)-1]
	actor := gameState.Entities[event.ActorID]

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Pending Event: %s\n", event.Type))
	sb.WriteString(fmt.Sprintf("- **Actor:** %s\n", actor.Name))
	for k, v := range event.Payload {
		sb.WriteString(fmt.Sprintf("- **%s:** %s\n", k, v))
	}
	sb.WriteString("\n## Potential Reactors\n")
	for _, r := range event.Reactors {
		reactor := gameState.Entities[r.EntityID]
		sb.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", reactor.Name, r.EntityID, r.Reaction))
	}

	return sb.String(), nil
}

func cmdReact(args []string, deps Deps) (string, error) {
	positional, _ := ParseFlags(args)
	if len(positional) < 2 {
		return "", NewUsageError("missing reactor or reaction name", "vd react <entity_id> <reaction_name>")
	}
	reactorID := positional[0]
	reactionName := positional[1]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", WrapSystemError(err, "failed to load state")
	}

	if len(gameState.PendingEvents) == 0 {
		return "", NewStateError("no pending events to react to", "Run 'vd pending' to see if there are any events.")
	}

	eventIdx := len(gameState.PendingEvents) - 1
	event := gameState.PendingEvents[eventIdx]

	// Validate reactor is in the list
	foundIdx := -1
	for i, r := range event.Reactors {
		if r.EntityID == reactorID && r.Reaction == reactionName {
			foundIdx = i
			break
		}
	}
	if foundIdx == -1 {
		return "", NewRuleError(fmt.Sprintf("entity %s cannot perform reaction %s for this event", reactorID, reactionName), "Check 'vd pending' for valid reactors.")
	}

	reactor := gameState.Entities[reactorID]
	actor := gameState.Entities[event.ActorID]
	
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s** uses **%s** against **%s**!\n\n", reactor.Name, reactionName, actor.Name))

	disrupted := false
	if reactionName == "attack_of_opportunity" {
		// Perform a Strike
		// For simplicity, use first weapon
		if len(reactor.WieldedWeapons) == 0 {
			return "", NewRuleError("reactor has no weapons", "An actor must have a weapon to perform an Attack of Opportunity.")
		}
		weapon := reactor.WieldedWeapons[0]
		
		attrMod := reactor.Abilities.Strength
		baseAttackBonus := reactor.Level + 2 + attrMod
		
		naturalRoll := deps.Roller.Roll(1, 20)[0]
		res := check.PerformCheckWithRoll(naturalRoll, baseAttackBonus, nil, actor.GetAC())
		
		sb.WriteString(fmt.Sprintf("- **Attack Roll:** %d + %d = **%d** vs AC %d\n", res.NaturalRoll, res.Modifiers, res.Total, res.DC))
		sb.WriteString(fmt.Sprintf("- **Result:** %s\n", res.Degree.String()))

		if res.Degree == check.Success || res.Degree == check.CriticalSuccess {
			dmgRoll, _ := dice.Parse(weapon.Damage)
			totalDmg := 0
			for _, g := range dmgRoll.Groups {
				count := g.Count
				if count < 0 { count = -count }
				results := deps.Roller.Roll(count, g.Sides)
				groupTotal := 0
				for _, r := range results { groupTotal += r }
				if g.Count < 0 {
					totalDmg -= groupTotal
				} else {
					totalDmg += groupTotal
				}
			}
			totalDmg += dmgRoll.Modifier + reactor.Abilities.Strength
			if res.Degree == check.CriticalSuccess {
				totalDmg *= 2
				disrupted = true // AoO disrupts on Critical Hit
				sb.WriteString(fmt.Sprintf("- **Damage:** **%d** (Critical!)\n", totalDmg))
			} else {
				sb.WriteString(fmt.Sprintf("- **Damage:** **%d**\n", totalDmg))
			}
			actor.HP -= totalDmg
			if actor.HP < 0 { actor.HP = 0 }
		}
	}

	// Mark reaction as used
	gameState.ReactionsUsed[reactorID] = true

	if disrupted {
		sb.WriteString("\n**CRITICAL HIT! Action disrupted.**\n")
		// Remove event
		gameState.PendingEvents = append(gameState.PendingEvents[:eventIdx], gameState.PendingEvents[eventIdx+1:]...)
	} else {
		// Remove reactor from list
		event.Reactors = append(event.Reactors[:foundIdx], event.Reactors[foundIdx+1:]...)
		if len(event.Reactors) == 0 {
			// Resume original action
			sb.WriteString("\nNo more reactors. Resuming action...\n")
			resumeOut, err := resolveEvent(event, gameState)
			if err != nil { return "", err }
			sb.WriteString(resumeOut)
			// Remove event
			gameState.PendingEvents = append(gameState.PendingEvents[:eventIdx], gameState.PendingEvents[eventIdx+1:]...)
		} else {
			// Update event in state
			gameState.PendingEvents[eventIdx] = event
			sb.WriteString(fmt.Sprintf("\n%d reactors remaining.\n", len(event.Reactors)))
		}
	}

	if err := deps.Store.Save(gameState); err != nil {
		return "", err
	}

	return sb.String(), nil
}

func cmdReactSkip(args []string, deps Deps) (string, error) {
	positional, _ := ParseFlags(args)
	
	gameState, err := deps.Store.Load()
	if err != nil {
		return "", WrapSystemError(err, "failed to load state")
	}

	if len(gameState.PendingEvents) == 0 {
		return "", NewStateError("no pending events", "There are no actions currently waiting for reactions.")
	}

	eventIdx := len(gameState.PendingEvents) - 1
	event := gameState.PendingEvents[eventIdx]

	if len(positional) > 0 {
		reactorID := positional[0]
		foundIdx := -1
		for i, r := range event.Reactors {
			if r.EntityID == reactorID {
				foundIdx = i
				break
			}
		}
		if foundIdx == -1 {
			return "", NewNotFoundError("Reactor", reactorID, "Check 'vd pending' for valid reactors.")
		}
		event.Reactors = append(event.Reactors[:foundIdx], event.Reactors[foundIdx+1:]...)
	} else {
		if len(event.Reactors) > 0 {
			event.Reactors = event.Reactors[1:]
		}
	}

	var sb strings.Builder
	if len(event.Reactors) == 0 {
		sb.WriteString("All reactions skipped. Resuming action...\n")
		resumeOut, err := resolveEvent(event, gameState)
		if err != nil { return "", err }
		sb.WriteString(resumeOut)
		gameState.PendingEvents = append(gameState.PendingEvents[:eventIdx], gameState.PendingEvents[eventIdx+1:]...)
	} else {
		gameState.PendingEvents[eventIdx] = event
		sb.WriteString(fmt.Sprintf("Reaction skipped. %d reactors remaining.\n", len(event.Reactors)))
	}

	if err := deps.Store.Save(gameState); err != nil {
		return "", WrapSystemError(err, "failed to save state")
	}

	return sb.String(), nil
}

func cmdReactSkipAll(args []string, deps Deps) (string, error) {
	gameState, err := deps.Store.Load()
	if err != nil {
		return "", WrapSystemError(err, "failed to load state")
	}

	if len(gameState.PendingEvents) == 0 {
		return "", NewStateError("no pending events", "There are no actions currently waiting for reactions.")
	}

	eventIdx := len(gameState.PendingEvents) - 1
	event := gameState.PendingEvents[eventIdx]

	var sb strings.Builder
	sb.WriteString("All reactions skipped. Resuming action...\n")
	resumeOut, err := resolveEvent(event, gameState)
	if err != nil { return "", err }
	sb.WriteString(resumeOut)
	
	gameState.PendingEvents = append(gameState.PendingEvents[:eventIdx], gameState.PendingEvents[eventIdx+1:]...)

	if err := deps.Store.Save(gameState); err != nil {
		return "", WrapSystemError(err, "failed to save state")
	}

	return sb.String(), nil
}

func resolveEvent(event state.PendingEvent, gameState *state.GameState) (string, error) {
	actor, ok := gameState.Entities[event.ActorID]
	if !ok { return "", NewNotFoundError("Actor", event.ActorID, "") }

	switch event.Type {
	case "movement":
		toZone := event.Payload["to"]
		fromZone := event.Payload["from"]
		actor.Position = toZone
		return fmt.Sprintf("**%s** finished moving from **%s** to **%s**.\n", actor.Name, fromZone, toZone), nil
	default:
		return "", &VDError{Category: CatSystem, Message: fmt.Sprintf("unknown event type: %s", event.Type)}
	}
}
