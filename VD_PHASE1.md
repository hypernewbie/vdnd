# VD Phase 1: Core CLI Skeleton

> **Status:** Partially implemented by Gemini - requires gap work

This document defines Phase 1 requirements, compares against current implementation, and lists remaining work.

---

## Original Design (from VD_PLAN.md)

### Required Files

| File | Purpose |
|------|---------|
| `cmd/vd/main.go` | Thin wrapper calling `cli.Run()` |
| `internal/cli/cli.go` | `Run()` function, command router |
| `internal/cli/cli_test.go` | Unit tests for `Run()` |
| `internal/cli/deps.go` | Dependencies struct + interfaces |
| `internal/cli/deps_test.go` | Test doubles (MockRoller, etc.) |
| `internal/cli/flags.go` | Flag parsing utilities |

### Required Interfaces

**Deps struct:**
```go
type Deps struct {
    Roller     Roller
    Store      StateStore
    Clock      Clock
    Stderr     io.Writer
    Cwd        string
}
```

**Roller interface:**
```go
type Roller interface {
    Roll(count, sides int) []int
}
```

With implementations: `CryptoRoller`, `FixedRoller`, `SeededRoller`

**Clock interface:**
```go
type Clock interface {
    Now() time.Time
}
```

With implementations: `RealClock`, `FixedClock`

### Required Functionality

- `Run(args []string, deps Deps) (stdout string, exitCode int)`
- Command routing for 1-word and 2-word commands
- `help` command
- `DefaultDeps()` factory function
- Basic tests covering: no args, help, unknown command

---

## Current Implementation (by Gemini)

### Files Present

| File | Status | Notes |
|------|--------|-------|
| `cmd/vd/main.go` | ✅ Present | Correct pattern |
| `internal/cli/cli.go` | ✅ Present | Has `Run()`, router |
| `internal/cli/cli_test.go` | ✅ Present | Basic tests |
| `internal/cli/deps.go` | ✅ Present | Has interfaces |
| `internal/cli/deps_test.go` | ❌ Missing | No dedicated test doubles file |
| `internal/cli/flags.go` | ✅ Present | Simple parser |

### Interface Coverage

| Interface | Status | Notes |
|-----------|--------|-------|
| `Deps` struct | ✅ Complete | All fields present |
| `Roller` interface | ✅ Complete | Has `Roll(count, sides int) []int` |
| `CryptoRoller` | ✅ Present | Uses crypto/rand |
| `FixedRoller` | ✅ Present | For deterministic tests |
| `SeededRoller` | ❌ Missing | Plan specified this, not implemented |
| `Clock` interface | ✅ Complete | Has `Now() time.Time` |
| `RealClock` | ✅ Present | |
| `FixedClock` | ✅ Present | |

### Test Coverage

| Test | Status | Notes |
|------|--------|-------|
| `TestRun_Basic` | ✅ Present | Covers no args, help, unknown |
| `TestParseFlags` | ✅ Present | Flag extraction |

---

## Gap Analysis

### Missing Components

| Component | Severity | Action |
|-----------|----------|--------|
| `deps_test.go` | Low | Create file with test helper functions |
| `SeededRoller` | Low | Add to deps.go (useful for reproducible scenarios) |

### Incorrect/Suboptimal

| Issue | Severity | Action |
|-------|----------|--------|
| Help text is minimal | Low | Expand to list all commands |
| No command-specific help | Medium | Add `vd help <command>` support |

### Missing Tests

| Test | Priority |
|------|----------|
| Test for `DefaultDeps()` returns valid deps | Low |
| Test `FixedRoller` panics when exhausted | Low |

---

## Remaining Work

### Priority 1 (Required for Phase 1 Complete)

1. **Add `SeededRoller`** - Needed for reproducible scenario tests
   ```go
   type SeededRoller struct {
       rng *rand.Rand
   }
   
   func NewSeededRoller(seed int64) *SeededRoller {
       return &SeededRoller{rng: rand.New(rand.NewSource(seed))}
   }
   
   func (r *SeededRoller) Roll(count, sides int) []int {
       results := make([]int, count)
       for i := range results {
           results[i] = r.rng.Intn(sides) + 1
       }
       return results
   }
   ```

2. **Create `deps_test.go`** with helper functions:
   ```go
   func TestDeps(t *testing.T) Deps {
       return Deps{
           Roller: &FixedRoller{},
           Store:  &state.MemoryStore{},
           Clock:  &FixedClock{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
           Stderr: io.Discard,
           Cwd:    t.TempDir(),
       }
   }
   ```

### Priority 2 (Nice to Have)

3. **Expand help text** to list all available commands
4. **Add `vd help <command>`** for command-specific help

---

## Verification

Run tests:
```bash
go test ./internal/cli/... -v
```

Expected: All pass, including any new tests added.
