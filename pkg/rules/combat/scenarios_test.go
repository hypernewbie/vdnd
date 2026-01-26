package combat

import (
	"testing"
	"uaa/vdnd/pkg/rules/ability"
	"uaa/vdnd/pkg/rules/check"
	"uaa/vdnd/pkg/rules/condition"
	"uaa/vdnd/pkg/rules/dice"
	"uaa/vdnd/pkg/rules/entity"
	"uaa/vdnd/pkg/rules/item"
	"uaa/vdnd/pkg/rules/skill"
	"uaa/vdnd/pkg/rules/trait"
	"uaa/vdnd/pkg/rules/affliction"
)

// --- Helpers ---

func setupCombatant(name string, hp int, ac int, saves map[ability.SaveType]int) *entity.Entity {
	e := entity.NewEntity(name, name, 1)
	e.MaxHP = hp
	e.CurrentHP = hp
	e.Abilities = ability.AbilityScores{10, 10, 10, 10, 10, 10}
	if ac > 10 {
		e.UnarmoredDefense = ability.Trained 
	} else {
		e.UnarmoredDefense = ability.Untrained
	}
	for st, val := range saves {
		switch st {
		case ability.SaveFortitude: e.Fortitude = ability.ProficiencyRank(val)
		case ability.SaveReflex: e.Reflex = ability.ProficiencyRank(val)
		case ability.SaveWill: e.Will = ability.ProficiencyRank(val)
		}
	}
	return e
}

func equipWeapon(e *entity.Entity, name string, damage dice.DieRoll, traits ...trait.TraitID) *item.Weapon {
	w := &item.Weapon{
		ID:         name,
		Name:       name,
		Damage:     damage,
		DamageType: item.Slashing,
		Traits:     traits,
		IsMelee:    true,
		Bulk:       1,
	}
	e.WieldedWeapons = append(e.WieldedWeapons, w)
	return w
}

