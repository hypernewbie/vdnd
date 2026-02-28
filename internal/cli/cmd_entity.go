package cli

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"uaa/vdnd/internal/parser"
	"uaa/vdnd/internal/state"
)

func applyEntityFlags(e *state.EntityState, flags map[string]string) {
	if val, ok := flags["name"]; ok {
		e.Name = val
	}
	if val, ok := flags["level"]; ok {
		lvl, _ := strconv.Atoi(val)
		e.Level = lvl
	}
	if val, ok := flags["hp"]; ok {
		hp, _ := strconv.Atoi(val)
		e.HP = hp
		e.MaxHP = hp
	}
	if val, ok := flags["maxhp"]; ok {
		hp, _ := strconv.Atoi(val)
		e.MaxHP = hp
	}
	if val, ok := flags["temphp"]; ok {
		hp, _ := strconv.Atoi(val)
		e.TempHP = hp
	}
	if val, ok := flags["ac"]; ok {
		ac, _ := strconv.Atoi(val)
		e.AC = ac
	}
	if val, ok := flags["speed"]; ok {
		spd, _ := strconv.Atoi(strings.TrimSuffix(val, "ft"))
		e.Speed = spd
	}
	if val, ok := flags["ancestry"]; ok {
		e.Ancestry = val
	}
	if val, ok := flags["class"]; ok {
		e.Class = val
	}
	if val, ok := flags["background"]; ok {
		e.Background = val
	}

	// Ability Scores
	if val, ok := flags["str"]; ok {
		e.Abilities.Strength, _ = strconv.Atoi(val)
	}
	if val, ok := flags["dex"]; ok {
		e.Abilities.Dexterity, _ = strconv.Atoi(val)
	}
	if val, ok := flags["con"]; ok {
		e.Abilities.Constitution, _ = strconv.Atoi(val)
	}
	if val, ok := flags["int"]; ok {
		e.Abilities.Intelligence, _ = strconv.Atoi(val)
	}
	if val, ok := flags["wis"]; ok {
		e.Abilities.Wisdom, _ = strconv.Atoi(val)
	}
	if val, ok := flags["cha"]; ok {
		e.Abilities.Charisma, _ = strconv.Atoi(val)
	}

	// Saves & Perception
	if val, ok := flags["fort"]; ok {
		e.Fortitude, _ = strconv.Atoi(val)
	}
	if val, ok := flags["ref"]; ok {
		e.Reflex, _ = strconv.Atoi(val)
	}
	if val, ok := flags["will"]; ok {
		e.Will, _ = strconv.Atoi(val)
	}
	if val, ok := flags["perception"]; ok {
		e.Perception, _ = strconv.Atoi(val)
	}

	// Skills (e.g. --skill stealth=5)
	if val, ok := flags["skill"]; ok {
		parts := strings.Split(val, "=")
		if len(parts) == 2 {
			if e.Skills == nil {
				e.Skills = make(map[string]int)
			}
			skillName := strings.ToLower(strings.TrimSpace(parts[0]))
			bonus, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			e.Skills[skillName] = bonus
		}
	}

	// Weapon (e.g. --weapon longsword:1d8:slashing)
	if val, ok := flags["weapon"]; ok {
		parts := strings.Split(val, ":")
		if len(parts) == 3 {
			w := state.WeaponState{
				ID:         parts[0],
				Damage:     parts[1],
				DamageType: parts[2],
			}
			e.WieldedWeapons = append(e.WieldedWeapons, w)
		}
	}
}

