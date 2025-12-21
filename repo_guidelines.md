Below is a set of **tiny, backend-only implementation prompts** you can hand to an engineer (or use yourself) to build this repo incrementally. Each prompt includes:

* **Goal**
* **Public method APIs** to expose
* **Definition of Done** (DoD) with **tests required**
* Notes on concurrency + lazy settlement where relevant

Assumptions locked in (from you):

* In-memory only
* Lazy settlement for Scrap
* Single player/world/season (no IDs yet)
* Energy ignored (v1)
* Component crafting: produces **exactly 1 Component**
* Cancel craft: **full Scrap refund**
* Components are only accumulated (no usage yet)
* Constants:

  * Scrap base rate = **1 scrap/sec** (3600/hr)
  * Unlock cost = **100 scrap**
  * Craft cost = **10 scrap**
  * Craft duration = **10 sec**
* Must be **concurrency-safe** (Mutex OK)
* 100% test coverage target

---

# Repo Architecture (target shape)

**Packages**

* `internal/clock` – clock abstraction for deterministic tests
* `internal/domain` – state structs + pure helpers (no mutex)
* `internal/service` – main API, mutex, lazy settlement, validation
* `internal/config` – constants/config struct

**Primary API surface**
Expose a Go type like:

```go
type GameService struct { ... }

func NewGameService(cfg config.Config, clk clock.Clock) *GameService
```

Everything else stays internal for now.

---

# Prompt 1 — Create repo skeleton + tooling

## Goal

Initialize a new Go module with a test setup that makes **100% coverage visible**.

## Do

* Create `go.mod` (module name can be placeholder like `scraps`).
* Add packages with empty files:

  * `internal/config/config.go`
  * `internal/clock/clock.go`
  * `internal/domain/state.go`
  * `internal/service/service.go`
* Add `Makefile` (or just document commands) for:

  * `go test ./... -coverprofile=coverage.out`
  * `go tool cover -func=coverage.out`

## Definition of Done

* `go test ./...` succeeds (even if tests are trivial placeholders)
* Running coverage outputs coverage numbers (even low for now)
* No external dependencies required

---

# Prompt 2 — Clock abstraction (deterministic time)

## Goal

Introduce a clock interface so lazy settlement is testable without sleeping.

## Public API

`internal/clock/clock.go`

```go
package clock

import "time"

type Clock interface {
    Now() time.Time
}

type RealClock struct{}
func (RealClock) Now() time.Time
```

Add a test-only fake clock in `internal/clock/fake_clock_test.go` (in-package tests are fine).

## Definition of Done

* Unit test verifies `FakeClock` can advance time deterministically
* 100% coverage for `internal/clock`

---

# Prompt 3 — Config constants

## Goal

Centralize tunables.

## Public API

`internal/config/config.go`

```go
package config

import "time"

type Config struct {
    ScrapPerSecond     int64         // 1
    UnlockCraftCost    int64         // 100
    CraftComponentCost int64         // 10
    CraftDuration      time.Duration // 10s
}

func Default() Config
```

## Definition of Done

* `Default()` returns the values you specified
* Unit tests assert exact defaults
* 100% coverage for `internal/config`

---

# Prompt 4 — Domain state (pure data)

## Goal

Define the minimal state for single-player progression.

## Public API

`internal/domain/state.go`

```go
package domain

import "time"

type State struct {
    Scrap                int64
    Components           int64
    CraftingUnlocked     bool

    // Lazy settlement
    LastSettledAt        time.Time

    // Crafting job (optional)
    ActiveCraft          *CraftJob
}

type CraftJob struct {
    StartedAt time.Time
    FinishesAt time.Time
    ScrapCost int64
}
```

No mutex here—domain is a dumb struct.

## Definition of Done

* Tests cover any constructors/helpers you add (if you add none, keep it simple)
* 100% coverage for domain package files you created

---

# Prompt 5 — Service: initialization + GetState snapshot

## Goal

Create the concurrency-safe service wrapper with state snapshot.

## Public API

`internal/service/service.go`

