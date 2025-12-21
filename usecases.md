Below is **one concrete, end-to-end early-game user flow**, written so an engineer, PM, or QA can all reason about the same thing.
This is **one happy path**, no branches, no edge cases yet.

---

# Concrete User Flow: From New Player to First Component Use

## User Story (plain English)

> As a new player, I want to understand how Scrap and Energy work, unlock crafting, craft my first Component, and use it to make an action more efficient — without failing or getting stuck.

---

## Step-by-Step Flow

### Step 1: Player enters the game (initial state)

**System state**

* Scrap = `S0`
* Energy = `E0`
* Component Crafting = **locked**
* Components = `0`
* No crafting jobs running

**Player experience**

* Player sees Scrap and Energy slowly increasing over time
* Player can perform basic actions that cost Scrap and take time

**Design intent**

* Player learns: *time + Scrap = progress*
* No complexity yet

---

### Step 2: Player notices Crafting is locked

**UI / UX**

* Player sees a disabled section or callout:

  > “Component Crafting – Locked”

* Tooltip or hint:

  > “Unlock to convert Scrap into efficiency upgrades.”

**System state**

* No change

**Design intent**

* Curiosity
* Clear promise: *Scrap can do more*

---

### Step 3: Player unlocks Component Crafting

**Player action**

* Player clicks “Unlock Component Crafting”
* Confirms Scrap cost

**System behavior**

* Deduct flat Scrap cost immediately
* Persist:

  * `componentCraftingUnlocked = true`

**System state after**

* Scrap = `S0 - UNLOCK_COST`
* Energy unchanged
* Component Crafting = **unlocked**

**Design intent**

* “Aha” moment
* Scrap permanently converted into capability
* No randomness, no regret trap

---

### Step 4: Player starts first Component craft

**Player action**

* Player selects “Craft Component”
* Confirms craft

**System behavior**

* Deduct Scrap immediately
* Start crafting timer
* Lock crafting queue (1 job max)

**System state**

* Scrap = `S1 - CRAFT_SCRAP_COST`
* Crafting job:

  * `started_at`
  * `finishes_at`
* Components = `0`

**Design intent**

* Introduce time-based crafting
* Player understands waiting is okay

---

### Step 5: Crafting completes

**System behavior**

* Timer completes
* Add `+1 Component` to inventory
* Clear crafting queue

**System state**

* Components = `1`
* No active crafting job

**Player experience**

* Visual confirmation
* Component appears as a usable resource

**Design intent**

* Reward loop closes
* No ambiguity about success

---

### Step 6: Player performs an action using a Component

**Player action**

* Player selects a Scrap-costing action
* Toggles “Use Component” for this action

**System behavior**

* Consume 1 Component
* Apply efficiency effect (e.g. reduced Scrap cost or faster completion)
* Execute action

**System state after**

* Components = `0`
* Action completes more efficiently than normal

**Design intent**

* Immediate, visible payoff
* Reinforces: *Components = efficiency*

---

### Step 7: Player returns to baseline loop

**System state**

* Player continues generating Scrap and Energy
* Can choose to:

  * Craft more Components
  * Save Scrap
  * Spend Scrap on other actions

**Design intent**

* No lock-in
* No dead ends
* Early mistakes are recoverable with time

---

## Why this flow matters

This single flow proves that:

* Scrap is always relevant
* Crafting is optional but rewarding
* Complexity is gated
* The game is understandable within one session
* Hardcore players optimize timing
* Casual players can just wait and progress

---

## What we intentionally did NOT include

* Energy math edge cases
* Failure states
* Multiple crafting queues
* Fuel
* Relics
* Strategic branching

Those come later.
