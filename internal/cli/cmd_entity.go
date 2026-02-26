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

func cmdEntityAdd(args []string, deps Deps) (string, error) {
	positional, flags := ParseFlags(args)
	if len(positional) < 1 {
		return "", NewUsageError("missing entity ID", "vd entity add <id> --file <path>")
	}
	id := positional[0]
	filePath := flags["file"]
	if filePath == "" {
		return "", NewUsageError("missing --file flag", "vd entity add <id> --file <path>")
	}

	f, err := os.Open(filePath)
	if err != nil {
		return "", WrapSystemError(err, fmt.Sprintf("failed to open file: %s", filePath))
	}
	defer f.Close()

	entity, err := parser.ParseEntity(f)
	if err != nil {
		return "", &VDError{
			Category: CatSystem,
			Message:  fmt.Sprintf("failed to parse entity from %s: %v", filePath, err),
			Hint:     "Ensure the file is a valid JSON character template.",
		}
	}
	entity.ID = id

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

	return fmt.Sprintf("Added entity: **%s** (%s)\n", entity.Name, id), nil
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
