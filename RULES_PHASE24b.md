# Phase 24b: Comprehensive Test Suite (Phases 18-24)

## Objective

This document defines an "onslaught" of comprehensive tests to verify the robustness of the implementations from Phase 18 through 24. These tests go beyond basic happy-path scenarios to cover edge cases, error conditions, and complex interactions.

## 1. Shield Mechanics Tests (Phase 18)

**Target File:** `pkg/rules/combat/shield_comprehensive_test.go`

1.  **Shield Breakage Mid-Combat**
    - **Scenario:** Entity blocks damage that reduces shield HP to 0 (Destroyed).
    - **Expect:** Shield blocks damage up to Hardness, remaining damage goes to entity. Shield `IsDestroyed()` becomes true. `IsRaised` becomes false immediately (cannot keep a destroyed shield raised).
    - **Check:** `RaiseShieldAction` should fail if shield is destroyed.

2.  **Snapshotting AC**
    - **Scenario:** Entity raises shield. Turn ends. Next entity attacks.
    - **Expect:** AC includes bonus.
    - **Scenario:** Entity turn starts.
    - **Expect:** Shield automatically lowers (`IsRaised = false`). AC returns to normal.

3.  **Damage Types**
    - **Scenario:** Reaction triggered by Fire (energy) damage.
    - **Expect:** `ShieldBlockReaction` returns `CanUse = false`. Shield cannot block energy damage.

4.  **Hardness Absorption**
    - **Scenario:** Incoming damage 3. Shield Hardness 5.
    - **Expect:** Damage to entity: 0. Damage to shield: 0.

## 2. Inventory & Bulk Tests (Phase 19)

**Target File:** `pkg/rules/entity/inventory_stress_test.go`

1.  **The "Bag of Rocks" Stress Test**
    - **Scenario:** Add 100 "Rock" items, each 1 Bulk.
    - **Expect:** Entity becomes `Encumbered` immediately, then `Immobilized` when exceeding `MaxBulk`. Movement speed becomes 0.

2.  **Fractional Bulk Arithmetic**
    - **Scenario:** Add 9 Light items.
    - **Expect:** Bulk = 0.
    - **Scenario:** Add 1 more Light item (Total 10).
    - **Expect:** Bulk = 1.
    - **Scenario:** Add 1000 coins (1 Bulk).
    - **Expect:** Total Bulk increases correctly.

3.  **Nested Containers**
    - **Scenario:** Put 50 items (Bulk 1) into a Backpack (Reduces 2 Bulk).
    - **Expect:** Total bulk calculation handles the reduction but respects the backpack's capacity (e.g. if backpack holds 4 bulk, putting 50 items in should fail or spill over). *Note: Basic implementation might not enforce capacity limits yet, verify behavior.*

4.  **Drop to Move**
    - **Scenario:** Entity is Immobilized due to bulk. Drop heaviest item.
    - **Expect:** Condition `Immobilized` removed. If still over limit, `Encumbered` remains.

## 3. Crafting & Economy Tests (Phase 20)

**Target File:** `pkg/rules/skill/crafting_edge_test.go`

1.  **Catastrophic Failure**
    - **Scenario:** Crafting check results in Critical Failure.
    - **Expect:** `MaterialsSpent` reduces by 10% (loss of materials). Progress does not increase.

2.  **Level 20 Economy**
    - **Scenario:** Level 20 Legendary crafter Earns Income.
    - **Expect:** Returns correct massive gold amount from table (e.g. 15+ gp/day).