func getEntitySummary(e *state.EntityState) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Entity Status: %s (%s)\n\n", e.Name, e.ID))
	sb.WriteString(fmt.Sprintf("**Level:** %d | **HP:** %d/%d (Temp: %d) | **AC:** %d | **Speed:** %dft\n",
		e.Level, e.HP, e.MaxHP, e.TempHP, e.AC, e.Speed))
	sb.WriteString(fmt.Sprintf("**Ancestry:** %s | **Class:** %s | **Background:** %s\n\n",
		e.Ancestry, e.Class, e.Background))

	sb.WriteString("### Ability Scores\n")
	sb.WriteString(fmt.Sprintf("- **STR:** %d (%+d) | **DEX:** %d (%+d) | **CON:** %d (%+d)\n",
		e.Abilities.Strength, e.GetAbilityModifier(e.Abilities.Strength),
		e.Abilities.Dexterity, e.GetAbilityModifier(e.Abilities.Dexterity),
		e.Abilities.Constitution, e.GetAbilityModifier(e.Abilities.Constitution)))
	sb.WriteString(fmt.Sprintf("- **INT:** %d (%+d) | **WIS:** %d (%+d) | **CHA:** %d (%+d)\n\n",
		e.Abilities.Intelligence, e.GetAbilityModifier(e.Abilities.Intelligence),
		e.Abilities.Wisdom, e.GetAbilityModifier(e.Abilities.Wisdom),
		e.Abilities.Charisma, e.GetAbilityModifier(e.Abilities.Charisma)))

	sb.WriteString("### Saves & Perception\n")
	sb.WriteString(fmt.Sprintf("- **Fort:** %+d | **Ref:** %+d | **Will:** %+d | **Perception:** %+d\n\n",
		e.Fortitude, e.Reflex, e.Will, e.Perception))

	if len(e.Skills) > 0 {
		sb.WriteString("### Skills\n")
		var skills []string
		for k, v := range e.Skills {
			skills = append(skills, fmt.Sprintf("%s: %+d", k, v))
		}
		sort.Strings(skills)
		sb.WriteString("- " + strings.Join(skills, ", ") + "\n\n")
	}

	// Yakka check
	var missing []string
	if e.MaxHP <= 0 { missing = append(missing, "HP") }
	if e.AC <= 0 { missing = append(missing, "AC") }
	if e.Level < 1 { missing = append(missing, "Level") }
	if e.Abilities.Strength == 0 && e.Abilities.Dexterity == 0 { missing = append(missing, "Ability Scores") }
	if e.Fortitude == 0 && e.Reflex == 0 && e.Will == 0 { missing = append(missing, "Saves") }
	if e.Perception == 0 { missing = append(missing, "Perception") }

	if len(missing) > 0 {
		sb.WriteString("⚠️ **NEEDS YAKKA (Missing Core Stats):**\n")
		for _, m := range missing {
			sb.WriteString(fmt.Sprintf("- %s\n", m))
		}
		sb.WriteString("\n*Hint: Use 'vd entity edit' to fill these in.*\n")
	} else {
		sb.WriteString("✅ **Mechanical Yakka Complete.** Entity is ready for combat.\n")
	}

	return sb.String()
}

func cmdEntityAdd(args []string, deps Deps) (string, error) {
	positional, flags := ParseFlags(args)
	if len(positional) < 1 {
		return "", NewUsageError("missing entity ID", "vd entity add <id> [--file <path>] [stats...]")
	}
	id := positional[0]
	filePath := flags["file"]

	var entity *state.EntityState
	var parseErr error

	if filePath != "" {
		f, err := os.Open(filePath)
		if err != nil {
			return "", WrapSystemError(err, fmt.Sprintf("failed to open file: %s", filePath))
		}
		defer f.Close()

		entity, parseErr = parser.ParseEntity(f)
		if parseErr != nil {
			return "", WrapSystemError(parseErr, fmt.Sprintf("failed to parse file: %s", filePath))
		}
	} else {
		entity = &state.EntityState{
			Skills: make(map[string]int),
		}
	}

	entity.ID = id
	applyEntityFlags(entity, flags)

	if entity.Name == "" {
		entity.Name = id
	}

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", WrapSystemError(err, "failed to load state")
	}

	if gameState.Entities == nil {
		gameState.Entities = make(map[string]*state.EntityState)
	}
	gameState.Entities[id] = entity

	if err := deps.Store.Save(gameState); err != nil {
		return "", WrapSystemError(err, "failed to save state")
	}

	return getEntitySummary(entity), nil
}

func cmdEntityEdit(args []string, deps Deps) (string, error) {
	positional, flags := ParseFlags(args)
	if len(positional) < 1 {
		return "", NewUsageError("missing entity ID", "vd entity edit <id> [stats...]")
	}
	id := positional[0]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", WrapSystemError(err, "failed to load state")
	}

	entity, ok := gameState.Entities[id]
	if !ok {
		return "", NewNotFoundError("Entity", id, "")
	}

	applyEntityFlags(entity, flags)

	if err := deps.Store.Save(gameState); err != nil {
		return "", WrapSystemError(err, "failed to save state")
	}

	return getEntitySummary(entity), nil
}

