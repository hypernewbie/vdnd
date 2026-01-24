package subsystem

import (
	"fmt"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/skill"
)

// ChaseRole defines whether participant is pursuer or quarry
type ChaseRole int

const (
	RolePursuer ChaseRole = iota
	RoleQuarry
)

// ChaseParticipant tracks a participant's position and state
type ChaseParticipant struct {
	Entity   *entity.Entity
	Role     ChaseRole
	Position int  // Higher = further ahead (quarry wants high, pursuer wants to match)
	HasActed bool // This round
}

// ChaseObstacle represents a challenge at a specific position
type ChaseObstacle struct {
	Position    int
	Name        string
	Description string
	Skill       ability.SkillID
	AltSkill    ability.SkillID // Alternative skill that can be used
	DC          int
	VPOnSuccess int
	VPOnCrit    int
	Penalty     string // What happens on failure
}

// Chase extends Subsystem with chase-specific mechanics
type Chase struct {
	*Subsystem
	Participants []ChaseParticipant
	Obstacles    []ChaseObstacle

	// Chase-specific settings
	StartingGap    int // Initial position difference
	CatchDistance  int // How close pursuer must get to catch (usually 0)
	EscapeDistance int // How far quarry must get to escape
}

// NewChase creates a chase encounter
// src: rules/rules/gamemastery-guide/chapter-3-subsystems.md (Chases)
func NewChase(id, name string, rounds, startingGap, escapeDistance int) *Chase {
	return &Chase{
		Subsystem:      NewSubsystem(id, name, SubsystemChase, escapeDistance, 0, rounds),
		Participants:   make([]ChaseParticipant, 0),
		Obstacles:      make([]ChaseObstacle, 0),
		StartingGap:    startingGap,
		CatchDistance:  0,
		EscapeDistance: escapeDistance,
	}
}

// AddPursuer adds a pursuer at position 0
func (c *Chase) AddPursuer(e *entity.Entity) {
	c.Participants = append(c.Participants, ChaseParticipant{
		Entity:   e,
		Role:     RolePursuer,
		Position: 0,
	})
}

// AddQuarry adds quarry at starting gap position
func (c *Chase) AddQuarry(e *entity.Entity) {
	c.Participants = append(c.Participants, ChaseParticipant{
		Entity:   e,
		Role:     RoleQuarry,
		Position: c.StartingGap,
	})
}

// AddObstacle adds an obstacle at a position
func (c *Chase) AddObstacle(position int, name, desc string, skillID ability.SkillID, dc, vpSuccess, vpCrit int) {
	c.Obstacles = append(c.Obstacles, ChaseObstacle{
		Position:    position,
		Name:        name,
		Description: desc,
		Skill:       skillID,
		DC:          dc,
		VPOnSuccess: vpSuccess,
		VPOnCrit:    vpCrit,
	})
}

// GetParticipant finds a participant by entity ID
func (c *Chase) GetParticipant(entityID string) *ChaseParticipant {
	for i := range c.Participants {
		if c.Participants[i].Entity.ID == entityID {
			return &c.Participants[i]
		}
	}
	return nil
}

// GetObstacleAt returns obstacle at position, or nil
func (c *Chase) GetObstacleAt(position int) *ChaseObstacle {
	for i := range c.Obstacles {
		if c.Obstacles[i].Position == position {
			return &c.Obstacles[i]
		}
	}
	return nil
}

// ChaseAction represents what a participant does on their turn
type ChaseAction int

const (
	ChaseActionStride ChaseAction = iota // Move 1 position
	ChaseActionDash                      // Athletics check to move extra
	ChaseActionOvercome                  // Handle obstacle
	ChaseActionHinder                    // Slow down opponent
)

// TakeChaseAction resolves a chase action
func (c *Chase) TakeChaseAction(entityID string, action ChaseAction, targetID string, naturalRoll int) ChaseActionResult {
	participant := c.GetParticipant(entityID)
	if participant == nil {
		return ChaseActionResult{Success: false, Description: "Participant not found"}
	}
	if participant.HasActed {
		return ChaseActionResult{Success: false, Description: "Already acted this round"}
	}

	result := ChaseActionResult{EntityID: entityID}

	switch action {
	case ChaseActionStride:
		result = c.resolveStride(participant)
	case ChaseActionDash:
		result = c.resolveDash(participant, naturalRoll)
	case ChaseActionOvercome:
		result = c.resolveOvercome(participant, naturalRoll)
	case ChaseActionHinder:
		target := c.GetParticipant(targetID)
		result = c.resolveHinder(participant, target, naturalRoll)
	}

	participant.HasActed = true
	c.checkChaseEnd()
	return result
}

type ChaseActionResult struct {
	EntityID      string
	Success       bool
	PositionDelta int
	Description   string
	CheckResult   check.CheckResult
}

func (c *Chase) resolveStride(p *ChaseParticipant) ChaseActionResult {
	// Stride always moves 1 position toward goal
	p.Position++
	if p.Role == RoleQuarry {
		return ChaseActionResult{Success: true, PositionDelta: 1, Description: "Moved ahead 1 position"}
	} else {
		return ChaseActionResult{Success: true, PositionDelta: 1, Description: "Closed gap by 1 position"}
	}
}

