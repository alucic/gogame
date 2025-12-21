package service

import (
	"errors"
	"sync"
	"time"

	"scraps/internal/clock"
	"scraps/internal/config"
	"scraps/internal/domain"
)

// GameService provides the concurrency-safe game API.
type GameService struct {
	mu  sync.Mutex
	cfg config.Config
	clock clock.Clock
	state domain.State
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
	snap := s.state
	if s.state.ActiveCraft != nil {
		ac := *s.state.ActiveCraft
		snap.ActiveCraft = &ac
	}
	return snap
}

// Settle mints scrap based on whole seconds elapsed since last settlement.
func (s *GameService) Settle() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	mint := s.settleLocked()
	return int64(mint)
}

// UnlockComponentCrafting unlocks component crafting and deducts the cost.
func (s *GameService) UnlockComponentCrafting() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.settleLocked()
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

// CraftComponent starts a single craft job and deducts scrap immediately.
func (s *GameService) CraftComponent() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.settleLocked()
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

// ClaimCraftedComponent claims a finished craft job and grants one component.
func (s *GameService) ClaimCraftedComponent() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
