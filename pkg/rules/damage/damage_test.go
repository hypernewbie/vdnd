package damage

import (
	"testing"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
	"uaa/vdnd/pkg/rules/trait"
)

func TestDamageRoll(t *testing.T) {
	// Simple roll
	dr := DamageRoll{
		BaseDice:   dice.DieRoll{Count: 1, Sides: 8, Modifier: 0},
		Modifier:   4,
		DamageType: item.Slashing,
	}
	// We'll trust dice.Roll() works as it's tested elsewhere.
	res := dr.Roll()
	if res.Amount < 5 || res.Amount > 12 {
		t.Errorf("DamageRoll 1d8+4 expected 5-12, got %d", res.Amount)
	}

	// Critical roll with deadly
	dr2 := DamageRoll{
		BaseDice:   dice.DieRoll{Count: 1, Sides: 8, Modifier: 0},
		Modifier:   4,
		DamageType: item.Piercing,
	}
	// Deadly d10
	deadly := dice.DieRoll{Count: 1, Sides: 10, Modifier: 0}
	resCrit := dr2.RollCritical(deadly, dice.DieRoll{})

	// (1d8+4)*2 + 1d10
	// min: (1+4)*2 + 1 = 11
	// max: (8+4)*2 + 10 = 34
	if resCrit.Amount < 11 || resCrit.Amount > 34 {
		t.Errorf("Critical DamageRoll 1d8+4 + deadly d10 expected 11-34, got %d", resCrit.Amount)
	}
}

func TestImmunity(t *testing.T) {
	target := entity.NewEntity("t1", "Ghost", 1)
	target.Immunities = append(target.Immunities, string(item.Fire), "precision")

	if !CheckImmunity(target, string(item.Fire), false, nil) {
		t.Error("Expected immunity to fire")
	}
	if !CheckImmunity(target, string(item.Slashing), true, nil) {
		t.Error("Expected immunity to precision damage")
	}
	if CheckImmunity(target, string(item.Slashing), false, nil) {
		t.Error("Did not expect immunity to slashing")
	}
}

func TestWeakness(t *testing.T) {
	target := entity.NewEntity("t1", "Zombie", 1)
	target.Weaknesses[string(item.Fire)] = 5
	target.Weaknesses["silver"] = 10

	w1 := CalculateWeakness(target, string(item.Fire), nil)
	if w1 != 5 {
		t.Errorf("Expected fire weakness 5, got %d", w1)
	}

	w2 := CalculateWeakness(target, string(item.Slashing), []trait.TraitID{trait.TraitID("silver")})
	if w2 != 10 {
		t.Errorf("Expected silver weakness 10, got %d", w2)
	}
}

func TestResistance(t *testing.T) {
	target := entity.NewEntity("t1", "Golem", 1)
	target.Resistances[string(item.Slashing)] = entity.ResistanceEntry{
		Amount: 5,
		Except: []string{"adamantine"},
	}

	r1 := CalculateResistance(target, string(item.Slashing), nil)
	if r1 != 5 {
		t.Errorf("Expected slashing resistance 5, got %d", r1)
	}

	r2 := CalculateResistance(target, string(item.Slashing), []trait.TraitID{trait.TraitID("adamantine")})
	if r2 != 0 {
		t.Errorf("Expected adamantine to bypass resistance, got %d", r2)
	}
}

func TestPipeline(t *testing.T) {
	target := entity.NewEntity("t1", "Hero", 1)
	target.MaxHP = 20
	target.CurrentHP = 20
	target.Weaknesses[string(item.Fire)] = 5

	dmg := DamageInstance{Amount: 10, Type: item.Fire}
	res := ProcessDamage(target, dmg, false)

	if res.FinalDamage != 15 {
		t.Errorf("Expected 15 damage (10 + 5 weakness), got %d", res.FinalDamage)
	}
	if target.CurrentHP != 5 {
		t.Errorf("Expected 5 HP remaining, got %d", target.CurrentHP)
	}

	// Reduce to 0
	dmg2 := DamageInstance{Amount: 10, Type: item.Slashing}
	res2 := ProcessDamage(target, dmg2, false)

	if target.CurrentHP != 0 {
		t.Error("Expected 0 HP")
	}
	if !target.Conditions.Has(condition.Dying) {
		t.Error("Expected Dying condition")
	}
	if res2.BecameDying != true {
		t.Error("Expected BecameDying to be true")
	}
}
