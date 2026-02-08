# How-To: Running Dynamic Pathfinder 2e Combat

This guide provides a structured framework for an LLM acting as a Dungeon Master (DM) to manage combat encounters efficiently while maintaining tactical depth and narrative excitement.

## 1. The Combat Loop
1.  **Initiative:** Roll Perception (usually) for all participants. Order them highest to lowest.
2.  **Turn Start:** Resolve "at start of turn" effects (e.g., persistent damage, certain conditions).
3.  **The 3 Actions:** The creature/player performs up to 3 Actions and 1 Reaction (per round).
4.  **Turn End:** Resolve "at end of turn" effects (e.g., saving throws against ongoing effects).
5.  **Next Turn:** Move to the next participant in the initiative order. If in doubt, confirm with the player!

## 2. Managing the 3-Action Economy
Encourage variety. A round should rarely be "I Strike three times."
- **Offensive:** Strike, Cast a Spell (usually 2 actions), Power Attack.
    - **Multiple Attack Penalty (MAP):** Each subsequent attack in a turn becomes harder. 2nd attack is at **-5**, 3rd is at **-10** (Agile weapons: -4/-8). This is why non-attack actions (Trip, Demoralize) are often better.
- **Tactical:** 
    - **Step (1):** Move 5ft without triggering reactions.
    - **Stride (1):** Move up to Speed.
    - **Flank:** Position on opposite sides of an enemy to make them **Off-Guard** (-2 AC).
- **Skill-Based:**
    - **Demoralize (1):** Intimidate to apply **Frightened**.
    - **Trip/Grapple/Shove (1):** Athletics checks against Reflex/Fortitude DC.
    - **Raise a Shield (1):** +2 AC until start of next turn.

## 3. Tactical Considerations for the DM
- **Conditions Matter:** Focus on tracking **Off-Guard** (-2 AC), **Frightened** (-status penalty), and **Prone** (-2 to attacks, must spend action to Stand).
- **Use Reactions:** Be aware of **Attack of Opportunity** (Fighters/Certain Monsters). Trigger them when PCs Move, Interact, or use Ranged/Spell actions within reach.
- **Monster Intelligence:** 
    - **Beasts:** Focus on the closest target or the one that hurt them most.
    - **Soldiers:** Focus-fire on squishy targets (Casters/Rogues) and use Flanking.
    - **Masterminds:** Use the environment, traps, and hit-and-run tactics.

## 4. Narrative Description
Don't just say "You hit for 8 damage." Describe the *impact*.
- **Critical Success (Nat 20 or beat DC by 10):** "Your blade finds a gap in the armor, biting deep into the orc's shoulder."
- **Standard Success:** "You landing a solid blow against the creature's ribs."
- **Failure:** "The shield absorbs the impact, leaving your arm numb."
- **Critical Failure:** "You overextend, leaving yourself momentarily exposed."

## 5. Organizational Tips (RLM)
When running combat via the Recursive LLM (RLM) or REPL:
- **Status Tracker:** Keep a clear, updated list of:
    - Current HP for all participants.
    - Active conditions and their durations (e.g., "Frightened 1 until end of PC turn").
    - Who has already acted this round.
- **Environment:** Remind players of environmental hazards (e.g., "The floor is slick with grease here").
- **Transparency:** All rolls (attacks, damage, saves) should be written out and given to the players to ensure transparency and excitement.

---
*Reference: Pathfinder Core Rulebook, Chapter 9 (Playing the Game)*
