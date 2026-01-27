package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"uaa/vdnd/internal/state"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/dice"
)

type SpellDefinition struct {
	Name       string
	Type       string // "attack", "save", "auto-hit", "utility"
	SaveType   string // "reflex", "fortitude", "will"
	BasicSave  bool
	Damage     string
	DamageType string
	Area       int
}

var builtInSpells = map[string]SpellDefinition{
	"fireball": {
		Name:       "fireball",
		Type:       "save",
		SaveType:   "reflex",
		BasicSave:  true,
		Damage:     "6d6",
		DamageType: "fire",
		Area:       20,
	},
	"magic_missile": {
		Name:       "magic_missile",
		Type:       "auto-hit",
		Damage:     "1d4+1",
		DamageType: "force",
	},
	"heal": {
		Name:       "heal",
		Type:       "utility",
		Damage:     "1d8+8",
		DamageType: "healing",
	},
	"chilling_darkness": {
		Name:       "chilling_darkness",
		Type:       "attack",
		Damage:     "5d6",
		DamageType: "cold",
	},
}

func cmdActionCast(args []string, deps Deps) (string, error) {
	positional, flags := ParseFlags(args)
	if len(positional) < 2 {
		return "", fmt.Errorf("usage: vd action cast <actor_id> <spell_name> [flags]")
	}
	actorID := positional[0]
	spellName := strings.ToLower(positional[1])

	gameState, err := deps.Store.Load()
	if err != nil {
		return "", err
	}

	actor, ok := gameState.Entities[actorID]
	if !ok {
		return "", fmt.Errorf("actor not found: %s", actorID)
	}

	// 1. Get Spell Definition
	spell, ok := builtInSpells[spellName]
	if !ok {
		// Generic spell if not built-in
		spell = SpellDefinition{Name: spellName}
	}

	// Override with flags
	if val, ok := flags["type"]; ok {
		spell.Type = val
	}
	if val, ok := flags["save"]; ok {
		spell.SaveType = val
	}
	if _, ok := flags["basic_save"]; ok {
		spell.BasicSave = true
	}
	if val, ok := flags["damage"]; ok {
		spell.Damage = val
	}
	if val, ok := flags["dmg_type"]; ok {
		spell.DamageType = val
	}

	dc := 0
	if val, ok := flags["dc"]; ok {
		dc, _ = strconv.Atoi(val)
	}

	attackMod := 0
	if val, ok := flags["attack_mod"]; ok {
		attackMod, _ = strconv.Atoi(val)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s** casts **%s**!\n\n", actor.Name, spell.Name))

	// 2. Resolve Targets
	var targets []*state.EntityState
	targetID := flags["target"]
	zoneName := flags["zone"]

	if targetID != "" {
		if t, ok := gameState.Entities[targetID]; ok {
			targets = append(targets, t)
		} else {
			return "", fmt.Errorf("target not found: %s", targetID)
		}
	} else if zoneName != "" {
		for _, e := range gameState.Entities {
			if e.Position == zoneName {
				targets = append(targets, e)
			}
		}
		// Sort targets by ID for deterministic output
		sort.Slice(targets, func(i, j int) bool {
			return targets[i].ID < targets[j].ID
		})
	}

	if len(targets) == 0 && spell.Type != "utility" {
		return "", fmt.Errorf("no targets specified or found")
	}

	// 3. Resolve Mechanics
	switch spell.Type {
	case "attack":
		if len(targets) == 0 {
			return "", fmt.Errorf("attack spells require a target")
		}
		target := targets[0]
		naturalRoll := deps.Roller.Roll(1, 20)[0]
		res := check.PerformCheckWithRoll(naturalRoll, attackMod, nil, target.GetAC())

		sb.WriteString(fmt.Sprintf("- **Spell Attack Roll:** %d + %d = **%d** vs AC %d\n", res.NaturalRoll, res.Modifiers, res.Total, res.DC))
		sb.WriteString(fmt.Sprintf("- **Result:** %s\n", res.Degree.String()))

		if res.Degree == check.Success || res.Degree == check.CriticalSuccess {
			dmg := rollSpellDamage(spell, deps, res.Degree == check.CriticalSuccess)
			applySpellDamage(target, dmg, spell.DamageType, &sb)
		}

	case "save":
		if dc == 0 {
			return "", fmt.Errorf("--dc is required for save spells")
		}
		// Roll damage once for all targets in AoE (standard PF2E)
		baseDmg := rollSpellDamage(spell, deps, false)
		
		for _, target := range targets {
			saveMod := 0
			switch strings.ToLower(spell.SaveType) {
			case "reflex":
				saveMod = target.Reflex
			case "fortitude":
				saveMod = target.Fortitude
			case "will":
				saveMod = target.Will
			}

			naturalRoll := deps.Roller.Roll(1, 20)[0]
			res := check.PerformCheckWithRoll(naturalRoll, saveMod, nil, dc)
			
			sb.WriteString(fmt.Sprintf("### %s (%s)\n", target.Name, target.ID))
			sb.WriteString(fmt.Sprintf("- **%s Save:** %d + %d = **%d** vs DC %d\n", spell.SaveType, res.NaturalRoll, res.Modifiers, res.Total, res.DC))
			sb.WriteString(fmt.Sprintf("- **Result:** %s\n", res.Degree.String()))

			finalDmg := baseDmg
			if spell.BasicSave {
				switch res.Degree {
				case check.CriticalSuccess:
					finalDmg = 0
				case check.Success:
					finalDmg /= 2
				case check.Failure:
					// Full damage
				case check.CriticalFailure:
					finalDmg *= 2
				}
			} else {
				// Non-basic saves usually have specific effects per degree.
				// For MVP, if not basic, Success/CritSuccess = 0, Failure = Full, CritFail = Double?
				// Actually, many spells just do nothing on success.
				if res.Degree >= check.Success {
					finalDmg = 0
				} else if res.Degree == check.CriticalFailure {
					finalDmg *= 2
				}
			}
			
			if finalDmg > 0 {
				applySpellDamage(target, finalDmg, spell.DamageType, &sb)
			} else {
				sb.WriteString("- No damage taken.\n")
			}
		}

	case "auto-hit":
		dmg := rollSpellDamage(spell, deps, false)
		for _, target := range targets {
			applySpellDamage(target, dmg, spell.DamageType, &sb)
		}

	case "utility":
		if spell.DamageType == "healing" {
			dmg := rollSpellDamage(spell, deps, false)
			for _, target := range targets {
				target.HP += dmg
				if target.HP > target.MaxHP {
					target.HP = target.MaxHP
				}
				sb.WriteString(fmt.Sprintf("- **%s** healed for **%d**. HP: %d/%d\n", target.Name, dmg, target.HP, target.MaxHP))
			}
		} else {
			sb.WriteString("- Spell cast successfully (No automated effect).\n")
		}
	}

	if err := deps.Store.Save(gameState); err != nil {
		return "", err
	}

	return sb.String(), nil
}

func rollSpellDamage(spell SpellDefinition, deps Deps, isCrit bool) int {
	d, err := dice.Parse(spell.Damage)
	if err != nil {
		return 0
	}
	results := deps.Roller.Roll(d.Count, d.Sides)
	total := 0
	for _, r := range results {
		total += r
	}
	total += d.Modifier
	if isCrit {
		total *= 2
	}
	return total
}

func applySpellDamage(target *state.EntityState, amount int, dmgType string, sb *strings.Builder) {
	// Apply Immunities/Weaknesses/Resistances (simplified version of cmdDamage logic)
	for _, imm := range target.Immunities {
		if strings.EqualFold(imm, dmgType) {
			sb.WriteString(fmt.Sprintf("- Immune to **%s**!\n", dmgType))
			return
		}
	}
	
	if val, ok := target.Weaknesses[dmgType]; ok {
		amount += val
		sb.WriteString(fmt.Sprintf("- Weakness to **%s**: +%d damage\n", dmgType, val))
	}
	if val, ok := target.Resistances[dmgType]; ok {
		amount -= val
		if amount < 0 { amount = 0 }
		sb.WriteString(fmt.Sprintf("- Resistance to **%s**: -%d damage\n", dmgType, val))
	}

	if amount > 0 {
		target.HP -= amount
		if target.HP < 0 { target.HP = 0 }
		sb.WriteString(fmt.Sprintf("- **Damage:** **%d** %s. HP: %d/%d\n", amount, dmgType, target.HP, target.MaxHP))
	} else {
		sb.WriteString("- No damage taken.\n")
	}
}
