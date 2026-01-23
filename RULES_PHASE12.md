# Phase 12: Encounter Management

## Agent Prompt

You are implementing the encounter management system for a Pathfinder 2E rules engine in Go. This handles initiative, turn order, round structure, and the overall flow of combat encounters.

**Your task:** Implement the `pkg/rules/encounter` package with full test coverage.

**Prerequisites:** Phases 1-8 should be complete (especially combat/turn management).

**Note:** This is M5 in the milestone plan—it comes before spells/skills because combat flow is needed to test actions properly.

---

## Context

### Source References
- Modes of play: `rules/rules/core-rulebook/chapter-9-playing-the-game.md:15`
- Initiative: `rules/rules/core-rulebook/chapter-9-playing-the-game.md:283`
- Rounds & turns: `rules/rules/core-rulebook/chapter-9-playing-the-game.md:468`

### Encounter Structure

```
1. ENCOUNTER START
   - GM calls for initiative
   - All participants roll initiative
   - Sort by result (highest first), break ties

2. ROUND (repeats until encounter ends)
   - Each participant takes their TURN in initiative order
   - Per turn: 3 actions + 1 reaction (modified by conditions)
   - Reactions can trigger on other creatures' turns
   - End-of-round effects resolve

3. ENCOUNTER END
   - When one side is defeated, flees, or objective met
   - Clean up temporary effects
```

### Initiative
- Usually: **Perception check** (d20 + Perception modifier)
- Some abilities use other skills (Stealth for ambushes, Deception for social encounters)
- Ties: GM decides, or higher modifier goes first, or players choose among tied PCs

### Delays and Readied Actions
- **Delay:** Drop in initiative order, take turn later (above creature that just acted)
- **Ready:** Prepare an action with a trigger, use reaction when trigger occurs

---

## File Structure

```
pkg/
└── rules/
    └── encounter/
        ├── encounter.go    # Encounter struct, lifecycle
        ├── initiative.go   # Initiative rolling, ordering
        ├── round.go        # Round/turn progression
        ├── events.go       # Event system for reactions
        └── encounter_test.go
```

---

## Implementation Plan

### 1. `pkg/rules/encounter/encounter.go`

```go
type EncounterState int
const (
    StateNotStarted EncounterState = iota
    StateRollingInitiative
    StateInProgress
    StateEnded
)

type Encounter struct {
    ID           string
    State        EncounterState
    Participants []*Participant
    CurrentRound int
    CurrentTurn  int  // Index into sorted initiative order
    
    // Event system for reactions
    EventQueue   *EventQueue
}

type Participant struct {
    Entity      *entity.Entity
    Initiative  int
    HasActed    bool  // This round
    IsDelaying  bool
    TurnState   *combat.TurnState
}

func NewEncounter(id string) *Encounter

// AddParticipant adds an entity to the encounter
func (e *Encounter) AddParticipant(ent *entity.Entity)

// RemoveParticipant removes an entity (fled, died, etc.)
func (e *Encounter) RemoveParticipant(entityID string)

// Start begins the encounter, rolls initiative, sorts order
func (e *Encounter) Start() error

// GetCurrentParticipant returns whose turn it is
func (e *Encounter) GetCurrentParticipant() *Participant

// NextTurn advances to the next participant's turn
func (e *Encounter) NextTurn() error

// EndEncounter marks the encounter as finished
func (e *Encounter) EndEncounter()

// IsOver returns true if encounter has ended
func (e *Encounter) IsOver() bool
```

### 2. `pkg/rules/encounter/initiative.go`

```go
type InitiativeType int
const (
    InitPerception InitiativeType = iota
    InitStealth
    InitDeception
    // etc.
)

// RollInitiative rolls initiative for all participants
func (e *Encounter) RollInitiative(initType InitiativeType) error

// RollInitiativeFor rolls initiative for a single entity
func RollInitiativeFor(ent *entity.Entity, initType InitiativeType) int

// SortByInitiative orders participants from highest to lowest
func (e *Encounter) SortByInitiative()

// ResolveTie determines order when two participants have same initiative
func ResolveTie(a, b *Participant) int  // -1 if a first, 1 if b first
```

