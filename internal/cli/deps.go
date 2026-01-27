package cli

import (
	"crypto/rand"
	"io"
	"math/big"
	mrand "math/rand"
	"os"
	"time"
	"uaa/vdnd/internal/state"
)

// Deps holds all external dependencies for the CLI.
// In production, use DefaultDeps(). In tests, inject mocks.
type Deps struct {
	Roller Roller
	Store  state.Store
	Clock  Clock
	Stderr io.Writer
	Cwd    string
}

// DefaultDeps returns production dependencies.
func DefaultDeps() Deps {
	cwd, _ := os.Getwd()
	return Deps{
		Roller: &CryptoRoller{},
		Store:  &state.FileStore{Root: cwd},
		Clock:  &RealClock{},
		Stderr: os.Stderr,
		Cwd:    cwd,
	}
}

// Roller abstracts dice rolling for testability.
type Roller interface {
	// Roll returns `count` individual die results, each 1 to `sides` inclusive.
	Roll(count, sides int) []int
}

// CryptoRoller uses crypto/rand for production.
type CryptoRoller struct{}

func (r *CryptoRoller) Roll(count, sides int) []int {
	results := make([]int, count)
	for i := range results {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(sides)))
		results[i] = int(n.Int64()) + 1
	}
	return results
}

// FixedRoller returns predetermined values. Panics if exhausted.
type FixedRoller struct {
	Results []int
	Index   int
}

func (r *FixedRoller) Roll(count, sides int) []int {
	if r.Index+count > len(r.Results) {
		panic("FixedRoller exhausted")
	}
	out := r.Results[r.Index : r.Index+count]
	r.Index += count
	return out
}

// SeededRoller uses a seeded PRNG for reproducible scenarios.
type SeededRoller struct {
	rng *mrand.Rand
}

func NewSeededRoller(seed int64) *SeededRoller {
	return &SeededRoller{rng: mrand.New(mrand.NewSource(seed))}
}

func (r *SeededRoller) Roll(count, sides int) []int {
	results := make([]int, count)
	for i := range results {
		results[i] = r.rng.Intn(sides) + 1
	}
	return results
}

// Clock abstracts time for testing time-based effects.
type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (c *RealClock) Now() time.Time { return time.Now() }

type FixedClock struct{ Time time.Time }

func (c *FixedClock) Now() time.Time { return c.Time }
