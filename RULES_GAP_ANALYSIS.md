## Triage: Hard Rules vs Soft DM

| Gap | Verdict | Reasoning |
|:---|:---|:---|
| **Counteract System** | ⚙️ **Implement (Light)** | It's just a standardised check formula: `(Caster Level + Prof) vs DC`. The *decision* of what to counteract is soft, but the roll itself is maths. Add a `vd check counteract <caster> --vs <effect_level>` command that applies the level-comparison logic. |
| **Magic Item Activation** | 🧠 **LLM Soft** | "Interact to pull a wand, Command word to fire it" is narrative pacing. The *effect* of the item (burst of fire, +2 to saves) can be hardcoded per-item, but tracking "did they use their 1 Interact action?" is just action economy the LLM already manages. Investiture (10 item limit) is a soft "you can only have 10 invested" rule the LLM can enforce narratively. |
| **Light & Vision** | 🧠 **LLM Soft** | Light levels are environmental storytelling. The *mechanical effect* (Concealed condition) is already in conditions. LLM can just `vd condition add goblin concealed --source "dim light"`. No need for a light state machine. |
| **Movement Modes (Fly/Swim/Climb)** | ⚙️ **Implement (Minimal)** | Add `MoveMode` enum to Entity (`Ground`, `Fly`, `Swim`, `Climb`, `Burrow`). `Stride` action checks `Speed` vs `FlySpeed` etc. The *terrain interaction* (Is it water? Can you climb this wall?) is soft. |
| **Formulas (Crafting)** | 🧠 **LLM Soft** | "Do you know the formula?" is a character sheet question. The LLM can check the PC markdown file. The *crafting roll* (Phase 20) is maths, but formula knowledge is narrative. |
| **XP & Leveling** | 🧠 **LLM Soft** | The plan explicitly states Level is a static int. Leveling up is a session-boundary event the LLM handles ("Congrats, you're level 6! Update your sheet"). No need for an XP tracker. |
| **Deities & Domains** | 🧠 **LLM Soft** | "Are you following your deity's edicts?" is roleplay. Divine Font choice (harm/heal) is a character build decision, not a runtime mechanic. |
| **Vehicles** | 🗑️ **Out of Scope** | Edge case. If someone runs a naval campaign, add it then. |
| **Secrets of Magic** | 🗑️ **Out of Scope** | Ley Lines are very niche. |
| **Interact Action** | ⚙️ **Implement (Stub)** | Add `vd action interact <actor> <object>` as a 1-action stub. It does nothing mechanically except decrement actions. The LLM describes the narrative outcome. Useful for tracking action economy. |
| **Take Cover** | ⚙️ **Implement** | This is a real +2/+4 AC bonus (circumstance). Add `vd action take_cover <actor>` that applies a "taking cover" condition with AC bonus. Resets on movement. |
| **Point Out / Avert Gaze** | 🧠 **LLM Soft** | "You point out the hidden goblin to your ally" → LLM removes Hidden from that ally's perspective. "You avert your gaze" → LLM applies -2 circumstance to attacks vs that creature. Edge cases the LLM can handle with `vd condition add`. |
| **Drop Prone / Stand** | ⚙️ **Implement** | Trivial. `vd action drop_prone <actor>` → adds Prone. `vd action stand <actor>` (1 action) → removes Prone. Prone is already a condition with effects. |

---

## Summary: What To Build

**Add to `pkg/rules`:**
1. `Counteract` check helper (level-vs-level comparison with degree of success).
2. `MoveMode` field on Entity + `FlySpeed`, `SwimSpeed`, `ClimbSpeed` ints.
3. `TakeCover` action → applies circumstance AC bonus condition.
4. `DropProne` / `Stand` actions → toggle Prone condition.
5. `Interact` action stub → just spends 1 action, returns "Interacted with X".

**Leave to LLM:**
- Light levels, vision, concealment application.
- Item activation decisions and investiture limits.
- Formula knowledge for crafting.
- XP, leveling, deity edicts.
- Point Out, Avert Gaze (just `condition add`).

---

## My Recommendation

Skip a Phase 25 for now. Instead, add a **"Glue Actions" section to Phase 7** (Actions & Combat) in your existing plan:

```markdown
### 7.4 Glue Actions
These are simple actions that primarily exist for action economy tracking:

- **Interact** — 1 action, no mechanical effect, LLM describes outcome.
- **Take Cover** — 1 action, applies +2 circumstance bonus to AC/Reflex until movement.
- **Drop Prone** — Free action, applies Prone condition.
- **Stand** — 1 action, removes Prone condition.
```

And add a **Phase 7.5 or Phase 16: Movement Modes**:

```markdown
### Movement Modes
Entity struct gains:
- `FlySpeed int`
- `SwimSpeed int`
- `ClimbSpeed int`
- `CurrentMoveMode MoveMode` (enum: Ground, Fly, Swim, Climb, Burrow)

Stride action checks appropriate speed based on mode.
LLM is responsible for setting mode appropriately (e.g., "you dive into the water" → `vd entity set rogue movemode swim`).
```

**Counteract** can be a helper function in `pkg/rules/check/` that's exposed via `vd check counteract`. No need for a whole phase.
