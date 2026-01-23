package state

import "uaa/vdnd/pkg/rules/ability"

type GameState struct {
	// Scene
	SceneName string           `json:"sceneName"`
	Positions map[string]*Zone `json:"positions"`

	// Entities
	Entities map[string]*EntityState `json:"entities"`

	// Combat
	InCombat         bool            `json:"inCombat"`
	Round            int             `json:"round"`
	InitiativeOrder  []string        `json:"initiativeOrder"`
	CurrentTurn      string          `json:"currentTurn"`
	TurnIndex        int             `json:"turnIndex"`
	ActionsRemaining int             `json:"actionsRemaining"`
	ReactionsUsed    map[string]bool `json:"reactionsUsed"`
	AttacksMade      int             `json:"attacksMade"` // For MAP calculation

	// Pending
	PendingEvents []PendingEvent `json:"pendingEvents,omitempty"`
}

type Zone struct {
	Name     string   `json:"name"`
	Size     string   `json:"size"` // small, medium, large
	Adjacent []string `json:"adjacent"`
	Near     []string `json:"near,omitempty"`
	Far      []string `json:"far,omitempty"`
	Cover    string   `json:"cover,omitempty"` // none, lesser, standard, greater
	Elevated bool     `json:"elevated,omitempty"`
	Notes    string   `json:"notes,omitempty"`
}

type EntityState struct {
	// Identity
	ID    string `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`

	// Flavour
	Ancestry   string `json:"ancestry,omitempty"`
	Class      string `json:"class,omitempty"`
	Background string `json:"background,omitempty"`

	// Combat stats
	HP     int `json:"hp"`
	MaxHP  int `json:"maxHp"`
	TempHP int `json:"tempHp,omitempty"`
	AC     int `json:"ac"`
	Speed  int `json:"speed"`

	// Saves (total bonus)
	Fortitude int `json:"fortitude"`
	Reflex    int `json:"reflex"`
	Will      int `json:"will"`

	// Abilities
	Abilities ability.AbilityScores `json:"abilities"`

	// Position
	Position    string   `json:"position"`
	EngagedWith []string `json:"engagedWith,omitempty"`

	// Conditions
	Conditions []ConditionInstance `json:"conditions,omitempty"`

	// Equipment (simplified for state)
	WieldedWeapons []WeaponState `json:"wieldedWeapons,omitempty"`
	WornArmor      *ArmorState   `json:"wornArmor,omitempty"`
	RaisedShield   bool          `json:"raisedShield,omitempty"`

	// Defences
	Immunities  []string       `json:"immunities,omitempty"`
	Weaknesses  map[string]int `json:"weaknesses,omitempty"`
	Resistances map[string]int `json:"resistances,omitempty"`

	// Special
	Reactions []string `json:"reactions,omitempty"` // Available reaction types
}

type ConditionInstance struct {
	ID       string `json:"id"`
	Value    int    `json:"value,omitempty"`
	Duration int    `json:"duration,omitempty"` // -1 = until removed
	Source   string `json:"source,omitempty"`
}

type WeaponState struct {
	ID         string `json:"id"`
	Damage     string `json:"damage"`
	DamageType string `json:"damageType"`
}

type ArmorState struct {
	ID      string `json:"id"`
	ACBonus int    `json:"acBonus"`
}

type PendingEvent struct {
	ID          string              `json:"id"`
	Type        string              `json:"type"` // "movement", "attack", "spell"
	Actor       string              `json:"actor"`
	Description string              `json:"description"`
	Reactors    []string            `json:"reactors"` // Entities that can react
	Reactions   []AvailableReaction `json:"reactions"`
}

type AvailableReaction struct {
	Entity   string `json:"entity"`
	Reaction string `json:"reaction"` // "attack_of_opportunity", "shield_block", etc
	Trigger  string `json:"trigger"`
}
