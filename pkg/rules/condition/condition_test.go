package condition

import (
	"testing"
)

type MockActor struct {
	TotalDamage int
	ID          string
}

func (m *MockActor) ApplyDamage(amount int) { m.TotalDamage += amount }
func (m *MockActor) GetID() string          { return m.ID }

func TestConditionTracker_Apply(t *testing.T) {
	tr := NewTracker()
	tr.Apply(NewValuedCondition(Frightened, 2, "Demoralize"))
	if tr.Value(Frightened) != 2 {
		t.Errorf("Expected Frightened 2, got %d", tr.Value(Frightened))
	}
	tr.Apply(NewValuedCondition(Frightened, 3, "Fear"))
	if tr.Value(Frightened) != 3 {
		t.Errorf("Expected Frightened 3, got %d", tr.Value(Frightened))
	}
}

func TestPersistentDamageStacking(t *testing.T) {
	tr := NewTracker()
	// Fire 5
	tr.Apply(NewPersistentDamage(5, "fire", "Torch"))
	// Fire 10 (should replace)
	tr.Apply(NewPersistentDamage(10, "fire", "Fireball"))
	// Acid 5 (should coexist)
	tr.Apply(NewPersistentDamage(5, "acid", "Splash"))

	all := tr.All()
	fireVal := 0
	acidVal := 0
	pdCount := 0
	for _, c := range all {
		if c.ID == PersistentDamage {
			pdCount++
			if c.DamageType == "fire" { fireVal = c.Value }
			if c.DamageType == "acid" { acidVal = c.Value }
		}
	}

	if pdCount != 2 { t.Errorf("Expected 2 PD instances, got %d", pdCount) }
	if fireVal != 10 { t.Errorf("Expected Fire 10, got %d", fireVal) }
	if acidVal != 5 { t.Errorf("Expected Acid 5, got %d", acidVal) }
}

func TestEndTurnPersistentDamage(t *testing.T) {
	// We want to test that damage is applied AND then the flat check happens.
	// Since FlatCheck uses dice.D20, we should probably mock it or just run multiple times.
	// For this test, we'll just verify damage application.
	tr := NewTracker()
	tr.Apply(NewPersistentDamage(10, "fire", "Fire"))
	actor := &MockActor{ID: "test"}

	tr.EndTurn(actor)

	if actor.TotalDamage != 10 {
		t.Errorf("Expected 10 damage applied, got %d", actor.TotalDamage)
	}
}

func TestFlatCheckRemoval(t *testing.T) {
	// To test deterministic removal, we'd need to mock check.FlatCheck.
	// Since we can't easily mock it without refactoring, we'll just ensure the logic exists 
	// in tracker.go and run it in a loop to see it eventually clears.
	tr := NewTracker()
	tr.Apply(NewPersistentDamage(10, "fire", "Fire"))
	
	removed := false
	for i := 0; i < 100; i++ {
		tr.EndTurn(&MockActor{})
		if tr.Value(PersistentDamage) == 0 {
			removed = true
			break
		}
	}
	if !removed {
		t.Error("Persistent damage was not removed after 100 turns (highly unlikely if DC 15 is working)")
	}
}

func TestValuedConditionDecay(t *testing.T) {
	tr := NewTracker()
	tr.Apply(NewValuedCondition(Frightened, 2, "Fear"))
	tr.Apply(NewValuedCondition(Drained, 1, "Ghoul"))
	
	tr.EndTurn(nil)
	
	if tr.Value(Frightened) != 1 {
		t.Errorf("Frightened should have decayed to 1, got %d", tr.Value(Frightened))
	}
	if tr.Value(Drained) != 1 {
		t.Errorf("Drained should NOT have decayed, got %d", tr.Value(Drained))
	}
	
	tr.EndTurn(nil)
	if tr.Value(Frightened) != 0 {
		t.Errorf("Frightened should have decayed to 0, got %d", tr.Value(Frightened))
	}
	if tr.Value(Drained) != 1 {
		t.Errorf("Drained should STILL be 1, got %d", tr.Value(Drained))
	}
}

func TestRelationalConditionLogic(t *testing.T) {
	tr := NewTracker()
	tr.ApplyRelative(Hidden, "observer-1", "Stealth")
	
	if !tr.HasRelative(Hidden, "observer-1") {
		t.Error("Should be hidden from observer-1")
	}
	if tr.HasRelative(Hidden, "observer-2") {
		t.Error("Should NOT be hidden from observer-2")
	}
	
	tr.RemoveRelative(Hidden, "observer-1")
	if tr.HasRelative(Hidden, "observer-1") {
		t.Error("Should no longer be hidden from observer-1")
	}
}