package service

import (
	"errors"
	"sync"
	"time"

	"scraps/internal/clock"
	"scraps/internal/config"
	"scraps/internal/domain"
)

type GameService struct {
	mu  sync.Mutex
	cfg config.Config
	clock clock.Clock
	state domain.State
}

var (
	ErrInsufficientScrap = errors.New("insufficient scrap")
	ErrAlreadyUnlocked   = errors.New("already unlocked")
	ErrCraftingLocked    = errors.New("crafting locked")
	ErrCraftInProgress   = errors.New("craft already in progress")
)

func NewGameService(cfg config.Config, clk clock.Clock, startTime time.Time) *GameService {
	return &GameService{
		cfg: cfg,
		clock: clk,
		state: domain.State{
			LastSettledAt: startTime,
		},
	}
}

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

func (s *GameService) Settle() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	mint := s.settleLocked()
	return int64(mint)
}

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
