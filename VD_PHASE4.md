# Phase 4: Combat Commands

## Goal
Implement the combat loop management commands. This includes entering/exiting combat, rolling initiative, and managing the turn order.

## Implementation Steps

1.  **Combat State Logic**
    *   Ensure `GameState` has necessary fields: `InCombat`, `Round`, `InitiativeOrder`, `CurrentTurn`, `TurnIndex`.

2.  **Command Handlers (`internal/cli/cmd_combat.go`)**
    *   `cmdCombatStart`: Sets `InCombat = true`, resets round/turn counters.
    *   `cmdCombatEnd`: Sets `InCombat = false`, clears initiative.
    *   `cmdCombatInitiative`:
        *   Iterates all entities in the scene.
        *   Rolls Perception (or specified skill) for each.
        *   Sorts by result (descending).
        *   Populates `InitiativeOrder` and sets `CurrentTurn` to the first entity.
    *   `cmdCombatNext`:
        *   Increments `TurnIndex`.
        *   If end of order, increments `Round` and resets `TurnIndex`.
        *   Updates `CurrentTurn`.
        *   *Future hook:* Trigger start-of-turn effects (tick conditions).
    *   `cmdCombatStatus`: Displays current round, active turn, and full initiative list.

3.  **CLI Routing**
    *   Register these commands in `internal/cli/cli.go`.

## Command Reference

| Command | Args | Description |
|---------|------|-------------|
| `vd combat start` | | Enter encounter mode |
| `vd combat end` | | Exit encounter mode |
| `vd combat initiative` | `--advantage <id>` | Roll initiative for all, optionally give advantage |
| `vd combat next` | | Advance to next turn |
| `vd combat status` | | Show initiative order, current turn, round |

## Testing Plan
*   **Unit Tests (`internal/cli/cmd_combat_test.go`)**:
    *   Test `combat start` initializes state correctly.
    *   Test `initiative` sorting logic (mock the Roller to ensure specific order).
    *   Test `next` wrapping around to round 2.
    *   Test `next` updates `CurrentTurn` correctly.