```go
package service

import (
  "sync"
  "scraps/internal/config"
  "scraps/internal/clock"
  "scraps/internal/domain"
)

type GameService struct {
    mu  sync.Mutex
    cfg config.Config
    clk clock.Clock
    st  domain.State
}

func NewGameService(cfg config.Config, clk clock.Clock, startTime time.Time) *GameService

// Returns a copy/snapshot. No pointers that allow mutation.
func (s *GameService) GetState() domain.State
```

Initialization rules:

* `LastSettledAt = startTime`
* `Scrap = 0`, `Components=0`, `CraftingUnlocked=false`, `ActiveCraft=nil`

## Definition of Done

* Tests:

  * `GetState()` returns expected initial state
  * Returned state is a copy (mutating returned value does not mutate internal state)
* Concurrency test:

  * Run `GetState()` concurrently from many goroutines with `-race` (add a test that spawns goroutines)

---

# Prompt 6 — Lazy settlement for Scrap (core mechanic)

## Goal

Implement settlement so scrap increases with elapsed time since `LastSettledAt`.

## Public API additions

```go
// Forces settlement and returns how much Scrap was minted by this call.
func (s *GameService) Settle() int64
```

Settlement rule:

* `elapsedSeconds = floor(now - LastSettledAt)`
* `mint = elapsedSeconds * cfg.ScrapPerSecond`
* `Scrap += mint`
* `LastSettledAt += elapsedSeconds seconds` (advance by whole seconds)
* If elapsed < 1s: mint 0 and do not change timestamps

## Definition of Done

* Tests (must be deterministic with FakeClock):

  * After 0.5s: no mint
  * After 1s: +1 scrap
  * After 10s: +10 scrap
  * Two consecutive calls without time advancing: second mints 0
  * Partial seconds carry: e.g. advance 1.9s, settle mints 1 and leaves 0.9s implicit (via LastSettledAt advance-by-whole-seconds)
* Concurrency test:

  * N goroutines call `Settle()` concurrently; final scrap equals exactly one settlement result (no double mint)
* `GetState()` should **not** auto-settle yet (keep explicit for clarity)

---

# Prompt 7 — Unlock Component Crafting

## Goal

Add the feature gate + Scrap sink.

## Public API additions

```go
var (
    ErrInsufficientScrap = errors.New("insufficient scrap")
    ErrAlreadyUnlocked   = errors.New("already unlocked")
)

func (s *GameService) UnlockComponentCrafting() error
```

Rules:

* Method performs `Settle()` first (inside same lock) so player gets credited before spending
* If already unlocked → `ErrAlreadyUnlocked`
* If scrap < `cfg.UnlockCraftCost` → `ErrInsufficientScrap`
* Else:

  * `Scrap -= UnlockCraftCost`
  * `CraftingUnlocked = true`

## Definition of Done

* Tests:

  * Unlock fails with insufficient scrap
  * Unlock succeeds at exactly 100 scrap
  * Unlock is idempotent-safe: second call returns `ErrAlreadyUnlocked`, no scrap change
  * Unlock calls settle implicitly: start at 99 scrap, advance time by 1 sec, unlock succeeds
* Concurrency tests:

  * Two goroutines attempt unlock simultaneously:

    * Exactly one succeeds
    * Scrap deducted only once
    * Other returns `ErrAlreadyUnlocked` (or insufficient if you prefer—pick one and test it; recommend `ErrAlreadyUnlocked` after success)

---

# Prompt 8 — Start crafting a Component (single queue)

## Goal

Allow starting a craft job; deduct scrap immediately.

## Public API additions

```go
var (
    ErrCraftingLocked    = errors.New("crafting locked")
    ErrCraftInProgress   = errors.New("craft already in progress")
)

func (s *GameService) StartCraftComponent() error
```

Rules:

* Calls `Settle()` first
* Requires `CraftingUnlocked == true` else `ErrCraftingLocked`
* Requires `ActiveCraft == nil` else `ErrCraftInProgress`
* Requires `Scrap >= cfg.CraftComponentCost` else `ErrInsufficientScrap`
* On success:

  * Deduct Scrap immediately by 10
  * Create `ActiveCraft` with:

    * `StartedAt = now`
    * `FinishesAt = now + cfg.CraftDuration`
    * `ScrapCost = cfg.CraftComponentCost`

