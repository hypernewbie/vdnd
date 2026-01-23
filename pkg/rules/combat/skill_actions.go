package combat

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/trait"
)

// Demoralize - Intimidation vs Will DC, applies Frightened
type DemoralizeAction struct{}

func (d *DemoralizeAction) Name() string     { return "Demoralize" }
func (d *DemoralizeAction) Cost() ActionCost { return CostOne }
func (d *DemoralizeAction) HasTrait(id trait.TraitID) bool {
	return id == trait.TraitAuditory || id == trait.TraitMental || id == trait.TraitEmotion || id == trait.TraitFear
}

func (d *DemoralizeAction) Validate(actor, target *entity.Entity, turn *TurnState) error {
	return nil
}

func (d *DemoralizeAction) Execute(actor, target *entity.Entity, turn *TurnState) ActionResult {
	if err := turn.SpendActions(d.Cost()); err != nil {
		return ActionResult{Success: false, Description: err.Error()}
	}

	intimidation := actor.GetSkillModifier(ability.SkillIntimidation)
	willDC := target.GetSaveDC(entity.SaveWill)

	res := check.PerformCheck(intimidation, nil, willDC)

	switch res.Degree {
	case check.CriticalSuccess:
		target.Conditions.Apply(condition.NewValuedCondition(condition.Frightened, 2, "Demoralize"))
		return ActionResult{Success: true, Degree: res.Degree, Description: "Target is Frightened 2"}
	case check.Success:
		target.Conditions.Apply(condition.NewValuedCondition(condition.Frightened, 1, "Demoralize"))
		return ActionResult{Success: true, Degree: res.Degree, Description: "Target is Frightened 1"}
	}

	return ActionResult{Success: false, Degree: res.Degree, Description: "Demoralize failed"}
}

// Grapple - Athletics vs Fortitude DC, applies Grabbed
type GrappleAction struct{}

func (g *GrappleAction) Name() string     { return "Grapple" }
func (g *GrappleAction) Cost() ActionCost { return CostOne }
func (g *GrappleAction) HasTrait(id trait.TraitID) bool {
	return id == trait.TraitAttack
}

func (g *GrappleAction) Validate(actor, target *entity.Entity, turn *TurnState) error {
	return nil
}

func (g *GrappleAction) Execute(actor, target *entity.Entity, turn *TurnState) ActionResult {
	if err := turn.SpendActions(g.Cost()); err != nil {
		return ActionResult{Success: false, Description: err.Error()}
	}

	athletics := actor.GetSkillModifier(ability.SkillAthletics)
	fortDC := target.GetSaveDC(entity.SaveFortitude)

	// Grapple has the attack trait, so it incurs and is affected by MAP
	mapPenalty := turn.GetMAP(false) // Grapple isn't agile
	modifiers := []check.Modifier{
		{Value: mapPenalty, Type: check.BonusUntyped, Source: "MAP"},
	}

	res := check.PerformCheck(athletics, modifiers, fortDC)
	turn.RecordAttack()

	switch res.Degree {
	case check.CriticalSuccess, check.Success:
		target.Conditions.Apply(condition.NewCondition(condition.FlatFooted, "Grappled"))
		target.Conditions.Apply(condition.NewCondition(condition.Grabbed, "Grappled"))
		return ActionResult{Success: true, Degree: res.Degree, Description: "Target is Grabbed"}
	}

	return ActionResult{Success: false, Degree: res.Degree, Description: "Grapple failed"}
}
