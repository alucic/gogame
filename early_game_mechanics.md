# Early Game Resource & Crafting System – Engineer Overview

## Scope

This document describes **early-game resource mechanics** and **component crafting** only.

Out of scope:

* Fuel (mid-game)
* Relics (late-game)
* Progression paths
* Combat balance
* Scaling systems

The goal is to establish a **simple, recoverable early loop** built around Scrap, Energy, and Components.

---

## 1. Resource Types

### Core Resources

#### Scrap

* Primary currency
* Required for all early-game actions
* Flat production rate for all players

#### Energy

* Pacing resource
* Flat production rate
* Affects how fast actions complete
* Does not block actions outright

---

### Strategic Resource

#### Component

* Tier-2 resource
* Represents efficiency improvements
* Crafted using Scrap and time
* Consumable
* Applied manually per action

Components do **not** increase power directly; they reduce costs or time, make things more efficient.

---

## 2. Resource Generation

### Scrap Generation

* All players generate Scrap passively over time
* Early-game Scrap income is **flat and identical** for all players

Formula:

```
ScrapPerHour = BASE_SCRAP
```

---

### Energy Generation

* All players generate Energy passively over time
* Energy influences action speed

Formula:

```
EnergyPerHour = BASE_ENERGY
```

Energy behavior is intentionally simple in early game:

* Low Energy = slower actions
* No hard failure or lockout

---

## 3. Tech Unlock: Component Crafting

### Name

Working name: **Component Crafting**
(Alternate name candidates: *Mechanics*, *Assembly*, *Fabrication*)

### Unlock Rules

* One-time unlock per player
* Costs a **flat amount of Scrap**
* Scrap spent is permanently removed (sink)
* Unlock is irreversible

Purpose:

* Create an “aha” moment
* Gate complexity
* Introduce strategic spending early

---

## 4. Component Crafting System

### Inputs

* Scrap
* Time
* (Energy passively affects speed)

### Output

* 1 Component per craft

### Crafting Rules

* Only **one crafting job** can run at a time (early-game)
* Scrap is deducted immediately on craft start
* Component is granted on completion
* No RNG
* No quality tiers

---

### Crafting Time (conceptual)

```
CraftTime = BASE_CRAFT_TIME
Adjusted by available Energy
```

Exact Energy math is intentionally flexible.

---

## 5. Component Usage

### Application

* Components are applied **manually per action**
* They are **consumed on use**
* No permanent bonuses

### Early-Game Effects (examples)

* Reduce Scrap cost of an action
* Reduce action completion time
* Reduce efficiency penalties

Components are **optional optimizations**, not requirements.

---

## 6. Early-Game Loop Summary

1. Player generates Scrap and Energy over time
2. Player spends Scrap on basic actions
3. Player unlocks Component Crafting using Scrap
4. Player crafts Components using Scrap + time
5. Player consumes Components to perform actions more efficiently
6. Mistakes are recoverable by waiting and generating more Scrap

---

## 7. Design Principles (Why this works)

* Minimal onboarding: only Scrap and Energy matter
* No early failure states
* No permanent power imbalance
* Skill expression comes from **timing and spending**
* System is easy to extend later without breaking fundamentals

---

## 8. Engineer Implementation Notes (Non-binding)

* Resources can be modeled as:

  * `type`
  * `amount`
  * `generation_rate`
* Component crafting should support:

  * One active job
  * Time-based completion
* Components should be:

  * Inventory items
  * Consumable
  * Applied explicitly to actions

---

## Open Items (Intentionally Deferred)

* Energy math specifics
* Multiple crafting queues
* Component variants
* Mid-game Fuel
* Late-game Relics