## Definition of Done

* Tests:

  * Cannot craft before unlock
  * Cannot craft with insufficient scrap
  * Can craft at exactly 10 scrap
  * Cannot start second craft while one is active
  * Starting craft settles first (e.g., at 9 scrap + advance 1 sec → can start)
* Concurrency tests:

  * Two goroutines call `StartCraftComponent()` simultaneously:

    * Exactly one succeeds, other gets `ErrCraftInProgress` (or locked/insufficient depending on ordering; recommended `ErrCraftInProgress` once active)

---

# Prompt 9 — Complete crafting (claim on finish)

## Goal

Convert finished craft job into 1 Component.

## Public API additions

```go
var ErrCraftNotComplete = errors.New("craft not complete")
var ErrNoActiveCraft    = errors.New("no active craft")

// Claim the finished craft; returns components gained (0 or 1)
func (s *GameService) ClaimCraftedComponent() (int64, error)
```

Rules:

* If `ActiveCraft == nil` → `ErrNoActiveCraft`
* If `now < ActiveCraft.FinishesAt` → `ErrCraftNotComplete`
* Else:

  * `Components += 1`
  * `ActiveCraft = nil`
  * return `1, nil`

## Definition of Done

* Tests:

  * Claim with no craft: error
  * Claim before finish: `ErrCraftNotComplete`
  * Claim at finish time: succeeds, adds 1 component, clears craft
  * Claim after finish time: succeeds
  * Claim twice: second returns `ErrNoActiveCraft`
* Concurrency tests:

  * Multiple goroutines attempt claim after completion:

    * Exactly one gets 1 component
    * Others get `ErrNoActiveCraft`

---

# Prompt 10 — Cancel crafting (full refund)

## Goal

Cancel active craft and refund scrap cost fully.

## Public API additions

```go
func (s *GameService) CancelCraft() error
```

Rules:

* If `ActiveCraft == nil` → `ErrNoActiveCraft`
* Else:

  * `Scrap += ActiveCraft.ScrapCost` (full refund)
  * `ActiveCraft = nil`

Cancel should work whether craft is complete or not (simplest behavior).

## Definition of Done

* Tests:

  * Cancel with no craft → `ErrNoActiveCraft`
  * Cancel immediately after start refunds 10 scrap and clears craft
  * Cancel after craft would have completed still refunds and clears (and does NOT grant component)
  * After cancel, starting craft is allowed again
* Concurrency tests:

  * Cancel and claim race:

    * At most one of {refund, component} happens
    * State remains consistent (no double benefit)
  * Implement deterministically by locking; write a test that attempts both in goroutines and asserts invariants:

    * `Components` is either 0 or 1
    * Scrap reflects either refund or not, but never both reward paths

---

# Prompt 11 — Service-level “advance simulation” helper (test-only optional)

## Goal

Make tests readable and keep coverage high without leaking internals.

Option A (recommended): keep fake clock usage in tests only
Option B: add a **test-only helper** in `_test.go`.

No production API needed.

## Definition of Done

* All tests remain deterministic
* `go test ./... -race` passes
* `go test ./... -cover` reports **100%** for this repo

---

# Notes on 100% coverage and concurrency

* Use `-race` in CI locally.
* Keep methods small and validation explicit so every branch is testable.
* For concurrency tests, you don’t need to prove ordering—prove **invariants**:

  * No double mint
  * No double unlock
  * No double craft start
  * No double claim
  * Cancel vs claim cannot both award value

---

## What you’ll have after these prompts

A tiny, rock-solid in-memory backend that supports these use cases:

* Accumulate Scrap via lazy settlement
* Unlock component crafting with flat Scrap cost
* Start 1 craft at a time
* Claim component after 10 seconds
* Cancel craft anytime for full refund
* Fully deterministic tests with 100% coverage
* Concurrency-safe behavior under race conditions

---

If you want, I can now turn this into the **actual Go code** in one shot (service + tests) following exactly these prompts and constants—just say “generate the code.”
