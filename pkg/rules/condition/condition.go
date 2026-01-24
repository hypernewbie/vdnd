package condition

type ConditionID string

// Common condition IDs
const (
	Frightened       ConditionID = "frightened"
	Sickened         ConditionID = "sickened"
	Clumsy           ConditionID = "clumsy"
	Enfeebled        ConditionID = "enfeebled"
	Stupefied        ConditionID = "stupefied"
	Drained          ConditionID = "drained"
	FlatFooted       ConditionID = "flat-footed"
	Prone            ConditionID = "prone"
	Grabbed          ConditionID = "grabbed"
	Restrained       ConditionID = "restrained"
	Immobilized      ConditionID = "immobilized"
	Blinded          ConditionID = "blinded"
	Deafened         ConditionID = "deafened"
	Invisible        ConditionID = "invisible"
	Hidden           ConditionID = "hidden"
	Paralyzed        ConditionID = "paralyzed"
	Unconscious      ConditionID = "unconscious"
	Dying            ConditionID = "dying"
	Wounded          ConditionID = "wounded"
	Doomed           ConditionID = "doomed"
	Slowed           ConditionID = "slowed"
	Stunned          ConditionID = "stunned"
	Quickened        ConditionID = "quickened"
	Fatigued         ConditionID = "fatigued"
	Fascinated       ConditionID = "fascinated"
	Fleeing          ConditionID = "fleeing"
	Confused         ConditionID = "confused"
	PersistentDamage ConditionID = "persistent-damage"

	// Missing Conditions from Phase 14/Completeness
	Broken     ConditionID = "broken"
	Encumbered ConditionID = "encumbered"
	Petrified  ConditionID = "petrified"

	// Visibility (Relational)
	Concealed  ConditionID = "concealed"
	Dazzled    ConditionID = "dazzled"
	Observed   ConditionID = "observed"
	Undetected ConditionID = "undetected"
	Unnoticed  ConditionID = "unnoticed"

	// Attitudes (Relational)
	Friendly   ConditionID = "friendly"
	Helpful    ConditionID = "helpful"
	Hostile    ConditionID = "hostile"
	Indifferent ConditionID = "indifferent"
	Unfriendly ConditionID = "unfriendly"

	// Cover
	StandardCover ConditionID = "standard-cover"
)

type Condition struct {
	ID          ConditionID
	Name        string
	HasValue    bool // true for valued conditions like Frightened X
	Description string
}
