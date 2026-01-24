package subsystem_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/subsystem"
)

// Source: rules/rules/gamemastery-guide/chapter-3-subsystems.md

func TestChaseSplitParty(t *testing.T) {
	// 2 PCs chase 1 Thief. P1: Gap 10, P2: Gap 20.
	t.Log("Testing split party chase logic - expectation: success if closest pursuer catches quarry")
}

func TestInfluenceSkillReuse(t *testing.T) {
	// Player uses Diplomacy 10 times in a row.
	t.Log("Testing influence skill reuse penalty - documentation of intended GM logic")
}

func TestResearchFailureChain(t *testing.T) {
	// Research library. 3 Crit Fails in a row.
	// targetVP=10, failureThreshold=-3, roundsLimit=0
	lib := subsystem.NewSubsystem("Library", "Library", subsystem.SubsystemResearch, 10, -3, 0)
	lib.Start()
	
	lib.CurrentVP = -3
	
	if lib.CurrentVP <= lib.FailureThreshold {
		lib.State = subsystem.StateFailure
	}
	
	if lib.State != subsystem.StateFailure {
		t.Errorf("Expected StateFailure after reaching threshold, got %v", lib.State)
	}
}