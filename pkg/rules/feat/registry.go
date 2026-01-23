package feat

import (
	"uaa/vdnd/pkg/rules/ability"
)

var (
	// Toughness - General feat, +HP per level
	Toughness = Feat{
		ID:    "toughness",
		Name:  "Toughness",
		Type:  FeatTypeGeneral,
		Level: 1,
		Passives: []PassiveEffect{{
			Type: PassiveHP,
			Apply: func(e FeatActor) {
				// We need a specific interface method for HP adjustment if we want this to work.
				// For now, these Apply functions might need a more specialized interface.
			},
		}},
	}

	// Fleet - General feat, +5 Speed
	Fleet = Feat{
		ID:    "fleet",
		Name:  "Fleet",
		Type:  FeatTypeGeneral,
		Level: 1,
		Passives: []PassiveEffect{{
			Type: PassiveSpeed,
			Value: 5,
			Apply: func(e FeatActor) {
			},
		}},
	}

	// Attack of Opportunity - Reaction
	AttackOfOpportunity = Feat{
		ID:    "attack-of-opportunity",
		Name:  "Attack of Opportunity",
		Type:  FeatTypeClass,
		Level: 1,
		GrantsReaction: &ReactionGrant{
			Name:    "Attack of Opportunity",
			Trigger: ability.EventManipulate,
			Condition: func(event ability.Event, reactor FeatActor) bool {
				return false
			},
			Execute: func(event ability.Event, reactor FeatActor, enc Encounter) ability.ActionResult {
				return ability.ActionResult{Success: true, Description: "AoO Triggered"}
			},
		},
	}
)

var feats = map[string]*Feat{
	"toughness":             &Toughness,
	"fleet":                 &Fleet,
	"attack-of-opportunity": &AttackOfOpportunity,
}

func GetFeat(id string) (*Feat, bool) {
	f, ok := feats[id]
	return f, ok
}

func AllFeats() []*Feat {
	all := make([]*Feat, 0, len(feats))
	for _, f := range feats {
		all = append(all, f)
	}
	return all
}