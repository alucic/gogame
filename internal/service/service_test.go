package service

import (
	"sync"
	"testing"
	"time"

	"scraps/internal/clock"
	"scraps/internal/config"
	"scraps/internal/domain"
)

type fakeClock struct {
	now time.Time
}

func (f *fakeClock) Now() time.Time {
	return f.now
}

func (f *fakeClock) Advance(d time.Duration) {
	f.now = f.now.Add(d)
}

func TestNewGameServiceInitialState(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	svc := NewGameService(config.Default(), clock.RealClock{}, start)

	got := svc.GetState()
	if got.Scrap != 0 || got.Components != 0 || got.CraftingUnlocked {
		t.Fatalf("unexpected initial counters: %+v", got)
	}
	if got.ActiveCraft != nil {
		t.Fatalf("expected nil ActiveCraft")
	}
	if !got.LastSettledAt.Equal(start) {
		t.Fatalf("expected LastSettledAt %v got %v", start, got.LastSettledAt)
	}
}

func TestGetStateReturnsCopy(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	svc := NewGameService(config.Default(), clock.RealClock{}, start)

	original := &domain.CraftJob{
		StartedAt:  start,
		FinishesAt: start.Add(10 * time.Second),
		ScrapCost:  10,
	}

	svc.mu.Lock()
	svc.state.Scrap = 5
	svc.state.ActiveCraft = original
	svc.mu.Unlock()

	snap := svc.GetState()
	snap.Scrap = 99
	if snap.ActiveCraft == nil {
		t.Fatalf("expected ActiveCraft in snapshot")
	}
	snap.ActiveCraft.ScrapCost = 999

	svc.mu.Lock()
	defer svc.mu.Unlock()
	if svc.state.Scrap != 5 {
		t.Fatalf("expected internal Scrap to remain 5 got %d", svc.state.Scrap)
	}
	if svc.state.ActiveCraft == nil {
		t.Fatalf("expected internal ActiveCraft")
	}
	if svc.state.ActiveCraft.ScrapCost != 10 {
		t.Fatalf("expected internal ScrapCost to remain 10 got %d", svc.state.ActiveCraft.ScrapCost)
	}
	if svc.state.ActiveCraft == snap.ActiveCraft {
		t.Fatalf("expected deep copy of ActiveCraft")
	}
}

func TestGetStateConcurrent(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	svc := NewGameService(config.Default(), clock.RealClock{}, start)

	var wg sync.WaitGroup
	const workers = 50
	const iterations = 200

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = svc.GetState()
			}
		}()
	}
	wg.Wait()
}

func TestSettleNoMintUnderOneSecond(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	svc := NewGameService(config.Default(), clk, start)

	clk.Advance(500 * time.Millisecond)
	if mint := svc.Settle(); mint != 0 {
		t.Fatalf("expected mint 0 got %d", mint)
	}

	got := svc.GetState()
	if got.Scrap != 0 {
		t.Fatalf("expected scrap 0 got %d", got.Scrap)
	}
	if !got.LastSettledAt.Equal(start) {
		t.Fatalf("expected LastSettledAt unchanged")
	}
}

func TestSettleOneSecond(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	svc := NewGameService(config.Default(), clk, start)

	clk.Advance(1 * time.Second)
	if mint := svc.Settle(); mint != 1 {
		t.Fatalf("expected mint 1 got %d", mint)
	}

	got := svc.GetState()
	if got.Scrap != 1 {
		t.Fatalf("expected scrap 1 got %d", got.Scrap)
	}
	if !got.LastSettledAt.Equal(start.Add(1 * time.Second)) {
		t.Fatalf("expected LastSettledAt advanced by 1s")
	}
}

func TestSettleTenSeconds(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	svc := NewGameService(config.Default(), clk, start)

	clk.Advance(10 * time.Second)
	if mint := svc.Settle(); mint != 10 {
		t.Fatalf("expected mint 10 got %d", mint)
	}

	got := svc.GetState()
	if got.Scrap != 10 {
		t.Fatalf("expected scrap 10 got %d", got.Scrap)
	}
	if !got.LastSettledAt.Equal(start.Add(10 * time.Second)) {
		t.Fatalf("expected LastSettledAt advanced by 10s")
	}
}

func TestSettleTwiceWithoutAdvance(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	svc := NewGameService(config.Default(), clk, start)

	clk.Advance(1 * time.Second)
	if mint := svc.Settle(); mint != 1 {
		t.Fatalf("expected mint 1 got %d", mint)
	}
	if mint := svc.Settle(); mint != 0 {
		t.Fatalf("expected mint 0 got %d", mint)
	}
}

