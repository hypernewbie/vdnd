package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"uaa/vdnd/internal/state"
)

func cmdQueryDistance(args []string, deps Deps) (string, error) {
	positional, _ := ParseFlags(args)
	if len(positional) < 2 {
		return "", fmt.Errorf("usage: vd query distance <entity1> <entity2>")
	}
	id1 := positional[0]
	id2 := positional[1]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	e1, ok := gameState.Entities[id1]
	if !ok {
		return "", fmt.Errorf("entity not found: %s", id1)
	}
	e2, ok := gameState.Entities[id2]
	if !ok {
		return "", fmt.Errorf("entity not found: %s", id2)
	}

	if e1.Position == e2.Position {
		return "Distance: **0 ft** (Same Zone).\n", nil
	}

	path, dist := findShortestPath(gameState, e1.Position, e2.Position)
	if path == nil {
		return "Distance: **Infinite** (No path found between zones).\n", nil
	}

	// 1 Zone = 30ft
	ft := dist * 30
	return fmt.Sprintf("Distance: **%d ft** (%d Zones).\nPath: %s\n", ft, dist, strings.Join(path, " -> ")), nil
}

func cmdQueryTargets(args []string, deps Deps) (string, error) {
	positional, flags := ParseFlags(args)
	if len(positional) < 1 {
		return "", fmt.Errorf("usage: vd query targets <entity_id> [--range <ft>] [--melee]")
	}
	actorID := positional[0]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	actor, ok := gameState.Entities[actorID]
	if !ok {
		return "", fmt.Errorf("actor not found: %s", actorID)
	}

	maxRange := 0
	if r, ok := flags["range"]; ok {
		maxRange, _ = strconv.Atoi(r)
	}
	onlyMelee := flags["melee"] == "true"

	var targets []string
	for id, target := range gameState.Entities {
		if id == actorID {
			continue
		}

		// Simple Ally/Enemy logic
		if isAlly(actorID, id) {
			continue
		}

		dist := 0
		if actor.Position != target.Position {
			_, d := findShortestPath(gameState, actor.Position, target.Position)
			if d == 0 { // No path
				continue
			}
			dist = d * 30
		}

		if onlyMelee && dist > 0 {
			// Check if engaged even if different zones? PF2E usually same zone.
			engaged := false
			for _, e := range actor.EngagedWith {
				if e == id {
					engaged = true
					break
				}
			}
			if !engaged {
				continue
			}
		}

		if maxRange > 0 && dist > maxRange {
			continue
		}

		targets = append(targets, id)
	}

	sort.Strings(targets)

	if len(targets) == 0 {
		return "No valid targets found.\n", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Valid Targets for %s\n\n", actor.Name))
	sb.WriteString("| ID | Name | Distance | Cover |\n")
	sb.WriteString("|----|------|----------|-------|\n")
	for _, id := range targets {
		target := gameState.Entities[id]
		dist := 0
		if actor.Position != target.Position {
			_, d := findShortestPath(gameState, actor.Position, target.Position)
			dist = d * 30
		}
		cover := getCover(gameState, actor, target)
		sb.WriteString(fmt.Sprintf("| %s | %s | %d ft | %s |\n", id, target.Name, dist, cover))
	}

	return sb.String(), nil
}

func cmdQueryFlanking(args []string, deps Deps) (string, error) {
	positional, _ := ParseFlags(args)
	if len(positional) < 2 {
		return "", fmt.Errorf("usage: vd query flanking <attacker> <target>")
	}
	attackerID := positional[0]
	targetID := positional[1]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	attacker, ok := gameState.Entities[attackerID]
	if !ok { return "", fmt.Errorf("attacker not found: %s", attackerID) }
	target, ok := gameState.Entities[targetID]
	if !ok { return "", fmt.Errorf("target not found: %s", targetID) }

	// Flanking requires being engaged in melee
	isEngaged := false
	for _, e := range attacker.EngagedWith {
		if e == targetID {
			isEngaged = true
			break
		}
	}
	// Also check position if EngagedWith is not fully updated
	if !isEngaged && attacker.Position == target.Position {
		isEngaged = true
	}

	if !isEngaged {
		return fmt.Sprintf("**%s** is not in melee range of **%s**.\n", attacker.Name, target.Name), nil
	}

	var allies []string
	for id, entity := range gameState.Entities {
		if id == attackerID || id == targetID {
			continue
		}
		if isAlly(attackerID, id) {
			// Check if ally is also engaging target
			allyEngaged := false
			for _, e := range entity.EngagedWith {
				if e == targetID {
					allyEngaged = true
					break
				}
			}
			if !allyEngaged && entity.Position == target.Position {
				allyEngaged = true
			}
			
			if allyEngaged {
				allies = append(allies, entity.Name)
			}
		}
	}

	if len(allies) > 0 {
		return fmt.Sprintf("Target **%s** IS flanked by **%s** and **%s**. (-2 AC flat-footed)\n", 
			target.Name, attacker.Name, strings.Join(allies, ", ")), nil
	}

	return fmt.Sprintf("Target **%s** is NOT flanked.\n", target.Name), nil
}

func cmdQueryCover(args []string, deps Deps) (string, error) {
	positional, _ := ParseFlags(args)
	if len(positional) < 2 {
		return "", fmt.Errorf("usage: vd query cover <attacker> <target>")
	}
	attackerID := positional[0]
	targetID := positional[1]

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	attacker, ok := gameState.Entities[attackerID]
	if !ok { return "", fmt.Errorf("attacker not found: %s", attackerID) }
	target, ok := gameState.Entities[targetID]
	if !ok { return "", fmt.Errorf("target not found: %s", targetID) }

	cover := getCover(gameState, attacker, target)
	if cover == "none" || cover == "" {
		return fmt.Sprintf("**%s** has no cover from **%s**.\n", target.Name, attacker.Name), nil
	}

	return fmt.Sprintf("**%s** has **%s** cover from **%s**.\n", target.Name, cover, attacker.Name), nil
}

// Helpers

func findShortestPath(gs *state.GameState, start, end string) ([]string, int) {
	if start == end {
		return []string{start}, 0
	}

	type node struct {
		id   string
		path []string
	}

	queue := []node{{id: start, path: []string{start}}}
	visited := map[string]bool{start: true}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		zone, ok := gs.Positions[curr.id]
		if !ok {
			continue
		}

		for _, adj := range zone.Adjacent {
			if adj == end {
				finalPath := append(curr.path, adj)
				return finalPath, len(finalPath) - 1
			}
			if !visited[adj] {
				visited[adj] = true
				newPath := make([]string, len(curr.path))
				copy(newPath, curr.path)
				queue = append(queue, node{id: adj, path: append(newPath, adj)})
			}
		}
	}

	return nil, 0
}

func isAlly(id1, id2 string) bool {
	// Simple prefix logic or Hero logic
	isHero1 := strings.HasPrefix(id1, "hero") || strings.HasPrefix(id1, "pc")
	isHero2 := strings.HasPrefix(id2, "hero") || strings.HasPrefix(id2, "pc")
	
	if isHero1 && isHero2 {
		return true
	}

	// Prefix check (e.g. goblin_1 and goblin_2)
	parts1 := strings.Split(id1, "_")
	parts2 := strings.Split(id2, "_")
	if len(parts1) > 1 && len(parts2) > 1 && parts1[0] == parts2[0] {
		return true
	}

	return false
}

func getCover(gs *state.GameState, attacker, target *state.EntityState) string {
	if attacker.Position == target.Position {
		return "none"
	}
	
	targetZone, ok := gs.Positions[target.Position]
	if !ok {
		return "none"
	}

	if targetZone.Cover != "" && targetZone.Cover != "none" {
		return targetZone.Cover
	}

	return "none"
}
