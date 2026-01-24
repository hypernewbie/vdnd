# Phase 22: Gamemastery Subsystems (Victory Points)

## Objective

Implement the Victory Points framework from the Gamemastery Guide. This provides a generic system for tracking progress in non-combat challenges like Chases, Research, Infiltration, and Social Influence encounters.

---

## 1. Victory Points Core Framework

**Target File:** `pkg/rules/subsystem/victory_points.go`

```go
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

// Subsystem represents a Victory Points challenge
// src: rules/rules/gamemastery-guide/chapter-1-gamemastery-basics.md (Victory Points)
type Subsystem struct {
    ID               string
    Name             string
    Type             SubsystemType
    Description      string
    
    // Victory conditions
    TargetVP         int  // VP needed for success
    FailureThreshold int  // Negative VP that ends in failure (typically negative)
    
    // Time limits
    RoundsLimit      int  // 0 = no limit
    CurrentRound     int
    
    // Current state
    CurrentVP        int
    State            SubsystemState
    
    // Tracking
    Contributions    map[string]int        // EntityID -> VP contributed
    ChecksAttempted  map[string]int        // EntityID -> number of checks made
    RoundHistory     []RoundResult         // History of each round
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
        s.Name, s.CurrentVP, s.TargetVP, s.CurrentRound, s.RoundsLimit, s.State)
}
```

---

## 2. Chase Subsystem

**Target File:** `pkg/rules/subsystem/chase.go`

Chases are VP subsystems with position tracking and obstacles.

```go
package subsystem

import (
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
    StartingGap   int  // Initial position difference
    CatchDistance int  // How close pursuer must get to catch (usually 0)
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
func (c *Chase) AddObstacle(position int, name, desc string, skill ability.SkillID, dc, vpSuccess, vpCrit int) {
    c.Obstacles = append(c.Obstacles, ChaseObstacle{
        Position:    position,
        Name:        name,
        Description: desc,
        Skill:       skill,
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
    EntityID     string
    Success      bool
    PositionDelta int
    Description  string
    CheckResult  check.CheckResult
}

func (c *Chase) resolveStride(p *ChaseParticipant) ChaseActionResult {
    // Stride always moves 1 position toward goal
    if p.Role == RoleQuarry {
        p.Position++
        return ChaseActionResult{Success: true, PositionDelta: 1, Description: "Moved ahead 1 position"}
    } else {
        p.Position++
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
    
    for _, p := range c.Participants {
        if p.Role == RoleQuarry && p.Position > maxQuarry {
            maxQuarry = p.Position
        }
        if p.Role == RolePursuer && p.Position < minPursuer {
            minPursuer = p.Position
        }
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
```

---

## 3. Research Subsystem

**Target File:** `pkg/rules/subsystem/research.go`

Research allows characters to uncover information over time.

```go
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
    Topics           []ResearchTopic
    AvailableSkills  []ability.SkillID // Skills that can be used
    MaxChecksPerDay  int                // Limit on daily checks
    ChecksToday      map[string]int     // EntityID -> checks made today
}

// NewResearch creates a research subsystem
// src: rules/rules/gamemastery-guide/chapter-3-subsystems.md (Research)
func NewResearch(id, name string, totalVP int) *Research {
    return &Research{
        Subsystem:       NewSubsystem(id, name, SubsystemResearch, totalVP, -10, 0),
        Topics:          make([]ResearchTopic, 0),
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
    r.updateTopics()
    for _, topic := range r.Topics {
        if topic.Uncovered && topic.VPCurrent == topic.VPRequired {
            result.TopicUncovered = &topic
            break
        }
    }
    
    return result
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
```

---

## 4. Influence Subsystem

**Target File:** `pkg/rules/subsystem/influence.go`

Social influence encounters for convincing NPCs.

