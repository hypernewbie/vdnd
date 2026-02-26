# VD CLI Manual

The `vd` (Virtual Dungeon) CLI is a stateless tool for managing Pathfinder 2E encounters via the command line. It is designed to be driven by an LLM Agent but is human-readable.

## Core Concepts

- **Stateless:** Every command loads the current state (default: `state.json`), performs an operation, and saves it back.
- **Deterministic:** Dice rolls are handled by an injectable provider (seeded or random).
- **Entities:** Characters and monsters are referenced by unique IDs (e.g., `hero`, `goblin_1`).
- **Zones:** The map is abstract, defined by named zones and their connections, not grid coordinates.

## Installation

```bash
go install ./cmd/vd
```

## Usage

```bash
vd <command> [subcommand] [flags]
```

---

## Command Reference

### 1. Scene Management
Create and persist game states.

- **`vd scene new <name>`**
  Creates a new empty scene/session. Resets all state.
  *Example:* `vd scene new dungeon_level_1`

- **`vd scene save <path>`**
  Saves the current state to a specific JSON file.
  *Example:* `vd scene save checkpoints/before_boss.json`

- **`vd scene load <path>`**
  Loads a state from a JSON file.
  *Example:* `vd scene load checkpoints/before_boss.json`

- **`vd status`**
  Shows a high-level overview of the current scene and entities.

### 2. Entity Management
Manage characters, monsters, and objects.

#### Entity File Format
The `--file` flag expects a simple Markdown file defining the entity.
**Example (`goblin.md`):**
```markdown
# Goblin Warrior
- Level: 1
- HP: 15/15
- AC: 16
- Speed: 25ft
- Fort: +5
- Ref: +7
- Will: +4
- Perception: +5
- Strength: 2
- Dexterity: 4
- Constitution: 2
- Intelligence: -1
- Wisdom: 0
- Charisma: 0
- Acrobatics: +5
- Stealth: +5
```

- **`vd entity add <id> --file <path>`**
  Adds a single entity defined in a markdown file.
  *Example:* `vd entity add valeros --file sheets/valeros.md`

- **`vd entity spawn <path> --count <N> --prefix <str>`**
  Spawns multiple copies of a monster template.
  *Example:* `vd entity spawn bestiary/goblin.md --count 3 --prefix gob`
  *Result:* Adds `gob_1`, `gob_2`, `gob_3`.

- **`vd entity list`**
  Lists all entities in the scene with basic stats (HP, AC, Init).
  *Flags:* `--zone <name>` (Filter by zone).

- **`vd entity get <id>`**
  Displays full details of an entity (Stats, Equipment, Conditions).
  *Flags:* `--field <name>` (Extract single value, e.g., "hp").

- **`vd entity set <id> <field> <value>`**
  Manually overrides a stat. Use with caution.
  *Example:* `vd entity set hero hp 50`

### 3. Basic Actions
Perform standard combat actions.

- **`vd action strike <actor> <target>`**
  Performs a melee or ranged attack. Handles hit/crit/miss and applies damage.
  *Flags:*
  - `--weapon <id>`: Specific weapon to use (defaults to first wielded).
  - `--map <0|1|2>`: Multi-Attack Penalty (0=None, 1=-5, 2=-10).

- **`vd action stride <actor> --to <zone>`**
  Moves an entity to an adjacent zone. **May trigger reactions.**
  *Example:* `vd action stride hero --to hallway`

- **`vd action step <actor> --to <zone>`**
  Moves 5ft (staying in zone or very close). Does **not** trigger reactions.

- **`vd action raise_shield <actor>`**
  Applies the `Screened` or equivalent AC bonus condition for 1 round.

- **`vd action cast <actor> <spell> [flags]`**
  Casts a spell.
  *Flags:*
  - `--target <id>`: Single target.
  - `--zone <name>`: Area of effect.
  - `--dc <N>`: Save DC (if saving throw).
  - `--save <type>`: reflex/fort/will.
  - `--damage <expr>`: e.g., "6d6".
  - `--type <str>`: e.g., "fire".
  *Example:* `vd action cast wizard fireball --zone room_a --dc 22 --damage 6d6 --type fire --basic_save`

### 4. Skill Actions
Perform specialized maneuvers and checks.

- **`vd action grapple <actor> <target>`**
  Athletics check vs Fortitude DC. Success applies `Grabbed`.

- **`vd action trip <actor> <target>`**
  Athletics check vs Reflex DC. Success applies `Prone`.

- **`vd action shove <actor> <target>`**
  Athletics check vs Fortitude DC. Success pushes target.

- **`vd action demoralize <actor> <target>`**
  Intimidation check vs Will DC. Success applies `Frightened`.

- **`vd action hide <actor>`**
  Stealth check vs Perception DC (Passive). Success applies `Hidden`.

- **`vd action seek <actor> <target>`**
  Perception check vs Stealth DC. Success removes `Hidden` condition.

### 5. Reactions & Interrupts
Handle events that pause the game flow (like Attack of Opportunity).

- **`vd pending`**
  Shows the current event waiting for resolution (e.g., "Goblin moving out of square").

- **`vd react <id> <reaction>`**
  Executes a reaction for the specified entity against the pending event.
  *Example:* `vd react fighter attack_of_opportunity`
  *Note:* If the reaction disrupts the trigger (e.g., Crit hit on Move), the pending action is cancelled.

- **`vd react skip [id]`**
  Declines the reaction opportunity for a specific entity or the current turn.

- **`vd react skip_all`**
  Declines reactions for *everyone* and allows the pending action to proceed immediately.

### 6. Conditions
Manage status effects.

- **`vd condition add <id> <condition> [value]`**
  Applies a condition.
  *Flags:* `--duration <rounds>`, `--source <str>`.
  *Example:* `vd condition add hero frightened 1`

- **`vd condition remove <id> <condition>`**
  Removes a condition completely.

- **`vd condition reduce <id> <condition> [amount]`**
  Lowers the value of a condition (e.g., end of turn decay).
  *Example:* `vd condition reduce hero frightened 1`

- **`vd condition list <id>`**
  Lists all active conditions on an entity.

### 7. Damage & Healing
Direct HP manipulation.

- **`vd damage <id> <amount> [type]`**
  Applies damage, automatically calculating **Immunities, Weaknesses, and Resistances**.
  *Example:* `vd damage skeleton 10 bludgeoning` (Trigger weakness!)

- **`vd heal <id> <amount>`**
  Restores HP, capped at MaxHP. Handles `Dying`/`Wounded` logic automatically.

- **`vd temp_hp <id> <amount>`**
  Sets temporary hit points (non-stacking).

### 8. Queries
Ask the system about the battlefield state.

- **`vd query distance <id1> <id2>`**
  Returns abstract distance (Melee, 1 Zone, 2 Zones...).

- **`vd query targets <id>`**
  Lists valid enemies within range/sight.
  *Flags:* `--range <ft>`, `--melee`.

- **`vd query flanking <id1> <id2>`**
  Checks if `id2` is flanked by `id1` and an ally.

- **`vd query cover <id1> <id2>`**
  Checks if `id2` has cover from `id1`.

### 9. Generic Rolls
Ad-hoc resolution.

- **`vd roll <expression>`**
  Rolls dice. Supports multiple groups, flat modifiers, and shorthand (e.g., `d20`).
  *Examples:* `vd roll d20+7`, `vd roll 2d8+1d6+4`, `vd roll +5`

- **`vd check <id> <skill>`**
  Performs a skill check for an entity (adding modifiers).
  *Flags:* `--dc <N>` (returns Success/Failure result).
  *Example:* `vd check rogue thievery --dc 20`
