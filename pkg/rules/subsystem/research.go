package subsystem

import (
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/entity"
)

// ResearchTopic represents a piece of information to uncover
type ResearchTopic struct {
	ID          string
	Name        string
	Description string
	VPRequired  int    // VP needed to uncover this topic
	VPCurrent   int    // Progress toward this topic
	Uncovered   bool
	SecretInfo  string // Revealed when uncovered
}

// Research extends Subsystem for library/investigation challenges
type Research struct {
	*Subsystem
	Topics          []ResearchTopic
	AvailableSkills []ability.SkillID // Skills that can be used
	MaxChecksPerDay int               // Limit on daily checks
	ChecksToday     map[string]int    // EntityID -> checks made today
}

// NewResearch creates a research subsystem
// src: rules/rules/gamemastery-guide/chapter-3-subsystems.md (Research)
func NewResearch(id, name string, totalVP int) *Research {
	return &Research{
		Subsystem: NewSubsystem(id, name, SubsystemResearch, totalVP, -10, 0),
		Topics:    make([]ResearchTopic, 0),
		AvailableSkills: []ability.SkillID{
			ability.SkillArcana,
			ability.SkillNature,
			ability.SkillOccultism,
			ability.SkillReligion,
			ability.SkillSociety,
		},
		MaxChecksPerDay: 2,
		ChecksToday:     make(map[string]int),
	}
}

// AddTopic adds a research topic
func (r *Research) AddTopic(id, name, description, secret string, vpRequired int) {
	r.Topics = append(r.Topics, ResearchTopic{
		ID:          id,
		Name:        name,
		Description: description,
		VPRequired:  vpRequired,
		SecretInfo:  secret,
	})
}

// CanResearch checks if entity can make another check today
func (r *Research) CanResearch(entityID string) bool {
	return r.ChecksToday[entityID] < r.MaxChecksPerDay
}

// Research performs a research check
func (r *Research) Research(actor *entity.Entity, skillID ability.SkillID, dc int, naturalRoll int) ResearchResult {
	result := ResearchResult{EntityID: actor.ID, Skill: skillID}

	if !r.CanResearch(actor.ID) {
		result.Description = "Already made maximum checks today"
		return result
	}

	// Validate skill is allowed
	allowed := false
	for _, s := range r.AvailableSkills {
		if s == skillID {
			allowed = true
			break
		}
	}
	if !allowed {
		result.Description = "Skill not applicable to this research"
		return result
	}

	// Perform check and contribute
	contribution := r.ContributeWithCheck(actor, skillID, dc, 1, 2, naturalRoll)
	r.ChecksToday[actor.ID]++

	result.Contribution = contribution
	result.VPEarned = contribution.VPEarned

	// Check if any topics are uncovered
	uncoveredBefore := r.getUncoveredCount()
	r.updateTopics()
	uncoveredAfter := r.getUncoveredCount()

	if uncoveredAfter > uncoveredBefore {
		// Topic was uncovered this check
		for _, topic := range r.Topics {
			if topic.Uncovered && topic.VPCurrent == topic.VPRequired {
				// This might be ambiguous if multiple topics uncovered, 
				// but research is usually sequential.
				result.TopicUncovered = &topic
			}
		}
	}

	return result
}

func (r *Research) getUncoveredCount() int {
	count := 0
	for _, t := range r.Topics {
		if t.Uncovered {
			count++
		}
	}
	return count
}

type ResearchResult struct {
	EntityID       string
	Skill          ability.SkillID
	Contribution   ContributionResult
	VPEarned       int
	TopicUncovered *ResearchTopic
	Description    string
}

// updateTopics allocates VP to topics in order
func (r *Research) updateTopics() {
	remaining := r.CurrentVP
	for i := range r.Topics {
		if r.Topics[i].Uncovered {
			remaining -= r.Topics[i].VPRequired
			continue
		}

		needed := r.Topics[i].VPRequired - r.Topics[i].VPCurrent
		if remaining >= needed {
			r.Topics[i].VPCurrent = r.Topics[i].VPRequired
			r.Topics[i].Uncovered = true
			remaining -= needed
		} else if remaining > 0 {
			r.Topics[i].VPCurrent += remaining
			remaining = 0
		}
	}
}

// NewDay resets daily check limits
func (r *Research) NewDay() {
	r.ChecksToday = make(map[string]int)
}

// GetUncoveredTopics returns all topics that have been researched
func (r *Research) GetUncoveredTopics() []ResearchTopic {
	uncovered := make([]ResearchTopic, 0)
	for _, t := range r.Topics {
		if t.Uncovered {
			uncovered = append(uncovered, t)
		}
	}
	return uncovered
}