func (c *Chase) resolveDash(p *ChaseParticipant, naturalRoll int) ChaseActionResult {
	// Athletics check DC 20 to move extra
	dc := 20
	res := skill.PerformSkillCheckWithRoll(p.Entity, ability.SkillAthletics, dc, naturalRoll)

	result := ChaseActionResult{CheckResult: res}

	switch res.Degree {
	case check.CriticalSuccess:
		p.Position += 2
		result.Success = true
		result.PositionDelta = 2
		result.Description = "Dashed 2 positions!"
	case check.Success:
		p.Position += 1
		result.Success = true
		result.PositionDelta = 1
		result.Description = "Dashed 1 position"
	case check.Failure:
		result.Success = false
		result.Description = "Failed to dash, no movement"
	case check.CriticalFailure:
		p.Position--
		if p.Position < 0 {
			p.Position = 0
		}
		result.Success = false
		result.PositionDelta = -1
		result.Description = "Stumbled! Lost 1 position"
	}

	return result
}

func (c *Chase) resolveOvercome(p *ChaseParticipant, naturalRoll int) ChaseActionResult {
	obstacle := c.GetObstacleAt(p.Position)
	if obstacle == nil {
		// No obstacle, just stride
		return c.resolveStride(p)
	}

	res := skill.PerformSkillCheckWithRoll(p.Entity, obstacle.Skill, obstacle.DC, naturalRoll)
	result := ChaseActionResult{CheckResult: res}

	switch res.Degree {
	case check.CriticalSuccess:
		c.CurrentVP += obstacle.VPOnCrit
		p.Position++
		result.Success = true
		result.PositionDelta = 1
		result.Description = fmt.Sprintf("Overcame %s brilliantly! +%d VP", obstacle.Name, obstacle.VPOnCrit)
	case check.Success:
		c.CurrentVP += obstacle.VPOnSuccess
		p.Position++
		result.Success = true
		result.PositionDelta = 1
		result.Description = fmt.Sprintf("Overcame %s. +%d VP", obstacle.Name, obstacle.VPOnSuccess)
	case check.Failure:
		result.Success = false
		result.Description = fmt.Sprintf("Failed to overcome %s", obstacle.Name)
	case check.CriticalFailure:
		c.CurrentVP--
		result.Success = false
		result.Description = fmt.Sprintf("Badly failed %s. -1 VP", obstacle.Name)
	}

	return result
}

func (c *Chase) resolveHinder(actor *ChaseParticipant, target *ChaseParticipant, naturalRoll int) ChaseActionResult {
	if target == nil {
		return ChaseActionResult{Success: false, Description: "No target specified"}
	}

	// Opposed check: actor's choice vs target's Reflex DC
	dc := target.Entity.GetSaveDC(ability.SaveReflex)

	// Can use Athletics, Acrobatics, or Deception
	res := skill.PerformSkillCheckWithRoll(actor.Entity, ability.SkillAthletics, dc, naturalRoll)
	result := ChaseActionResult{CheckResult: res}

	switch res.Degree {
	case check.CriticalSuccess:
		target.Position -= 2
		if target.Position < 0 {
			target.Position = 0
		}
		result.Success = true
		result.Description = fmt.Sprintf("Severely hindered %s! They lose 2 positions", target.Entity.Name)
	case check.Success:
		target.Position--
		if target.Position < 0 {
			target.Position = 0
		}
		result.Success = true
		result.Description = fmt.Sprintf("Hindered %s! They lose 1 position", target.Entity.Name)
	case check.Failure:
		result.Success = false
		result.Description = "Failed to hinder target"
	case check.CriticalFailure:
		actor.Position--
		if actor.Position < 0 {
			actor.Position = 0
		}
		result.Success = false
		result.Description = "Fumbled! Lost 1 position"
	}

	return result
}

// checkChaseEnd determines if chase has concluded
func (c *Chase) checkChaseEnd() {
	// Find closest pursuer and furthest quarry
	maxQuarry := 0
	minPursuer := 9999

	hasQuarry := false
	hasPursuer := false

	for _, p := range c.Participants {
		if p.Role == RoleQuarry {
			hasQuarry = true
			if p.Position > maxQuarry {
				maxQuarry = p.Position
			}
		}
		if p.Role == RolePursuer {
			hasPursuer = true
			if p.Position < minPursuer {
				minPursuer = p.Position
			}
		}
	}

	if !hasQuarry || !hasPursuer {
		return
	}

	gap := maxQuarry - minPursuer

	// Quarry escapes
	if gap >= c.EscapeDistance {
		c.State = StateSuccess // Quarry succeeded
	}

	// Pursuer catches
	if gap <= c.CatchDistance {
		c.State = StateFailure // Quarry caught (pursuer succeeded)
	}
}

// AdvanceChaseRound resets acted flags and advances round
func (c *Chase) AdvanceChaseRound() error {
	for i := range c.Participants {
		c.Participants[i].HasActed = false
	}
	return c.AdvanceRound()
}

// GetGap returns current gap between closest pursuer and furthest quarry
func (c *Chase) GetGap() int {
	maxQuarry := 0
	minPursuer := 9999

	for _, p := range c.Participants {
		if p.Role == RoleQuarry && p.Position > maxQuarry {
			maxQuarry = p.Position
		}
		if p.Role == RolePursuer && p.Position < minPursuer {
			minPursuer = p.Position
		}
	}

	return maxQuarry - minPursuer
}
