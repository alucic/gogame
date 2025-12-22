package service

import (
	"errors"
	"sync"
	"time"

	"scraps/internal/clock"
	"scraps/internal/commands"
	"scraps/internal/config"
	"scraps/internal/domain"
	"scraps/internal/events"
)

// GameService provides the concurrency-safe game API.
type GameService struct {
	mu  sync.Mutex
	cfg config.Config
	clock clock.Clock
	state domain.State
	eventSequence int64
	events        []events.Event
}

// Result is the outcome of executing a command.
type Result struct {
	State  domain.State
	Events []events.Event
}

var (
	// ErrInsufficientScrap indicates the player lacks enough scrap.
	ErrInsufficientScrap = errors.New("insufficient scrap")
	// ErrAlreadyUnlocked indicates the crafting technology is already unlocked.
	ErrAlreadyUnlocked   = errors.New("already unlocked")
	// ErrCraftingLocked indicates crafting has not been unlocked yet.
	ErrCraftingLocked    = errors.New("crafting locked")
	// ErrCraftInProgress indicates a craft job is already active.
	ErrCraftInProgress   = errors.New("craft already in progress")
	// ErrNoActiveCraft indicates there is no active craft job.
	ErrNoActiveCraft     = errors.New("no active craft")
	// ErrCraftNotComplete indicates the craft job has not finished yet.
	ErrCraftNotComplete  = errors.New("craft not complete")
)

// NewGameService initializes a new game service with empty state.
func NewGameService(cfg config.Config, clk clock.Clock, startTime time.Time) *GameService {
	return &GameService{
		cfg: cfg,
		clock: clk,
		state: domain.State{
			LastSettledAt: startTime,
		},
	}
}

// GetState returns a snapshot of the current state.
func (s *GameService) GetState() domain.State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked()
}

// Settle mints scrap based on whole seconds elapsed since last settlement.
func (s *GameService) Settle() int64 {
	command := &commands.Settle{
		CommandIDValue: "settle",
	}
	_, _ = s.Execute(command)
	return int64(command.MintedScrap)
}

// UnlockComponentCrafting unlocks component crafting and deducts the cost.
func (s *GameService) UnlockComponentCrafting() error {
	command := commands.UnlockComponentCrafting{
		CommandIDValue: "unlock_component_crafting",
	}
	_, err := s.Execute(command)
	return err
}

// CraftComponent starts a single craft job and deducts scrap immediately.
func (s *GameService) CraftComponent() error {
	command := commands.CraftComponent{
		CommandIDValue: "craft_component",
	}
	_, err := s.Execute(command)
	return err
}

// ClaimCraftedComponent claims a finished craft job and grants one component.
func (s *GameService) ClaimCraftedComponent() (int64, error) {
	command := &commands.ClaimCraftedComponent{
		CommandIDValue: "claim_crafted_component",
	}
	_, err := s.Execute(command)
	return int64(command.ComponentsGained), err
}

// CancelCraft cancels an active craft job and refunds its scrap cost.
func (s *GameService) CancelCraft() error {
	command := commands.CancelCraft{
		CommandIDValue: "cancel_craft",
	}
	_, err := s.Execute(command)
	return err
}

// ListEvents returns events after the given ID, up to limit entries.
func (s *GameService) ListEvents(sinceID int64, limit int) []events.Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	var filtered []events.Event
	for _, ev := range s.events {
		if ev.ID <= sinceID {
			continue
		}
		filtered = append(filtered, ev)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}

	out := make([]events.Event, len(filtered))
	copy(out, filtered)
	return out
}

// Execute runs a command and returns the resulting state snapshot.
func (s *GameService) Execute(cmd commands.Command) (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result Result
	var err error
	var eventsList []events.Event

	switch command := cmd.(type) {
	case commands.SyncState:
		s.settleLocked()
	case *commands.Settle:
		command.MintedScrap = s.settleLocked()
	case commands.UnlockComponentCrafting:
		s.settleLocked()
		err = s.unlockComponentCraftingLocked()
	case commands.CraftComponent:
		s.settleLocked()
		err = s.craftComponentLocked()
	case *commands.ClaimCraftedComponent:
		var gained uint64
		gained, err = s.claimCraftedComponentLocked()
		command.ComponentsGained = gained
	case commands.CancelCraft:
		err = s.cancelCraftLocked()
	}

	s.eventSequence++
	eventItem := events.Event{
		ID:        s.eventSequence,
		At:        s.clock.Now(),
		CommandID: cmd.CommandID(),
		Type:      events.EventType(cmd.Name()),
		Data:      nil,
	}
	s.events = append(s.events, eventItem)
	eventsList = append(eventsList, eventItem)

	result.State = s.snapshotLocked()
	result.Events = eventsList
	return result, err
}

func (s *GameService) snapshotLocked() domain.State {
	snap := s.state
	if s.state.ActiveCraft != nil {
		ac := *s.state.ActiveCraft
		snap.ActiveCraft = &ac
	}
	return snap
}

func (s *GameService) settleLocked() uint64 {
	now := s.clock.Now()
	elapsed := now.Sub(s.state.LastSettledAt).Seconds()
	elapsedSeconds := int64(elapsed)
	if elapsedSeconds <= 0 {
		return 0
	}

	mint := uint64(elapsedSeconds) * s.cfg.BaseScrapProduction
	s.state.Scrap += mint
	s.state.LastSettledAt = s.state.LastSettledAt.Add(time.Duration(elapsedSeconds) * time.Second)
	return mint
}

func (s *GameService) unlockComponentCraftingLocked() error {
	if s.state.CraftingUnlocked {
		return ErrAlreadyUnlocked
	}
	if s.state.Scrap < s.cfg.CraftComponentTechnologyCost {
		return ErrInsufficientScrap
	}

	s.state.Scrap -= s.cfg.CraftComponentTechnologyCost
	s.state.CraftingUnlocked = true
	return nil
}

func (s *GameService) craftComponentLocked() error {
	if !s.state.CraftingUnlocked {
		return ErrCraftingLocked
	}
	if s.state.ActiveCraft != nil {
		return ErrCraftInProgress
	}
	if s.state.Scrap < s.cfg.CraftComponentCost {
		return ErrInsufficientScrap
	}

	now := s.clock.Now()
	s.state.Scrap -= s.cfg.CraftComponentCost
	s.state.ActiveCraft = &domain.CraftJob{
		StartedAt:  now,
		FinishesAt: now.Add(time.Duration(s.cfg.CraftDurationSecs) * time.Second),
		ScrapCost:  s.cfg.CraftComponentCost,
	}
	return nil
}

func (s *GameService) claimCraftedComponentLocked() (uint64, error) {
	if s.state.ActiveCraft == nil {
		return 0, ErrNoActiveCraft
	}
	if s.clock.Now().Before(s.state.ActiveCraft.FinishesAt) {
		return 0, ErrCraftNotComplete
	}

	s.state.Components += 1
	s.state.ActiveCraft = nil
	return 1, nil
}

func (s *GameService) cancelCraftLocked() error {
	if s.state.ActiveCraft == nil {
		return ErrNoActiveCraft
	}

	s.state.Scrap += s.state.ActiveCraft.ScrapCost
	s.state.ActiveCraft = nil
	return nil
}
