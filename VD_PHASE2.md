# VD Phase 2: State Management

> **Status:** Partially implemented by Gemini - requires gap work

This document defines Phase 2 requirements, compares against current implementation, and lists remaining work.

---

## Original Design (from VD_PLAN.md)

### Required Files

| File | Purpose |
|------|---------|
| `internal/state/state.go` | GameState struct and related types |
| `internal/state/store.go` | Store interface |
| `internal/state/file_store.go` | File-based persistence |
| `internal/state/memory_store.go` | In-memory for tests |
| `internal/output/formatter.go` | Markdown output builder |
| `internal/output/templates.go` | Common output patterns |

### Required State Structures

**GameState** (from VD_PLAN lines 872-888):
```go
type GameState struct {
    SceneName        string
    Positions        map[string]*Zone
    Entities         map[string]*EntityState
    InCombat         bool
    Round            int
    InitiativeOrder  []string
    CurrentTurn      string
    TurnIndex        int
    ActionsRemaining int
    ReactionsUsed    map[string]bool
    AttacksMade      int
    PendingEvents    []PendingEvent
}
```

**Store interface** (from VD_PLAN lines 188-204):
```go
type StateStore interface {
    Load() (*GameState, error)
    Save(state *GameState) error
    Exists() bool
}
```

### Required Commands

| Command | Behaviour |
|---------|-----------|
| `scene new <name>` | Create new session, error if exists |
| `scene save` | Explicit checkpoint |
| `scene load <name>` | Load from template directory |

### Required Output Formatting

The plan specifies an `internal/output/` package with:
- `Output` struct with fluent builder API
- Methods: `Header()`, `Field()`, `Section()`, `ListItem()`, `Table()`, `Result()`

---

## Current Implementation (by Gemini)

### Files Present

| File | Status | Notes |
|------|--------|-------|
| `internal/state/state.go` | ✅ Present | Has GameState, all types |
| `internal/state/store.go` | ⚠️ Combined | Has Store + FileStore + MemoryStore in one file |
| `internal/state/file_store.go` | ❌ Missing | Combined into store.go |
| `internal/state/memory_store.go` | ❌ Missing | Combined into store.go |
| `internal/output/formatter.go` | ❌ Missing | Not implemented |
| `internal/output/templates.go` | ❌ Missing | Not implemented |
| `internal/cli/cmd_scene.go` | ✅ Present | Scene commands |
| `internal/cli/cmd_scene_test.go` | ✅ Present | Scene tests |

### State Structure Coverage

| Struct/Type | Status | Notes |
|-------------|--------|-------|
| `GameState` | ✅ Complete | All fields present |
| `Zone` | ✅ Complete | Has size, adjacent, near, far, cover, elevated |
| `EntityState` | ✅ Complete | Has all fields including abilities via pkg/rules |
| `ConditionInstance` | ✅ Present | ID, Value, Duration, Source |
| `WeaponState` | ✅ Present | Simplified for state |
| `ArmorState` | ✅ Present | Simplified for state |
| `PendingEvent` | ✅ Present | For reaction system |
| `AvailableReaction` | ✅ Present | Entity, Reaction, Trigger |

### Store Implementation

| Component | Status | Notes |
|-----------|--------|-------|
| `Store` interface | ✅ Present | Has Load/Save/Exists |
| `FileStore` | ✅ Present | Saves to state.json |
| `MemoryStore` | ✅ Present | For tests |

### Command Coverage

| Command | Status | Notes |
|---------|--------|-------|
| `scene new` | ✅ Works | Creates state, errors if exists |
| `scene save` | ⚠️ Stub | Returns static message, doesn't actually re-save |
| `scene load` | ⚠️ Stub | Returns "not implemented" |

---

## Gap Analysis

### Missing Components

| Component | Severity | Action |
|-----------|----------|--------|
| `internal/output/formatter.go` | **High** | Create output builder for structured markdown |
| `internal/output/templates.go` | Medium | Create common output patterns |
| Separate store files | Low | Optional - current combined file is fine |

### Incorrect/Suboptimal

| Issue | Severity | Action |
|-------|----------|--------|
| `scene save` is a no-op | Medium | Should load current state and re-save (acts as checkpoint) |
| `scene load` not implemented | Medium | Should load from template directories |
| No state validation | Low | Consider adding validation on Load() |

### Missing Tests

| Test | Priority |
|------|----------|
| `FileStore` integration test (actual file I/O) | Medium |
| `GameState` JSON round-trip test | Medium |
| `scene save` actually writes | Medium |

---

## Remaining Work

### Priority 1 (Required for Phase 2 Complete)

1. **Create `internal/output/formatter.go`**
   
   This is critical - all command output should use consistent formatting:
   ```go
   package output
   
   import "strings"
   
   type Output struct {
       buf strings.Builder
   }
   
   func New() *Output { return &Output{} }
   
   func (o *Output) Header(level int, text string) *Output {
       o.buf.WriteString(strings.Repeat("#", level) + " " + text + "\n")
       return o
   }
   
   func (o *Output) Field(label, value string) *Output {
       o.buf.WriteString("**" + label + ":** " + value + "\n")
       return o
   }
   
   func (o *Output) Section(title string) *Output {
       o.buf.WriteString("\n## " + title + "\n")
       return o
   }
   
   func (o *Output) ListItem(text string) *Output {
       o.buf.WriteString("- " + text + "\n")
       return o
   }
   
   func (o *Output) Table(headers []string, rows [][]string) *Output {
       o.buf.WriteString("| " + strings.Join(headers, " | ") + " |\n")
       o.buf.WriteString("|" + strings.Repeat("---|", len(headers)) + "\n")
       for _, row := range rows {
           o.buf.WriteString("| " + strings.Join(row, " | ") + " |\n")
       }
       return o
   }
   
   func (o *Output) Result(label, value string) *Output {
       o.buf.WriteString("- **" + label + ":** " + value + "\n")
       return o
   }
   
   func (o *Output) String() string { return o.buf.String() }
   ```

2. **Fix `scene save`** to actually reload and save state:
   ```go
   func cmdSceneSave(args []string, deps Deps) (string, error) {
       state, err := deps.Store.Load()
       if err != nil {
           return "", fmt.Errorf("no active session: %w", err)
       }
       if err := deps.Store.Save(state); err != nil {
           return "", fmt.Errorf("failed to save: %w", err)
       }
       return "Scene saved.", nil
   }
   ```

### Priority 2 (Should Have)

3. **Implement `scene load`** for template directories
4. **Add state validation** on Load()
5. **Split store.go** into separate files (optional, for cleanliness)

### Priority 3 (Nice to Have)

6. **Output templates** for common patterns (action results, status, etc.)
7. **FileStore** integration tests

---

## Verification

Run tests:
```bash
go test ./internal/state/... ./internal/cli/... -v
```

Manual verification:
```bash
cd /tmp/test_session
vd scene new "Test Campaign"
cat state.json  # Should have valid JSON
vd scene save
```

---

## Design Notes

### Why Output Formatting Matters

The LLM orchestrator parses the CLI output. Consistent markdown structure makes parsing reliable:
- `# HEADER` indicates action type
- `**Field:** value` for key-value data
- `## Section` for grouping
- Tables for lists of options/targets

Without the output package, each command will have inconsistent formatting, making the system harder to use.
