package cli

import (
	"fmt"
	"strconv"
	"strings"
	"uaa/vdnd/internal/state"
)

func cmdDamage(args []string, deps Deps) (string, error) {
	positional, flags := ParseFlags(args)
	if len(positional) < 2 {
		return "", fmt.Errorf("usage: vd damage <entity_id> <amount> [type]")
	}
	entityID := positional[0]
	amount, _ := strconv.Atoi(positional[1])
	dmgType := "untyped"
	if len(positional) > 2 {
		dmgType = strings.ToLower(positional[2])
	}

	source := flags["from"]
	isCrit := flags["crit"] == "true"

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	entity, ok := gameState.Entities[entityID]
	if !ok {
		return "", fmt.Errorf("entity not found: %s", entityID)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s** takes damage", entity.Name))
	if source != "" {
		sb.WriteString(fmt.Sprintf(" from **%s**", source))
	}
	sb.WriteString(":\n")

	initialAmount := amount
	
	// 1. Immunities
	for _, imm := range entity.Immunities {
		if strings.ToLower(imm) == dmgType {
			sb.WriteString(fmt.Sprintf("- Immune to **%s**! (0 damage)\n", dmgType))
			amount = 0
			break
		}
	}

	if amount > 0 {
		// 2. Weaknesses
		if val, ok := entity.Weaknesses[dmgType]; ok {
			amount += val
			sb.WriteString(fmt.Sprintf("- Weakness to **%s**: +%d damage\n", dmgType, val))
		}
		// 3. Resistances
		if val, ok := entity.Resistances[dmgType]; ok {
			amount -= val
			if amount < 0 {
				amount = 0
			}
			sb.WriteString(fmt.Sprintf("- Resistance to **%s**: -%d damage\n", dmgType, val))
		}
	}

	finalDmg := amount
	sb.WriteString(fmt.Sprintf("- **Final Damage:** %d %s (from %d base)\n", finalDmg, dmgType, initialAmount))

	// 4. Temp HP
	if finalDmg > 0 && entity.TempHP > 0 {
		absorbed := entity.TempHP
		if absorbed > finalDmg {
			absorbed = finalDmg
		}
		entity.TempHP -= absorbed
		finalDmg -= absorbed
		sb.WriteString(fmt.Sprintf("- **Temp HP absorbed:** %d (%d remaining)\n", absorbed, entity.TempHP))
	}

	// 5. HP Reduction
	if finalDmg > 0 {
		entity.HP -= finalDmg
		if entity.HP <= 0 {
			entity.HP = 0
			// 6. Death/Dying Check
			handleDying(entity, isCrit, &sb)
		}
		sb.WriteString(fmt.Sprintf("- **%s** HP: %d/%d\n", entity.Name, entity.HP, entity.MaxHP))
	} else if initialAmount > 0 && amount > 0 {
		// All absorbed by Temp HP
		sb.WriteString(fmt.Sprintf("- **%s** HP: %d/%d (unchanged)\n", entity.Name, entity.HP, entity.MaxHP))
	}

	if err := deps.Store.Save(gameState); err != nil {
		return "", err
	}

	return sb.String(), nil
}

func handleDying(entity *state.EntityState, isCrit bool, sb *strings.Builder) {
	// Find Wounded and Dying conditions
	dyingIdx := -1
	woundedVal := 0
	for i, c := range entity.Conditions {
		if c.ID == "dying" {
			dyingIdx = i
		} else if c.ID == "wounded" {
			woundedVal = c.Value
		}
	}

	increase := 1
	if isCrit {
		increase = 2
	}

	if dyingIdx != -1 {
		entity.Conditions[dyingIdx].Value += increase
		sb.WriteString(fmt.Sprintf("- **Dying** increased to %d!\n", entity.Conditions[dyingIdx].Value))
	} else {
		newDying := 1 + woundedVal
		if isCrit {
			newDying += 1
		}
		entity.Conditions = append(entity.Conditions, state.ConditionInstance{
			ID:    "dying",
			Value: newDying,
		})
		sb.WriteString(fmt.Sprintf("- **%s** is falling! **Dying %d**\n", entity.Name, newDying))
	}
}

func cmdHeal(args []string, deps Deps) (string, error) {
	positional, flags := ParseFlags(args)
	if len(positional) < 2 {
		return "", fmt.Errorf("usage: vd heal <entity_id> <amount>")
	}
	entityID := positional[0]
	amount, _ := strconv.Atoi(positional[1])
	source := flags["from"]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	entity, ok := gameState.Entities[entityID]
	if !ok {
		return "", fmt.Errorf("entity not found: %s", entityID)
	}

	wasDying := false
	dyingIdx := -1
	for i, c := range entity.Conditions {
		if c.ID == "dying" {
			wasDying = true
			dyingIdx = i
			break
		}
	}

	entity.HP += amount
	if entity.HP > entity.MaxHP {
		entity.HP = entity.MaxHP
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s** is healed for %d HP", entity.Name, amount))
	if source != "" {
		sb.WriteString(fmt.Sprintf(" by **%s**", source))
	}
	sb.WriteString(".\n")
	sb.WriteString(fmt.Sprintf("- **%s** HP: %d/%d\n", entity.Name, entity.HP, entity.MaxHP))

	if wasDying {
		// Remove Dying
		entity.Conditions = append(entity.Conditions[:dyingIdx], entity.Conditions[dyingIdx+1:]...)
		// Add/Increase Wounded
		woundedIdx := -1
		for i, c := range entity.Conditions {
			if c.ID == "wounded" {
				woundedIdx = i
				break
			}
		}
		if woundedIdx != -1 {
			entity.Conditions[woundedIdx].Value++
			sb.WriteString(fmt.Sprintf("- **Dying** removed. **Wounded** increased to %d.\n", entity.Conditions[woundedIdx].Value))
		} else {
			entity.Conditions = append(entity.Conditions, state.ConditionInstance{
				ID:    "wounded",
				Value: 1,
			})
			sb.WriteString("- **Dying** removed. **Wounded 1** added.\n")
		}
	}

	if err := deps.Store.Save(gameState); err != nil {
		return "", err
	}

	return sb.String(), nil
}

func cmdTempHP(args []string, deps Deps) (string, error) {
	positional, _ := ParseFlags(args)
	if len(positional) < 2 {
		return "", fmt.Errorf("usage: vd temp_hp <entity_id> <amount>")
	}
	entityID := positional[0]
	amount, _ := strconv.Atoi(positional[1])

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	entity, ok := gameState.Entities[entityID]
	if !ok {
		return "", fmt.Errorf("entity not found: %s", entityID)
	}

	if amount > entity.TempHP {
		entity.TempHP = amount
	}

	if err := deps.Store.Save(gameState); err != nil {
		return "", err
	}

	return fmt.Sprintf("**%s** now has %d **Temp HP**.\n", entity.Name, entity.TempHP), nil
}
