package llm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"uaa/vdnd/internal/cli"
	"uaa/vdnd/internal/llm/vdhelpers"
	"uaa/vdnd/internal/state"
)

// Helper to create deps with a temp file store and fixed roller
func newTestDeps(t *testing.T, rolls []int) cli.Deps {
	tmpDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(tmpDir, "rules"), 0755); err != nil {
		t.Fatal(err)
	}

	return cli.Deps{
		Roller: &cli.FixedRoller{Results: rolls},
		Store:  &state.FileStore{Root: tmpDir},
		Clock:  &cli.RealClock{},
		Stderr: os.Stderr,
		Cwd:    tmpDir,
	}
}

func createEntityFile(t *testing.T, dir, name, displayName, content string) string {
	path := filepath.Join(dir, name)
	fullContent := "# " + displayName + "\n" + content
	if err := os.WriteFile(path, []byte(fullContent), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOrchestrator_CombatFlow(t *testing.T) {
	// We need rolls for:
	// 1. Strike attack roll (1d20) -> 15
	// 2. Strike damage roll (1d8) -> 5
	deps := newTestDeps(t, []int{15, 5})
	
	heroPath := createEntityFile(t, deps.Cwd, "hero.md", "Hero", `
- HP: 20/20
- AC: 15
- Speed: 25ft
- Str: 4
- Dex: 2
`)
	goblinPath := createEntityFile(t, deps.Cwd, "goblin.md", "Goblin", `
- HP: 10/10
- AC: 13
- Speed: 25ft
- Str: 1
- Dex: 3
`)

	ctx := context.Background()
	p := NewDummyProvider("test")
	o := NewOrchestrator(ctx, p, deps)

	execute := func(name, args string) string {
		call := ToolCall{
			Name:      name,
			Arguments: args,
		}
		return o.executeTool(call)
	}

	// 1. Create scene
	execute("vd_scene_new", `{"name": "test_scene"}`)

	// 2. Add entities
	execute("vd", `{"cmd": "entity add hero --file `+heroPath+`"}`)
	execute("vd", `{"cmd": "entity add goblin --file `+goblinPath+`"}`)

	// 3. Manual Weapon Injection
	gs, _ := deps.Store.Load()
	hero := gs.Entities["hero"]
	hero.WieldedWeapons = append(hero.WieldedWeapons, state.WeaponState{
		ID:         "longsword",
		Damage:     "1d8",
		DamageType: "slashing",
	})
	deps.Store.Save(gs)

	// 4. Strike
	// Attack: 15 (roll) + 4 (str) + 2 (trained) + 0 (level) = 21 vs AC 13 -> Success
	// Damage: 5 (roll) + 4 (str) = 9 slashing
	res := execute("vd_action_strike", `{"actor": "hero", "target": "goblin"}`)
	var result vdhelpers.VDResult
	json.Unmarshal([]byte(res), &result)
	if result.ExitCode != 0 {
		t.Fatalf("Strike failed: %s", result.Error)
	}
	if !strings.Contains(result.Stdout, "rolled [5]") || !strings.Contains(result.Stdout, "9") {
		t.Errorf("Unexpected strike result: %s", result.Stdout)
	}

	// 5. Verify goblin HP
	res = execute("vd_status", "{}")
	json.Unmarshal([]byte(res), &result)
	// 10 - 9 = 1
	if !strings.Contains(result.Stdout, "1/10") {
		t.Errorf("Expected goblin to have 1 HP, got: %s", result.Stdout)
	}

	// 6. Apply more damage
	execute("vd_damage", `{"id": "goblin", "amount": 5}`)
	
	// 7. Final status
	res = execute("vd_status", "{}")
	json.Unmarshal([]byte(res), &result)
	if !strings.Contains(result.Stdout, "0/10") {
		t.Errorf("Expected goblin to have 0 HP, got: %s", result.Stdout)
	}
}
