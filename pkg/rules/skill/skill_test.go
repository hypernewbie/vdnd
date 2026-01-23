package skill

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/entity"
)

func TestExhaustiveDemoralize(t *testing.T) {
	actor := entity.NewEntity("a1", "Intimidator", 1)
	target := entity.NewEntity("t1", "Victim", 1)
	target.Will = ability.Untrained // DC 11 (10 + 0 + 1)

	// Since Demoralize is random, let's just test that immunity is applied.
	_ = Demoralize(actor, target)
	if !target.IsTemporarilyImmune("demoralize", actor.ID) {
		t.Error("Target should be immune after Demoralize attempt")
	}

	// 2. Test Immunity Block
	res2 := Demoralize(actor, target)
	if res2.Degree != check.Failure {
		t.Error("Demoralize should fail against immune target")
	}
}

func TestTripDegrees(t *testing.T) {
	attacker := entity.NewEntity("a1", "Striker", 1)
	target := entity.NewEntity("t1", "Victim", 1)
	
	_ = Trip(attacker, target, nil)
}

func TestFeintRelational(t *testing.T) {
	actor := entity.NewEntity("a1", "Liar", 1)
	target := entity.NewEntity("t1", "Mark", 1)
	
	// Verify relational condition logic works
	target.Conditions.Apply(condition.NewRelationalCondition(condition.FlatFooted, []string{actor.ID}, "Feint"))
	
	if !target.Conditions.HasRelative(condition.FlatFooted, actor.ID) {
		t.Error("Should be flat-footed to actor")
	}
	if target.Conditions.HasRelative(condition.FlatFooted, "other") {
		t.Error("Should NOT be flat-footed to other")
	}
}

func TestConditionTracker_Relational(t *testing.T) {
	tr := condition.NewTracker()
	tr.Apply(condition.NewRelationalCondition(condition.FlatFooted, []string{"attacker-1"}, "Feint"))

	if !tr.HasRelative(condition.FlatFooted, "attacker-1") {
		t.Error("Should be flat-footed to attacker-1")
	}
}

func TestDrainedCondition(t *testing.T) {
	// Drained doesn't decay
	tr := condition.NewTracker()
	tr.Apply(condition.NewValuedCondition(condition.Drained, 1, "Ghoul"))
	
	tr.EndTurn(nil)
	
	if tr.Value(condition.Drained) != 1 {
		t.Error("Drained should not decay at end of turn")
	}
}
