package entity

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/item"
)

func TestSize(t *testing.T) {
	if Medium.Space() != 5 {
		t.Errorf("Medium space should be 5, got %d", Medium.Space())
	}
	if Large.Space() != 10 {
		t.Errorf("Large space should be 10, got %d", Large.Space())
	}
	if Tiny.Reach() != 0 {
		t.Errorf("Tiny reach should be 0, got %d", Tiny.Reach())
	}
	if Medium.String() != "Medium" {
		t.Errorf("Medium string should be Medium, got %s", Medium.String())
	}
}

func TestEntityCreation(t *testing.T) {
	e := NewPC("pc1", "Valeros", 1, "Human", "Fighter", "Noble")
	if e.Name != "Valeros" || e.Class != "Fighter" {
		t.Error("PC fields not populated correctly")
	}
	if e.Conditions == nil {
		t.Error("Conditions tracker not initialized")
	}

	e.Abilities.Strength = 18
	clone := e.Clone()
	if clone.Abilities.Strength != 18 {
		t.Error("Clone did not copy ability scores")
	}
	clone.Abilities.Strength = 10
	if e.Abilities.Strength != 18 {
		t.Error("Modifying clone affected original (shallow copy?)")
	}
}

func TestACCalculation(t *testing.T) {
	e := NewEntity("e1", "Target", 1)
	e.Abilities.Dexterity = 16           // +3 mod
	e.UnarmoredDefense = ability.Trained // +3 bonus at lvl 1

	// Unarmored: 10 + 3 (dex) + 3 (prof) = 16
	if ac := e.GetAC(); ac != 16 {
		t.Errorf("Expected AC 16, got %d", ac)
	}

	// With Leather Armor (+1 AC, Dex Cap 4)
	e.WornArmor = &item.LeatherArmor
	e.ArmorProficiencies[item.LightArmor] = ability.Trained
	// 10 + 3 (dex) + 3 (prof) + 1 (item) = 17
	if ac := e.GetAC(); ac != 17 {
		t.Errorf("Expected AC 17 with leather, got %d", ac)
	}

	// Dex exceeds cap
	e.Abilities.Dexterity = 22 // +6 mod
	// Cap is 4, so 10 + 4 (dex cap) + 3 (prof) + 1 (item) = 18
	if ac := e.GetAC(); ac != 18 {
		t.Errorf("Expected AC 18 with dex cap, got %d", ac)
	}

	// Conditions
	e.Conditions.Apply(condition.NewCondition(condition.FlatFooted, "Flank"))
	// 18 - 2 = 16
	if ac := e.GetAC(); ac != 16 {
		t.Errorf("Expected AC 16 while flat-footed, got %d", ac)
	}
}

func TestSaveCalculation(t *testing.T) {
	e := NewEntity("e1", "Target", 5)
	e.Abilities.Constitution = 14 // +2
	e.Abilities.Dexterity = 18    // +4
	e.Abilities.Wisdom = 12       // +1

	e.Fortitude = ability.Trained // 2 + 7 = 9
	e.Reflex = ability.Expert     // 4 + 9 = 13
	e.Will = ability.Untrained    // 0

	if f := e.GetFortitude(); f != 9 {
		t.Errorf("Expected Fortitude 9, got %d", f)
	}
	if r := e.GetReflex(); r != 13 {
		t.Errorf("Expected Reflex 13, got %d", r)
	}
	if w := e.GetWill(); w != 1 {
		t.Errorf("Expected Will 1 (untrained), got %d", w)
	}

	// Modifiers
	e.Conditions.Apply(condition.NewValuedCondition(condition.Frightened, 2, "Fear"))
	// All reduce by 2
	if f := e.GetFortitude(); f != 7 {
		t.Errorf("Expected Fortitude 7 with Frightened 2, got %d", f)
	}
}

func TestDamageAndHP(t *testing.T) {
	e := NewEntity("e1", "Target", 1)
	e.MaxHP = 20
	e.CurrentHP = 20

	// Basic damage
	e.TakeDamage(5, "slashing")
	if e.CurrentHP != 15 {
		t.Errorf("Expected 15 HP, got %d", e.CurrentHP)
	}

	// Immunity
	e.Immunities = append(e.Immunities, "fire")
	e.TakeDamage(10, "fire")
	if e.CurrentHP != 15 {
		t.Error("Took damage from immune type")
	}

	// Resistance
	e.Resistances["cold"] = 5
	e.TakeDamage(8, "cold") // Takes 8 - 5 = 3
	if e.CurrentHP != 12 {
		t.Errorf("Expected 12 HP after resistance, got %d", e.CurrentHP)
	}

	// Weakness
	e.Weaknesses["acid"] = 5
	e.TakeDamage(5, "acid") // Takes 5 + 5 = 10
	if e.CurrentHP != 2 {
		t.Errorf("Expected 2 HP after weakness, got %d", e.CurrentHP)
	}

	// Temp HP
	e.AddTempHP(10)
	e.TakeDamage(8, "slashing") // Temp HP absorbs all
	if e.CurrentHP != 2 || e.TempHP != 2 {
		t.Errorf("Expected 2 HP and 2 TempHP, got %d and %d", e.CurrentHP, e.TempHP)
	}
}

func TestDying(t *testing.T) {
	e := NewEntity("e1", "Target", 1)
	e.MaxHP = 10
	e.CurrentHP = 5

	// Drop to 0
	e.TakeDamage(10, "slashing")
	e.CheckDying(false)
	if !e.IsDying() || e.Conditions.Value(condition.Dying) != 1 {
		t.Error("Expected Dying 1")
	}
	if !e.IsUnconscious() {
		t.Error("Expected Unconscious at 0 HP")
	}

	// Damage at 0 HP
	e.CheckDying(false)
	if e.Conditions.Value(condition.Dying) != 2 {
		t.Errorf("Expected Dying 2 after more damage, got %d", e.Conditions.Value(condition.Dying))
	}

	// Recovery Success
	e.RecoveryCheck(true, false)
	if e.Conditions.Value(condition.Dying) != 1 {
		t.Error("Expected Dying 1 after recovery success")
	}

	// Stabilize
	e.RecoveryCheck(true, false)
	if e.IsDying() {
		t.Error("Expected not dying after stabilizing")
	}
	if e.Conditions.Value(condition.Wounded) != 1 {
		t.Error("Expected Wounded 1 after stabilizing")
	}

	// Drop to 0 again with Wounded
	e.CurrentHP = 5
	e.TakeDamage(10, "slashing")
	e.CheckDying(false)
	// Should be 1 (base) + 1 (wounded) = 2
	if e.Conditions.Value(condition.Dying) != 2 {
		t.Errorf("Expected Dying 2 due to wounded, got %d", e.Conditions.Value(condition.Dying))
	}

	// Death
	e.CheckDying(true) // Crit at 0 HP adds 2 -> Dying 4
	if !e.IsDead() {
		t.Error("Expected dead at Dying 4")
	}
}