**RollInitiative Pseudocode:**
```
func (e *Encounter) RollInitiative(initType InitiativeType) error:
    for _, p := range e.Participants:
        switch initType:
        case InitPerception:
            mod := p.Entity.GetPerception()
        case InitStealth:
            mod := p.Entity.GetSkillModifier(Stealth)
        // etc.
        
        roll := dice.Roll(dice.D20)
        p.Initiative = roll + mod
    
    e.SortByInitiative()
    e.State = StateInProgress
    e.CurrentRound = 1
    e.CurrentTurn = 0
    return nil
```

**SortByInitiative Pseudocode:**
```
func (e *Encounter) SortByInitiative():
    sort.SliceStable(e.Participants, func(i, j int) bool:
        if e.Participants[i].Initiative != e.Participants[j].Initiative:
            return e.Participants[i].Initiative > e.Participants[j].Initiative
        # Tie-breaker: higher modifier goes first
        return ResolveTie(e.Participants[i], e.Participants[j]) < 0
    )
```

### 3. `pkg/rules/encounter/round.go`

```go
// StartRound begins a new round
func (e *Encounter) StartRound() error

// StartTurn begins the current participant's turn
func (e *Encounter) StartTurn() (*combat.TurnState, error)

// EndTurn ends the current participant's turn
func (e *Encounter) EndTurn() error

// Delay causes current participant to delay their turn
func (e *Encounter) Delay() error

// ResumeFromDelay lets a delaying participant take their turn
func (e *Encounter) ResumeFromDelay(entityID string) error
```

**StartTurn Pseudocode:**
```
func (e *Encounter) StartTurn() (*combat.TurnState, error):
    p := e.GetCurrentParticipant()
    if p == nil:
        return nil, errors.New("no current participant")
    
    # Skip if delaying
    if p.IsDelaying:
        return nil, errors.New("participant is delaying")
    
    # Create turn state
    turn := combat.NewTurn(p.Entity)
    p.TurnState = turn
    
    # Process start-of-turn effects
    p.Entity.Conditions.StartTurn()
    
    # Refresh reaction
    turn.ReactionUsed = false
    
    return turn, nil
```

**EndTurn Pseudocode:**
```
func (e *Encounter) EndTurn() error:
    p := e.GetCurrentParticipant()
    
    # Process end-of-turn effects
    p.Entity.Conditions.EndTurn()
    
    # Persistent damage checks happen here
    e.ProcessPersistentDamage(p.Entity)
    
    p.HasActed = true
    p.TurnState = nil
    
    # Advance to next turn
    e.CurrentTurn++
    
    # Check if round is over
    if e.CurrentTurn >= len(e.Participants):
        e.EndRound()
    
    return nil
```

**EndRound / StartRound:**
```
func (e *Encounter) EndRound():
    # Reset HasActed for all
    for _, p := range e.Participants:
        p.HasActed = false
    
    e.CurrentRound++
    e.CurrentTurn = 0

func (e *Encounter) StartRound() error:
    # Handle round-start effects if any
    return nil
```

### 4. `pkg/rules/encounter/events.go`

Event system for triggering reactions.

```go
type EventType int
const (
    EventMove EventType = iota       // Entity moved
    EventManipulate                  // Manipulate action used
    EventConcentrate                 // Concentrate action used  
    EventStrike                      // Strike made
    EventCast                        // Spell cast
    EventDamaged                     // Entity took damage
)

type Event struct {
    Type     EventType
    Actor    *entity.Entity
    Target   *entity.Entity
    Position string  // For movement events
    Details  map[string]interface{}
}

type EventQueue struct {
    events   []Event
    handlers map[EventType][]EventHandler
}

type EventHandler func(event Event, encounter *Encounter) bool  // Returns true if handled

// Emit adds an event to the queue
func (q *EventQueue) Emit(event Event)

// Process handles all queued events, checking for reactions
func (q *EventQueue) Process(encounter *Encounter)

// RegisterHandler adds a handler for an event type (for reactions)
func (q *EventQueue) RegisterHandler(eventType EventType, handler EventHandler)
```