```go
package subsystem

import (
    "uaa/vdnd/pkg/rules/ability"
    "uaa/vdnd/pkg/rules/check"
    "uaa/vdnd/pkg/rules/entity"
)

// InfluenceTarget represents an NPC to influence
type InfluenceTarget struct {
    Entity       *entity.Entity
    CurrentVP    int
    InfluenceVP  int      // VP needed to influence
    Resistances  []ability.SkillID // Skills they resist
    Weaknesses   []ability.SkillID // Skills that work well
    Discovery    InfluenceDiscovery
}

// InfluenceDiscovery tracks what PCs know about the target
type InfluenceDiscovery struct {
    Discovered       bool
    ResistancesKnown bool
    WeaknessesKnown  bool
}

// Influence tracks a social influence encounter
type Influence struct {
    *Subsystem
    Targets        []InfluenceTarget
    RoundsPerPhase int // Social rounds before checking progress
}

// NewInfluence creates an influence encounter
// src: rules/rules/gamemastery-guide/chapter-3-subsystems.md (Influence)
func NewInfluence(id, name string, rounds int) *Influence {
    return &Influence{
        Subsystem:      NewSubsystem(id, name, SubsystemInfluence, 0, -99, rounds),
        Targets:        make([]InfluenceTarget, 0),
        RoundsPerPhase: 3,
    }
}

// AddTarget adds an NPC target
func (inf *Influence) AddTarget(e *entity.Entity, vpNeeded int, resists, weaknesses []ability.SkillID) {
    inf.Targets = append(inf.Targets, InfluenceTarget{
        Entity:      e,
        InfluenceVP: vpNeeded,
        Resistances: resists,
        Weaknesses:  weaknesses,
    })
    inf.TargetVP += vpNeeded // Total VP is sum of all targets
}

// Discover attempts to learn about a target
func (inf *Influence) Discover(actor *entity.Entity, targetID string, skillID ability.SkillID, naturalRoll int) DiscoverResult {
    target := inf.getTarget(targetID)
    if target == nil {
        return DiscoverResult{Description: "Target not found"}
    }
    
    dc := target.Entity.GetSaveDC(ability.SaveWill)
    res := ContributeWithCheck(inf.Subsystem, actor, skillID, dc, 0, 0, naturalRoll)
    
    result := DiscoverResult{
        CheckResult: res,
    }
    
    switch res.Degree {
    case check.CriticalSuccess:
        target.Discovery.Discovered = true
        target.Discovery.ResistancesKnown = true
        target.Discovery.WeaknessesKnown = true
        result.Learned = "Learned all resistances and weaknesses"
    case check.Success:
        target.Discovery.Discovered = true
        // Learn one or the other
        if len(target.Weaknesses) > 0 {
            target.Discovery.WeaknessesKnown = true
            result.Learned = "Learned weaknesses"
        } else {
            target.Discovery.ResistancesKnown = true
            result.Learned = "Learned resistances"
        }
    case check.Failure:
        result.Description = "Failed to learn anything useful"
    case check.CriticalFailure:
        result.Description = "Target is now suspicious"
    }
    
    return result
}

type DiscoverResult struct {
    CheckResult ContributionResult
    Learned     string
    Description string
}

// InfluenceTarget attempts to sway a target
func (inf *Influence) InfluenceTarget(actor *entity.Entity, targetID string, skillID ability.SkillID, naturalRoll int) InfluenceResult {
    target := inf.getTarget(targetID)
    if target == nil {
        return InfluenceResult{Description: "Target not found"}
    }
    
    dc := target.Entity.GetSaveDC(ability.SaveWill)
    
    // Apply modifiers for weaknesses/resistances
    modifier := 0
    for _, r := range target.Resistances {
        if r == skillID {
            modifier -= 2 // Harder if they resist this approach
            break
        }
    }
    for _, w := range target.Weaknesses {
        if w == skillID {
            modifier += 2 // Easier if this is their weakness
            break
        }
    }
    
    effectiveDC := dc - modifier // Lower DC = easier
    res := ContributeWithCheck(inf.Subsystem, actor, skillID, effectiveDC, 1, 2, naturalRoll)
    
    target.CurrentVP += res.VPEarned
    
    result := InfluenceResult{
        CheckResult: res,
        VPEarned:    res.VPEarned,
        TargetVP:    target.CurrentVP,
        TargetMax:   target.InfluenceVP,
    }
    
    if target.CurrentVP >= target.InfluenceVP {
        result.Influenced = true
        result.Description = fmt.Sprintf("%s has been influenced!", target.Entity.Name)
    }
    
    return result
}

type InfluenceResult struct {
    CheckResult ContributionResult
    VPEarned    int
    TargetVP    int
    TargetMax   int
    Influenced  bool
    Description string
}

func (inf *Influence) getTarget(entityID string) *InfluenceTarget {
    for i := range inf.Targets {
        if inf.Targets[i].Entity.ID == entityID {
            return &inf.Targets[i]
        }
    }
    return nil
}

// GetInfluencedCount returns how many targets have been influenced
func (inf *Influence) GetInfluencedCount() int {
    count := 0
    for _, t := range inf.Targets {
        if t.CurrentVP >= t.InfluenceVP {
            count++
        }
    }
    return count
}
```

