package trait

import (
	"testing"
)

func TestNewTrait(t *testing.T) {
	tr := NewTrait(TraitAgile, "Agile", CategoryWeapon)
	if tr.ID != TraitAgile || tr.Name != "Agile" || tr.Category != CategoryWeapon || tr.Parameter != "" {
		t.Errorf("NewTrait failed: got %+v", tr)
	}
}

func TestNewParameterizedTrait(t *testing.T) {
	tr := NewParameterizedTrait(TraitDeadly, "Deadly", CategoryWeapon, "d10")
	if tr.ID != TraitDeadly || tr.Parameter != "d10" {
		t.Errorf("NewParameterizedTrait failed: got %+v", tr)
	}
}

func TestRegistry(t *testing.T) {
	reg := DefaultRegistry()

	tests := []struct {
		id    TraitID
		found bool
	}{
		{TraitAgile, true},
		{TraitFire, true},
		{TraitAttack, true},
		{"humanoid", true},
		{"nonexistent", false},
	}

	for _, tt := range tests {
		_, ok := reg.Get(tt.id)
		if ok != tt.found {
			t.Errorf("Registry.Get(%s) found=%v, want %v", tt.id, ok, tt.found)
		}
	}

	// Test Register
	custom := NewTrait("custom", "Custom", CategoryGeneral)
	reg.Register(custom)
	if !reg.Has("custom") {
		t.Error("Registry.Has failed to find registered trait")
	}

	// Test AllInCategory
	damageTraits := reg.AllInCategory(CategoryDamage)
	if len(damageTraits) < 5 {
		t.Errorf("Too few damage traits: %d", len(damageTraits))
	}
}

func TestTraitSet(t *testing.T) {
	ts := TraitSet{TraitAgile, TraitFinesse}

	if !ts.HasTrait(TraitAgile) {
		t.Error("TraitSet.HasTrait failed to find present trait")
	}
	if ts.HasTrait(TraitReach) {
		t.Error("TraitSet.HasTrait found absent trait")
	}

	empty := TraitSet{}
	if empty.HasTrait(TraitAgile) {
		t.Error("Empty TraitSet.HasTrait returned true")
	}

	traits := ts.Traits()
	if len(traits) != 2 || traits[0] != TraitAgile || traits[1] != TraitFinesse {
		t.Errorf("TraitSet.Traits() returned wrong slice: %v", traits)
	}
}

func TestHasAnyAllTraits(t *testing.T) {
	ts := TraitSet{TraitFire, TraitCold, TraitAttack}

	// HasAnyTrait
	if !HasAnyTrait(ts, TraitFire, TraitAcid) {
		t.Error("HasAnyTrait failed to find one match")
	}
	if HasAnyTrait(ts, TraitAcid, TraitSonic) {
		t.Error("HasAnyTrait found match in none")
	}
	if HasAnyTrait(ts) { // Empty query
		t.Error("HasAnyTrait with no query returned true")
	}

	// HasAllTraits
	if !HasAllTraits(ts, TraitFire, TraitAttack) {
		t.Error("HasAllTraits failed to find all present")
	}
	if HasAllTraits(ts, TraitFire, TraitAcid) {
		t.Error("HasAllTraits returned true with missing trait")
	}
	if !HasAllTraits(ts) { // Empty query - vacuously true
		t.Error("HasAllTraits with no query returned false")
	}
}