**Example: Attack of Opportunity**
```
func AttackOfOpportunityHandler(event Event, encounter *Encounter) bool:
    if event.Type != EventManipulate && event.Type != EventMove:
        return false
    
    # Find creatures with AoO that are engaged with the target
    for _, p := range encounter.Participants:
        if p.Entity.HasAbility("Attack of Opportunity"):
            if p.Entity.IsEngagedWith(event.Actor.ID):
                if !p.TurnState.ReactionUsed:
                    # Offer reaction
                    # If taken, perform Strike against event.Actor
                    return true
    
    return false
```

---

## Test Plan

### Encounter Lifecycle Tests

| Test | Action | Expected |
|------|--------|----------|
| Create encounter | NewEncounter("combat1") | State = NotStarted |
| Add participants | AddParticipant x 4 | 4 participants |
| Start encounter | Start() | State = InProgress, initiatives rolled |
| Get current | GetCurrentParticipant | Highest initiative |
| End encounter | EndEncounter() | State = Ended |

### Initiative Tests

| Test | Participants | Rolls | Expected Order |
|------|--------------|-------|----------------|
| Simple ordering | A, B, C | 15, 20, 10 | B, A, C |
| Tie with different mods | A (+5), B (+3) | Both roll 15 | A first (+5 > +3) |
| Same roll, same mod | A, B both 15, both +5 | Either order (stable) | - |

### Turn Progression Tests

| Test | Setup | Action | Expected |
|------|-------|--------|----------|
| First turn | Start encounter | StartTurn | TurnState for first participant |
| End turn advances | On turn 0 | EndTurn | CurrentTurn = 1 |
| Round wraps | 3 participants, on turn 2 | EndTurn | CurrentRound = 2, CurrentTurn = 0 |
| HasActed set | EndTurn | | HasActed = true for that participant |
| HasActed reset | End round | | All HasActed = false |

### Delay Tests

| Test | Setup | Action | Expected |
|------|-------|--------|----------|
| Delay sets flag | During turn | Delay() | IsDelaying = true, advances turn |
| Resume from delay | B is delaying, A ends turn | ResumeFromDelay("B") | B goes, then C |
| Can't act while delaying | IsDelaying = true | StartTurn | Error |

### Condition Integration Tests

| Test | Condition | Expected |
|------|-----------|----------|
| Stunned loses actions | Stunned 2 | NewTurn returns 1 action (3 - 2) |
| Slowed 1 | Slowed 1 | 2 actions |
| Quickened | Quickened | 4 actions |
| Frightened reduces at end | Frightened 2 | After EndTurn: Frightened 1 |

### Event System Tests

| Test | Event | Handler Registered | Expected |
|------|-------|-------------------|----------|
| Move event emitted | Stride action | - | Event in queue |
| Handler called | Manipulate event | AoO handler | Handler invoked |
| Reaction consumed | AoO triggered | - | TurnState.ReactionUsed = true |

---

## Validation Checklist

- [ ] All tests pass
- [ ] `go vet` reports no issues  
- [ ] `go fmt` applied
- [ ] Initiative sorts correctly with tie-breaking
- [ ] Turn order cycles through all participants
- [ ] Rounds increment when all have acted
- [ ] Delay/resume works correctly
- [ ] Start/end turn triggers condition effects
- [ ] Event system can trigger reactions

---

## Notes for Implementation

1. **Initiative ties:** Rules say GM decides. For automation, use higher modifier as tiebreaker, then entity ID for determinism.

2. **Delay position:** When resuming from delay, you go immediately after the trigger (not at your old position).

3. **Ready action:** More complex—stores an action with a trigger. Not implementing in Phase 12, defer to later.

4. **Persistent damage:** At end of turn, take persistent damage, then roll DC 15 flat check to end it. This is part of EndTurn.

5. **Reactions:** The event system enables reactions but actual reaction abilities (AoO, Shield Block) are implemented separately. This phase just provides the hooks.

6. **Removing participants:** When a creature dies or flees, remove from initiative. Handle gracefully if it was their turn.
