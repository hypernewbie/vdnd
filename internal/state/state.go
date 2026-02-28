package state

import (
	"fmt"
	"strings"
	"uaa/vdnd/pkg/rules/ability"
)

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

// Validate checks that the GameState has valid structure.
func (g *GameState) Validate() error {
	if g.SceneName == "" {
		return fmt.Errorf("scene name cannot be empty")
	}
	if g.Positions == nil {
		return fmt.Errorf("positions map cannot be nil")
	}
	if g.Entities == nil {
		return fmt.Errorf("entities map cannot be nil")
	}
	if g.ReactionsUsed == nil {
		return fmt.Errorf("reactionsUsed map cannot be nil")
	}
	for id, entity := range g.Entities {
		if entity == nil {
			return fmt.Errorf("entity %s is nil", id)
		}
		// We no longer fail the entire state if one entity has bad stats.
		// Use entity.Validate() for specific checks.
	}
	return nil
}

func (e *EntityState) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("entity ID cannot be empty")
	}
	if e.Name == "" {
		return fmt.Errorf("entity name cannot be empty")
	}
	if e.MaxHP <= 0 {
		return fmt.Errorf("entity %s has invalid MaxHP: %d", e.ID, e.MaxHP)
	}
	if e.AC <= 0 {
		return fmt.Errorf("entity %s has invalid AC: %d", e.ID, e.AC)
	}
	return nil
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

	// Perception (total bonus)
	Perception int `json:"perception"`

	// Skills (total bonus)
	Skills map[string]int `json:"skills,omitempty"`

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

func (e *EntityState) GetAC() int {
	ac := e.AC
	for _, c := range e.Conditions {
		if c.ID == "raised_shield" {
			ac += 2
		}
	}
	return ac
}

func (e *EntityState) GetSkillModifier(skillID string) int {
	skillID = strings.ToLower(skillID)
	if skillID == "perception" {
		return e.Perception
	}
	if val, ok := e.Skills[skillID]; ok {
		return val
	}
	// Fallback to ability modifier + level (assume trained for MVP if not specified?)
	// Actually, let's just use ability modifier if not specified.
	return 0 // For now, if it's not in the map, return 0. 
	// In a real implementation we'd do more complex calculation.
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
	ID       string            `json:"id"`
	Type     string            `json:"type"` // "movement", "strike", "spell"
	ActorID  string            `json:"actorId"`
	TargetID string            `json:"targetId,omitempty"`
	Payload  map[string]string `json:"payload,omitempty"`
	Reactors []AvailableReaction `json:"reactors"`
}

type AvailableReaction struct {
	EntityID string `json:"entityId"`
	Reaction string `json:"reaction"` // "attack_of_opportunity", "shield_block", etc
}
