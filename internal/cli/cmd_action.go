package cli

import (
	"fmt"
	"strconv"
	"strings"
	"uaa/vdnd/internal/state"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/dice"
)

func cmdActionStrike(args []string, deps Deps) (string, error) {
	positional, flags := ParseFlags(args)
	if len(positional) < 2 {
		return "", NewUsageError("missing actor or target ID", "vd action strike <actor_id> <target_id> [--weapon <id>] [--map <0|1|2>]")
	}
	actorID := positional[0]
	targetID := positional[1]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", WrapSystemError(err, "failed to load state")
	}

	actor, ok := gameState.Entities[actorID]
	if !ok {
		return "", NewNotFoundError("Actor", actorID, "")
	}
	target, ok := gameState.Entities[targetID]
	if !ok {
		return "", NewNotFoundError("Target", targetID, "")
	}

	// 1. Select Weapon
	var weapon *state.WeaponState
	weaponID := flags["weapon"]
	if weaponID != "" {
		for _, w := range actor.WieldedWeapons {
			if w.ID == weaponID {
				weapon = &w
				break
			}
		}
		if weapon == nil {
			return "", NewNotFoundError("Weapon", weaponID, fmt.Sprintf("Use 'vd entity get %s' to see wielded weapons.", actorID))
		}
	} else if len(actor.WieldedWeapons) > 0 {
		weapon = &actor.WieldedWeapons[0]
	} else {
		return "", NewRuleError("actor has no weapons wielded", "An actor must have at least one weapon wielded to Strike. Use 'vd entity set' to add weapons.")
	}

	// 2. Calculate Modifiers
	// Base attack bonus: Level + 2 (Trained) + STR or DEX modifier
	// For simplicity in Phase 5, we use STR for all melee and DEX for ranged.
	// We'll assume Melee for now unless we can distinguish.
	attrMod := actor.Abilities.Strength
	// If it's a "finesse" weapon we should use DEX, but we don't have traits in WeaponState yet.
	// Let's just use what's in the state.
	
	baseAttackBonus := actor.Level + 2 + attrMod

	// MAP
	mapVal := gameState.AttacksMade
	if overrideMap := flags["map"]; overrideMap != "" {
		mapVal, _ = strconv.Atoi(overrideMap)
	}
	penalty := 0
	if mapVal == 1 {
		penalty = -5
	} else if mapVal >= 2 {
		penalty = -10
	}

	// 3. Roll Attack
	naturalRoll := deps.Roller.Roll(1, 20)[0]
	res := check.PerformCheckWithRoll(naturalRoll, baseAttackBonus, []check.Modifier{
		{Value: penalty, Source: "MAP"},
	}, target.GetAC())

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s** strikes **%s** with **%s**!\n\n", actor.Name, target.Name, weapon.ID))
	sb.WriteString(fmt.Sprintf("- **Attack Roll:** %d + %d = **%d** vs AC %d\n", res.NaturalRoll, res.Modifiers, res.Total, res.DC))
	sb.WriteString(fmt.Sprintf("- **Result:** %s\n", res.Degree.String()))

	// 4. Handle Damage
	if res.Degree == check.Success || res.Degree == check.CriticalSuccess {
		dmgRoll, err := dice.Parse(weapon.Damage)
		if err != nil {
			return "", NewRuleError(fmt.Sprintf("invalid damage expression: %s", weapon.Damage), "Ensure weapon damage is in standard dice format like '1d6+2'.")
		}
		
		totalDmg := 0
		var dmgBreakdown []string
		for _, g := range dmgRoll.Groups {
			count := g.Count
			if count < 0 { count = -count }
			results := deps.Roller.Roll(count, g.Sides)
			groupTotal := 0
			for _, r := range results {
				groupTotal += r
			}
			if g.Count < 0 {
				totalDmg -= groupTotal
				dmgBreakdown = append(dmgBreakdown, fmt.Sprintf("-%v", results))
			} else {
				totalDmg += groupTotal
				dmgBreakdown = append(dmgBreakdown, fmt.Sprintf("%v", results))
			}
		}

		totalDmg += dmgRoll.Modifier
		// Melee adds STR to damage
		totalDmg += actor.Abilities.Strength

		if res.Degree == check.CriticalSuccess {
			totalDmg *= 2
			sb.WriteString(fmt.Sprintf("- **Damage:** (rolled %s + %d + %d STR) x 2 = **%d** %s\n", 
				strings.Join(dmgBreakdown, " + "), dmgRoll.Modifier, actor.Abilities.Strength, totalDmg, weapon.DamageType))
		} else {
			sb.WriteString(fmt.Sprintf("- **Damage:** rolled %s + %d + %d STR = **%d** %s\n", 
				strings.Join(dmgBreakdown, " + "), dmgRoll.Modifier, actor.Abilities.Strength, totalDmg, weapon.DamageType))
		}

		target.HP -= totalDmg
		if target.HP < 0 {
			target.HP = 0
		}
		sb.WriteString(fmt.Sprintf("- **%s** HP: %d/%d\n", target.Name, target.HP, target.MaxHP))
	}

	// 5. Update State
	gameState.AttacksMade++
	if err := deps.Store.Save(gameState); err != nil {
		return "", fmt.Errorf("failed to save state: %w", err)
	}

	return sb.String(), nil
}

