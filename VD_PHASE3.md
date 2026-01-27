# Phase 3: Entity Commands

## Goal
Implement the core entity management commands for the `vd` CLI. These commands allow adding, retrieving, modifying, and listing entities within the game state.

## Implementation Steps

1.  **Entity Parser Integration**
    *   Ensure `internal/parser/entity.go` is fully integrated and can parse the markdown format used for entities.
    *   Verify `EntityState` conversion from parsed data.

2.  **Command Handlers (`internal/cli/cmd_entity.go`)**
    *   `cmdEntityAdd`: Reads a markdown file, parses it, creates an `EntityState`, and adds it to `GameState.Entities`.
    *   `cmdEntitySpawn`: Creates N entities from a bestiary template (simplified version of `add` for generic mobs).
    *   `cmdEntityGet`: Retrieves an entity by ID and prints its details (or a specific field).
    *   `cmdEntitySet`: Updates a specific field on an entity (e.g., HP, position).
    *   `cmdEntityList`: Iterates through `GameState.Entities` and prints a summary table. Supports filtering by zone.

3.  **CLI Routing**
    *   Register these commands in `internal/cli/cli.go`.

## Command Reference

| Command | Args | Description |
|---------|------|-------------|
| `vd entity add <id>` | `--file <path>` | Add entity from markdown file |
| `vd entity spawn <template>` | `--count N --prefix str` | Spawn N entities from bestiary template |
| `vd entity get <id>` | `--field <name>` | Get entity stats (optionally single field) |
| `vd entity set <id> <field> <value>` | | Set entity field directly |
| `vd entity list` | `--zone <name>` | List all entities (optionally filtered by zone) |

## Testing Plan
*   **Unit Tests (`internal/cli/cmd_entity_test.go`)**:
    *   Test adding a valid entity file.
    *   Test adding a duplicate ID (should fail or overwrite depending on design, likely fail).
    *   Test getting non-existent entity.
    *   Test setting fields (int, string types).
    *   Test listing with and without filters.
