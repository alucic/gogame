# gogame

# Resource spending
You don't spend all at once, you can work toward something constantly for as long as you have resources. If something costs 100 and takes 10h to build, and you build 2 hours, you can pause it and come back to it later and continue 8h. In other words, time and money spent towards doing something accumulate, and each building has resource per hour cost.

# kraftat se mogu relics
- maksimalno tjedno mozes imat 20h speedup queue po 1h izkraftano tako da ljudi mogu planirat malo bolje FS i tajming
- Fleet speed kraft artifact, ili neki booster za produkciju s gornjim tjednim limitom

Meni trenutno 8h50min FS ne odgovara, bilo bi odlicno kad bi ga moga uštelit s tim itemima na nesto sta mi paše

### Core resources
- Scrap
- Energy
- Fuel

Scrap income should be:
- Predictable (casual-friendly)
- Exploitable via how you spend it (hardcore depth)

Energy – the pacing brake
Energy should limit how much you can do at once, not how rich you are.
**Collection method** Power infrastructure, controlled zones

Fuel

#### Strategic resources
##### Data
##### Components
- Unlock power
- Shape the tech tree
- Enable non-military victories
- Encourage conflict over specific assets

**Collection Method** Advanced production chains, rare sites
##### Relics

### Technology
Component Crafting


### Crafting
Crafting is the first game mechanic that a player is presented with. Crafting takes scraps and converts them to components given time. Crafting is unlocked by Component Crafting technology.


## Epic: Core Resource Generation

- UC1: Player gains Scrap at a flat hourly rate
- UC2: Player gains Energy at a flat hourly rate
- UC3: Actions take longer when Energy is low

## Usage

## Service API

The primary API is `internal/service.GameService`, which manages in-memory
game state with a mutex and exposes:

- `NewGameService`
- `GetState`
- `Settle`
- `UnlockComponentCrafting`
- `CraftComponent`
- `ClaimCraftedComponent`
- `CancelCraft`
- `Execute`

## Commands and Events

Commands are typed request objects (in `internal/commands`) that describe
what operation to run against the game state. The `Execute` method accepts a
command and returns a `Result` containing the post-command state snapshot.

Events are lightweight records (in `internal/events`) that can be emitted
during command execution. They are returned in the `Result.Events` slice and
can later drive UI updates or analytics. Currently the API returns an empty
event list, but the type is in place for future use.

Run tests with:

```sh
make test
```

Run coverage with:

```sh
make coverage
```

Run race detection with:

```sh
go test ./... -race
```
