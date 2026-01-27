package cli

import (
	"fmt"
	"strconv"
	"strings"
	"uaa/vdnd/internal/state"
)

func cmdConditionAdd(args []string, deps Deps) (string, error) {
	positional, flags := ParseFlags(args)
	if len(positional) < 2 {
		return "", fmt.Errorf("usage: vd condition add <entity_id> <condition_id> [value] [--duration <rounds>] [--source <string>]")
	}
	entityID := positional[0]
	conditionID := strings.ToLower(positional[1])
	
	value := 0
	if len(positional) > 2 {
		value, _ = strconv.Atoi(positional[2])
	}

	duration := -1
	if d, ok := flags["duration"]; ok {
		duration, _ = strconv.Atoi(d)
	}
	source := flags["source"]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	entity, ok := gameState.Entities[entityID]
	if !ok {
		return "", fmt.Errorf("entity not found: %s", entityID)
	}

	// Check if exists
	found := false
	for i, c := range entity.Conditions {
		if c.ID == conditionID {
			// If it exists, update value if higher
			if value > c.Value {
				entity.Conditions[i].Value = value
			}
			// Update duration/source if provided? PF2E usually takes the "best" or "longest"
			// For now, let's just update them if provided.
			if duration != -1 {
				entity.Conditions[i].Duration = duration
			}
			if source != "" {
				entity.Conditions[i].Source = source
			}
			found = true
			break
		}
	}

	if !found {
		entity.Conditions = append(entity.Conditions, state.ConditionInstance{
			ID:       conditionID,
			Value:    value,
			Duration: duration,
			Source:   source,
		})
	}

	if err := deps.Store.Save(gameState); err != nil {
		return "", err
	}

	valStr := ""
	if value > 0 {
		valStr = fmt.Sprintf(" %d", value)
	}
	return fmt.Sprintf("Added condition **%s**%s to **%s**.\n", conditionID, valStr, entity.Name), nil
}

func cmdConditionRemove(args []string, deps Deps) (string, error) {
	positional, _ := ParseFlags(args)
	if len(positional) < 2 {
		return "", fmt.Errorf("usage: vd condition remove <entity_id> <condition_id>")
	}
	entityID := positional[0]
	conditionID := strings.ToLower(positional[1])

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	entity, ok := gameState.Entities[entityID]
	if !ok {
		return "", fmt.Errorf("entity not found: %s", entityID)
	}

	newConditions := make([]state.ConditionInstance, 0, len(entity.Conditions))
	removed := false
	for _, c := range entity.Conditions {
		if c.ID == conditionID {
			removed = true
			continue
		}
		newConditions = append(newConditions, c)
	}
	entity.Conditions = newConditions

	if !removed {
		return fmt.Sprintf("**%s** does not have condition **%s**.\n", entity.Name, conditionID), nil
	}

	if err := deps.Store.Save(gameState); err != nil {
		return "", err
	}

	return fmt.Sprintf("Removed condition **%s** from **%s**.\n", conditionID, entity.Name), nil
}

func cmdConditionReduce(args []string, deps Deps) (string, error) {
	positional, _ := ParseFlags(args)
	if len(positional) < 2 {
		return "", fmt.Errorf("usage: vd condition reduce <entity_id> <condition_id> [amount]")
	}
	entityID := positional[0]
	conditionID := strings.ToLower(positional[1])
	
	amount := 1
	if len(positional) > 2 {
		amount, _ = strconv.Atoi(positional[2])
	}

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	entity, ok := gameState.Entities[entityID]
	if !ok {
		return "", fmt.Errorf("entity not found: %s", entityID)
	}

	found := false
	for i, c := range entity.Conditions {
		if c.ID == conditionID {
			entity.Conditions[i].Value -= amount
			if entity.Conditions[i].Value <= 0 {
				// Remove it
				entity.Conditions = append(entity.Conditions[:i], entity.Conditions[i+1:]...)
			}
			found = true
			break
		}
	}

	if !found {
		return fmt.Sprintf("**%s** does not have condition **%s**.\n", entity.Name, conditionID), nil
	}

	if err := deps.Store.Save(gameState); err != nil {
		return "", err
	}

	return fmt.Sprintf("Reduced condition **%s** on **%s** by %d.\n", conditionID, entity.Name, amount), nil
}

func cmdConditionList(args []string, deps Deps) (string, error) {
	positional, _ := ParseFlags(args)
	if len(positional) < 1 {
		return "", fmt.Errorf("usage: vd condition list <entity_id>")
	}
	entityID := positional[0]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	entity, ok := gameState.Entities[entityID]
	if !ok {
		return "", fmt.Errorf("entity not found: %s", entityID)
	}

	if len(entity.Conditions) == 0 {
		return fmt.Sprintf("**%s** has no active conditions.\n", entity.Name), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Conditions for %s\n\n", entity.Name))
	sb.WriteString("| Condition | Value | Duration | Source |\n")
	sb.WriteString("|-----------|-------|----------|--------|\n")
	for _, c := range entity.Conditions {
		durStr := "Infinite"
		if c.Duration != -1 {
			durStr = fmt.Sprintf("%d rnd", c.Duration)
		}
		sb.WriteString(fmt.Sprintf("| %s | %d | %s | %s |\n", c.ID, c.Value, durStr, c.Source))
	}

	return sb.String(), nil
}
