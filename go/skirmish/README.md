# Skirmish DICE (GUI Edition)

A standalone 2D tactical tabletop wargame in Go using **Ebitengine v2**, faithfully implementing the rules and mechanics of **Skirmish Dice** by *Gray Army Gaming*.

The game launches directly into a full GUI window with no command prompt or console window.

---

## Game Overview

In *Skirmish Dice*, polyhedral dice represent your warriors on the battlefield. The number of sides on the die dictates the rank, health, and combat prowess of the unit.

### Unit Ranks & Costs (70 Point Recommended Warband)

| Die Type | Rank | Limit | Cost | Base Radius |
| :--- | :--- | :--- | :--- | :--- |
| **d10** | Commander | Max 1 | 10 Points | 0.65" |
| **d8** | Captain | Max 1 | 8 Points | 0.58" |
| **d6** | Soldier | No Limit | 6 Points | 0.52" |
| **d4** | Mercenary | No Limit | 4 Points | 0.45" |

- **Ranged Weapons (Shooters)**: Shooters are indicated by a gold/brass coin base under the die.
- **Battlefield**: Standard 36" $\times$ 36" tabletop with tactical terrain:
  - **Soft Cover** (Hedges/Bushes): $+1$ cover defense bonus. Does not impede movement.
  - **Hard Cover** (Stone Walls/Ruins): $+3$ cover defense bonus. Crossing over costs $+1$ movement step.
  - **Pillars/Blocks**: Solid obstacles that block Line of Sight completely.

---

## Round Structure

### 1. Deployment
1. Both players roll a **d20**.
2. The winner chooses their battle edge (North vs. South).
3. The *other* player places the first die within **6 inches** of their edge.
4. Players alternate placing one die at a time until all units are deployed.

### 2. Initiative
Each player rolls a **d20** and adds their current count of **Casualties** (underdog catch-up mechanic). Highest total takes Turn 1.

### 3. Activation (4 Turns: A-B-A-B)
1. **Generate Activation Points**: Active player rolls **2d6 $+$ Casualties** for their points pool.
2. **One Die at a Time**: You must fully resolve one die's actions before activating another.
3. **Movement**:
   - 1st Move ($0\text{--}6"$): 1 Point
   - 2nd Move ($6\text{--}12"$): $+3$ Points (Total 4)
   - 3rd Move ($12\text{--}18"$): $+5$ Points (Total 9)
   - *Obstacle Penalty*: Moving over an obstacle higher than the die costs one extra movement step.
4. **Shooting (Author Rules Errata)**:
   - Paying **1 Activation Point** allows a ranged unit to fire at any target in Line of Sight (unlimited range).
   - Attacker rolls die vs. Defender die $+$ Cover ($+1$ Soft, $+3$ Hard).
   - If Attacker roll $>$ Defender roll: target is **Wounded** and turned to the **"1" face**.
   - *Restriction*: A unit may move then shoot, but once it shoots, it cannot move again that turn.
   - Units showing "1" cannot perform actions for the rest of the round.
5. **Melee (Charging)**:
   - Charging into base-to-base contact costs no extra points beyond movement.
   - **Attacker Advantage**: Attacker rolls two dice of their type and takes the highest. Defender rolls one die.
   - Highest roll wins; loser is removed as a casualty.
   - **Ties**: Attacker rolls advantage die. If still tied, combat is unresolved and the attacker bounces back **2 inches**.
   - **Wounded Targets**: Engaging a unit showing "1" in melee automatically eliminates it without rolling.

### 4. End of Round
1. **Morale Test**: After Turn 4, both players roll a **d20**. You must roll strictly greater than your number of casualties ($roll > \text{casualties}$).
   - Failure: Your warband flees the field and you lose.
   - If both fail: The player with fewer casualties wins.
2. **Casualty Recovery Check**:
   - All surviving dice showing "1" roll their die:
   - **Even Roll**: Recovers and returns to normal face for the next round.
   - **Odd Roll**: Succumbs to wounds and is removed as a casualty.

---

## Assets (`skirmish.png` & `skirmish.ico`)

The game automatically searches for:
1. `skirmish.png`: Placed in this directory (`go/skirmish/skirmish.png`). Used as the decorative header banner in-game and as the high-res window icon source.
2. `skirmish.ico`: Placed in this directory (`go/skirmish/skirmish.ico`). Embedded into the Windows `.exe` using `windres` during the build.

> **Note**: Clean procedural fallback artwork and icons are built-in, so the game runs gracefully even if the image files are not yet present.

---

## Building

Run the PowerShell build script:
```powershell
./build.ps1
```

The resulting binaries are placed in:
- `bin/win/x64/SkirmishDice.exe` (Windows 64-bit GUI)
- `bin/win/x86/SkirmishDice.exe` (Windows 32-bit GUI)

When double-clicked or executed from a terminal, the game launches directly into the GUI with no console window.