---

## 5. Tests

**Target File:** `pkg/rules/subsystem/subsystem_test.go`

```go
package subsystem_test

import (
    "testing"
    "uaa/vdnd/pkg/rules/ability"
    "uaa/vdnd/pkg/rules/check"
    "uaa/vdnd/pkg/rules/entity"
    "uaa/vdnd/pkg/rules/subsystem"
)

func TestVictoryPointsBasic(t *testing.T) {
    sub := subsystem.NewSubsystem("test", "Test Challenge", subsystem.SubsystemCustom, 10, -5, 5)
    
    if err := sub.Start(); err != nil {
        t.Fatalf("Failed to start: %v", err)
    }
    
    // Contribute some VP
    result := sub.Contribute("player1", check.Success, 2, 3)
    
    if result.VPEarned != 2 {
        t.Errorf("Expected 2 VP, got %d", result.VPEarned)
    }
    if sub.CurrentVP != 2 {
        t.Errorf("Expected current VP 2, got %d", sub.CurrentVP)
    }
    
    // Crit success
    result = sub.Contribute("player2", check.CriticalSuccess, 2, 3)
    if result.VPEarned != 3 {
        t.Errorf("Expected 3 VP on crit, got %d", result.VPEarned)
    }
}

func TestVictoryPointsCompletion(t *testing.T) {
    sub := subsystem.NewSubsystem("test", "Test", subsystem.SubsystemCustom, 5, -3, 10)
    sub.Start()
    
    // Reach target
    sub.Contribute("p1", check.CriticalSuccess, 2, 5)
    
    if !sub.IsComplete() {
        t.Error("Should be complete at target VP")
    }
    if !sub.IsSuccess() {
        t.Error("Should be successful")
    }
}

func TestVictoryPointsFailure(t *testing.T) {
    sub := subsystem.NewSubsystem("test", "Test", subsystem.SubsystemCustom, 10, -3, 10)
    sub.Start()
    
    // Three crit failures = -3 VP
    sub.Contribute("p1", check.CriticalFailure, 1, 2)
    sub.Contribute("p1", check.CriticalFailure, 1, 2)
    sub.Contribute("p1", check.CriticalFailure, 1, 2)
    
    if !sub.IsComplete() {
        t.Error("Should be complete at failure threshold")
    }
    if sub.IsSuccess() {
        t.Error("Should not be successful")
    }
}

func TestChase(t *testing.T) {
    chase := subsystem.NewChase("chase1", "Rooftop Chase", 10, 3, 8)
    
    pursuer := entity.NewEntity("guard", "City Guard", 5)
    pursuer.SkillProficiencies[ability.SkillAthletics] = ability.Trained
    
    quarry := entity.NewEntity("thief", "Nimble Thief", 4)
    quarry.SkillProficiencies[ability.SkillAcrobatics] = ability.Expert
    
    chase.AddPursuer(pursuer)
    chase.AddQuarry(quarry)
    chase.Start()
    
    if chase.GetGap() != 3 {
        t.Errorf("Initial gap should be 3, got %d", chase.GetGap())
    }
    
    // Quarry strides
    result := chase.TakeChaseAction("thief", subsystem.ChaseActionStride, "", 0)
    if result.PositionDelta != 1 {
        t.Errorf("Stride should move 1, got %d", result.PositionDelta)
    }
    
    if chase.GetGap() != 4 {
        t.Errorf("Gap should be 4 after quarry stride, got %d", chase.GetGap())
    }
}

func TestResearch(t *testing.T) {
    research := subsystem.NewResearch("library", "Ancient Library", 10)
    research.AddTopic("dragons", "Dragon Origins", "What created dragons?", "Dragons were born from primordial chaos", 3)
    research.AddTopic("weakness", "Dragon Weakness", "How to defeat dragons?", "Cold iron disrupts their magic", 5)
    research.Start()
    
    scholar := entity.NewEntity("scholar", "Scholar", 6)
    scholar.SkillProficiencies[ability.SkillArcana] = ability.Expert
    
    // First research check
    result := research.Research(scholar, ability.SkillArcana, 15, 18)
    
    t.Logf("Research result: %d VP earned", result.VPEarned)
    
    if !research.CanResearch(scholar.ID) {
        // Should still be able to research (max 2 per day)
        if research.ChecksToday[scholar.ID] >= research.MaxChecksPerDay {
            t.Log("At max checks, expected")
        }
    }
}

func TestInfluence(t *testing.T) {
    influence := subsystem.NewInfluence("council", "City Council Meeting", 5)
    
    mayor := entity.NewEntity("mayor", "Mayor Thornwood", 8)
    influence.AddTarget(mayor, 4, 
        []ability.SkillID{ability.SkillIntimidation}, // Resists intimidation
        []ability.SkillID{ability.SkillDiplomacy})    // Weak to diplomacy
    
    influence.Start()
    
    diplomat := entity.NewEntity("bard", "Silver-Tongued Bard", 6)
    diplomat.SkillProficiencies[ability.SkillDiplomacy] = ability.Expert
    
    // Discover first
    discResult := influence.Discover(diplomat, "mayor", ability.SkillSociety, 15)
    t.Logf("Discovery: %s", discResult.Learned)
    
    // Try to influence with weakness skill
    infResult := influence.InfluenceTarget(diplomat, "mayor", ability.SkillDiplomacy, 18)
    t.Logf("Influence: %d/%d VP", infResult.TargetVP, infResult.TargetMax)
}
```

