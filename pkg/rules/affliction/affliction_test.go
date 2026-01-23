package affliction

import (
	"testing"
	"uaa/vdnd/pkg/rules/check"
)

func TestAfflictionStageStateMachine(t *testing.T) {
	aff := &Affliction{
		ID:       "centipede-venom",
		Name:     "Centipede Venom",
		MaxStage: 3,
		Stages: []Stage{
			{Number: 1},
			{Number: 2},
			{Number: 3},
		},
	}

	inst := NewInstance(aff, "Bite")
	inst.CurrentStage = 1

	// 1. Critical Success: -2 stages -> Stage 0 (Cured)
	inst.ProcessSave(check.CriticalSuccess)
	if !inst.IsCured() {
		t.Errorf("Expected cured (stage 0) on Crit Success, got Stage %d", inst.CurrentStage)
	}

	// 2. Reset and Test Success: -1 stage -> Stage 0 (Cured)
	inst.CurrentStage = 1
	inst.ProcessSave(check.Success)
	if !inst.IsCured() {
		t.Errorf("Expected cured on Success from Stage 1, got Stage %d", inst.CurrentStage)
	}

	// 3. Reset and Test Failure: +1 stage -> Stage 2
	inst.CurrentStage = 1
	inst.ProcessSave(check.Failure)
	if inst.CurrentStage != 2 {
		t.Errorf("Expected Stage 2 on Failure from Stage 1, got %d", inst.CurrentStage)
	}

	// 4. Reset and Test Crit Failure: +2 stages -> Stage 3
	inst.CurrentStage = 1
	inst.ProcessSave(check.CriticalFailure)
	if inst.CurrentStage != 3 {
		t.Errorf("Expected Stage 3 on Crit Failure from Stage 1, got %d", inst.CurrentStage)
	}

	// 5. Overflow: Crit Failure from Stage 2 should cap at Stage 3
	inst.CurrentStage = 2
	inst.ProcessSave(check.CriticalFailure)
	if inst.CurrentStage != 3 {
		t.Errorf("Expected Stage 3 (capped) on Crit Failure from Stage 2, got %d", inst.CurrentStage)
	}
}

func TestAfflictionTick(t *testing.T) {
	aff := &Affliction{
		ID:         "venom",
		OnsetDelay: 1,
		Interval:   2,
	}
	inst := NewInstance(aff, "Test")

	// T=0: OnsetDelay 1
	if inst.Tick() { t.Error("Should not require save yet") }
	// T=1: Onset happens
	if !inst.Tick() { t.Error("Should require initial save after onset") }
	
	// T=2: Interval 2 (1/2)
	if inst.Tick() { t.Error("Should not require save yet") }
	// T=3: Interval 2 (2/2)
	if !inst.Tick() { t.Error("Should require save after interval") }
}

func TestTickWithRoll(t *testing.T) {
	aff := &Affliction{
		ID:         "test",
		MaxStage:   5,
		OnsetDelay: 0,
		Interval:   1,
	}
	inst := NewInstance(aff, "Test")
	
	// Stage 1 -> Fail -> Stage 2
	inst.TickWithRoll(10, 15) // Roll 10 vs DC 15 = Fail
	if inst.CurrentStage != 2 { t.Errorf("Expected Stage 2, got %d", inst.CurrentStage) }
	
	// Stage 2 -> Crit Fail -> Stage 4
	inst.TickWithRoll(1, 15) // Roll 1 vs DC 15 = Crit Fail
	if inst.CurrentStage != 4 { t.Errorf("Expected Stage 4, got %d", inst.CurrentStage) }
	
	// Stage 4 -> Crit Success -> Stage 2
	inst.TickWithRoll(20, 15) // Roll 20 vs DC 15 = Crit Success
	if inst.CurrentStage != 2 { t.Errorf("Expected Stage 2, got %d", inst.CurrentStage) }
}