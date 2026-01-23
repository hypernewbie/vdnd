# Rules Completeness Survey - Exhaustive

This document lists every mechanical item found in the source material (`rules/`) that is currently missing from the implementation (`pkg/rules/`).

## Skills
*   **Lore**: Missing the entire Lore skill system (including dynamic subcategories like *Vampire Lore*, *Sailing Lore*, *Military Lore*, *Planar Lore*, etc.).

## Missing Skill Actions
Compiled from `rules/rules/actions/` and `rules/compendium/skills.md`:

### Acrobatics
*   Balance
*   Tumble Through
*   Maneuver in Flight
*   Squeeze

### Athletics
*   Climb
*   Swim
*   High Jump
*   Long Jump
*   Disarm
*   Force Open

### Deception
*   Create a Diversion
*   Feint
*   Impersonate
*   Lie

### Diplomacy
*   Gather Information
*   Make an Impression
*   Request

### Intimidation
*   Coerce

### Medicine
*   Administer First Aid
*   Treat Disease
*   Treat Poison

### Stealth
*   Sneak
*   Conceal an Object

### Thievery
*   Disable a Device
*   Pick a Lock
*   Steal
*   Palm an Object

### Nature
*   Command an Animal

### General / Combat / Exploration Actions
*   Aid, Escape, Point Out, Ready, Delay, Stand, Drop Prone, Crawl, Mount, Release, Take Cover, Interact, Activate an Item, Cast a Spell (Base action), Sustain a Spell, Dismiss, Identify Magic, Decipher Writing, Learn a Spell, Earn Income, Identify Alchemy, Repair, Subsist, Sense Direction, Track, Sense Motive, Investigate, Scout, Search.

## Missing Conditions
Found in `rules/rules/conditions.md` but not in `pkg/rules/condition/registry.go`:

*   **Broken** (Object condition)
*   **Concealed**
*   **Dazzled**
*   **Encumbered**
*   **Observed**
*   **Petrified**
*   **Undetected**
*   **Unnoticed**
*   **Attitudes**: Friendly, Helpful, Hostile, Indifferent, Unfriendly.

## Missing Traits
Categorized from `rules/rules/traits/`:

### Rarity
*   Common, Uncommon, Rare, Unique.

### Traditions & Schools
*   **Traditions**: Arcane, Divine, Occult, Primal.
*   **Schools**: Abjuration, Conjuration, Divination, Enchantment, Evocation, Illusion, Necromancy, Transmutation.

### Classes
Alchemist, Barbarian, Bard, Champion, Cleric, Druid, Fighter, Gunslinger, Inventor, Investigator, Kineticist, Magus, Monk, Oracle, Ranger, Rogue, Sorcerer, Summoner, Swashbuckler, Thaumaturge, Witch, Wizard.

### Ancestries
Dwarf, Elf, Gnome, Halfling, Human, Lizardfolk, Ratfolk, Kobold, Orc, etc. (Totaling ~30+ missing).

### Weapon Properties (Missing as Traits/Enums)
Backswing, Brace, Bulwark, Capacity, Combination, Concussive, Disarm, Double-Barrel, Fatal Aim, Free-Hand, Injection, Integrated, Jousting, Kickback, Nonlethal, Parry, Range-Increment, Repeating, Scatter, Shove, Trip, Twin, Unarmed, Volley.

### Miscellaneous Tags
Aura, Bomb, Cantrip, Consumable, Curse, Death, Downtime, Emotion, Environmental, Exploration, Extradimensional, Flourish, Fortune, Grimoire, Healing, Incapacitation, Infused, Invested, Light, Metamagic, Morph, Mutagen, Oil, Open, Persistent Damage, Poison, Polymorph, Potion, Precious, Scroll, Scrying, Secret, Shadow, Sleep, Stance, Summoned, Talisman, Teleportation, Trap, Virulent, Wand.

## Missing Tables and Lists (Data)

### Languages
*   **Common**: Common, Draconic, Dwarven, Elven, Gnomish, Goblin, Halfling, Jotun, Orcish, Sylvan, Undercommon.
*   **Uncommon**: Abyssal, Aklo, Aquan, Auran, Celestial, Gnoll, Ignan, Necril, Shadowtongue, Terran.

### Alignments
*   Lawful Good (LG), Lawful Neutral (LN), Lawful Evil (LE)
*   Neutral Good (NG), True Neutral (N), Neutral Evil (NE)
*   Chaotic Good (CG), Chaotic Neutral (CN), Chaotic Evil (CE)

### Materials
*   Silver, Cold Iron, Adamantine, Mithral, Darkwood.

### Light Levels
*   Bright Light, Dim Light, Darkness.

### Terrain Types
*   Normal Terrain, Difficult Terrain, Greater Difficult Terrain, Hazardous Terrain.

## Missing Systems
*   **Feat System**: No registry for General, Skill, or Class Feats (~1000+ entries).
*   **Afflictions**: No staged progression system for Poisons, Diseases, Curses.
*   **Hazards**: Registry for traps and environmental dangers.
*   **Spells**: Massive compendium of 1,200+ spells from `rules/compendium/spells/`.
