package cli

import (
	"fmt"
	"strings"
)

// CommandHandler func(args []string, deps Deps) (string, error)
type CommandHandler func(args []string, deps Deps) (string, error)

var commands = map[string]CommandHandler{
	"help":               cmdHelp,
	"scene new":          cmdSceneNew,
	"scene save":         cmdSceneSave,
	"scene load":         cmdSceneLoad,
	"status":             cmdStatus,
	"entity add":         cmdEntityAdd,
	"entity edit":        cmdEntityEdit,
	"entity get":         cmdEntityGet,
	"entity set":         cmdEntityEdit, // Alias set to edit
	"entity list":        cmdEntityList,
	"entity spawn":       cmdEntitySpawn,
	"action strike":      cmdActionStrike,
	"action stride":      cmdActionStride,
	"action step":        cmdActionStep,
	"action raise_shield": cmdActionRaiseShield,
	"action cast":         cmdActionCast,
	"pending":            cmdPending,
	"react":              cmdReact,
	"react skip":         cmdReactSkip,
	"react skip_all":     cmdReactSkipAll,
	"condition add":      cmdConditionAdd,
	"condition remove":   cmdConditionRemove,
	"condition reduce":   cmdConditionReduce,
	"condition list":     cmdConditionList,
	"damage":             cmdDamage,
	"heal":               cmdHeal,
	"temp_hp":            cmdTempHP,
	"query distance":     cmdQueryDistance,
	"query targets":      cmdQueryTargets,
	"query flanking":     cmdQueryFlanking,
	"query cover":        cmdQueryCover,
	"roll":               cmdRoll,
	"check":              cmdCheck,
	"action grapple":     cmdActionGrapple,
	"action trip":        cmdActionTrip,
	"action shove":       cmdActionShove,
	"action demoralize":  cmdActionDemoralize,
	"action hide":        cmdActionHide,
	"action seek":        cmdActionSeek,
}

// Run is the main entry point. Takes CLI args and dependencies, returns output and exit code.
// This is the function that main.go calls and tests exercise.
func Run(args []string, deps Deps) (stdout string, exitCode int) {
	if len(args) == 0 {
		return helpText(), 0
	}

	// Build command key from first 1-2 args
	cmd, cmdArgs := parseCommand(args)

	handler, ok := commands[cmd]
	if !ok {
		return fmt.Sprintf("unknown command: %s\n\nRun 'vd help' for usage.", cmd), 1
	}

	// Execute handler
	result, err := handler(cmdArgs, deps)
	if err != nil {
		fmt.Fprintln(deps.Stderr, err)
		return "", 1
	}

	return result, 0
}

func parseCommand(args []string) (string, []string) {
	if len(args) == 0 {
		return "", nil
	}

	// Check for 2-word command
	if len(args) >= 2 {
		potentialCmd := args[0] + " " + args[1]
		if _, ok := commands[potentialCmd]; ok {
			return potentialCmd, args[2:]
		}
	}

	// Check for 1-word command
	if _, ok := commands[args[0]]; ok {
		return args[0], args[1:]
	}

	// Default to first arg as command key (even if unknown, let map lookup fail)
	return args[0], args[1:]
}

func cmdHelp(args []string, deps Deps) (string, error) {
	return helpText(), nil
}

func cmdStatus(args []string, deps Deps) (string, error) {
	state, err := deps.Store.Load()
	if err != nil {
		return "", WrapSystemError(err, "failed to load state")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Scene: %s\n\n", state.SceneName))

	if len(state.Entities) == 0 {
		sb.WriteString("No entities in scene.\n")
	} else {
		sb.WriteString("## Entities\n")
		for id, e := range state.Entities {
			sb.WriteString(fmt.Sprintf("- **%s** (%s): HP %d/%d\n", id, e.Name, e.HP, e.MaxHP))
		}
	}

	return sb.String(), nil
}

func helpText() string {
	return `vd - Pathfinder 2E CLI

Usage:
  vd <command> [args]

Commands:
  help                    Show this help message
  status                  Show current scene status
  scene new <name>        Create a new scene
  scene save <path>       Save current scene to file
  scene load <path>       Load scene from file
  entity add <id> [--file path] [stats...]  Add entity
  entity edit <id> [stats...]               Update entity stats
  entity get <id>                           Show entity details
  entity list                               List all entities
  entity spawn <path>     Spawn multiple entities from template
  action strike <actor> <target> [--weapon <id>] [--map <0|1|2>]
  action stride <actor> --to <zone>
  action step <actor> --to <zone>
  action raise_shield <actor>
  action cast <actor> <spell> [flags]
  pending                 Show pending events requiring reactions
  react <id> <reaction>   Perform a reaction
  react skip [id]         Skip one or more reactions
  react skip_all          Skip all reactions for current event
  condition add <id> <cond> [val]  Add condition
  condition remove <id> <cond>     Remove condition
  condition reduce <id> <cond> [n] Reduce condition value
  condition list <id>              List entity conditions
  damage <id> <amt> [type]         Apply damage to entity
  heal <id> <amt>                  Heal entity
  temp_hp <id> <amt>               Set temporary HP
  query distance <id1> <id2>       Calculate distance between entities
  query targets <id> [--range ft]  List valid attack targets
  query flanking <id1> <id2>       Check if id1 flanks id2
  query cover <id1> <id2>          Check cover id2 has from id1
  roll <expression>                Roll dice (e.g. 2d6+4)
  check <id> <skill> [--dc N]      Perform a skill check
  action grapple <actor> <target>  Attempt to grapple target
  action trip <actor> <target>     Attempt to trip target
  action shove <actor> <target>    Attempt to shove target
  action demoralize <actor> <target> Attempt to demoralize target
  action hide <actor> [--dc N]     Attempt to hide
  action seek <actor> <target>     Attempt to find hidden target
`
}
