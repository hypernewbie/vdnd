package combat

import (
	"uaa/vdnd/pkg/rules/ability" // Needed for SeekAction
	"uaa/vdnd/pkg/rules/check"   // Kept for types if needed, or removing if unused
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/skill"
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

	res := skill.Demoralize(actor, target)

	desc := "Demoralize failed"
	if res.Degree >= check.Success {
		desc = "Target is Frightened"
		if res.Degree == check.CriticalSuccess {
			desc += " 2"
		} else {
			desc += " 1"
		}
	}

	return ActionResult{Success: res.Degree >= check.Success, Degree: res.Degree, Description: desc}
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

	mapPenalty := turn.GetMAP(false)
	modifiers := []check.Modifier{
		{Value: mapPenalty, Type: check.BonusUntyped, Source: "MAP"},
	}

	res := skill.Grapple(actor, target, modifiers)
	turn.RecordAttack()

	desc := "Grapple failed"
	if res.Degree >= check.Success {
		desc = "Target is Grabbed"
		if res.Degree == check.CriticalSuccess {
			desc += " (Restrained)"
		}
		// Flat-footed is applied by skill.Grapple call implicitly?
		// Actually, standard Grapple only implies Grabbed which IMPLIES Flat-Footed/Off-Guard.
		// The previous implementation applied both. skill.Grapple should ideally handle it or condition logic should.
		// For now we assume skill.Grapple handles condition application.
	}

	return ActionResult{Success: res.Degree >= check.Success, Degree: res.Degree, Description: desc}
}

// Trip - Athletics vs Reflex DC
type TripAction struct{}

func (t *TripAction) Name() string     { return "Trip" }
func (t *TripAction) Cost() ActionCost { return CostOne }
func (t *TripAction) HasTrait(id trait.TraitID) bool {
	return id == trait.TraitAttack
}
func (t *TripAction) Validate(actor, target *entity.Entity, turn *TurnState) error {
	return nil
}

func (t *TripAction) Execute(actor, target *entity.Entity, turn *TurnState) ActionResult {
	if err := turn.SpendActions(t.Cost()); err != nil {
		return ActionResult{Success: false, Description: err.Error()}
	}

	mapPenalty := turn.GetMAP(false)
	modifiers := []check.Modifier{
		{Value: mapPenalty, Type: check.BonusUntyped, Source: "MAP"},
	}

	res := skill.Trip(actor, target, modifiers)
	turn.RecordAttack()

	desc := "Trip failed"
	if res.Degree >= check.Success {
		desc = "Target is Prone"
		if res.Degree == check.CriticalSuccess {
			desc += " + Damage"
		}
	} else if res.Degree == check.CriticalFailure {
		desc = "Attacker fell Prone!"
	}

	return ActionResult{Success: res.Degree >= check.Success, Degree: res.Degree, Description: desc}
}

// Shove - Athletics vs Fortitude DC
type ShoveAction struct{}

func (s *ShoveAction) Name() string     { return "Shove" }
func (s *ShoveAction) Cost() ActionCost { return CostOne }
func (s *ShoveAction) HasTrait(id trait.TraitID) bool {
	return id == trait.TraitAttack
}
func (s *ShoveAction) Validate(actor, target *entity.Entity, turn *TurnState) error {
	return nil
}

func (s *ShoveAction) Execute(actor, target *entity.Entity, turn *TurnState) ActionResult {
	if err := turn.SpendActions(s.Cost()); err != nil {
		return ActionResult{Success: false, Description: err.Error()}
	}

	mapPenalty := turn.GetMAP(false)
	modifiers := []check.Modifier{
		{Value: mapPenalty, Type: check.BonusUntyped, Source: "MAP"},
	}

	res := skill.Shove(actor, target, modifiers)
	turn.RecordAttack()

	desc := "Shove failed"
	if res.Degree >= check.Success {
		desc = "Target Shoved 5ft" // Actual movement left to user or managed elsewhere
	} else if res.Degree == check.CriticalFailure {
		desc = "Attacker fell Prone!"
	}

	return ActionResult{Success: res.Degree >= check.Success, Degree: res.Degree, Description: desc}
}

// Hide - Stealth vs DC (Secret)
type HideAction struct{}

func (h *HideAction) Name() string                   { return "Hide" }
func (h *HideAction) Cost() ActionCost               { return CostOne }
func (h *HideAction) HasTrait(id trait.TraitID) bool { return id == trait.TraitSecret }
func (h *HideAction) Validate(actor, target *entity.Entity, turn *TurnState) error {
	return nil
}

func (h *HideAction) Execute(actor, target *entity.Entity, turn *TurnState) ActionResult {
	if err := turn.SpendActions(h.Cost()); err != nil {
		return ActionResult{Success: false, Description: err.Error()}
	}

	dc := 20 // Default standard DC
	if target != nil {
		// If hiding from specific target, use their Perception DC
		dc = 10 + target.GetPerception()
	}

	res := skill.Hide(actor, dc, nil)

	desc := "Hide result: " + res.Degree.String()
	if res.Degree >= check.Success {
		desc += " (Hidden)"
	}

	return ActionResult{Success: res.Degree >= check.Success, Degree: res.Degree, Description: desc}
}

// Seek - Perception vs Stealth DC (Secret)
type SeekAction struct{}

func (s *SeekAction) Name() string                   { return "Seek" }
func (s *SeekAction) Cost() ActionCost               { return CostOne }
func (s *SeekAction) HasTrait(id trait.TraitID) bool { return id == trait.TraitSecret }
func (s *SeekAction) Validate(actor, target *entity.Entity, turn *TurnState) error {
	return nil
}

func (s *SeekAction) Execute(actor, target *entity.Entity, turn *TurnState) ActionResult {
	if err := turn.SpendActions(s.Cost()); err != nil {
		return ActionResult{Success: false, Description: err.Error()}
	}

	dc := 20 // Default standard DC
	if target != nil {
		// If seeking specific target, use their Stealth DC
		mod := target.GetSkillModifier(ability.SkillStealth)
		dc = 10 + mod
	}

	res := skill.Seek(actor, dc, nil)

	desc := "Seek result: " + res.Degree.String()

	return ActionResult{Success: res.Degree >= check.Success, Degree: res.Degree, Description: desc}
}

// RecallKnowledge - Check against LevelBasedDC
type RecallKnowledgeAction struct {
	Skill ability.SkillID
}

func (r *RecallKnowledgeAction) Name() string     { return "Recall Knowledge" }
func (r *RecallKnowledgeAction) Cost() ActionCost { return CostOne }
func (r *RecallKnowledgeAction) HasTrait(id trait.TraitID) bool {
	return id == trait.TraitConcentrate || id == trait.TraitSecret
}
func (r *RecallKnowledgeAction) Validate(actor, target *entity.Entity, turn *TurnState) error {
	return nil
}

func (r *RecallKnowledgeAction) Execute(actor, target *entity.Entity, turn *TurnState) ActionResult {
	if err := turn.SpendActions(r.Cost()); err != nil {
		return ActionResult{Success: false, Description: err.Error()}
	}

	dc := 15 // Default
	if target != nil {
		dc = skill.LevelBasedDC(target.Level)
	}

	// Use generic PerformSkillCheck from skill package via RecallKnowledge wrapper?
	// skill.RecallKnowledge returns string, result.
	info, res := skill.RecallKnowledge(actor, r.Skill, dc)

	desc := "Recall Knowledge: " + res.Degree.String()
	if res.Degree >= check.Success {
		desc += " - " + info
	}

	return ActionResult{Success: res.Degree >= check.Success, Degree: res.Degree, Description: desc}
}
