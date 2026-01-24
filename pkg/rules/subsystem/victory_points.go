package subsystem

import (
	"fmt"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/skill"
)

// SubsystemType identifies the kind of challenge
type SubsystemType string

const (
	SubsystemChase        SubsystemType = "chase"
	SubsystemResearch     SubsystemType = "research"
	SubsystemInfiltration SubsystemType = "infiltration"
	SubsystemInfluence    SubsystemType = "influence"
	SubsystemCustom       SubsystemType = "custom"
)

// SubsystemState tracks the current state
type SubsystemState int

const (
	StateNotStarted SubsystemState = iota
	StateInProgress
	StateSuccess
	StateFailure
)

func (s SubsystemState) String() string {
	switch s {
	case StateNotStarted:
		return "Not Started"
	case StateInProgress:
		return "In Progress"
	case StateSuccess:
		return "Success"
	case StateFailure:
		return "Failure"
	default:
		return "Unknown"
	}
}

// Subsystem represents a Victory Points challenge
// src: rules/rules/gamemastery-guide/chapter-1-gamemastery-basics.md (Victory Points)
type Subsystem struct {
	ID               string
	Name             string
	Type             SubsystemType
	Description      string

	// Victory conditions
	TargetVP         int // VP needed for success
	FailureThreshold int // Negative VP that ends in failure (typically negative)

	// Time limits
	RoundsLimit  int // 0 = no limit
	CurrentRound int

	// Current state
	CurrentVP int
	State     SubsystemState

	// Tracking
	Contributions   map[string]int     // EntityID -> VP contributed
	ChecksAttempted map[string]int     // EntityID -> number of checks made
	RoundHistory    []RoundResult      // History of each round
}

type RoundResult struct {
	Round         int
	Contributions []ContributionResult
	VPDelta       int
	TotalVP       int
}

type ContributionResult struct {
	EntityID    string
	EntityName  string
	SkillUsed   ability.SkillID
	Degree      check.DegreeOfSuccess
	VPEarned    int
	Description string
}

func NewSubsystem(id, name string, subsystemType SubsystemType, targetVP, failureThreshold, roundsLimit int) *Subsystem {
	return &Subsystem{
		ID:               id,
		Name:             name,
		Type:             subsystemType,
		TargetVP:         targetVP,
		FailureThreshold: failureThreshold,
		RoundsLimit:      roundsLimit,
		CurrentRound:     0,
		CurrentVP:        0,
		State:            StateNotStarted,
		Contributions:    make(map[string]int),
		ChecksAttempted:  make(map[string]int),
		RoundHistory:     make([]RoundResult, 0),
	}
}

// Start begins the subsystem
func (s *Subsystem) Start() error {
	if s.State != StateNotStarted {
		return fmt.Errorf("subsystem already started")
	}
	s.State = StateInProgress
	s.CurrentRound = 1
	return nil
}

// Contribute adds VP based on a check result
func (s *Subsystem) Contribute(entityID string, degree check.DegreeOfSuccess, vpOnSuccess, vpOnCritSuccess int) ContributionResult {
	result := ContributionResult{
		EntityID: entityID,
		Degree:   degree,
	}

	switch degree {
	case check.CriticalSuccess:
		result.VPEarned = vpOnCritSuccess
	case check.Success:
		result.VPEarned = vpOnSuccess
	case check.Failure:
		result.VPEarned = 0
	case check.CriticalFailure:
		result.VPEarned = -1 // Lose 1 VP on crit fail
	}

	s.CurrentVP += result.VPEarned
	s.Contributions[entityID] += result.VPEarned
	s.ChecksAttempted[entityID]++

	s.checkCompletion()
	return result
}

// ContributeWithCheck performs a skill check and contributes
func (s *Subsystem) ContributeWithCheck(actor *entity.Entity, skillID ability.SkillID, dc int, vpOnSuccess, vpOnCritSuccess int, naturalRoll int) ContributionResult {
	var res check.CheckResult
	if naturalRoll > 0 {
		res = skill.PerformSkillCheckWithRoll(actor, skillID, dc, naturalRoll)
	} else {
		res = skill.PerformSkillCheck(actor, skillID, dc)
	}

	result := s.Contribute(actor.ID, res.Degree, vpOnSuccess, vpOnCritSuccess)
	result.EntityName = actor.Name
	result.SkillUsed = skillID
	return result
}

// AdvanceRound moves to the next round
func (s *Subsystem) AdvanceRound() error {
	if s.State != StateInProgress {
		return fmt.Errorf("subsystem not in progress")
	}

	s.CurrentRound++

	// Check time limit
	if s.RoundsLimit > 0 && s.CurrentRound > s.RoundsLimit {
		s.State = StateFailure
		return fmt.Errorf("time limit exceeded")
	}

	return nil
}

// checkCompletion checks if the subsystem has ended
func (s *Subsystem) checkCompletion() {
	if s.CurrentVP >= s.TargetVP {
		s.State = StateSuccess
	} else if s.CurrentVP <= s.FailureThreshold {
		s.State = StateFailure
	}
}

// IsComplete returns true if the subsystem has concluded
func (s *Subsystem) IsComplete() bool {
	return s.State == StateSuccess || s.State == StateFailure
}

// IsSuccess returns true if completed successfully
func (s *Subsystem) IsSuccess() bool {
	return s.State == StateSuccess
}

// GetProgress returns current VP as a percentage of target
func (s *Subsystem) GetProgress() float64 {
	if s.TargetVP == 0 {
		return 0
	}
	return float64(s.CurrentVP) / float64(s.TargetVP) * 100
}

// Summary returns a formatted status string
func (s *Subsystem) Summary() string {
	return fmt.Sprintf("%s: %d/%d VP (Round %d/%d) - %s",
		s.Name, s.CurrentVP, s.TargetVP, s.CurrentRound, s.RoundsLimit, s.State.String())
}