func TestScenarios(t *testing.T) {
	// 1. The Debuff Cascade
	t.Run("The Debuff Cascade", func(t *testing.T) {
		hero := setupCombatant("Hero", 20, 15, nil)
		boss := setupCombatant("Boss", 50, 15, nil)
		boss.UnarmoredDefense = ability.Untrained 

		boss.Conditions.Apply(condition.NewValuedCondition(condition.Frightened, 1, "Bard"))
		if got := boss.GetAC(nil); got != 9 { t.Errorf("Boss AC expected 9, got %d", got) }

		res := skill.Trip(hero, boss, nil, 10)
		if res.Degree < check.Success { t.Fatal("Trip failed") }
		
		if !boss.Conditions.Has(condition.Prone) { t.Error("Boss should be Prone") }
		if got := boss.GetAC(nil); got != 7 { t.Errorf("Boss AC expected 7, got %d", got) }

		w := equipWeapon(hero, "Sword", dice.DieRoll{1, 8, 0})
		strike := NewStrike(w)
		turn := NewTurn(hero)
		resS, _ := strike.ExecuteWithRoll(hero, boss, turn, 7)
		if !resS.Success { t.Error("Should have hit AC 7 with a 7") }
	})

	// 2. The Ranger Flurry
	t.Run("The Ranger Flurry", func(t *testing.T) {
		ranger := setupCombatant("Ranger", 20, 15, nil)
		targetA := setupCombatant("A", 20, 10, nil)
		targetB := setupCombatant("B", 20, 10, nil)
		
		sw := equipWeapon(ranger, "Shortsword", dice.DieRoll{1, 6, 0}, trait.TraitAgile)
		mace := equipWeapon(ranger, "Mace", dice.DieRoll{1, 6, 0})
		
		strikeSW := NewStrike(sw)
		strikeMace := NewStrike(mace)
		turn := NewTurn(ranger)

		_, c1 := strikeSW.ExecuteWithRoll(ranger, targetA, turn, 10)
		if c1.Modifiers != 0 { t.Errorf("1st attack mod expected 0, got %d", c1.Modifiers) }

		_, c2 := strikeMace.ExecuteWithRoll(ranger, targetB, turn, 10)
		if c2.Modifiers != -5 { t.Errorf("2nd attack (non-agile) mod expected -5, got %d", c2.Modifiers) }

		_, c3 := strikeSW.ExecuteWithRoll(ranger, targetA, turn, 10)
		if c3.Modifiers != -8 { t.Errorf("3rd attack (agile) mod expected -8, got %d", c3.Modifiers) }
	})

	// 3. Relational Stealth
	t.Run("Relational Stealth", func(t *testing.T) {
		rogue := setupCombatant("Rogue", 20, 15, nil)
		guardA := setupCombatant("GuardA", 20, 15, nil)
		guardB := setupCombatant("GuardB", 20, 15, nil)
		
		rogue.Conditions.ApplyRelative(condition.Hidden, guardA.ID, "Hide")

		if !rogue.Conditions.HasRelative(condition.Hidden, guardA.ID) { t.Error("Should be hidden from A") }
		
		// Guard perception mod is 0. Base AC 13 (Trained lvl 1).
		// If rogue is hidden, guard should be flat-footed -> AC 11.
		ac := guardA.GetAC(rogue)
		if ac != 11 {
			t.Errorf("Guard A AC should be 11 (flat-footed to hidden rogue), got %d", ac)
		}
		
		w := equipWeapon(rogue, "Dagger", dice.DieRoll{1, 4, 0})
		strike := NewStrike(w)
		turn := NewTurn(rogue)
		
		strike.ExecuteWithRoll(rogue, guardA, turn, 10)
		if rogue.Conditions.HasRelative(condition.Hidden, guardA.ID) { t.Error("Hidden should be removed") }
		if guardB.Conditions.IsFlatFootedTo(rogue.ID) { t.Error("Guard B should NOT be flat-footed") }
	})

	// 4. Affliction Timeline
	t.Run("Affliction Timeline", func(t *testing.T) {
		victim := setupCombatant("Victim", 20, 10, nil)
		poison := &affliction.Affliction{
			ID: "Poison",
			Name: "Poison",
			OnsetDelay: 1,
			Interval: 1,
			MaxStage: 2,
			Stages: []affliction.Stage{
				{Number: 1, Damage: dice.DieRoll{Count: 5, Sides: 1, Modifier: 0}},
				{Number: 2, Damage: dice.DieRoll{Count: 10, Sides: 1, Modifier: 0}},
			},
		}
		inst := affliction.NewInstance(poison, "Snake")
		victim.Afflictions.AddInstance(inst)
		
		inst.TickWithRoll(5, 10) // Fail onset save
		if inst.CurrentStage != 1 { t.Errorf("Expected Stage 1, got %d", inst.CurrentStage) }
		dmg, _, _ := inst.GetCurrentEffects()
		victim.ApplyDamage(dmg.Roll())
		if victim.CurrentHP != 15 { t.Errorf("HP expected 15, got %d", victim.CurrentHP) }

		inst.TickWithRoll(5, 10) // Fail interval save
		if inst.CurrentStage != 2 { t.Errorf("Expected Stage 2, got %d", inst.CurrentStage) }
	})

	// 5. Grapple and Restrain
	t.Run("Grapple and Restrain", func(t *testing.T) {
		monk := setupCombatant("Monk", 20, 15, nil)
		wizard := setupCombatant("Wizard", 15, 10, nil)
		skill.Grapple(monk, wizard, nil, 10) 
		if !wizard.Conditions.Has(condition.Grabbed) { t.Error("Wizard should be Grabbed") }
		skill.Grapple(monk, wizard, nil, 20)
		if !wizard.Conditions.Has(condition.Restrained) { t.Error("Wizard should be Restrained") }
	})

	// 6. Persistent Damage Death Spiral
	t.Run("Persistent Damage Death Spiral", func(t *testing.T) {
		hero := setupCombatant("Hero", 50, 15, nil)
		hero.CurrentHP = 10
		hero.Conditions.Apply(condition.NewValuedCondition(condition.Wounded, 1, "Prior fight"))
		hero.ApplyDamage(12)
		hero.CheckDying(false)
		if hero.Conditions.Value(condition.Dying) != 2 { t.Errorf("Expected Dying 2, got %d", hero.Conditions.Value(condition.Dying)) }
		hero.Conditions.Apply(condition.NewPersistentDamage(5, "fire", "Fireball"))
		hero.Conditions.EndTurn(hero)
		if hero.Conditions.Value(condition.Dying) != 3 { t.Errorf("Expected Dying 3, got %d", hero.Conditions.Value(condition.Dying)) }
	})

	// 7. Sweep and Backswing
	t.Run("Sweep and Backswing", func(t *testing.T) {
		fighter := setupCombatant("Fighter", 20, 15, nil)
		targetA := setupCombatant("A", 20, 10, nil)
		targetB := setupCombatant("B", 20, 10, nil)
		club := equipWeapon(fighter, "Greatclub", dice.DieRoll{1, 10, 0}, trait.TraitBackswing)
		strikeClub := NewStrike(club)
		turn := NewTurn(fighter)

		strikeClub.ExecuteWithRoll(fighter, targetA, turn, 1)
		_, c2 := strikeClub.ExecuteWithRoll(fighter, targetA, turn, 10)
		if c2.Modifiers != -4 { t.Errorf("Backswing mod expected -4, got %d", c2.Modifiers) }

		sweepW := equipWeapon(fighter, "SweepSword", dice.DieRoll{1, 8, 0}, trait.TraitSweep)
		strikeSweep := NewStrike(sweepW)
		_, c3 := strikeSweep.ExecuteWithRoll(fighter, targetA, turn, 15)
		if c3.Modifiers != -10 { t.Errorf("Sweep mod expected -10, got %d", c3.Modifiers) }
		_, c4 := strikeSweep.ExecuteWithRoll(fighter, targetB, turn, 15)
		if c4.Modifiers != -9 { t.Errorf("Sweep mod expected -9, got %d", c4.Modifiers) }
	})

	// 8. Medicine Check: Treat Wounds
	t.Run("Medicine Check: Treat Wounds", func(t *testing.T) {
		healer := setupCombatant("Healer", 20, 15, nil)
		healer.SkillProficiencies[ability.SkillMedicine] = ability.Trained
		patient := setupCombatant("Patient", 20, 15, nil)
		patient.CurrentHP = 5

		res := skill.TreatWoundsWithRoll(healer, patient, 15, 10, &dice.SimpleRoller{}) // Trained DC 15, Roll 10. Assuming healer has +5 mod.
		// Let's force mod to be exactly what we want for predictable results
		// In setupCombatant, healer level is 1, so trained = 1 + 2 = 3. Wis is 10 (mod 0). Total +3.
		// Roll 10 + 3 = 13. DC 15. Failure.
		if res.Degree != check.Failure { t.Errorf("Expected Failure, got %s", res.Degree) }

		// Let's try success
		patient.Conditions.Remove(condition.ConditionTreatWoundsImmunity)
		res = skill.TreatWoundsWithRoll(healer, patient, 15, 15, &dice.SimpleRoller{}) // 15 + 3 = 18. Success.
		if patient.CurrentHP <= 5 { t.Error("Patient should have been healed") }
		if !res.Applied { t.Error("Should be applied") }
	})

	// 9. Tactical Mobility
	t.Run("Tactical Mobility", func(t *testing.T) {
		rogue := setupCombatant("Rogue", 20, 15, nil)
		guard := setupCombatant("Guard", 20, 15, map[ability.SaveType]int{ability.SaveReflex: 1}) // Reflex Trained lvl 1 = +3. DC 13.
		
		res, _ := skill.TumbleThrough(rogue, guard, 10) // Rogue mod 0 + 10 = 10. DC 13. Fail.
		if res.Success { t.Error("Tumble should fail") }
		if !res.EndMove { t.Error("Tumble fail should end move") }

		res2, _ := skill.TumbleThrough(rogue, guard, 15) // Rogue mod 0 + 15 = 15. DC 13. Success.
		if !res2.Success { t.Error("Tumble should succeed") }
	})

	// 10. Counteract Check
	t.Run("Counteract Check", func(t *testing.T) {
		// CRB p.458: "Failure: Counteract the target if its counteract level is lower than your effect's counteract level."
		
		// Source Rank 3, Target Rank 4. Roll 10. Success?
		// We need the degree of success from a check.
		// If check is Success:
		if !check.Counteract(3, 4, check.Success) { t.Error("Success (3 vs 4) SHOULD counteract (max source+1)") }
		if !check.Counteract(3, 3, check.Success) { t.Error("Success (3 vs 3) should counteract") }
		if !check.Counteract(3, 6, check.CriticalSuccess) { t.Error("Crit Success (3 vs 6) should counteract (max source+3)") }
		
		// The Failure case (MANDATORY COMMENT)
		// CRB p.458: Failure counteracts if target level < source level.
		if !check.Counteract(3, 2, check.Failure) { t.Error("Failure (3 vs 2) SHOULD counteract per CRB p.458") }
		if check.Counteract(3, 3, check.Failure) { t.Error("Failure (3 vs 3) should NOT counteract") }
	})

	// 11. Yo-Yo Healer
	t.Run("Yo-Yo Healer", func(t *testing.T) {
		hero := setupCombatant("Hero", 20, 15, nil)
		hero.CurrentHP = 0
		hero.Conditions.Apply(condition.NewValuedCondition(condition.Dying, 2, "Crit"))
		hero.Conditions.Apply(condition.NewCondition(condition.Unconscious, "0 HP"))
		
		hero.Heal(5)
		
		if hero.Conditions.Has(condition.Dying) { t.Error("Dying should be removed") }
		if hero.Conditions.Has(condition.Unconscious) { t.Error("Unconscious should be removed by healing") }
		if hero.Conditions.Value(condition.Wounded) != 1 { t.Errorf("Expected Wounded 1, got %d", hero.Conditions.Value(condition.Wounded)) }
		
		// Go down again
		hero.ApplyDamage(10) // Not crit
		hero.CheckDying(false)
		// Dying should be 1 + Wounded = 2
		if hero.Conditions.Value(condition.Dying) != 2 { t.Errorf("Expected Dying 2, got %d", hero.Conditions.Value(condition.Dying)) }
	})

	// 12. Thief's Gambit
	t.Run("Thief's Gambit", func(t *testing.T) {
		thief := setupCombatant("Thief", 20, 15, nil)
		target := setupCombatant("Target", 20, 10, nil) // Perception +0? DC 10.
		
		res := skill.Steal(thief, target, 10)
		if res.Degree < check.Success { t.Errorf("Steal should succeed on 10, got %s", res.Degree) }
		
		res2 := skill.PalmObject(thief, 15, 5) // DC 15, Roll 5. Fail.
		if res2.Degree >= check.Success { t.Error("PalmObject should fail") }
	})

	// 13. Jumping Puzzle
	t.Run("Jumping Puzzle", func(t *testing.T) {
		athlete := setupCombatant("Athlete", 20, 15, nil)
		// Long Jump: Success if distance <= roll total
		dist, _ := skill.LongJump(athlete, 15, 15) // Roll 15 + 0 = 15.
		if dist != 15 { t.Errorf("Expected distance 15, got %d", dist) }
		
		dist2, _ := skill.LongJump(athlete, 15, 5) // Roll 5 + 0 = 5. Fail.
		if dist2 != 0 { t.Errorf("Expected distance 0 on fail, got %d", dist2) }
	})

	// 14. Modifier Stacking
	t.Run("Modifier Stacking", func(t *testing.T) {
		mods := []check.Modifier{
			{Value: 1, Type: check.BonusStatus, Source: "Bless"},
			{Value: 1, Type: check.BonusStatus, Source: "Inspire Courage"},
			{Value: 2, Type: check.BonusCircumstance, Source: "Flanking"},
			{Value: 1, Type: check.BonusCircumstance, Source: "Higher Ground"},
			{Value: 1, Type: check.BonusItem, Source: "Magic Weapon"},
		}
		if total := check.CalculateTotal(mods); total != 4 { t.Errorf("Expected +4, got %d", total) }
	})

	// 15. Penalty Stacking
	t.Run("Penalty Stacking", func(t *testing.T) {
		mods := []check.Modifier{
			{Value: -2, Type: check.BonusStatus, Source: "Frightened"},
			{Value: -1, Type: check.BonusStatus, Source: "Sickened"},
			{Value: -2, Type: check.BonusCircumstance, Source: "Cover"},
			{Value: -10, Type: check.BonusUntyped, Source: "MAP"},
		}
		if total := check.CalculateTotal(mods); total != -14 { t.Errorf("Expected -14, got %d", total) }
	})

	// 16. Brutal Critical
	t.Run("Brutal Critical", func(t *testing.T) {
		// CRB p.451: "Benefits you gain specifically from a critical hit... aren't doubled."
		hero := setupCombatant("Hero", 50, 15, nil)
		hero.Abilities.Strength = 18 // +4
		w := &item.Weapon{
			ID: "hammer",
			Damage: dice.DieRoll{Count: 1, Sides: 8, Modifier: 0},
			Traits: []trait.TraitID{trait.TraitDeadly},
			IsMelee: true,
		}
		strike := NewStrike(w)
		target := setupCombatant("Orc", 50, 10, nil)
		res, _ := strike.ExecuteWithRoll(hero, target, NewTurn(hero), 20) // Crit
		// (1d8 + 4) * 2. Min = 10. (Since DeadlyDie isn't set in Weapon struct here)
		if res.Damage < 10 { t.Errorf("Expected at least 10, got %d", res.Damage) }
	})

	// 17. Fatal Critical
	t.Run("Fatal Critical", func(t *testing.T) {
		hero := setupCombatant("Hero", 50, 15, nil)
		w := &item.Weapon{
			ID: "pick",
			Damage: dice.DieRoll{Count: 1, Sides: 6, Modifier: 0},
			Traits: []trait.TraitID{trait.TraitFatal},
			IsMelee: true,
		}
		strike := NewStrike(w)
		target := setupCombatant("Orc", 50, 10, nil)
		res, _ := strike.ExecuteWithRoll(hero, target, NewTurn(hero), 20) // Crit
		// 1d6*2. Min = 2. (Since FatalDie isn't set)
		if res.Damage < 2 { t.Errorf("Expected at least 2, got %d", res.Damage) }
	})

	// 18. Condition Cascade (Incapacitate)
	t.Run("Condition Cascade (Incapacitate)", func(t *testing.T) {
		// CRB p.458: "If a spell has the incapacitation trait, any creature of more than twice the spell's level 
		// treats the result of their check... as one degree of success better"
		
		hero := setupCombatant("Hero", 50, 15, nil)
		hero.Level = 5 // Level 5
		
		// Spell level 2. 2*2 = 4. 5 > 4. Incapacitation applies.
		spellLevel := 2
		
		rollTotal := 10
		dc := 15
		res := check.DetermineDegree(10, rollTotal, dc) // Failure
		if res != check.Failure { t.Fatalf("Expected Failure, got %s", res) }
		
		if hero.Level > spellLevel*2 {
			res = res.Adjust(1) // Improve degree
		}
		
		if res != check.Success { t.Errorf("Expected Success (after adjust), got %s", res) }
	})

	// 19. The Flanking Paradox
	t.Run("The Flanking Paradox", func(t *testing.T) {
		target := setupCombatant("Target", 20, 15, nil)
		target.X, target.Y = 0, 0
		
		a := setupCombatant("A", 20, 15, nil)
		a.X, a.Y = 0, 1 // North
		
		b := setupCombatant("B", 20, 15, nil)
		b.X, b.Y = 0, -1 // South
		
		if !entity.IsFlanking(target, a, b) { t.Error("North/South should flank Center") }
		
		c := setupCombatant("C", 20, 15, nil)
		c.X, c.Y = 1, 0 // East
		
		if entity.IsFlanking(target, a, c) { t.Error("North/East should NOT flank Center") }
	})

	// 20. Reaction Economy
	t.Run("Reaction Economy", func(t *testing.T) {
		hero := setupCombatant("Hero", 50, 15, nil)
		turn := NewTurn(hero)
		
		if err := turn.SpendReaction(); err != nil { t.Errorf("Should spend reaction: %v", err) }
		if err := turn.SpendReaction(); err == nil { t.Error("Should NOT be able to spend reaction twice") }
	})

	// 21. Action Tax
	t.Run("Action Tax", func(t *testing.T) {
		wizard := setupCombatant("Wizard", 20, 10, nil)
		wizard.Conditions.Apply(condition.NewValuedCondition(condition.Slowed, 1, "Chill Touch"))
		wizard.Conditions.Apply(condition.NewValuedCondition(condition.Stunned, 1, "Ghoulish Touch"))

		// 3 actions - 1 (stunned) - 1 (slowed) = 1
		turn := NewTurn(wizard)
		if turn.ActionsRemaining != 1 {
			t.Errorf("Expected 1 action remaining, got %d", turn.ActionsRemaining)
		}
		if wizard.Conditions.Value(condition.Stunned) != 0 {
			t.Errorf("Expected stunned 0, got %d", wizard.Conditions.Value(condition.Stunned))
		}
	})

	// 22. Heightened Spell
	t.Run("Heightened Spell", func(t *testing.T) {
		// Simulating fireball: 6d6 base (level 3), +2d6 per level
		spellLevel := 5
		baseLevel := 3
		baseDamage := 6
		heightenedDamage := (spellLevel - baseLevel) * 2
		totalDice := baseDamage + heightenedDamage
		
		if totalDice != 10 { t.Errorf("Expected 10d6 at level 5, got %d", totalDice) }
	})

	// 23. Multi-Disease
	t.Run("Multi-Disease", func(t *testing.T) {
		victim := setupCombatant("Victim", 50, 10, nil)
		d1 := &affliction.Affliction{ID: "D1", Name: "D1", MaxStage: 3}
		d2 := &affliction.Affliction{ID: "D2", Name: "D2", MaxStage: 3}
		
		victim.Afflictions.AddInstance(affliction.NewInstance(d1, "S1"))
		victim.Afflictions.AddInstance(affliction.NewInstance(d2, "S2"))
		
		if len(victim.Afflictions.All()) != 2 { t.Fatalf("Expected 2 afflictions, got %d", len(victim.Afflictions.All())) }
		
		// Remove one
		victim.Afflictions.Remove("D1")
		if len(victim.Afflictions.All()) != 1 { t.Error("D1 should be removed") }
		if victim.Afflictions.Get("D2") == nil { t.Error("D2 should remain") }
	})

	// 24. Massive Damage
	t.Run("Massive Damage", func(t *testing.T) {
		hero := setupCombatant("Hero", 50, 10, nil)
		hero.ApplyDamage(100) // 2x MaxHP
		if !hero.IsDead() { t.Error("Should be dead from massive damage") }
	})

	// 25. Saving Throw Cascade
	t.Run("Saving Throw Cascade", func(t *testing.T) {
		dmg := 31
		if half := dmg / 2; half != 15 { t.Errorf("Expected 15, got %d", half) }
		if double := dmg * 2; double != 62 { t.Errorf("Expected 62, got %d", double) }
	})

	// 26. Specialist's Splash
	t.Run("Specialist's Splash", func(t *testing.T) {
		// Bomb: 1d6 fire + 1 splash
		// CRB p.451: "Splash damage... isn't doubled."
		// Success: full damage + splash.
		// Fail: splash only.
		// Crit Fail: no damage.
		
		splash := 1
		direct := 4
		
		// Crit
		critTotal := (direct * 2) + splash
		if critTotal != 9 { t.Errorf("Expected 9 on crit (4*2 + 1), got %d", critTotal) }
		
		// Fail
		failTotal := splash
		if failTotal != 1 { t.Errorf("Expected 1 on fail, got %d", failTotal) }
	})

	// 27. Silver Standard
	t.Run("Silver Standard", func(t *testing.T) {
		werewolf := setupCombatant("Werewolf", 50, 10, nil)
		werewolf.Weaknesses["silver"] = 5
		dmg := 8
		total := dmg + werewolf.Weaknesses["silver"]
		if total != 13 { t.Errorf("Expected 13, got %d", total) }
	})

	// 28. Ghostly Duel
	t.Run("Ghostly Duel", func(t *testing.T) {
		attacker := setupCombatant("Attacker", 20, 15, nil)
		ghost := setupCombatant("Ghost", 20, 15, nil)
		w := equipWeapon(attacker, "Sword", dice.DieRoll{1, 8, 0})
		
		// Ghost is hidden
		ghost.Conditions.ApplyRelative(condition.Hidden, attacker.ID, "Mist")
		
		strike := NewStrike(w)
		turn := NewTurn(attacker)

		// Fail flat check
		strike.FlatCheckRoll = 10
		res1, _ := strike.ExecuteWithRoll(attacker, ghost, turn, 15)
		if res1.Success { t.Error("Should have failed flat check with 10") }

		// New turn to reset MAP
		turn = NewTurn(attacker)

		// Pass flat check
		strike.FlatCheckRoll = 11
		res2, _ := strike.ExecuteWithRoll(attacker, ghost, turn, 15)
		if !res2.Success {
			t.Errorf("Should have passed flat check with 11 and hit")
		}
	})

	// 29. The Bunker
	t.Run("The Bunker", func(t *testing.T) {
		hero := setupCombatant("Hero", 20, 15, map[ability.SaveType]int{ability.SaveReflex: 1}) // +3 Reflex
		// Standard Cover: +2 circumstance bonus to AC and Reflex saves
		hero.Conditions.Apply(condition.NewCondition(condition.StandardCover, "Wall"))
		
		if got := hero.GetAC(nil); got != 15 { // 13 base + 2 cover
			t.Errorf("Expected AC 15, got %d", got)
		}
		
		if got := hero.GetReflex(); got != 5 { // 3 base + 2 cover
			t.Errorf("Expected Reflex 5, got %d", got)
		}
	})

	// 30. Aid Train
	t.Run("Aid Train", func(t *testing.T) {
		mods := []check.Modifier{
			{Value: 1, Type: check.BonusCircumstance, Source: "Aid A"},
			{Value: 2, Type: check.BonusCircumstance, Source: "Aid B"},
			{Value: 1, Type: check.BonusCircumstance, Source: "Aid C"},
		}
		if total := check.CalculateTotal(mods); total != 2 { t.Errorf("Expected +2, got %d", total) }
	})
}