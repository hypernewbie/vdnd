package feat

// FeatTracker manages feats for an entity
type FeatTracker struct {
	feats map[string]*Feat
}

func NewFeatTracker() *FeatTracker {
	return &FeatTracker{
		feats: make(map[string]*Feat),
	}
}

func (t *FeatTracker) Add(feat *Feat) {
	t.feats[feat.ID] = feat
}

func (t *FeatTracker) Has(featID string) bool {
	_, ok := t.feats[featID]
	return ok
}

func (t *FeatTracker) Get(featID string) *Feat {
	return t.feats[featID]
}

func (t *FeatTracker) All() []*Feat {
	all := make([]*Feat, 0, len(t.feats))
	for _, f := range t.feats {
		all = append(all, f)
	}
	return all
}

// GetGrantedActions returns all actions from feats
func (t *FeatTracker) GetGrantedActions() []*ActionGrant {
	actions := make([]*ActionGrant, 0)
	for _, f := range t.feats {
		if f.GrantsAction != nil {
			actions = append(actions, f.GrantsAction)
		}
	}
	return actions
}

// GetGrantedReactions returns all reactions from feats
func (t *FeatTracker) GetGrantedReactions() []*ReactionGrant {
	reactions := make([]*ReactionGrant, 0)
	for _, f := range t.feats {
		if f.GrantsReaction != nil {
			reactions = append(reactions, f.GrantsReaction)
		}
	}
	return reactions
}

// ApplyPassives applies all passive effects to entity
func (t *FeatTracker) ApplyPassives(e FeatActor) {
	for _, f := range t.feats {
		for _, p := range f.Passives {
			if p.Apply != nil {
				p.Apply(e)
			}
		}
	}
}