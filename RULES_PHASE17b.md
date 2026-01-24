# Phase 18: Massive Integration Test Suite

## Objective
Create a comprehensive integration test suite that verifies the interactions between Skills, Combat, Conditions, and Afflictions. These tests must be exhaustive, simulating complex gameplay scenarios to ensure the engine holds up under pressure.

**Target File**: `pkg/rules/combat/scenarios_test.go` (Create this new file).

## Helper Infrastructure
First, define helper functions within the test file to make the scenarios readable:
*   `setupCombatant(name string, hp int, ac int, saves map[ability.SaveType]int) *entity.Entity`
*   `equipWeapon(e *entity.Entity, name string, damage dice.DieRoll, traits ...trait.TraitID)`

## Scenario List

You must implement **ALL** of the following scenarios as individual `t.Run("Scenario Name", ...)` tests.

### 1. The Debuff Cascade (Attributes & Conditions)
*   **Setup**: Hero (Fighter) vs Boss (Orc Warlord).
*   **Action 1 (Bard Ally/Simulated)**: Boss is hit by `Demoralize`.
    *   *Logic*: Apply `Frightened 1`.
    *   *Assert*: Boss AC is -1 (Status penalty). Saves are -1.
*   **Action 2 (Hero)**: Hero uses `Trip` action.
    *   *Logic*: Roll Athletics vs (Reflex DC - 1). Assume Success.
    *   *Assert*: Boss is now `Prone`. Boss AC is now -3 total (-1 Status from Fear, -2 Circumstance from Prone).
*   **Action 3 (Hero)**: Hero uses `Strike`.
    *   *Logic*: Roll Attack vs (Boss AC - 3).
    *   *Assert*: Hit calculation uses the modified AC.

### 2. The Ranger Flurry (MAP & Trait Logic)
*   **Setup**: Ranger with `Shortsword` (Agile, Main) and `Mace` (Shove, Offhand).
*   **Action 1**: Strike Target A with Shortsword.
    *   *Assert*: MAP is 0.
*   **Action 2**: Strike Target B with Mace (NOT Agile).
    *   *Assert*: MAP is -5 (Standard 2nd attack).
*   **Action 3**: Strike Target A with Shortsword (Agile).
    *   *Assert*: MAP is -8 (3rd attack, Agile: -8 instead of -10).
*   **Action 4**: Strike Target A with Shortsword again.
    *   *Assert*: MAP remains -8 (Max penalty).

### 3. Relational Stealth (The "Schrödinger's Rogue")
*   **Setup**: Rogue, Guard A, Guard B.
*   **Action 1**: Rogue `Hide` check.
    *   *Result*: Beat Guard A's Perception, Fail against Guard B's Perception.
    *   *Assert*:
        *   `rogue.Conditions.HasRelative(Hidden, GuardA.ID)` is **TRUE**.
        *   `rogue.Conditions.HasRelative(Hidden, GuardB.ID)` is **FALSE**.
*   **Action 2**: Rogue Strikes Guard A.
    *   *Assert*: Guard A is `Flat-footed` (since Rogue is hidden from them). Rogue loses `Hidden` condition after attack.
*   **Action 3**: Rogue Strikes Guard B.
    *   *Assert*: Guard B is **NOT** `Flat-footed`.

### 4. Affliction Timeline (Poison Progression)
*   **Setup**: Victim with 20 HP. Poison: Onset 1 round, Interval 1 round, Stage 1 (5 dmg), Stage 2 (10 dmg).
*   **Turn 1**: Victim bitten.
    *   *Logic*: `AfflictionTracker.Add(...)`.
    *   *Assert*: No damage yet (Onset).
*   **Turn 2**: Tick.
    *   *Logic*: Advance time. Onset completes.
    *   *Assert*: Save needed.
    *   *Action*: Fail Save.
    *   *Assert*: Stage 1. Damage applied (5). HP = 15.
*   **Turn 3**: Tick.
    *   *Logic*: Interval pass. Save needed.
    *   *Action*: Fail Save again.
    *   *Assert*: Advances to Stage 2. Damage applied (10). HP = 5.

### 5. Grapple and Restrain
*   **Setup**: Monk vs Wizard.
*   **Action 1**: Monk `Grapple`.
    *   *Logic*: Success.
    *   *Assert*: Wizard has `Grabbed`. Wizard is `Flat-footed`.