func TestSettlePartialSecondsCarry(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	svc := NewGameService(config.Default(), clk, start)

	clk.Advance(1900 * time.Millisecond)
	if mint := svc.Settle(); mint != 1 {
		t.Fatalf("expected mint 1 got %d", mint)
	}
	got := svc.GetState()
	if got.Scrap != 1 {
		t.Fatalf("expected scrap 1 got %d", got.Scrap)
	}
	if !got.LastSettledAt.Equal(start.Add(1 * time.Second)) {
		t.Fatalf("expected LastSettledAt advanced by 1s")
	}
	if mint := svc.Settle(); mint != 0 {
		t.Fatalf("expected mint 0 got %d", mint)
	}
}

func TestSettleConcurrent(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	svc := NewGameService(config.Default(), clk, start)

	clk.Advance(10 * time.Second)

	var wg sync.WaitGroup
	const workers = 50

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			_ = svc.Settle()
		}()
	}
	wg.Wait()

	got := svc.GetState()
	if got.Scrap != 10 {
		t.Fatalf("expected scrap 10 got %d", got.Scrap)
	}
	if !got.LastSettledAt.Equal(start.Add(10 * time.Second)) {
		t.Fatalf("expected LastSettledAt advanced by 10s")
	}
}

func TestUnlockComponentCraftingInsufficient(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.CraftComponentTechnologyCost = 100
	clk := &fakeClock{now: start}
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.Scrap = 99
	svc.mu.Unlock()

	if err := svc.UnlockComponentCrafting(); err != ErrInsufficientScrap {
		t.Fatalf("expected ErrInsufficientScrap got %v", err)
	}
}

func TestUnlockComponentCraftingAtCost(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.CraftComponentTechnologyCost = 100
	clk := &fakeClock{now: start}
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.Scrap = 100
	svc.mu.Unlock()

	if err := svc.UnlockComponentCrafting(); err != nil {
		t.Fatalf("expected success got %v", err)
	}

	got := svc.GetState()
	if got.Scrap != 0 {
		t.Fatalf("expected scrap 0 got %d", got.Scrap)
	}
	if !got.CraftingUnlocked {
		t.Fatalf("expected CraftingUnlocked true")
	}
}

func TestUnlockComponentCraftingIdempotent(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.CraftComponentTechnologyCost = 100
	clk := &fakeClock{now: start}
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.Scrap = 100
	svc.mu.Unlock()

	if err := svc.UnlockComponentCrafting(); err != nil {
		t.Fatalf("expected success got %v", err)
	}
	if err := svc.UnlockComponentCrafting(); err != ErrAlreadyUnlocked {
		t.Fatalf("expected ErrAlreadyUnlocked got %v", err)
	}

	got := svc.GetState()
	if got.Scrap != 0 {
		t.Fatalf("expected scrap 0 got %d", got.Scrap)
	}
}

func TestUnlockComponentCraftingSettlesFirst(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.CraftComponentTechnologyCost = 100
	cfg.BaseScrapProduction = 1
	clk := &fakeClock{now: start}
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.Scrap = 99
	svc.mu.Unlock()

	clk.Advance(1 * time.Second)
	if err := svc.UnlockComponentCrafting(); err != nil {
		t.Fatalf("expected success got %v", err)
	}
}

func TestUnlockComponentCraftingConcurrent(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.CraftComponentTechnologyCost = 100
	clk := &fakeClock{now: start}
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.Scrap = 100
	svc.mu.Unlock()

	startCh := make(chan struct{})
	errCh := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			<-startCh
			errCh <- svc.UnlockComponentCrafting()
		}()
	}
	close(startCh)
	wg.Wait()
	close(errCh)

	var success, already int
	for err := range errCh {
		if err == nil {
			success++
		} else if err == ErrAlreadyUnlocked {
			already++
		} else {
			t.Fatalf("unexpected error %v", err)
		}
	}

	if success != 1 || already != 1 {
		t.Fatalf("expected 1 success and 1 ErrAlreadyUnlocked got %d success %d already", success, already)
	}

	got := svc.GetState()
	if got.Scrap != 0 {
		t.Fatalf("expected scrap 0 got %d", got.Scrap)
	}
	if !got.CraftingUnlocked {
		t.Fatalf("expected CraftingUnlocked true")
	}
}
