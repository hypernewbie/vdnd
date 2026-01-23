package condition

import (
	"testing"
	"uaa/vdnd/pkg/check"
)

func TestConditionTracker_Apply(t *testing.T) {
	tr := NewTracker()

	// Valued condition
	tr.Apply(NewValuedCondition(Frightened, 2, "Demoralize"))
	if tr.Value(Frightened) != 2 {
		t.Errorf("Expected Frightened 2, got %d", tr.Value(Frightened))
	}

	// Stacking - higher value
	tr.Apply(NewValuedCondition(Frightened, 3, "Scare to Death"))
	if tr.Value(Frightened) != 3 {
		t.Errorf("Expected Frightened 3 after higher apply, got %d", tr.Value(Frightened))
	}

	// Stacking - lower value ignored
	tr.Apply(NewValuedCondition(Frightened, 1, "Mean Glance"))
	if tr.Value(Frightened) != 3 {
		t.Errorf("Expected Frightened 3 after lower apply, got %d", tr.Value(Frightened))
	}

	// Persistent damage stacking
	tr.Apply(NewPersistentDamage(5, "fire", "Torch"))
	tr.Apply(NewPersistentDamage(8, "fire", "Fireball"))
	tr.Apply(NewPersistentDamage(3, "bleed", "Knife"))

	all := tr.All()
	countFire := 0
	countBleed := 0
	for _, c := range all {
		if c.ID == PersistentDamage {
			if c.DamageType == "fire" {
				countFire++
				if c.Value != 8 {
					t.Errorf("Expected fire damage 8, got %d", c.Value)
				}
			}
			if c.DamageType == "bleed" {
				countBleed++
				if c.Value != 3 {
					t.Errorf("Expected bleed damage 3, got %d", c.Value)
				}
			}
		}
	}
	if countFire != 1 || countBleed != 1 {
		t.Errorf("Expected 1 fire and 1 bleed persistent damage instances, got %d and %d", countFire, countBleed)
	}
}

func TestConditionTracker_ReduceAndRemove(t *testing.T) {
	tr := NewTracker()
	tr.Apply(NewValuedCondition(Frightened, 3, "Fear"))
	tr.Reduce(Frightened, 1)
	if tr.Value(Frightened) != 2 {
		t.Errorf("Expected Frightened 2 after reduce, got %d", tr.Value(Frightened))
	}

	tr.Reduce(Frightened, 5)
	if tr.Has(Frightened) {
		t.Error("Expected Frightened to be removed after reducing to 0")
	}

	tr.Apply(NewCondition(FlatFooted, "Flank"))
	tr.Remove(FlatFooted)
	if tr.Has(FlatFooted) {
		t.Error("Expected Flat-footed to be removed")
	}
}

func TestConditionTracker_EndTurn(t *testing.T) {
	tr := NewTracker()
	tr.Apply(NewValuedCondition(Frightened, 2, "Fear"))
	tr.Apply(NewValuedCondition(Sickened, 2, "Gas"))

	// End turn: Frightened reduces, Sickened does not
	tr.EndTurn()
	if tr.Value(Frightened) != 1 {
		t.Errorf("Expected Frightened 1 after end turn, got %d", tr.Value(Frightened))
	}
	if tr.Value(Sickened) != 2 {
		t.Errorf("Expected Sickened 2 after end turn, got %d", tr.Value(Sickened))
	}

	// End turn again: Frightened removed
	tr.EndTurn()
	if tr.Has(Frightened) {
		t.Error("Expected Frightened removed after 2 turns")
	}

	// Duration test
	c := NewCondition(Invisible, "Potion")
	c.Duration = 1
	tr.Apply(c)
	tr.EndTurn()
	if tr.Has(Invisible) {
		t.Error("Expected duration-based Invisible to be removed after end turn")
	}
}

func TestConditionTracker_GetModifiers(t *testing.T) {
	tr := NewTracker()
	tr.Apply(NewValuedCondition(Frightened, 2, "Fear"))
	tr.Apply(NewCondition(FlatFooted, "Flank"))
	tr.Apply(NewValuedCondition(Clumsy, 1, "Web"))

	// Universal modifiers
	mods := tr.GetModifiers()
	foundFrightened := false
	for _, m := range mods {
		if m.Source == "Frightened" && m.Value == -2 && m.Type == check.BonusStatus {
			foundFrightened = true
		}
	}
	if !foundFrightened {
		t.Error("GetModifiers missing Frightened status penalty")
	}

	// AC modifiers
	acMods := tr.GetACModifiers()
	foundFlatFooted := false
	foundClumsy := false
	for _, m := range acMods {
		if m.Source == "Flat-footed" && m.Value == -2 && m.Type == check.BonusCircumstance {
			foundFlatFooted = true
		}
		if m.Source == "Clumsy" && m.Value == -1 && m.Type == check.BonusStatus {
			foundClumsy = true
		}
	}
	if !foundFlatFooted || !foundClumsy {
		t.Errorf("GetACModifiers missing expected penalties: flat=%v, clumsy=%v", foundFlatFooted, foundClumsy)
	}

	// Melee Attack modifiers
	tr.Apply(NewValuedCondition(Enfeebled, 3, "Curse"))
	atkMods := tr.GetAttackModifiers(true)
	foundEnfeebled := false
	for _, m := range atkMods {
		if m.Source == "Enfeebled" && m.Value == -3 {
			foundEnfeebled = true
		}
	}
	if !foundEnfeebled {
		t.Error("GetAttackModifiers(melee) missing Enfeebled penalty")
	}

	// Ranged Attack modifiers (should have Clumsy but not Enfeebled)
	rangedMods := tr.GetAttackModifiers(false)
	foundEnfeebled = false
	foundClumsy = false
	for _, m := range rangedMods {
		if m.Source == "Enfeebled" {
			foundEnfeebled = true
		}
		if m.Source == "Clumsy" {
			foundClumsy = true
		}
	}
	if foundEnfeebled || !foundClumsy {
		t.Errorf("GetAttackModifiers(ranged) wrong penalties: enfeebled=%v (want false), clumsy=%v (want true)", foundEnfeebled, foundClumsy)
	}
}
