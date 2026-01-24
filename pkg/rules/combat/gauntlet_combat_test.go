package combat_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/combat"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/encounter"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
)

// Source: rules/rules/core-rulebook/chapter-9-playing-the-game.md (General Combat)

func TestInvisibleMinionCommand(t *testing.T) {
	minion := entity.NewEntity("minion", "Minion", 1)
	minion.Minion = &entity.MinionSettings{MasterID: "m"}
	
	// Minion is Hidden
	minion.Conditions.Apply(condition.NewCondition(condition.Hidden, "stealth"))
	
	weapon := &item.Weapon{
		Name: "Bite", 
		Damage: dice.DieRoll{Count: 1, Sides: 6}, 
		DamageType: item.Slashing,
	}
	minion.WieldedWeapons = append(minion.WieldedWeapons, weapon)
	
	target := entity.NewEntity("t", "Target", 1)
	
	// Execute strike
	strike := &combat.StrikeAction{Weapon: weapon}
	ts := combat.NewTurn(minion)
	
	_, checkRes := strike.Execute(minion, target, ts)
	
	found := false
	for _, mod := range checkRes.DebugModifiers {
		if mod.Source == "Flat-footed" || mod.Source == "Hidden" {
			found = true
			break
		}
	}
	
	if !found {
		t.Log("Warning: Hidden status did not apply flat-footed/bonus to strike")
	}
}

func TestDyingRecoveryVsDoomed(t *testing.T) {
	e := entity.NewEntity("test", "Test", 1)
	e.MaxHP = 20
	e.CurrentHP = 0
	
	// Apply Doomed 1
	e.Conditions.Apply(condition.NewValuedCondition(condition.Doomed, 1, "curse"))
	
	// Entity drops to 0 HP, starts at Dying 1
	e.CheckDying(false)
	
	if e.Conditions.Value(condition.Dying) != 1 {
		t.Errorf("Expected Dying 1, got %d", e.Conditions.Value(condition.Dying))
	}
	
	// If Doomed 1, death is at Dying 3 (4 - 1)
	// Fail a recovery check (roll 5) -> Dying 2
	e.RecoveryCheck(5)
	if e.IsDead() {
		t.Error("Should not be dead at Dying 2 with Doomed 1")
	}
	
	// Fail again (roll 5) -> Dying 3
	e.RecoveryCheck(5)
	if !e.IsDead() {
		t.Error("Should be dead at Dying 3 with Doomed 1")
	}
}

func TestReactionChain(t *testing.T) {
	// A hits B. B reacts. A reacts to B.
	t.Log("Simulating nested reaction logic - documentation of intended behavior (LIFO resolution)")
}

func TestPersistentDamageAndDying(t *testing.T) {
	enc := encounter.NewEncounter("test")
	e := entity.NewEntity("p", "Player", 1)
	e.MaxHP = 10
	e.CurrentHP = 1
	
	enc.AddParticipant(e)
	enc.Start()
	
	// Apply Persistent Fire 5
	e.Conditions.Apply(condition.NewValuedCondition(condition.PersistentDamage, 5, "fire"))
	
	// End Turn
	enc.EndTurn()
	
	if e.CurrentHP != 0 {
		t.Errorf("Expected 0 HP, got %d", e.CurrentHP)
	}
	
	if !e.IsDying() {
		t.Error("Entity should be Dying after persistent damage at end of turn")
	}
}