*   **Action 2**: Monk `Grapple` again (to upgrade).
    *   *Logic*: Critical Success.
    *   *Assert*: Wizard now `Restrained`. Wizard is `Flat-footed` and `Immobilized`.
*   **Action 3**: Wizard tries to `Move`.
    *   *Assert*: Action denied/fails (Immobilized).

### 6. Persistent Damage Death Spiral
*   **Setup**: Hero at 10 HP. `Wounded 1`.
*   **Action 1**: Fireball hits. Hero takes 12 damage.
    *   *Assert*: HP 0. `Dying 1` + `Wounded 1` => `Dying 2`.
    *   *Logic*: Apply `Persistent Fire (5)`.
*   **Turn End**:
    *   *Logic*: Persistent Damage triggers (5).
    *   *Assert*: Taking damage while Dying increments Dying value. `Dying 3`.
    *   *Logic*: Roll Flat Check for Fire. Fail. Fire remains.

### 7. Sweep and Backswing
*   **Setup**: Fighter with Greatclub (`Backswing`, `Shove`) and a `Sweep` weapon.
*   **Action 1**: Miss Target A with Greatclub.
*   **Action 2**: Strike Target A with Greatclub (Second attack).
    *   *Assert*: MAP -5. Bonus +1 (Circumstance) from `Backswing` triggering on previous miss.
*   **Action 3**: Switch to Sweep weapon. Strike Target A.
    *   *Assert*: MAP -10. No Sweep bonus (Target A was just attacked).
*   **Action 4**: Strike Target B with Sweep weapon.
    *   *Assert*: MAP -10. **Sweep Bonus +1** applies (Target B != Target A, and previous strike was same weapon logic). *Note: Standard Sweep requires previous strike to be same turn? Check rule implementation implies 'StrikesMade', ensure logic holds.*

### 8. Medicine Check: Treat Wounds
*   **Setup**: Doctor (Wis 14, Trained Medicine) vs Patient (Injured, 10/20 HP).
*   **Action**: `Treat Wounds` DC 15.
    *   *Roll*: 18 (Success).
    *   *Assert*: Patient HP increases by 2d8. Patient has "Immunity: Treat Wounds (10min or 1hr)".
*   **Repeat Action**: Try to treat again immediately.
    *   *Assert*: Fails automatically or returns error due to immunity.

### 9. Tactical Mobility (Tumble Through & Reactions)
*   **Setup**: Rogue (Acrobatics Expert) vs Giant (Reach 10ft).
*   **Action 1**: Rogue uses `Tumble Through` against Giant's Reflex DC.
    *   *Result*: Success.
    *   *Assert*: Rogue moves through Giant's space.
    *   *Logic*: Check if this triggers `Attack of Opportunity` (future proofing). If Tumble is success, AoO should validly *not* trigger (or be avoided).
*   **Action 2**: Rogue moves away.
    *   *Assert*: Normal movement consumes actions.

### 10. Counteract Check (Dispel Magic logic template)
*   **Setup**: Cleric vs active "Grease" spell (Level 1, DC 15).
*   **Action**: Cleric attempts to Dispel (Counteract).
    *   *Logic*: Roll Caster Level vs DC.
    *   *Table Check*:
        *   Crit Success: Counteract Level + 3. (Dispel)
        *   Success: Counteract Level + 1. (Dispel)
        *   Failure: Counteract Level < Target Level? No effect. (CRB p.458: "Failure: Counteract the target if its counteract level is lower than your effect's counteract level.")
    *   *Assert*: Ensure the math of "Counteract Level" works properly (Spell Level vs Dispel Level).

### 11. The Yo-Yo Healer (Dying & Recovery Loop)
*   **Setup**: Fighter at 0 HP, `Dying 2`, `Unconscious`.
*   **Action 1**: Recovery Check (Flat Check).
    *   *Roll*: 12 (Success).
    *   *Assert*: `Dying` reduces to 1. Still `Unconscious`.
*   **Action 2**: Cleric casts 2-action `Heal` (Level 1).
    *   *Logic*: Fighter gains 12 HP.
    *   *Assert*:
        *   `Dying` removed.
        *   `Unconscious` removed.
        *   GAINS `Wounded 1`.
        *   Current HP = 12.
*   **Action 3**: Fighter takes 20 damage immediately.
    *   *Assert*: HP 0.
    *   *Logic*: Dying value = `Wounded 1` + 1 = `Dying 2`.