func cmdActionStride(args []string, deps Deps) (string, error) {
	positional, flags := ParseFlags(args)
	if len(positional) < 1 {
		return "", NewUsageError("missing actor ID", "vd action stride <actor_id> --to <zone_id>")
	}
	actorID := positional[0]
	toZone := flags["to"]
	if toZone == "" {
		return "", NewUsageError("missing --to flag", "vd action stride <actor_id> --to <zone_id>")
	}

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", WrapSystemError(err, "failed to load state")
	}

	actor, ok := gameState.Entities[actorID]
	if !ok {
		return "", NewNotFoundError("Actor", actorID, "")
	}

	if _, ok := gameState.Positions[toZone]; !ok {
		return "", NewNotFoundError("Zone", toZone, "Check available zones in the scene description or map.")
	}

	// Phase 6: Reaction Trigger Detection (Attack of Opportunity)
	var reactors []state.AvailableReaction
	for id, entity := range gameState.Entities {
		if id == actorID {
			continue
		}
		// Simplified: if in same zone and has AoO and hasn't used reaction
		if entity.Position == actor.Position {
			hasAoO := false
			for _, r := range entity.Reactions {
				if r == "attack_of_opportunity" {
					hasAoO = true
					break
				}
			}
			if hasAoO && !gameState.ReactionsUsed[id] {
				reactors = append(reactors, state.AvailableReaction{
					EntityID: id,
					Reaction: "attack_of_opportunity",
				})
			}
		}
	}

	if len(reactors) > 0 {
		event := state.PendingEvent{
			ID:      fmt.Sprintf("evt_%d", deps.Clock.Now().UnixNano()),
			Type:    "movement",
			ActorID: actorID,
			Payload: map[string]string{"to": toZone, "from": actor.Position, "mode": "stride"},
			Reactors: reactors,
		}
		gameState.PendingEvents = append(gameState.PendingEvents, event)
		if err := deps.Store.Save(gameState); err != nil {
			return "", fmt.Errorf("failed to save state: %w", err)
		}

		var names []string
		for _, r := range reactors {
			names = append(names, gameState.Entities[r.EntityID].Name)
		}
		return fmt.Sprintf("Movement paused! **%s** triggers reactions from: %s.\nRun `vd pending` to resolve.", 
			actor.Name, strings.Join(names, ", ")), nil
	}

	oldZone := actor.Position
	actor.Position = toZone

	if err := deps.Store.Save(gameState); err != nil {
		return "", fmt.Errorf("failed to save state: %w", err)
	}

	return fmt.Sprintf("**%s** strided from **%s** to **%s**.\n", actor.Name, oldZone, toZone), nil
}

func cmdActionStep(args []string, deps Deps) (string, error) {
	// Identical to stride for Phase 5, but semantic difference for Phase 6 (no reactions)
	positional, flags := ParseFlags(args)
	if len(positional) < 1 {
		return "", NewUsageError("missing actor ID", "vd action step <actor_id> --to <zone_id>")
	}
	actorID := positional[0]
	toZone := flags["to"]
	if toZone == "" {
		return "", NewUsageError("missing --to flag", "vd action step <actor_id> --to <zone_id>")
	}

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", WrapSystemError(err, "failed to load state")
	}

	actor, ok := gameState.Entities[actorID]
	if !ok {
		return "", NewNotFoundError("Actor", actorID, "")
	}

	if _, ok := gameState.Positions[toZone]; !ok {
		return "", NewNotFoundError("Zone", toZone, "Check available zones in the scene description or map.")
	}

	oldZone := actor.Position
	actor.Position = toZone

	if err := deps.Store.Save(gameState); err != nil {
		return "", fmt.Errorf("failed to save state: %w", err)
	}

	return fmt.Sprintf("**%s** stepped from **%s** to **%s**.\n", actor.Name, oldZone, toZone), nil
}

func cmdActionRaiseShield(args []string, deps Deps) (string, error) {
	positional, _ := ParseFlags(args)
	if len(positional) < 1 {
		return "", fmt.Errorf("usage: vd action raise_shield <actor_id>")
	}
	actorID := positional[0]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load state: %w", err)
	}

	actor, ok := gameState.Entities[actorID]
	if !ok {
		return "", fmt.Errorf("actor not found: %s", actorID)
	}

	// Check if already raised
	for _, c := range actor.Conditions {
		if c.ID == "raised_shield" {
			return fmt.Sprintf("**%s** already has their shield raised.\n", actor.Name), nil
		}
	}

	actor.RaisedShield = true
	actor.Conditions = append(actor.Conditions, state.ConditionInstance{
		ID:       "raised_shield",
		Source:   "Action",
		Duration: 1,
	})

	if err := deps.Store.Save(gameState); err != nil {
		return "", fmt.Errorf("failed to save state: %w", err)
	}

	return fmt.Sprintf("**%s** raised their shield (+2 AC).\n", actor.Name), nil
}
