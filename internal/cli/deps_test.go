package cli

import (
	"io"
	"testing"
	"time"
	"uaa/vdnd/internal/state"
)

// NewTestDeps returns test dependencies suitable for unit tests.
func NewTestDeps(t *testing.T) Deps {
	return Deps{
		Roller: &FixedRoller{},
		Store:  &state.MemoryStore{},
		Clock:  &FixedClock{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		Stderr: io.Discard,
		Cwd:    t.TempDir(),
	}
}

// TestDefaultDeps tests that DefaultDeps returns a valid dependencies struct.
func TestDefaultDeps(t *testing.T) {
	deps := DefaultDeps()

	if deps.Roller == nil {
		t.Error("DefaultDeps().Roller should not be nil")
	}
	if deps.Store == nil {
		t.Error("DefaultDeps().Store should not be nil")
	}
	if deps.Clock == nil {
		t.Error("DefaultDeps().Clock should not be nil")
	}
	if deps.Stderr == nil {
		t.Error("DefaultDeps().Stderr should not be nil")
	}
	if deps.Cwd == "" {
		t.Error("DefaultDeps().Cwd should not be empty")
	}
}

// TestFixedRollerExhaustion tests that FixedRoller panics when exhausted.
func TestFixedRollerExhaustion(t *testing.T) {
	roller := &FixedRoller{Results: []int{1, 2}}

	// Should work
	result := roller.Roll(1, 6)
	if len(result) != 1 || result[0] != 1 {
		t.Errorf("First roll failed: got %v", result)
	}

	// Should work
	result = roller.Roll(1, 6)
	if len(result) != 1 || result[0] != 2 {
		t.Errorf("Second roll failed: got %v", result)
	}

	// Should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected FixedRoller to panic when exhausted")
		}
	}()
	roller.Roll(1, 6)
}

// TestSeededRoller tests that SeededRoller produces reproducible results.
func TestSeededRoller(t *testing.T) {
	roller1 := NewSeededRoller(42)
	time.Sleep(time.Millisecond) // Ensure different instantiations don't affect results
	roller2 := NewSeededRoller(42)

	result1 := roller1.Roll(5, 6)
	result2 := roller2.Roll(5, 6)

	if len(result1) != 5 || len(result2) != 5 {
		t.Fatal("Roll results should have 5 elements")
	}

	for i := range result1 {
		if result1[i] != result2[i] {
			t.Errorf("Seeded roller results differ at index %d: %d vs %d", i, result1[i], result2[i])
		}
		if result1[i] < 1 || result1[i] > 6 {
			t.Errorf("Roll result out of range: %d", result1[i])
		}
	}
}
