package hazard_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/hazard"
)

func TestHazardDetection(t *testing.T) {
	trap := hazard.NewHazard("t1", "Hidden Pit", 1)
	trap.StealthDC = 20

	observer := entity.NewEntity("o1", "Observer", 1)
	observer.Perception = ability.Trained

	_ = trap.Detect(observer)
}

func TestHazardDisabling(t *testing.T) {
	trap := hazard.NewHazard("t1", "Mechanical Trap", 1)
	opt := hazard.DisableOption{Skill: ability.SkillThievery, DC: 15}
	trap.DisableOptions = append(trap.DisableOptions, opt)

	actor := entity.NewEntity("a1", "Rogue", 1)
	actor.SkillProficiencies[ability.SkillThievery] = ability.Trained

	if !trap.CanDisable(actor, opt) {
		t.Error("Trained actor should be able to disable")
	}

	res := trap.AttemptDisable(actor, opt)
	if res.Degree >= check.Success && !trap.IsDisabled {
		t.Error("Hazard should be disabled on success")
	}
}

func TestHazardTrigger(t *testing.T) {
	trap := hazard.NewHazard("t1", "Pit", 1)
	trap.Trigger = hazard.TriggerCondition{
		Type: hazard.TriggerEnter,
		Area: "zone-a",
	}

	actor := entity.NewEntity("a1", "Hero", 1)
	event := ability.Event{
		Type:     ability.EventMove,
		Actor:    actor,
		Position: "zone-a",
	}

	if !trap.CheckTrigger(event) {
		t.Error("Hazard should trigger when entering zone-a")
	}

	if !trap.IsTriggered {
		t.Error("Hazard should be marked as triggered")
	}

	if trap.CheckTrigger(event) {
		t.Error("Simple hazard should not trigger twice")
	}
}

func TestDamageEffect(t *testing.T) {
	trap := hazard.NewHazard("t1", "Spike Pit", 1)
	target := entity.NewEntity("t1", "Victim", 1)
	target.MaxHP = 20
	target.CurrentHP = 20

	effect := &hazard.DamageEffect{
		Damage:      dice.DieRoll{Count: 2, Sides: 6, Modifier: 0}, // 2d6
		SaveType:    ability.SaveReflex,
		SaveDC:      100, // Guaranteed failure for testing
		IsBasicSave: true,
	}

	results := effect.Apply(trap, []*entity.Entity{target})

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if target.CurrentHP >= 20 {
		t.Error("Target should have taken damage")
	}
}

func TestHazardTakeDamage(t *testing.T) {
	trap := hazard.NewHazard("t1", "Statue", 1)
	trap.HP = 20
	trap.CurrentHP = 20
	trap.Hardness = 5

	applied := trap.TakeDamage(10, "bludgeoning")
	if applied != 5 {
		t.Errorf("Expected 5 applied damage, got %d", applied)
	}
	if trap.CurrentHP != 15 {
		t.Errorf("Expected 15 HP remaining, got %d", trap.CurrentHP)
	}

	trap.TakeDamage(20, "slashing")
	if trap.CurrentHP != 0 {
		t.Error("Should be at 0 HP")
	}
	if !trap.IsDisabled {
		t.Error("Destroyed hazard should be disabled")
	}
}