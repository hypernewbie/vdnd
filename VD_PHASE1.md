# VD Phase 1: Core CLI Skeleton

> **Status:** ✅ Implemented (by Gemini 3 Pro, who jumped the gun)

This phase establishes the testable CLI architecture with dependency injection.

---

## What Was Built

### File Structure

```
cmd/vd/
└── main.go              # Thin entry point

internal/cli/
├── cli.go               # Run() function, command router
├── cli_test.go          # Basic Run() tests
├── deps.go              # Dependencies struct, interfaces
├── flags.go             # Simple --flag parser
├── cmd_scene.go         # Scene commands (scene new/save/load)
└── cmd_scene_test.go    # Scene command tests
```

---

## Core Components

### The `Run()` Function

The central entry point that all tests can call directly:

```go
func Run(args []string, deps Deps) (stdout string, exitCode int)
```

- Takes CLI arguments as a slice (no `os.Args` coupling)
- Takes dependencies struct (no globals)
- Returns output string and exit code (no `os.Exit` coupling)
- Testable with any combination of args and mock deps

### Command Routing

Two-level command parsing supports both single-word (`help`) and two-word (`scene new`) commands:

```go
var commands = map[string]CommandHandler{
    "help":       cmdHelp,
    "scene new":  cmdSceneNew,
    "scene save": cmdSceneSave,
    "scene load": cmdSceneLoad,
}
```

The router checks for 2-word commands first, then falls back to 1-word.

### Dependency Injection

All external dependencies injected via `Deps` struct:

```go
type Deps struct {
    Roller Roller       // Dice rolling (injectable for deterministic tests)
    Store  state.Store  // State persistence (file or memory)
    Clock  Clock        // Time (injectable for time-based effects)
    Stderr io.Writer    // Error output
    Cwd    string       // Working directory
}
```

Production defaults via `DefaultDeps()`, tests inject mocks.

---

## Interfaces

### Roller (Dice)

```go
type Roller interface {
    Roll(count, sides int) []int
}
```

Implementations:
- `CryptoRoller` - Production, uses crypto/rand
- `FixedRoller` - Tests, returns predetermined values

### Clock

```go
type Clock interface {
    Now() time.Time
}
```

Implementations:
- `RealClock` - Production
- `FixedClock` - Tests, returns fixed time

---

## Flag Parsing

Simple `--key value` parser, no external dependencies:

```go
func ParseFlags(args []string) (positional []string, flags map[string]string)
```

Handles:
- `--key value` pairs
- `--boolFlag` (value becomes "true")
- Preserves positional arguments

---

## Tests

### `TestRun_Basic`

Table-driven test covering:
- No args → shows help
- `help` command → shows help
- Unknown command → returns error

### `TestParseFlags`

Verifies flag extraction works correctly.

---

## Usage

```bash
# Build
go build -o vd.exe ./cmd/vd

# Run
./vd help
./vd scene new "My Campaign"
```

---

## What's NOT Done Yet

- Most command handlers (entity, combat, action, etc.) - just stubs in the registry
- Output formatting helpers (`internal/output/`)
- Scenario test infrastructure

---

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| No Cobra | LLM consumer doesn't need shell features, simpler testing |
| Hand-rolled router | Full control, trivial to test |
| `Deps` struct | Single point of injection, easy to mock |
| `crypto/rand` roller | Proper randomness for production |
