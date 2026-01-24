package encounter_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/encounter"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/hazard"
)

func TestEncounterWithHazard(t *testing.T) {
	enc := encounter.NewEncounter("trapped_room")

	// Add party
	fighter := entity.NewEntity("fighter", "Bold Fighter", 5)
	fighter.Position = "trap_room"
	enc.AddParticipant(fighter)

	// Add complex hazard
	trap := hazard.GetComplexHazard("spinning_blade_pillar")
	trap.Position = "trap_room"
	enc.AddHazard(trap)

	// Roll initiatives
	enc.RollInitiative(encounter.InitPerception)

	if err := enc.Start(); err != nil {
		t.Fatalf("Failed to start encounter: %v", err)
	}

	// Verify hazard is in initiative
	found := false
	for _, p := range enc.Participants {
		if p.Type == encounter.ParticipantHazard {
			found = true
			t.Logf("Hazard initiative: %d", p.Initiative)
		}
	}

	if !found {
		t.Error("Hazard should be in participants")
	}
}

func TestHazardTurnInEncounter(t *testing.T) {
	enc := encounter.NewEncounter("test")

	victim := entity.NewEntity("victim", "Victim", 5)
	victim.MaxHP = 50
	victim.CurrentHP = 50
	victim.Position = "danger_zone"
	enc.AddParticipant(victim)

	trap := hazard.GetComplexHazard("poisoned_dart_gallery")
	trap.Position = "danger_zone"
	enc.AddHazard(trap)

	enc.Start()

	// Execute hazard turn
	result := enc.ExecuteHazardTurn(trap.ID)

	t.Logf("Hazard turn result: %d actions, %d damage",
		len(result.ActionResults), result.TotalDamage)

	if len(result.ActionResults) == 0 {
		t.Error("Hazard should have taken actions")
	}
}

func TestSimpleHazardNotInInitiative(t *testing.T) {
	enc := encounter.NewEncounter("simple_test")

	// Simple hazard (not complex)
	pit := hazard.NewHazard("pit_trap", "Pit Trap", 2)
	pit.Type = hazard.HazardTrap
	pit.Complexity = hazard.ComplexitySimple

	enc.AddHazard(pit)

	// Simple hazards should not be added
	for _, p := range enc.Participants {
		if p.Type == encounter.ParticipantHazard {
			t.Error("Simple hazards should not be in initiative")
		}
	}
}
