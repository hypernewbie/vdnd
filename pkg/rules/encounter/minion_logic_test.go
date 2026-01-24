package encounter_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/combat"
	"uaa/vdnd/pkg/rules/encounter"
	"uaa/vdnd/pkg/rules/entity"
)

func TestUncommandedMinion(t *testing.T) {
	e := encounter.NewEncounter("enc")
	
	m := entity.NewEntity("m", "Minion", 1)
	m.Minion = &entity.MinionSettings{Type: entity.MinionFamiliar}
	
	e.AddParticipant(m)
	e.Start()
	
	p := e.GetCurrentParticipant()
	_, _ = e.StartTurn()
	
	if p.Entity.Minion == nil {
		t.Fatal("Expected minion settings")
	}
	
	if p.TurnState.ActionsRemaining != 0 {
		t.Errorf("Uncommanded minion should have 0 actions, got %d", p.TurnState.ActionsRemaining)
	}
}

func TestCommandMinionActions(t *testing.T) {
	e := encounter.NewEncounter("enc")
	
	master := entity.NewEntity("master", "Master", 1)
	minion := entity.NewEntity("minion", "Minion", 1)
	minion.Minion = &entity.MinionSettings{Type: entity.MinionFamiliar, MasterID: "master"}
	master.MinionIDs = []string{"minion"}
	
	e.AddParticipant(master)
	e.AddParticipant(minion)
	e.Start()
	
	// Master's turn
	masterP := e.GetParticipantByID("master")
	_, _ = e.StartTurn()
	
	// Command it.
	cmd := &combat.CommandMinionAction{TargetMinionID: "minion"}
	res := cmd.Execute(master, nil, masterP.TurnState)
	
	// Handle the grant
	e.HandleActionResult(res)
	
	// Minion turn
	e.EndTurn() 
	
	_, _ = e.StartTurn() 
	
	p := e.GetCurrentParticipant()
	if p.TurnState.ActionsRemaining != 2 {
		t.Errorf("Expected 2 actions for commanded minion, got %d", p.TurnState.ActionsRemaining)
	}
}

func TestOrphanedMinion(t *testing.T) {
	e := encounter.NewEncounter("enc")
	
	master := entity.NewEntity("master", "Master", 1)
	minion := entity.NewEntity("minion", "Minion", 1)
	minion.Minion = &entity.MinionSettings{Type: entity.MinionFamiliar, MasterID: "master"}
	
	e.AddParticipant(master)
	e.AddParticipant(minion)
	e.Start()
	
	e.EndTurn() // Skip master
	_, err := e.StartTurn()
	if err != nil {
		t.Errorf("Minion turn should still start without master: %v", err)
	}
	p := e.GetCurrentParticipant()
	if p.TurnState.ActionsRemaining != 0 {
		t.Error("Orphaned minion still starts with 0 actions")
	}
}