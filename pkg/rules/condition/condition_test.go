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
	tr := NewTracker()
	tr.Apply(NewPersistentDamage(10, "fire", "Fire"))
	actor := &MockActor{ID: "test"}

	tr.EndTurn(actor)

	if actor.TotalDamage != 10 {
		t.Errorf("Expected 10 damage applied, got %d", actor.TotalDamage)
	}
}

func TestConditionTracker_EndTurn(t *testing.T) {
	tr := NewTracker()
	tr.Apply(NewValuedCondition(Frightened, 2, "Fear"))
	tr.EndTurn(nil)
	if tr.Value(Frightened) != 1 {
		t.Errorf("Expected Frightened 1, got %d", tr.Value(Frightened))
	}
}
