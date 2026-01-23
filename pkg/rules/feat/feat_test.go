package feat_test

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/feat"
	"uaa/vdnd/pkg/rules/trait"
)

// MockFeatEntity for testing prerequisites
type MockFeatEntity struct {
	Level    int
	Abilities map[ability.Ability]int
	Feats    []string
	Skills   map[ability.SkillID]ability.ProficiencyRank
	Traits   []trait.TraitID
}

func (m *MockFeatEntity) GetLevel() int { return m.Level }
func (m *MockFeatEntity) GetAbilityScore(ab ability.Ability) int { return m.Abilities[ab] }
func (m *MockFeatEntity) HasFeat(featID string) bool {
	for _, f := range m.Feats {
		if f == featID {
			return true
		}
	}
	return false
}
func (m *MockFeatEntity) HasSkillRank(skillID ability.SkillID, rank ability.ProficiencyRank) bool {
	return m.Skills[skillID] >= rank
}
func (m *MockFeatEntity) HasTrait(traitID trait.TraitID) bool {
	for _, t := range m.Traits {
		if t == traitID {
			return true
		}
	}
	return false
}

// MockFeatActor
type MockFeatActor struct {
	ID   string
	Name string
}

func (m *MockFeatActor) GetID() string   { return m.ID }
func (m *MockFeatActor) GetName() string { return m.Name }

// MockTurnState for testing action grants
type MockTurnState struct {
	ActionsSpent ability.ActionCost
}

func (m *MockTurnState) SpendActions(cost ability.ActionCost) error {
	m.ActionsSpent = cost
	return nil
}
func (m *MockTurnState) SpendReaction() error { return nil }
func (m *MockTurnState) RecordAttack()        {}
func (m *MockTurnState) CanAct() bool         { return true }

func TestPrerequisites(t *testing.T) {
	f := &feat.Feat{
		ID:    "test-feat",
		Level: 3,
		Prerequisites: []feat.Prerequisite{
			{RequiredTrait: trait.TraitFighter},
		},
	}

	e := &MockFeatEntity{
		Level:  2,
		Traits: []trait.TraitID{trait.TraitFighter},
	}

	if f.MeetsPrerequisites(e) {
		t.Error("Should fail due to level 2 < 3")
	}

	e.Level = 3
	if !f.MeetsPrerequisites(e) {
		t.Error("Should pass with Level 3 and Fighter trait")
	}

	e2 := &MockFeatEntity{
		Level:  3,
		Traits: []trait.TraitID{},
	}
	if f.MeetsPrerequisites(e2) {
		t.Error("Should fail due to missing Fighter trait")
	}
}

func TestActionGrant(t *testing.T) {
	f := &feat.Feat{
		ID:   "power-attack",
		Name: "Power Attack",
		GrantsAction: &feat.ActionGrant{
			Name: "Power Attack",
			Cost: ability.CostTwo,
			Execute: func(actor, target feat.FeatActor, turn feat.TurnState) ability.ActionResult {
				turn.SpendActions(ability.CostTwo)
				return ability.ActionResult{Success: true, Description: "Power Attacked!"}
			},
		},
	}

	tracker := feat.NewFeatTracker()
	tracker.Add(f)

	actions := tracker.GetGrantedActions()
	if len(actions) != 1 || actions[0].Name != "Power Attack" {
		t.Fatal("Action not granted")
	}

	actor := &MockFeatActor{ID: "a1", Name: "Hero"}
	mockTurn := &MockTurnState{}
	res := actions[0].Execute(actor, nil, mockTurn)

	if !res.Success || res.Description != "Power Attacked!" {
		t.Error("Execution failed")
	}
	if mockTurn.ActionsSpent != ability.CostTwo {
		t.Errorf("Expected 2 actions spent, got %d", mockTurn.ActionsSpent)
	}
}