---

## 6. Execution Checklist

- [ ] Create `pkg/rules/subsystem/victory_points.go` with core Subsystem
- [ ] Implement `Contribute()` and `ContributeWithCheck()`
- [ ] Create `pkg/rules/subsystem/chase.go` with Chase mechanics
- [ ] Implement chase actions: Stride, Dash, Overcome, Hinder
- [ ] Create `pkg/rules/subsystem/research.go` with Research mechanics
- [ ] Implement topic discovery and daily limits
- [ ] Create `pkg/rules/subsystem/influence.go` with Influence mechanics
- [ ] Implement discovery and influence checks with weakness/resistance
- [ ] Create `pkg/rules/subsystem/subsystem_test.go`
- [ ] Run `go test -v ./pkg/rules/...` and ensure 100% pass

---

## 7. CLI Commands

```bash
# Create a chase
vd subsystem chase create rooftop_chase --rounds 10 --gap 3 --escape 8
vd subsystem chase add_pursuer rooftop_chase guard_1
vd subsystem chase add_quarry rooftop_chase thief_1
vd subsystem chase start rooftop_chase

# Take chase actions
vd subsystem chase action rooftop_chase thief_1 stride
vd subsystem chase action rooftop_chase guard_1 dash
vd subsystem chase status rooftop_chase

# Output:
# **Rooftop Chase** (Round 2/10)
# | Participant | Role | Position |
# |-------------|------|----------|
# | Nimble Thief | Quarry | 5 |
# | City Guard | Pursuer | 2 |
# **Gap:** 3 positions
# **Escape at:** 8 | **Catch at:** 0

# Research
vd subsystem research create library_research --target 10
vd subsystem research add_topic library_research dragons "Dragon Origins" 3
vd subsystem research check library_research scholar_1 arcana
vd subsystem research status library_research

# Output:
# **Ancient Library Research** (5/10 VP)
# **Topics:**
# - [UNCOVERED] Dragon Origins (3 VP) - "Dragons were born from primordial chaos"
# - [2/5] Dragon Weakness

# Influence
vd subsystem influence create council_meeting --rounds 5
vd subsystem influence add_target council_meeting mayor --vp 4 --resists intimidation --weak diplomacy
vd subsystem influence discover council_meeting bard_1 mayor society
vd subsystem influence sway council_meeting bard_1 mayor diplomacy

# Output:
# **Influence Check**
# Target: Mayor Thornwood
# Skill: Diplomacy (+2 weakness bonus)
# DC: 18 (adjusted from 20)
# Roll: 17 + 10 = 27 (Critical Success)
# **VP Earned:** 2
# **Target Progress:** 2/4
```
