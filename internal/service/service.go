package service

import (
	"sync"
	"time"

	"scraps/internal/clock"
	"scraps/internal/config"
	"scraps/internal/domain"
)

type GameService struct {
	mu  sync.Mutex
	cfg config.Config
	clk clock.Clock
	st  domain.State
}

func NewGameService(cfg config.Config, clk clock.Clock, startTime time.Time) *GameService {
	return &GameService{
		cfg: cfg,
		clk: clk,
		st: domain.State{
			LastSettledAt: startTime,
		},
	}
}

func (s *GameService) GetState() domain.State {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := s.st
	if s.st.ActiveCraft != nil {
		ac := *s.st.ActiveCraft
		snap.ActiveCraft = &ac
	}
	return snap
}

func (s *GameService) Settle() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clk.Now()
	elapsed := now.Sub(s.st.LastSettledAt).Seconds()
	elapsedSeconds := int64(elapsed)
	if elapsedSeconds <= 0 {
		return 0
	}

	mint := uint64(elapsedSeconds) * s.cfg.BaseScrapProduction
	s.st.Scrap += mint
	s.st.LastSettledAt = s.st.LastSettledAt.Add(time.Duration(elapsedSeconds) * time.Second)
	return int64(mint)
}