3.  **Repair Invalid Targets**
    - **Scenario:** Attempt to repair a Potion (Consumable).
    - **Expect:** Should fail (consumables don't have HP/broken state in the same way, or cannot be repaired).

4.  **Under-levelled Crafting**
    - **Scenario:** Level 1 entity tries to craft Level 10 item.
    - **Expect:** Setup phase might start, but daily checks should fail due to insufficient proficiency/level requirements.

## 4. Ritual Edge Cases (Phase 21)

**Target File:** `pkg/rules/spell/ritual_edge_test.go`

1.  **The "Crowded" Ritual**
    - **Scenario:** Ritual requires 2 secondaries. Provided 10 secondaries.
    - **Expect:** Only first 2 (or max supported) checks count, or all count? Rules say "primary caster casts... secondary casters...". Verify partial success logic with many inputs.

2.  **Material Refund Logic**
    - **Scenario:** Cast `Resurrect`. Result: Failure (not Critical).
    - **Expect:** Outcome `RefundMaterials` is true.
    - **Scenario:** Cast `Resurrect`. Result: Critical Failure.
    - **Expect:** Outcome `RefundMaterials` is false. Backlash applies (Doomed 1).

3.  **Attribute Mismatch**
    - **Scenario:** Entity with -1 CHA tries to lead a high-stakes social ritual.
    - **Expect:** Check fails, ritual fails. Math should check out.

## 5. Subsystem Simulations (Phase 22)

**Target File:** `pkg/rules/subsystem/simulation_test.go`

1.  **The "Impossible" Chase**
    - **Scenario:** Quarry moves +2 every turn. Pursuer moves +0.
    - **Expect:** Gap increases. Chase ends in `StateSuccess` (Quarry escapes) once `EscapeDistance` reached.

2.  **Influence Stalemate**
    - **Scenario:** Influence target requires 10 VP. Rounds limit 5. Players earn 1 VP/round.
    - **Expect:** `StateFailure` (Time limit exceeded) at end of round 5.

3.  **Research Dead End**
    - **Scenario:** Research topic requires Skill X. Player only has Skill Y.
    - **Expect:** `Research` call returns error or failure description "Skill not applicable".

## 6. Complex Hazard Interaction (Phase 23)

**Target File:** `pkg/rules/encounter/hazard_advanced_test.go`

1.  **Interleaved Turns**
    - **Scenario:** Initiative: PC (20) -> Hazard (15) -> PC (10).
    - **Expect:** PC moves. Hazard acts (executes routine). PC acts. Hazard does NOT react to PC movement unless it has a Trigger.

2.  **Hazard Reset**
    - **Scenario:** Hazard routine includes `RoutineReset` at end of turn.
    - **Expect:** `IsTriggered` becomes false.
    - **Scenario:** PC walks into trigger zone again.
    - **Expect:** Hazard triggers again (Active loop).

3.  **Disable Mid-Routine**
    - **Scenario:** Hazard has 3 actions. PC delays, interrupts hazard after Action 1 (if system supports it) to disable.
    - **Expect:** If disabled, subsequent routine actions (2 and 3) do not fire. *Note: System might run routine atomically. Verify if atomic or interruptible.*

## 7. Minion Autonomy (Phase 24)

**Target File:** `pkg/rules/encounter/minion_logic_test.go`

1.  **Uncommanded Minion**
    - **Scenario:** Turn starts. Master does NOT command minion.
    - **Expect:** Minion has 0 actions. Passes turn immediately.

2.  **Command Limit**
    - **Scenario:** Master spends 3 actions to Command -> Command -> Command.
    - **Expect:** Minion acts. Minion acts. Minion acts?
    - **Rule Check:** "A minion can't be commanded more than once per turn" (standard rule).
    - **Test:** Verify if repeated commands grant extra actions or fail.

3.  **Orphaned Minion**
    - **Scenario:** Master dies (participant removed/dead).
    - **Expect:** Minion typically doesn't act or flees. Test turn logic doesn't crash if Master entity is nil/dead.

## Implementation Steps for Tests

1.  Create `pkg/rules/simulation_suite_test.go` as a master test runner for these complex scenarios.
2.  Use the table-driven test pattern for standard edge cases.
3.  Use mock Entity helpers to quickly spawn level 1-20 characters with specific stats.

```go
// Example helper for suite
func CreateTestActor(level int, str, dex, int, wis, cha, con int) *entity.Entity {
    e := entity.NewEntity(fmt.Sprintf("actor_%d", level), "Test Actor", level)
    e.Abilities.Set(ability.Strength, str)
    // ... set others
    return e
}
```
