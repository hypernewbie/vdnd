package condition

// Registry holds all known condition definitions
type Registry struct {
	conditions map[ConditionID]Condition
}

func NewRegistry() *Registry {
	return &Registry{
		conditions: make(map[ConditionID]Condition),
	}
}

func DefaultRegistry() *Registry {
	r := NewRegistry()

	list := []Condition{
		{Frightened, "Frightened", true, "Status penalty to all checks and DCs."},
		{Sickened, "Sickened", true, "Status penalty to all checks and DCs. Can't ingest."},
		{Clumsy, "Clumsy", true, "Status penalty to DEX-based checks and DCs."},
		{Enfeebled, "Enfeebled", true, "Status penalty to STR-based checks."},
		{Stupefied, "Stupefied", true, "Status penalty to INT/WIS/CHA checks."},
		{Drained, "Drained", true, "Status penalty to CON checks; lose X*level max HP."},
		{FlatFooted, "Flat-footed", false, "-2 circumstance penalty to AC."},
		{Prone, "Prone", false, "Flat-footed, -2 circumstance to attacks."},
		{Grabbed, "Grabbed", false, "Flat-footed, immobilized."},
		{Restrained, "Restrained", false, "Flat-footed, immobilized, restricted actions."},
		{Immobilized, "Immobilized", false, "Can't use move actions."},
		{Blinded, "Blinded", false, "Can't see."},
		{Deafened, "Deafened", false, "Can't hear."},
		{Invisible, "Invisible", false, "Can't be seen."},
		{Hidden, "Hidden", false, "DC 11 flat check to target."},
		{Paralyzed, "Paralyzed", false, "Can't act, flat-footed."},
		{Unconscious, "Unconscious", false, "Can't act, flat-footed, -4 to AC/Perception/Reflex."},
		{Dying, "Dying", true, "Unconscious, dying."},
		{Wounded, "Wounded", true, "Increases dying value when gaining it."},
		{Doomed, "Doomed", true, "Reduces max dying value."},
		{Slowed, "Slowed", true, "Lose actions at start of turn."},
		{Stunned, "Stunned", true, "Lose total actions."},
		{Quickened, "Quickened", false, "Gain 1 extra action."},
		{Fatigued, "Fatigued", false, "-1 status penalty to AC and saves."},
		{Fascinated, "Fascinated", false, "-2 status to Perception and skill checks."},
		{Fleeing, "Fleeing", false, "Must run away."},
		{Confused, "Confused", false, "Attack randomly, flat-footed."},
		{PersistentDamage, "Persistent Damage", true, "Damage at end of turn."},
	}

	for _, c := range list {
		r.conditions[c.ID] = c
	}

	return r
}

func (r *Registry) Get(id ConditionID) (Condition, bool) {
	c, ok := r.conditions[id]
	return c, ok
}
