package combat

import (
	"fmt"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/trait"
)

// CommandMinionAction
// src: rules/actions/command-an-animal.md (adapted for minions)
type CommandMinionAction struct {
	TargetMinionID string
}

func (c *CommandMinionAction) Name() string             { return "Command Minion" }
func (c *CommandMinionAction) Cost() ability.ActionCost { return ability.CostOne }
func (c *CommandMinionAction) HasTrait(id trait.TraitID) bool {
	return id == trait.TraitAuditory || id == trait.TraitConcentrate
}

func (c *CommandMinionAction) Validate(actor, target *entity.Entity, turn *TurnState) error {
	if c.TargetMinionID == "" {
		return fmt.Errorf("no minion specified")
	}

	// Verify ownership
	owns := false
	for _, id := range actor.MinionIDs {
		if id == c.TargetMinionID {
			owns = true
			break
		}
	}

	if !owns {
		return fmt.Errorf("you do not own this minion")
	}

	return nil
}

func (c *CommandMinionAction) Execute(actor, _ *entity.Entity, turn *TurnState) ability.ActionResult {
	if err := turn.SpendActions(c.Cost()); err != nil {
		return ability.ActionResult{Success: false, Description: err.Error()}
	}

	// The caller (Encounter) needs to handle the logic of *finding* the minion and granting actions.
	// The Execute signature only gives us actor/target (which is nil here usually).
	// However, we can return a specific Result metadata that the Encounter interprets.
	// ARCHITECTURAL NOTE: Using Meta map to communicate side-effects decouples Action logic from Encounter state.
	return ability.ActionResult{
		Success:     true,
		Description: "Minion commanded",
		// The encounter loop must inspect this and grant actions
		Meta: map[string]interface{}{
			"GrantActions": 2,
			"TargetID":     c.TargetMinionID,
		},
	}
}
