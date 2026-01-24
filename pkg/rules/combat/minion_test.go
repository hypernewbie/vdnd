package combat_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/combat"
	"uaa/vdnd/pkg/rules/entity"
)

func TestCommandMinion(t *testing.T) {
	master := entity.NewEntity("master", "Master", 1)
	master.MinionIDs = []string{"minion1"}

	turn := combat.NewTurn(master)

	cmd := &combat.CommandMinionAction{TargetMinionID: "minion1"}

	// 1. Validation
	err := cmd.Validate(master, nil, turn)
	if err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	// 2. Execution
	res := cmd.Execute(master, nil, turn)
	if !res.Success {
		t.Error("Command should succeed")
	}

	if res.Meta["GrantActions"] != 2 {
		t.Error("Should grant 2 actions")
	}

	// 3. Cost
	if turn.ActionsRemaining != 2 { // Started at 3, spent 1
		t.Errorf("Expected 2 actions remaining, got %d", turn.ActionsRemaining)
	}
}