func cmdEntityGet(args []string, deps Deps) (string, error) {
	positional, flags := ParseFlags(args)
	if len(positional) < 1 {
		return "", NewUsageError("missing entity ID", "vd entity get <id> [--field <name>]")
	}
	id := positional[0]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", WrapSystemError(err, "failed to load state")
	}

	entity, ok := gameState.Entities[id]
	if !ok {
		return "", NewNotFoundError("Entity", id, "")
	}

	field := flags["field"]
	if field != "" {
		switch strings.ToLower(field) {
		case "hp":
			return fmt.Sprintf("%d/%d", entity.HP, entity.MaxHP), nil
		case "ac":
			return strconv.Itoa(entity.AC), nil
		case "level":
			return strconv.Itoa(entity.Level), nil
		case "name":
			return entity.Name, nil
		case "position":
			return entity.Position, nil
		default:
			return "", NewUsageError(fmt.Sprintf("unknown field: %s", field), "Available fields: hp, ac, level, name, position")
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s (%s)\n", entity.Name, entity.ID))
	sb.WriteString(fmt.Sprintf("- **Level:** %d\n", entity.Level))
	sb.WriteString(fmt.Sprintf("- **HP:** %d/%d\n", entity.HP, entity.MaxHP))
	sb.WriteString(fmt.Sprintf("- **AC:** %d\n", entity.AC))
	sb.WriteString(fmt.Sprintf("- **Speed:** %dft\n", entity.Speed))
	sb.WriteString(fmt.Sprintf("- **Position:** %s\n", entity.Position))
	// Add more fields as needed

	return sb.String(), nil
}

func cmdEntitySet(args []string, deps Deps) (string, error) {
	positional, _ := ParseFlags(args)
	if len(positional) < 3 {
		return "", NewUsageError("missing arguments for entity set", "vd entity set <id> <field> <value>")
	}
	id := positional[0]
	field := strings.ToLower(positional[1])
	value := positional[2]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", WrapSystemError(err, "failed to load state")
	}

	entity, ok := gameState.Entities[id]
	if !ok {
		return "", NewNotFoundError("Entity", id, "")
	}

	switch field {
	case "hp":
		val, _ := strconv.Atoi(value)
		entity.HP = val
	case "max_hp":
		val, _ := strconv.Atoi(value)
		entity.MaxHP = val
	case "ac":
		val, _ := strconv.Atoi(value)
		entity.AC = val
	case "position":
		entity.Position = value
	case "name":
		entity.Name = value
	default:
		return "", NewUsageError(fmt.Sprintf("unsupported field for set: %s", field), "Supported fields: hp, max_hp, ac, position, name")
	}

	if err := deps.Store.Save(gameState); err != nil {
		return "", WrapSystemError(err, "failed to save state")
	}

	return fmt.Sprintf("Updated %s for **%s**: %s\n", field, id, value), nil
}

func cmdEntityList(args []string, deps Deps) (string, error) {
	_, flags := ParseFlags(args)
	zoneFilter := flags["zone"]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", WrapSystemError(err, "failed to load state")
	}

	if len(gameState.Entities) == 0 {
		return "No entities found.\n", nil
	}

	var ids []string
	for id := range gameState.Entities {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var sb strings.Builder
	sb.WriteString("| ID | Name | Level | HP | Position |\n")
	sb.WriteString("|----|------|-------|----|----------|\n")

	count := 0
	for _, id := range ids {
		e := gameState.Entities[id]
		if zoneFilter != "" && e.Position != zoneFilter {
			continue
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %d | %d/%d | %s |\n",
			id, e.Name, e.Level, e.HP, e.MaxHP, e.Position))
		count++
	}

	if count == 0 && zoneFilter != "" {
		return fmt.Sprintf("No entities found in zone: %s\n", zoneFilter), nil
	}

	return sb.String(), nil
}

func cmdEntitySpawn(args []string, deps Deps) (string, error) {
	positional, flags := ParseFlags(args)
	if len(positional) < 1 {
		return "", NewUsageError("missing template path", "vd entity spawn <template_path> [--count N] [--prefix str]")
	}
	templatePath := positional[0]
	countStr := flags["count"]
	count := 1
	if countStr != "" {
		count, _ = strconv.Atoi(countStr)
	}
	prefix := flags["prefix"]
	if prefix == "" {
		prefix = "mob"
	}

	f, err := os.Open(templatePath)
	if err != nil {
		return "", WrapSystemError(err, fmt.Sprintf("failed to open template: %s", templatePath))
	}
	defer f.Close()

	templateEntity, err := parser.ParseEntity(f)
	if err != nil {
		return "", &VDError{
			Category: CatSystem,
			Message:  fmt.Sprintf("failed to parse template: %v", err),
			Hint:     "Ensure the template file is a valid JSON character template.",
		}
	}

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", WrapSystemError(err, "failed to load state")
	}

	if gameState.Entities == nil {
		gameState.Entities = make(map[string]*state.EntityState)
	}

	for i := 1; i <= count; i++ {
		id := fmt.Sprintf("%s_%d", prefix, i)
		// Check for collision
		for {
			if _, exists := gameState.Entities[id]; !exists {
				break
			}
			i++
			id = fmt.Sprintf("%s_%d", prefix, i)
		}

		newEntity := *templateEntity // Shallow copy
		newEntity.ID = id
		// If template has a name like "Goblin", we might want "Goblin 1"
		newEntity.Name = fmt.Sprintf("%s %d", templateEntity.Name, i)
		
		gameState.Entities[id] = &newEntity
	}

	if err := deps.Store.Save(gameState); err != nil {
		return "", fmt.Errorf("failed to save state: %w", err)
	}

	return fmt.Sprintf("Spawned %d entities with prefix **%s**\n", count, prefix), nil
}
