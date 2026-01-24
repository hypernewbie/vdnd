# Phase 24c: The Gauntlet - Extensive Edge Case & Regression Suite

## Objective
This phase aims to break the engine. We validate complex interactions between disparate systems (e.g., Minions + Hazards + Stealth + Conditions) to uncover logic gaps.

---

## 1. Complex Combat Interactions
**Source:** `rules/rules/core-rulebook/chapter-9-playing-the-game.md` (General Combat)

**Target File:** `pkg/rules/combat/gauntlet_combat_test.go`

1.  **Invisible Minion Command**
    - **Scenario:** Minion is Hidden/Invisible. Master Commands minion to Strike target.
    - **Tricky Part:** Does the Minion's attack benefit from Flat-Footed (due to being hidden)?
    - **Expect:** Yes. The Command action grants actions, but the Minion executes them with its own state.
    - **Verification:** Minion Strike roll should have "Hidden Attacker" bonuses (if implemented) or Target has -2 AC.

2.  **Dying Recovery vs. Doomed**
    - **Scenario:** Entity with Doomed 1 drops to 0 HP.
    - **Tricky Part:** "Dying 1 + Doomed 1 = Dying 2".
    - **Expect:** Entity starts at Dying 2.
    - **Scenario:** Entity fails recovery check by 1.
    - **Expect:** Dying 3 (Condition increment) -> Dying 4 (Death) if Doomed applies on increments too (it doesn't, usually just max/start). Wait, *PF2E Rule:* "If you are Doomed, you die at Dying 4 - Doomed Value."
    - **Verification:** Entity dies at Dying 3 (if Doomed 1).

3.  **Reaction Chain (The "No U" Loop)**
    - **Scenario:** Fighter A hits Fighter B. Fighter B uses `Retributive Strike`. Fighter A uses `Attack of Opportunity` triggered by B's strike.
    - **Tricky Part:** Nested reactions. Infinite loops?
    - **Expect:** Reactions resolve LIFO or FIFO? Usually, reaction interrupts trigger. A AoO interrupts B's Retrib Strike.
    - **Verification:** A's AoO resolves. If B takes damage, does Retrib Strike fail or continue? (Usually logic: Reaction spent, effect happens unless disrupted).

4.  **Persistent Damage & Dying**
    - **Scenario:** Entity at 1 HP takes 5 Persistent Fire damage at end of turn.
    - **Tricky Part:** Drops to 0 HP. Does Dying value increase immediately?
    - **Expect:** Yes. End of turn cleanup must handle state change (Unconscious/Dying) correctly before passing turn.

---

## 2. Inventory & Bulk Gauntlet
**Source:** `rules/rules/core-rulebook/chapter-6-equipment.md` (Bulk, Carry Limit)

**Target File:** `pkg/rules/entity/gauntlet_inventory_test.go`

1.  **Recursive Containers (Bag of Holding inside Bag of Holding)**
    - **Scenario:** Put Backpack A inside Backpack B. Put Item inside Backpack A.
    - **Tricky Part:** Logic loop or Bulk calculation depth.
    - **Expect:** Bulk should calculate recursivley. (Backpack A reduces item bulk. Backpack B reduces Backpack A's bulk).
    - **Verification:** Total bulk is correct. No infinite recursion crash.

2.  **Dropping a Container**
    - **Scenario:** Drop a Backpack containing 50 items.
    - **Tricky Part:** Do the items vanish? Do they spill?
    - **Expect:** Items remain "inside" the dropped backpack item. Inventory list removes 1 item (Backpack).
    - **Verification:** Entity bulk drops significantly. Items not lost (can be picked up).

3.  **Coin Accumulation Overflow**
    - **Scenario:** Add 1,000,000,000 copper pieces.
    - **Expect:** Bulk calculation shouldn't overflow integer bounds (unlikely in Go int, but good check). Bulk should be ~1,000,000. Entity Immobilized.

---

## 3. Hazard & Stealth Gauntlet
**Source:** `rules/rules/core-rulebook/chapter-9-playing-the-game.md` (Environment, Hazards)

**Target File:** `pkg/rules/encounter/gauntlet_hazard_test.go`

1.  **Stealth vs Hazard Passive Perception**
    - **Scenario:** Rogue Sneaks past Simple Hazard. Rogue Stealth exceed Hazard Stealth DC? Or Perception?
    - **Tricky Part:** Hazards don't always have Perception. Some use Stealth DC as detection DC.
    - **Expect:** If Rogue Roll > Hazard DC, Hazard does NOT trigger.
    - **Verification:** Trigger logic respects sneak result.

2.  **Flying over a Pressure Plate**
    - **Scenario:** Character uses `Fly` speed to move over floor trigger.
    - **Tricky Part:** "EntitiesAtPosition" usually ignores Z-axis/mode of travel in simple abstract mapping.
    - **Expect:** Ideally, Fly prevents floor trigger. Logic might need "MovementMode" check.
    - **Check:** Does existing system support `MoveMode` (Walk/Fly)? If not, note as limitation or feature gap.

3.  **Hazard Initiatives Matches PC**
    - **Scenario:** Hazard rolls Initiative 15. PC rolls 15.
    - **Tricky Part:** Tie-breaking. Rules say "Enemies usually go first" or "PCs go first"?
    - **Expect:** Deterministic tie-breaking (e.g. check modifier, or assume Enemy/Hazard wins vs PC). System shouldn't crash or skip turns.

---

## 4. Spell & Ritual Gauntlet
**Source:** `rules/rules/core-rulebook/chapter-7-spells.md` (Rituals, Casting)

**Target File:** `pkg/rules/spell/gauntlet_magic_test.go`

1.  **Counteracting a Ritual**
    - **Scenario:** Enemy casts `Dispel Magic` on an active Ritual effect (e.g. `Create Undead` minion).
    - **Tricky Part:** Counteract checks.
    - **Expect:** Not fully implemented? We need to verify if interacting with ritual outputs works.

2.  **Ritual with Dead Secondary Caster**
    - **Scenario:** A Secondary Caster dies mid-ritual (e.g. from Backlash of previous attempt or external damage).
    - **Expect:** Ritual fails? Or continues with penalty?
    - **Check:** `CastRitual` verifies caster health/status before rolling?

3.  **Cantrip Scaling**
    - **Scenario:** Level 20 Wizard casts `Electric Arc` (Cantrip).
    - **Tricky Part:** Damage scaling. Should be auto-heightened to Rank 10.
    - **Expect:** Damage = 10d4 + MOD? (Whatever the formula is).
    - **Verification:** Ensure cantrips scale with character level/max spell rank automatically.

---

## 5. Gamemastery Subsystem Gauntlet
**Source:** `rules/rules/gamemastery-guide/chapter-3-subsystems.md`

**Target File:** `pkg/rules/subsystem/gauntlet_subsystem_test.go`

1.  **Chase: Split Party**
    - **Scenario:** 2 PCs chase 1 Thief. PC A moves ahead. PC B stays back.
    - **Tricky Part:** The "Gap" mechanics usually track Closest Pursuer vs Quarry.
    - **Expect:** If PC A catches Thief, Chase ends in Success. PC B's position is irrelevant.
    - **Verification:** StateSuccess triggers correctly even if one participant lags.

2.  **Influence: Skill Reuse Penalty**
    - **Scenario:** Player uses Diplomacy 10 times in a row.
    - **Tricky Part:** Some GM rules impose penalties for repetitive arguments.
    - **Expect:** Check if system supports "diminishing returns" or "hard modifiers". Even if not enforced, check if `modifier` argument handles it.

3.  **Research: Critical Failure Chain**
    - **Scenario:** Research library. 3 Crit Fails in a row.
    - **Tricky Part:** Does library "Close"? (Victory Points <= FailureThreshold).
    - **Expect:** If `FailureThreshold` is reached, `StateFailure`. Further checks impossible.

---

## 6. Implementation Plan for The Gauntlet

1.  Create `pkg/rules/gauntlet_suite_test.go` (shared helpers).
2.  Implement `gauntlet_combat_test.go` (focus on Doomed/Dying/Reactions).
3.  Implement `gauntlet_inventory_test.go` (Recursive containers).
4.  Implement `gauntlet_hazard_test.go` (Stealth interaction).

---

## 7. Data Integrity & Weird Logic (Engine Stability)

**Target File:** `pkg/rules/gauntlet/integrity_test.go`

1.  **The "Schrödinger's Entity" (Invariant Check)**
    - **Scenario:** Manually set Entity HP to 20 but force add `Dying 1` condition.
    - **Tricky Part:** Game state is logically impossible.
    - **Expect:** Self-correction (remove Dying) OR validation error. System invariants must hold.
    - **Verification:** `ValidateEntityState(e)` returns error.

2.  **The "Equipment Paradox" (Pointer Safety)**
    - **Scenario:** Actor A equips Sword X. Actor B (a Clone) equips Sword X. Modify Sword X damage on Actor A.
    - **Tricky Part:** Go pointer sharing.
    - **Expect:** Actor B's sword stats remain unchanged.
    - **Verification:** Deep copy logic in `Clone()` is robust.

3.  **The "Time Travel" Turn (State Rollback)**
    - **Scenario:** Manually decrement `CurrentRound` (GM Undo).
    - **Tricky Part:** Duration expirations. Condition duration math `StartRound + Duration`.
    - **Expect:** Conditions generated in "future" rounds are purged? Or durations effectively extend?
    - **Verification:** System handles negative time deltas without panic.

4.  **The "Circular Grapple" (Topology)**
    - **Scenario:** A grapples B. B grapples C. C grapples A.
    - **Tricky Part:** Movement logic. Attempt `Stride` with A.
    - **Expect:** A drags B, who drags C, who drags A. Movement blocked (Immobilized) or infinite mass?
    - **Verification:** Movement fails gracefully ("Cannot move while grappled" usually covers it), but check for recursion stack overflow.

5.  **"Zero-Stat" Man (Input Validation)**
    - **Scenario:** Entity created with 0 in all attributes, 0 Speed, 0 HP.
    - **Tricky Part:** Division by zero logic (modifiers, derived stats).
    - **Expect:** `(0-10)/2 = -5` mod. `Speed 0` allows no movement.
    - **Verification:** No panics.

6.  **"The 25-Hour Ritual" (Fatigue Boundaries)**
    - **Scenario:** Caster begins a 1-day ritual.
    - **Tricky Part:** Staying awake > 16 hours causes Fatigue.
    - **Expect:** Does the system apply `Fatigued` mid-cast? If so, does the check suffer the penalty?
    - **Verification:** Check application hierarchy (Condition check -> Skill Check).