### 12. Thief's Gambit (Steal & Palm Object)
*   **Setup**: Thief vs Merchant (Perception DC 15). Item: "Key" (Negligible bulk).
*   **Action 1**: `Palm Object` (Key).
    *   *Result*: Success (Roll 16).
    *   *Assert*: Rogue is holding Key continuously? Or just flavor?
    *   *Game State*: If implemented, item moves to inventory or "held invisible".
*   **Action 2**: `Steal` from pocket.
    *   *Result*: Failure (Roll 10).
    *   *Assert*: Merchant notices (Condition `Observed`? or just "Aware").
    *   *Result 2*: Success (Roll 20).
    *   *Assert*: Item transfer from Merchant Inventory to Thief Inventory.

### 13. Jumping Puzzle (High Jump & Long Jump combos)
*   **Setup**: Barbarian (Athletics Master) vs Chasm (25ft).
*   **Action**: `Long Jump`.
    *   *Calculations*: Speed 25, DC 25.
    *   *Roll*: 30.
    *   *Assert*: Distance jumped is 30ft? Or capped by Speed? (Rule: Can't jump further than Speed without feats like *Cloud Jump*).
    *   *Test Limit*: Ensure the return value is clamped to Speed if necessary, or flagged.

### 14. The Modifier Stacking Nightmare (Bonus Type Rules)
*   **Setup**: Fighter with multiple buffs applied.
*   **Scenario**: The engine must correctly apply PF2e's "same type doesn't stack" rule.
*   **Buffs Applied**:
    *   `Bless` spell: +1 Status bonus to attacks.
    *   `Inspire Courage` (Bard): +1 Status bonus to attacks.
    *   `Flanking`: +2 Circumstance bonus to attacks.
    *   `Higher Ground`: +1 Circumstance bonus to attacks.
    *   `Magic Weapon`: +1 Item bonus to attacks.
*   **Expected Attack Modifier Breakdown**:
    *   Status: MAX(+1, +1) = **+1** (only highest Status applies).
    *   Circumstance: MAX(+2, +1) = **+2** (only highest Circumstance applies).
    *   Item: **+1** (only one Item bonus).
    *   **Total Bonus: +4** (NOT +6).
*   **Assert**: Final attack modifier includes exactly +4 from bonuses, not the sum of all.
*   **WHY**: This is the most commonly misunderstood rule in PF2e. If the engine adds all bonuses, combat becomes trivially easy.

### 15. The Penalty Stacking Exception (Untyped Always Stack)
*   **Setup**: Fighter under multiple debuffs.
*   **Penalties Applied**:
    *   `Frightened 2`: -2 Status penalty to attacks.
    *   `Sickened 1`: -1 Status penalty to attacks.
    *   `Clumsy 1`: -1 Status penalty to attacks (DEX-based).
    *   `Enfeebled 2`: -2 Status penalty to attacks (if STR-based).
    *   `Cover (Standard)`: -2 Circumstance penalty to attacks (from target).
    *   `Prone Attacker`: -2 Circumstance penalty to attacks.
    *   `MAP (3rd attack)`: -10 Untyped penalty.
*   **Expected Attack Modifier for STR-based melee attack**:
    *   Status: MAX(-2 Frightened, -1 Sickened, -2 Enfeebled) = **-2** (worst Status).
    *   Circumstance: MAX(-2 Cover, -2 Prone) = **-2** (worst Circumstance).
    *   Untyped: **-10** (always stacks).
    *   **Total Penalty: -14**.
*   **Assert**: Engine correctly takes worst of each typed penalty, but sums untyped.
*   **WHY**: Untyped penalties (like MAP) are designed to always hurt. Engine must distinguish.

### 16. The Brutal Critical (Weapon Specialisation & Deadly)
*   **Setup**: Level 7 Fighter with `+1 Striking Warhammer` (Deadly d10), Weapon Specialisation.
*   **Attack**: Critical Hit against Orc.
*   **Damage Calculation**:
    *   Base: 2d8 (Striking rune) + 4 (STR).
    *   Doubled on Crit: (2d8 + 4) × 2 = 4d8 + 8.
    *   Deadly d10: +1d10 (rolled separately, NOT doubled).
    *   Weapon Specialisation: +2 damage (added BEFORE doubling).
*   **Expected Minimum Damage**: (4 [4d8 min] + 4 [STR] + 2 [Spec]) × 2 = **20** + 1 [1d10 min] = **21**. No wait, math: ((2d8 + 4 + 2) × 2) + 1d10.
    *   Min: ((2+4+2)×2) + 1 = 16 + 1 = **17**.
    *   Max: ((16+4+2)×2) + 10 = 44 + 10 = **54**.
*   **Assert**: Deadly dice are NOT doubled. Specialisation IS doubled.
*   **WHY**: CRB p.451: "Benefits you gain specifically from a critical hit... aren't doubled." Weapon Spec is a base damage bonus, so it IS doubled.

### 17. The Fatal Critical (Pick Crit Math)
*   **Setup**: Rogue with `War Pick` (Fatal d10, base d6).
*   **Attack**: Critical Hit.
*   **Damage Calculation with Fatal**:
    *   On Crit: Base die SIZE changes from d6 → d10.
    *   Doubled: 2d10 (from 1d10 base, doubled) + STR × 2.
    *   Fatal Bonus: +1d10 (extra die, NOT doubled).
*   **Compared to normal crit without Fatal**: Would be 2d6 + STR × 2.
*   **Assert**: Die size upgrade happens BEFORE doubling. Extra fatal die is added after.
*   **WHY**: Fatal vs Deadly is a common confusion. Fatal upgrades the base die; Deadly adds extra dice.

### 18. The Condition Cascade (Incapacitate + Unconscious + Dying)
*   **Setup**: Hero at 5 HP hit by `Sleep` spell (Incapacitation, Will save).
*   **Scenario 1**: Hero is Level 5, Spell is Level 4.
    *   *Logic*: Hero's level > Spell level × 2? No (5 < 8). Incapacitation doesn't upgrade result.
    *   *Roll*: Critical Failure.
    *   *Assert*: Hero falls `Unconscious`.
*   **Scenario 2**: Hero is Level 10, Spell is Level 4.
    *   *Logic*: Hero's level > Spell level × 2? Yes (10 > 8). Result upgrades by one step.
    *   *Roll*: Critical Failure → becomes **Failure**.
    *   *Assert*: Hero does NOT fall unconscious (Failure effect is different).
*   **WHY**: Incapacitation trait protects high-level creatures. Engine must check level comparison.

### 19. The Flanking Paradox (Three Combatants)
*   **Setup**: Hero, Ally, and Boss in triangle formation.
*   **Positioning Logic**:
    *   Hero is on Boss's north side.
    *   Ally is on Boss's south side (directly opposite Hero).
    *   Boss is flanked.
*   **Assert 1**: Hero attacking Boss gets Flanking (+2 Circumstance, Boss is Flat-footed to Hero).
*   **Assert 2**: Ally attacking Boss gets Flanking (+2 Circumstance, Boss is Flat-footed to Ally).
*   **Now Ally moves to Boss's East side (90° from Hero)**:
*   **Assert 3**: Neither Hero nor Ally are flanking anymore (not on opposite sides).
*   **Assert 4**: Boss is no longer Flat-footed to either.
*   **WHY**: Flanking is positional and relational. The engine must track geometry, not just "2 allies nearby".

### 20. The Reaction Economy (Attack of Opportunity)
*   **Setup**: Fighter with `Attack of Opportunity` reaction, Goblin enemy.
*   **Goblin Turn**:
    *   Goblin uses `Stride` to move away from Fighter.
    *   *Trigger*: Goblin leaves a square within Fighter's reach.
*   **Assert 1**: Fighter's AoO triggers automatically (or prompts).
*   **Assert 2**: Fighter's reaction is now spent for the round.
*   **Goblin continues moving, triggers another potential AoO**:
*   **Assert 3**: No additional AoO (reaction spent).
*   **Next Goblin (Goblin B) takes turn, triggers AoO**:
*   **Assert 4**: Still no AoO (reaction doesn't refresh until Fighter's turn).
*   **WHY**: Reaction economy is fundamental. One reaction per round, period.

### 21. The Action Tax (Slowed + Stunned Interaction)
*   **Setup**: Wizard with `Slowed 2` (only 1 action) takes `Stunned 2` at start of turn.
*   **PF2e Rule**: Stunned removes actions at start of turn. Slowed reduces max actions.
*   **Calculation**:
    *   Base actions: 3.
    *   Slowed 2: Max actions = 3 - 2 = **1**.
    *   Stunned 2: Lose 2 actions... but only have 1.
    *   Remaining Stunned: 2 - 1 = **1** (carries to next turn).
*   **Assert 1**: Wizard has **0 actions** this turn.
*   **Assert 2**: Wizard still has `Stunned 1` that must apply next turn.
*   **Next Turn**: Stunned 1 removes 1 action.
    *   Max: 1 (still Slowed 2). Stunned removes 1. **0 actions again**.
*   **Assert 3**: Wizard is effectively locked down for 2 turns.
*   **WHY**: Stunned + Slowed is devastating. Engine must correctly sequence the reduction.

### 22. The Heightened Spell Cascade (Fireball Scaling)
*   **Setup**: Wizard casts `Fireball` at different spell levels.
*   **Base (Level 3)**: 6d6 fire damage.
*   **Heightened (+1)**: +2d6 per level.
*   **Test Cases**:
    *   Level 3: 6d6. Assert min=6, max=36.
    *   Level 5: 6d6 + 4d6 = 10d6. Assert min=10, max=60.
    *   Level 9: 6d6 + 12d6 = 18d6. Assert min=18, max=108.
*   **Assert**: Damage formula correctly scales with (SpellLevel - 3) × 2.
*   **WHY**: Heightening is the core spell scaling mechanic. Must work for all +1 heighten spells.

### 23. The Multi-Disease Nightmare (Concurrent Afflictions)
*   **Setup**: PC infected with BOTH `Sewer Plague` (Disease) AND `Giant Centipede Venom` (Poison).
*   **Affliction 1 (Disease)**: Stage 1 = Sickened 1. Interval = 1 day.
*   **Affliction 2 (Poison)**: Stage 1 = 1d6 damage. Interval = 1 round.
*   **Day 1, Round 5**:
    *   Poison Tick: Save vs Poison DC. Fail. Stage 2 (2d6 damage). Take 7 damage.
    *   Disease: No tick (interval is daily).
*   **Day 1, Round 6**:
    *   Poison Tick: Crit Success. Stage 2 → Stage 0. **CURED**.
*   **Day 2**:
    *   Disease Tick: Fail save. Stage 2 = Sickened 2.
    *   Poison: Already cured, no effect.
*   **Assert**: Two afflictions run on independent timers. Curing one doesn't affect the other.
*   **WHY**: Players often have multiple afflictions. Engine must track each independently.

### 24. The Massive Damage Rule (Instant Death)
*   **Setup**: Hero with 50 Max HP, currently at 50 HP.
*   **Attack**: Dragon's breath deals **100 damage** (double max HP).
*   **PF2e Massive Damage Rule**: If damage ≥ (Max HP × 2), instant death (no Dying, just Dead).
*   **Assert 1**: Hero does not gain `Dying`. Hero gains `Dead` condition.
*   **Edge Case**: Hero at 50 HP takes exactly 99 damage.
*   **Assert 2**: Hero is `Dying 1` (not Dead, as 99 < 100).
*   **WHY**: Massive damage is brutal but fair. Exact threshold must be checked.

### 25. The Saving Throw Cascade (Basic Save Degrees)
*   **Setup**: Wizard casts `Fireball` (Basic Reflex Save) for 30 damage.
*   **Target Results**:
    *   Target A: Critical Success → **0 damage**.
    *   Target B: Success → **15 damage** (half).
    *   Target C: Failure → **30 damage** (full).
    *   Target D: Critical Failure → **60 damage** (double).
*   **Assert**: Each degree applies correct multiplier (0, 0.5, 1, 2).
*   **Edge Case**: 31 damage halved = 15.5 → **15** (round down).
*   **Assert**: Damage always rounds down.
*   **WHY**: Basic saves are the most common save type. Rounding errors would accumulate.

### 26. The Specialist's Splash (Alchemist Fire logic)
*   **Setup**: Alchemist throws `Lesser Alchemist's Fire` at Goblin (AC 15).
*   **Scenario A (Hit)**: Roll 18.
    *   *Effect*: 1d8 Fire + 1 Splash Fire.
    *   *Assert*: Goblin takes Main + Splash. Adjacent enemies take 1 Splash.
*   **Scenario B (Critical Hit)**: Roll 25.
    *   *Effect*: (1d8 × 2) Fire + 1 Splash Fire.
    *   *Assert*: Main damage doubled. Splash damage **NOT** doubled. (CRB p.451: "You don't multiply splash damage on a critical hit.")
*   **Scenario C (Miss)**: Roll 10.
    *   *Effect*: 1 Splash Fire.
    *   *Assert*: Main target takes splash. Adjacent take splash. (CRB p.544: "If an attack with a splash weapon fails, succeeds, or critically succeeds, all creatures... take the listed splash damage.")
*   **Scenario D (Crit Fail)**: Roll 1.
    *   *Assert*: No damage to anyone.

### 27. The Silver Standard (Material Weakness/Resistance)
*   **Setup**: Hero with `Silver Dagger` vs Devil (Resist Physical 5 (Except Silver)) vs Werewolf (Weakness Silver 5).
*   **Attack vs Devil**: Hit for 8 piercing.
    *   *Logic*: Damage type is "Piercing + Silver".
    *   *Assert*: Devil takes full 8 (Resistance bypassed). (CRB p.453: "Resistance 10 to physical damage (except silver) would reduce... unless that damage was dealt by a silver weapon.")
*   **Attack vs Werewolf**: Hit for 8 piercing.
    *   *Logic*: Weakness triggers.
    *   *Assert*: Werewolf takes 8 + 5 = **13** damage. (CRB p.453: "Increase the damage you take by the value of the weakness.")
*   **Control vs Devil**: Normal Steel Dagger hits for 8.
    *   *Assert*: Devil takes 8 - 5 = **3** damage.

### 28. The Ghostly Duel (Invisibility & Flat Checks)
*   **Setup**: Hero targets Invisible Stalker (Status: `Undetected`).
*   **Action**: Hero attempts `Strike`.
    *   *Prerequisite*: Hero must pick a square. (Assume correct square for test).
    *   *Logic*: Target is `Secretly` in square. Hero attacks.
*   **Mechanic**: Before Attack Roll, must pass Flat Check DC 11 (Undetected).
    *   *Roll 1*: Flat Check 5 (Fail).
    *   *Assert*: Attack roll never happens. Action spent. Result "Miss". (CRB p.464: "If you fail this check, the attack misses.")
    *   *Roll 2*: Flat Check 15 (Pass).
    *   *Assert*: Proceed to Attack Roll vs AC.

### 29. The Bunker (Greater Cover vs Reflex)
*   **Setup**: Hero behind `Greater Cover` (+4 AC). Dragon breathes fire (Reflex Save).
*   **Rule**: Cover grants bonus to Reflex Saves against area effects originating from blocked direction.
*   **Action**: Dragon Breath (Cone).
    *   *Logic*: Hero rolls Reflex + 4 (Circumstance).
    *   *Assert*: Final save result includes the cover bonus. (CRB p.477: "Cover grants you a +2 circumstance bonus to Reflex saves... Greater cover increases... to +4.")

### 30. The Aid Train (Cooperative Nature)
*   **Setup**: Hero attacks Boss. Three allies attempt to `Aid`.
*   **Ally 1**: Rolls Aid (Success) -> +1 Circumstance.
*   **Ally 2**: Rolls Aid (Crit Success) -> +2 (Master proficiency? Let's assume +2 base for crit).
    *   *Note*: Crit success on Aid grants +2, +3 (Master), or +4 (Legendary). Assume Trained (+2).
*   **Ally 3**: Rolls Aid (Success) -> +1 Circumstance.
*   **Resolution**: Hero attacks.
*   **Assert**: Hero gains **+2 Circumstance** bonus total. (Bonuses don't stack, highest applies).
*   **WHY**: Essential group tactic. Engine must filter multiple Aids. (CRB p.444: "If you have multiple bonuses of the same type, you can use only the highest bonus.")

## Deliverables
1.  Create `pkg/rules/combat/scenarios_test.go`.
2.  Implement all **30** scenarios with strict assertion logic.
3.  Each scenario must include setup helpers to construct entities, weapons, and spells.
4.  Use deterministic rolls (`ExecuteWithRoll`, `PerformCheckWithRoll`) for reproducibility.
5.  **Specific Commenting Requirements**:
    *   **Scenario 10 (Counteract)**: You MUST include the text from CRB p.458 regarding Failure ("Counteract the target if its counteract level is lower...") to prove the logic.
    *   **Scenario 16 (Brutal Critical)**: You MUST include the text from CRB p.451 ("Benefits you gain specifically from a critical hit... aren't doubled") to justify why Weapon Specialization is doubled but Deadly is not.
    *   For all other tests, brief comments explaining the assertion are sufficient.
