package service

import (
	"sync"
	"testing"
	"time"

	"scraps/internal/clock"
	"scraps/internal/commands"
	"scraps/internal/config"
	"scraps/internal/domain"
	"scraps/internal/events"
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

func TestExecuteSyncStateReturnsSnapshot(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	svc := NewGameService(config.Default(), clk, start)

	svc.mu.Lock()
	svc.state.Scrap = 7
	svc.mu.Unlock()

	cmd := commands.SyncState{ID: "sync-1"}
	result, err := svc.Execute(cmd)
	if err != nil {
		t.Fatalf("expected nil error got %v", err)
	}
	if result.State.Scrap != 7 {
		t.Fatalf("expected scrap 7 got %d", result.State.Scrap)
	}
}

func TestExecuteConcurrentSyncState(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	svc := NewGameService(config.Default(), clk, start)

	var wg sync.WaitGroup
	const workers = 50

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			cmd := commands.SyncState{ID: "sync-1"}
			_, _ = svc.Execute(cmd)
		}()
	}
	wg.Wait()
}

func TestExecuteSyncStateNoSettlementEventUnderOneSecond(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	svc := NewGameService(config.Default(), clk, start)

	clk.Advance(500 * time.Millisecond)
	result, err := svc.Execute(commands.SyncState{ID: "sync-1"})
	if err != nil {
		t.Fatalf("expected nil error got %v", err)
	}
	if len(result.Events) != 0 {
		t.Fatalf("expected 0 events got %d", len(result.Events))
	}
}

func TestExecuteSyncStateSettlementEvent(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	svc := NewGameService(config.Default(), clk, start)

	clk.Advance(10 * time.Second)
	result, err := svc.Execute(commands.SyncState{ID: "sync-1"})
	if err != nil {
		t.Fatalf("expected nil error got %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 event got %d", len(result.Events))
	}
	ev := result.Events[0]
	if ev.Type != events.EventTypeScrapSettled {
		t.Fatalf("expected ScrapSettled event got %s", ev.Type)
	}
	data, ok := ev.Data.(events.ScrapSettledData)
	if !ok {
		t.Fatalf("expected ScrapSettledData payload")
	}
	if data.Minted != 10 {
		t.Fatalf("expected Minted 10 got %d", data.Minted)
	}
	if !data.From.Equal(start) || !data.To.Equal(start.Add(10*time.Second)) {
		t.Fatalf("unexpected From/To: %+v", data)
	}
	if !ev.At.Equal(start.Add(10 * time.Second)) {
		t.Fatalf("expected At %v got %v", start.Add(10*time.Second), ev.At)
	}
	if ev.CommandID != "sync-1" {
		t.Fatalf("expected CommandID sync-1 got %s", ev.CommandID)
	}
}

func TestExecuteSyncStateNoEventWhenNoTimePasses(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	svc := NewGameService(config.Default(), clk, start)

	clk.Advance(1 * time.Second)
	first, err := svc.Execute(commands.SyncState{ID: "sync-1"})
	if err != nil {
		t.Fatalf("expected nil error got %v", err)
	}
	if len(first.Events) != 1 {
		t.Fatalf("expected 1 event got %d", len(first.Events))
	}

	second, err := svc.Execute(commands.SyncState{ID: "sync-2"})
	if err != nil {
		t.Fatalf("expected nil error got %v", err)
	}
	if len(second.Events) != 0 {
		t.Fatalf("expected 0 events got %d", len(second.Events))
	}
}

func TestExecuteEventIDsIncrement(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	svc := NewGameService(config.Default(), clk, start)

	clk.Advance(1 * time.Second)
	first, err := svc.Execute(commands.SyncState{ID: "sync-1"})
	if err != nil {
		t.Fatalf("expected nil error got %v", err)
	}
	if len(first.Events) != 1 || first.Events[0].ID != 1 {
		t.Fatalf("expected first event ID 1 got %+v", first.Events)
	}

	clk.Advance(1 * time.Second)
	second, err := svc.Execute(commands.SyncState{ID: "sync-2"})
	if err != nil {
		t.Fatalf("expected nil error got %v", err)
	}
	if len(second.Events) != 1 || second.Events[0].ID != 2 {
		t.Fatalf("expected second event ID 2 got %+v", second.Events)
	}
}

func TestListEventsFilteringAndLimit(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	svc := NewGameService(config.Default(), clk, start)

	clk.Advance(1 * time.Second)
	_, _ = svc.Execute(commands.SyncState{ID: "sync-1"})
	clk.Advance(1 * time.Second)
	_, _ = svc.Execute(commands.SyncState{ID: "sync-2"})
	clk.Advance(1 * time.Second)
	_, _ = svc.Execute(commands.SyncState{ID: "sync-3"})

	all := svc.ListEvents(0, 0)
	if len(all) != 3 {
		t.Fatalf("expected 3 events got %d", len(all))
	}

	filtered := svc.ListEvents(1, 1)
	if len(filtered) != 1 || filtered[0].ID != 2 {
		t.Fatalf("expected only event ID 2 got %+v", filtered)
	}
}

func TestListEventsReturnsCopy(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	svc := NewGameService(config.Default(), clk, start)

	clk.Advance(1 * time.Second)
	_, _ = svc.Execute(commands.SyncState{ID: "sync-1"})

	first := svc.ListEvents(0, 1)
	if len(first) != 1 {
		t.Fatalf("expected 1 event got %d", len(first))
	}
	first[0].Type = "mutated"

	second := svc.ListEvents(0, 1)
	if second[0].Type == "mutated" {
		t.Fatalf("expected internal event to remain unchanged")
	}
}

func TestExecuteSyncStateConcurrentEvents(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	svc := NewGameService(config.Default(), clk, start)

	clk.Advance(10 * time.Second)

	var wg sync.WaitGroup
	const workers = 50

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer wg.Done()
			cmd := commands.SyncState{ID: "sync"}
			_, _ = svc.Execute(cmd)
		}(i)
	}
	wg.Wait()

	events := svc.ListEvents(0, 0)
	if len(events) != 1 {
		t.Fatalf("expected 1 event got %d", len(events))
	}

	seen := make(map[uint64]struct{}, len(events))
	var last uint64
	for _, ev := range events {
		if _, ok := seen[ev.ID]; ok {
			t.Fatalf("duplicate event ID %d", ev.ID)
		}
		seen[ev.ID] = struct{}{}
		if ev.ID <= last {
			t.Fatalf("expected increasing IDs, got %d after %d", ev.ID, last)
		}
		last = ev.ID
	}
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

func TestExecuteUnlockComponentCraftingEmitsEvent(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.CraftComponentTechnologyCost = 100
	cfg.BaseScrapProduction = 0
	clk := &fakeClock{now: start}
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.Scrap = 100
	svc.mu.Unlock()

	result, err := svc.Execute(commands.UnlockComponentCrafting{ID: "unlock-1"})
	if err != nil {
		t.Fatalf("expected success got %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 event got %d", len(result.Events))
	}
	ev := result.Events[0]
	if ev.Type != events.EventTypeCraftingUnlocked {
		t.Fatalf("expected CraftingUnlocked event got %s", ev.Type)
	}
	if ev.CommandID != "unlock-1" {
		t.Fatalf("expected CommandID unlock-1 got %s", ev.CommandID)
	}
	data, ok := ev.Data.(events.CraftingUnlockedData)
	if !ok {
		t.Fatalf("expected CraftingUnlockedData payload")
	}
	if data.Cost != 100 {
		t.Fatalf("expected Cost 100 got %d", data.Cost)
	}
}

func TestExecuteUnlockComponentCraftingEmitsSettlementThenUnlock(t *testing.T) {
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
	result, err := svc.Execute(commands.UnlockComponentCrafting{ID: "unlock-1"})
	if err != nil {
		t.Fatalf("expected success got %v", err)
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected 2 events got %d", len(result.Events))
	}
	if result.Events[0].Type != events.EventTypeScrapSettled {
		t.Fatalf("expected first event ScrapSettled got %s", result.Events[0].Type)
	}
	if result.Events[1].Type != events.EventTypeCraftingUnlocked {
		t.Fatalf("expected second event CraftingUnlocked got %s", result.Events[1].Type)
	}
	if result.Events[0].CommandID != "unlock-1" || result.Events[1].CommandID != "unlock-1" {
		t.Fatalf("expected CommandID unlock-1 on both events")
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

func TestCraftComponentLocked(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	clk := &fakeClock{now: start}
	svc := NewGameService(cfg, clk, start)

	if err := svc.CraftComponent(); err != ErrCraftingLocked {
		t.Fatalf("expected ErrCraftingLocked got %v", err)
	}
}

func TestCraftComponentInsufficientScrap(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.CraftComponentCost = 10
	clk := &fakeClock{now: start}
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.CraftingUnlocked = true
	svc.state.Scrap = 9
	svc.mu.Unlock()

	if err := svc.CraftComponent(); err != ErrInsufficientScrap {
		t.Fatalf("expected ErrInsufficientScrap got %v", err)
	}
}

func TestCraftComponentAtCost(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.CraftComponentCost = 10
	cfg.CraftDurationSecs = 10
	clk := &fakeClock{now: start}
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.CraftingUnlocked = true
	svc.state.Scrap = 10
	svc.mu.Unlock()

	if err := svc.CraftComponent(); err != nil {
		t.Fatalf("expected success got %v", err)
	}

	got := svc.GetState()
	if got.Scrap != 0 {
		t.Fatalf("expected scrap 0 got %d", got.Scrap)
	}
	if got.ActiveCraft == nil {
		t.Fatalf("expected ActiveCraft")
	}
	if !got.ActiveCraft.StartedAt.Equal(start) {
		t.Fatalf("expected StartedAt %v got %v", start, got.ActiveCraft.StartedAt)
	}
	if !got.ActiveCraft.FinishesAt.Equal(start.Add(10 * time.Second)) {
		t.Fatalf("expected FinishesAt %v got %v", start.Add(10*time.Second), got.ActiveCraft.FinishesAt)
	}
	if got.ActiveCraft.ScrapCost != 10 {
		t.Fatalf("expected ScrapCost 10 got %d", got.ActiveCraft.ScrapCost)
	}
}

func TestCraftComponentInProgress(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	clk := &fakeClock{now: start}
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.CraftingUnlocked = true
	svc.state.Scrap = 20
	svc.state.ActiveCraft = &domain.CraftJob{
		StartedAt:  start,
		FinishesAt: start.Add(10 * time.Second),
		ScrapCost:  10,
	}
	svc.mu.Unlock()

	if err := svc.CraftComponent(); err != ErrCraftInProgress {
		t.Fatalf("expected ErrCraftInProgress got %v", err)
	}
}

func TestCraftComponentSettlesFirst(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.CraftComponentCost = 10
	cfg.BaseScrapProduction = 1
	clk := &fakeClock{now: start}
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.CraftingUnlocked = true
	svc.state.Scrap = 9
	svc.mu.Unlock()

	clk.Advance(1 * time.Second)
	if err := svc.CraftComponent(); err != nil {
		t.Fatalf("expected success got %v", err)
	}
}

func TestCraftComponentConcurrent(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.CraftComponentCost = 10
	clk := &fakeClock{now: start}
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.CraftingUnlocked = true
	svc.state.Scrap = 10
	svc.mu.Unlock()

	startCh := make(chan struct{})
	errCh := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			<-startCh
			errCh <- svc.CraftComponent()
		}()
	}
	close(startCh)
	wg.Wait()
	close(errCh)

	var success, inProgress int
	for err := range errCh {
		if err == nil {
			success++
		} else if err == ErrCraftInProgress {
			inProgress++
		} else {
			t.Fatalf("unexpected error %v", err)
		}
	}

	if success != 1 || inProgress != 1 {
		t.Fatalf("expected 1 success and 1 ErrCraftInProgress got %d success %d in progress", success, inProgress)
	}

	got := svc.GetState()
	if got.Scrap != 0 {
		t.Fatalf("expected scrap 0 got %d", got.Scrap)
	}
	if got.ActiveCraft == nil {
		t.Fatalf("expected ActiveCraft")
	}
}

func TestExecuteStartCraftComponentLocked(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.BaseScrapProduction = 0
	clock := &fakeClock{now: start}
	svc := NewGameService(cfg, clock, start)

	_, err := svc.Execute(commands.StartCraftComponent{ID: "start-1"})
	if err != ErrCraftingLocked {
		t.Fatalf("expected ErrCraftingLocked got %v", err)
	}
}

func TestExecuteStartCraftComponentInsufficient(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.CraftComponentCost = 10
	cfg.BaseScrapProduction = 0
	clock := &fakeClock{now: start}
	svc := NewGameService(cfg, clock, start)

	svc.mu.Lock()
	svc.state.CraftingUnlocked = true
	svc.state.Scrap = 9
	svc.mu.Unlock()

	_, err := svc.Execute(commands.StartCraftComponent{ID: "start-1"})
	if err != ErrInsufficientScrap {
		t.Fatalf("expected ErrInsufficientScrap got %v", err)
	}
}

func TestExecuteStartCraftComponentInProgress(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.CraftComponentCost = 10
	cfg.BaseScrapProduction = 0
	clock := &fakeClock{now: start}
	svc := NewGameService(cfg, clock, start)

	svc.mu.Lock()
	svc.state.CraftingUnlocked = true
	svc.state.Scrap = 20
	svc.state.ActiveCraft = &domain.CraftJob{
		StartedAt:  start,
		FinishesAt: start.Add(10 * time.Second),
		ScrapCost:  10,
	}
	svc.mu.Unlock()

	_, err := svc.Execute(commands.StartCraftComponent{ID: "start-1"})
	if err != ErrCraftInProgress {
		t.Fatalf("expected ErrCraftInProgress got %v", err)
	}
}

func TestExecuteStartCraftComponentEmitsEvent(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.CraftComponentCost = 10
	cfg.CraftDurationSecs = 10
	cfg.BaseScrapProduction = 0
	clock := &fakeClock{now: start}
	svc := NewGameService(cfg, clock, start)

	svc.mu.Lock()
	svc.state.CraftingUnlocked = true
	svc.state.Scrap = 10
	svc.mu.Unlock()

	result, err := svc.Execute(commands.StartCraftComponent{ID: "start-1"})
	if err != nil {
		t.Fatalf("expected success got %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 event got %d", len(result.Events))
	}
	ev := result.Events[0]
	if ev.Type != events.EventTypeComponentCraftStarted {
		t.Fatalf("expected ComponentCraftStarted got %s", ev.Type)
	}
	if ev.CommandID != "start-1" {
		t.Fatalf("expected CommandID start-1 got %s", ev.CommandID)
	}
	data, ok := ev.Data.(events.ComponentCraftStartedData)
	if !ok {
		t.Fatalf("expected ComponentCraftStartedData payload")
	}
	if data.Cost != 10 {
		t.Fatalf("expected Cost 10 got %d", data.Cost)
	}
	if !data.FinishesAt.Equal(start.Add(10 * time.Second)) {
		t.Fatalf("expected FinishesAt %v got %v", start.Add(10*time.Second), data.FinishesAt)
	}
}

func TestExecuteStartCraftComponentSettlementThenStart(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.CraftComponentCost = 10
	cfg.CraftDurationSecs = 10
	cfg.BaseScrapProduction = 1
	clock := &fakeClock{now: start}
	svc := NewGameService(cfg, clock, start)

	svc.mu.Lock()
	svc.state.CraftingUnlocked = true
	svc.state.Scrap = 9
	svc.mu.Unlock()

	clock.Advance(1 * time.Second)
	result, err := svc.Execute(commands.StartCraftComponent{ID: "start-1"})
	if err != nil {
		t.Fatalf("expected success got %v", err)
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected 2 events got %d", len(result.Events))
	}
	if result.Events[0].Type != events.EventTypeScrapSettled {
		t.Fatalf("expected first event ScrapSettled got %s", result.Events[0].Type)
	}
	if result.Events[1].Type != events.EventTypeComponentCraftStarted {
		t.Fatalf("expected second event ComponentCraftStarted got %s", result.Events[1].Type)
	}
	if result.Events[0].CommandID != "start-1" || result.Events[1].CommandID != "start-1" {
		t.Fatalf("expected CommandID start-1 on both events")
	}
}

func TestExecuteStartCraftComponentConcurrent(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.CraftComponentCost = 10
	cfg.BaseScrapProduction = 0
	clock := &fakeClock{now: start}
	svc := NewGameService(cfg, clock, start)

	svc.mu.Lock()
	svc.state.CraftingUnlocked = true
	svc.state.Scrap = 10
	svc.mu.Unlock()

	startCh := make(chan struct{})
	errCh := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			<-startCh
			_, err := svc.Execute(commands.StartCraftComponent{ID: "start-1"})
			errCh <- err
		}()
	}
	close(startCh)
	wg.Wait()
	close(errCh)

	var success, inProgress int
	for err := range errCh {
		if err == nil {
			success++
		} else if err == ErrCraftInProgress {
			inProgress++
		} else {
			t.Fatalf("unexpected error %v", err)
		}
	}
	if success != 1 || inProgress != 1 {
		t.Fatalf("expected 1 success and 1 ErrCraftInProgress got %d success %d in progress", success, inProgress)
	}

	eventsList := svc.ListEvents(0, 0)
	if len(eventsList) != 1 {
		t.Fatalf("expected 1 event got %d", len(eventsList))
	}
	if eventsList[0].Type != events.EventTypeComponentCraftStarted {
		t.Fatalf("expected ComponentCraftStarted got %s", eventsList[0].Type)
	}
}

func TestClaimCraftedComponentNoActiveCraft(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	cfg := config.Default()
	cfg.BaseScrapProduction = 0
	svc := NewGameService(cfg, clk, start)

	if _, err := svc.ClaimCraftedComponent(); err != ErrNoActiveCraft {
		t.Fatalf("expected ErrNoActiveCraft got %v", err)
	}
}

func TestClaimCraftedComponentNotComplete(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	cfg := config.Default()
	cfg.BaseScrapProduction = 0
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.ActiveCraft = &domain.CraftJob{
		StartedAt:  start,
		FinishesAt: start.Add(10 * time.Second),
		ScrapCost:  10,
	}
	svc.mu.Unlock()

	clk.Advance(9 * time.Second)
	if _, err := svc.ClaimCraftedComponent(); err != ErrCraftNotComplete {
		t.Fatalf("expected ErrCraftNotComplete got %v", err)
	}
}

func TestClaimCraftedComponentAtFinish(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	cfg := config.Default()
	cfg.BaseScrapProduction = 0
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.ActiveCraft = &domain.CraftJob{
		StartedAt:  start,
		FinishesAt: start.Add(10 * time.Second),
		ScrapCost:  10,
	}
	svc.mu.Unlock()

	clk.Advance(10 * time.Second)
	if gained, err := svc.ClaimCraftedComponent(); err != nil || gained != 1 {
		t.Fatalf("expected gain 1 got %d err %v", gained, err)
	}

	got := svc.GetState()
	if got.Components != 1 {
		t.Fatalf("expected Components 1 got %d", got.Components)
	}
	if got.ActiveCraft != nil {
		t.Fatalf("expected ActiveCraft cleared")
	}
}

func TestClaimCraftedComponentAfterFinish(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	cfg := config.Default()
	cfg.BaseScrapProduction = 0
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.ActiveCraft = &domain.CraftJob{
		StartedAt:  start,
		FinishesAt: start.Add(10 * time.Second),
		ScrapCost:  10,
	}
	svc.mu.Unlock()

	clk.Advance(15 * time.Second)
	if gained, err := svc.ClaimCraftedComponent(); err != nil || gained != 1 {
		t.Fatalf("expected gain 1 got %d err %v", gained, err)
	}
}

func TestClaimCraftedComponentTwice(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	cfg := config.Default()
	cfg.BaseScrapProduction = 0
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.ActiveCraft = &domain.CraftJob{
		StartedAt:  start,
		FinishesAt: start.Add(10 * time.Second),
		ScrapCost:  10,
	}
	svc.mu.Unlock()

	clk.Advance(12 * time.Second)
	if gained, err := svc.ClaimCraftedComponent(); err != nil || gained != 1 {
		t.Fatalf("expected gain 1 got %d err %v", gained, err)
	}
	if _, err := svc.ClaimCraftedComponent(); err != ErrNoActiveCraft {
		t.Fatalf("expected ErrNoActiveCraft got %v", err)
	}
}

func TestClaimCraftedComponentConcurrent(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	cfg := config.Default()
	cfg.BaseScrapProduction = 0
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.ActiveCraft = &domain.CraftJob{
		StartedAt:  start,
		FinishesAt: start.Add(10 * time.Second),
		ScrapCost:  10,
	}
	svc.mu.Unlock()

	clk.Advance(10 * time.Second)

	startCh := make(chan struct{})
	errCh := make(chan error, 20)
	resultCh := make(chan int64, 20)

	var wg sync.WaitGroup
	const workers = 20

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			<-startCh
			gained, err := svc.ClaimCraftedComponent()
			resultCh <- gained
			errCh <- err
		}()
	}
	close(startCh)
	wg.Wait()
	close(resultCh)
	close(errCh)

	var success, noActive int
	var totalGained int64
	for gained := range resultCh {
		totalGained += gained
	}
	for err := range errCh {
		if err == nil {
			success++
		} else if err == ErrNoActiveCraft {
			noActive++
		} else {
			t.Fatalf("unexpected error %v", err)
		}
	}

	if success != 1 || noActive != workers-1 {
		t.Fatalf("expected 1 success and %d ErrNoActiveCraft got %d success %d no active", workers-1, success, noActive)
	}
	if totalGained != 1 {
		t.Fatalf("expected total gained 1 got %d", totalGained)
	}

	got := svc.GetState()
	if got.Components != 1 {
		t.Fatalf("expected Components 1 got %d", got.Components)
	}
	if got.ActiveCraft != nil {
		t.Fatalf("expected ActiveCraft cleared")
	}
}

func TestClaimCraftedComponentEmitsEventOnSuccess(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := &fakeClock{now: start}
	cfg := config.Default()
	cfg.BaseScrapProduction = 0
	svc := NewGameService(cfg, clock, start)

	svc.mu.Lock()
	svc.state.ActiveCraft = &domain.CraftJob{
		StartedAt:  start,
		FinishesAt: start.Add(10 * time.Second),
		ScrapCost:  10,
	}
	svc.mu.Unlock()

	clock.Advance(10 * time.Second)
	result, err := svc.Execute(&commands.ClaimCraftedComponent{ID: "claim-1"})
	if err != nil {
		t.Fatalf("expected success got %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 event got %d", len(result.Events))
	}
	ev := result.Events[0]
	if ev.Type != events.EventTypeComponentCraftClaimed {
		t.Fatalf("expected ComponentCraftClaimed got %s", ev.Type)
	}
	if ev.CommandID != "claim-1" {
		t.Fatalf("expected CommandID claim-1 got %s", ev.CommandID)
	}
	data, ok := ev.Data.(events.ComponentCraftClaimedData)
	if !ok {
		t.Fatalf("expected ComponentCraftClaimedData payload")
	}
	if data.Gained != 1 {
		t.Fatalf("expected Gained 1 got %d", data.Gained)
	}
}

func TestClaimCraftedComponentNoEventOnFailure(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := &fakeClock{now: start}
	cfg := config.Default()
	cfg.BaseScrapProduction = 0
	svc := NewGameService(cfg, clock, start)

	result, err := svc.Execute(&commands.ClaimCraftedComponent{ID: "claim-1"})
	if err != ErrNoActiveCraft {
		t.Fatalf("expected ErrNoActiveCraft got %v", err)
	}
	if len(result.Events) != 0 {
		t.Fatalf("expected 0 events got %d", len(result.Events))
	}
}

func TestClaimCraftedComponentConcurrentEmitsSingleEvent(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := &fakeClock{now: start}
	cfg := config.Default()
	cfg.BaseScrapProduction = 0
	svc := NewGameService(cfg, clock, start)

	svc.mu.Lock()
	svc.state.ActiveCraft = &domain.CraftJob{
		StartedAt:  start,
		FinishesAt: start.Add(10 * time.Second),
		ScrapCost:  10,
	}
	svc.mu.Unlock()

	clock.Advance(10 * time.Second)

	startCh := make(chan struct{})
	var wg sync.WaitGroup
	const workers = 20
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			<-startCh
			_, _ = svc.Execute(&commands.ClaimCraftedComponent{ID: "claim-1"})
		}()
	}
	close(startCh)
	wg.Wait()

	eventsList := svc.ListEvents(0, 0)
	if len(eventsList) != 1 {
		t.Fatalf("expected 1 event got %d", len(eventsList))
	}
	if eventsList[0].Type != events.EventTypeComponentCraftClaimed {
		t.Fatalf("expected ComponentCraftClaimed got %s", eventsList[0].Type)
	}
}

func TestCancelCraftNoActiveCraft(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	svc := NewGameService(config.Default(), &fakeClock{now: start}, start)

	if err := svc.CancelCraft(); err != ErrNoActiveCraft {
		t.Fatalf("expected ErrNoActiveCraft got %v", err)
	}
}

func TestCancelCraftRefundsAndClears(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	svc := NewGameService(config.Default(), &fakeClock{now: start}, start)

	svc.mu.Lock()
	svc.state.ActiveCraft = &domain.CraftJob{
		StartedAt:  start,
		FinishesAt: start.Add(10 * time.Second),
		ScrapCost:  10,
	}
	svc.state.Scrap = 0
	svc.mu.Unlock()

	if err := svc.CancelCraft(); err != nil {
		t.Fatalf("expected success got %v", err)
	}

	got := svc.GetState()
	if got.Scrap != 10 {
		t.Fatalf("expected scrap 10 got %d", got.Scrap)
	}
	if got.ActiveCraft != nil {
		t.Fatalf("expected ActiveCraft cleared")
	}
}

func TestCancelCraftAfterCompletionRefundsNoComponent(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	cfg := config.Default()
	cfg.BaseScrapProduction = 0
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.ActiveCraft = &domain.CraftJob{
		StartedAt:  start,
		FinishesAt: start.Add(10 * time.Second),
		ScrapCost:  10,
	}
	svc.state.Scrap = 0
	svc.mu.Unlock()

	clk.Advance(20 * time.Second)
	if err := svc.CancelCraft(); err != nil {
		t.Fatalf("expected success got %v", err)
	}

	got := svc.GetState()
	if got.Scrap != 10 {
		t.Fatalf("expected scrap 10 got %d", got.Scrap)
	}
	if got.Components != 0 {
		t.Fatalf("expected Components 0 got %d", got.Components)
	}
	if got.ActiveCraft != nil {
		t.Fatalf("expected ActiveCraft cleared")
	}
}

func TestCancelCraftAllowsNewCraft(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.CraftComponentCost = 10
	clk := &fakeClock{now: start}
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.CraftingUnlocked = true
	svc.state.ActiveCraft = &domain.CraftJob{
		StartedAt:  start,
		FinishesAt: start.Add(10 * time.Second),
		ScrapCost:  10,
	}
	svc.state.Scrap = 0
	svc.mu.Unlock()

	if err := svc.CancelCraft(); err != nil {
		t.Fatalf("expected success got %v", err)
	}
	if err := svc.CraftComponent(); err != nil {
		t.Fatalf("expected craft success got %v", err)
	}
}

func TestCancelCraftVsClaimRace(t *testing.T) {
	// Both cancel and claim race against a completed craft; exactly one outcome should win.
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := &fakeClock{now: start}
	cfg := config.Default()
	cfg.BaseScrapProduction = 0
	svc := NewGameService(cfg, clk, start)

	svc.mu.Lock()
	svc.state.ActiveCraft = &domain.CraftJob{
		StartedAt:  start,
		FinishesAt: start.Add(10 * time.Second),
		ScrapCost:  10,
	}
	svc.state.Scrap = 0
	svc.mu.Unlock()

	clk.Advance(10 * time.Second)

	startCh := make(chan struct{})
	errCh := make(chan error, 2)
	claimCh := make(chan int64, 1)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-startCh
		errCh <- svc.CancelCraft()
	}()
	go func() {
		defer wg.Done()
		<-startCh
		gained, err := svc.ClaimCraftedComponent()
		if err == nil {
			claimCh <- gained
		}
		errCh <- err
	}()
	close(startCh)
	wg.Wait()
	close(errCh)
	close(claimCh)

	got := svc.GetState()
	if got.Components > 1 {
		t.Fatalf("expected Components <= 1 got %d", got.Components)
	}
	if got.Components == 1 && got.Scrap != 0 {
		t.Fatalf("expected no refund when component claimed")
	}
	if got.Components == 0 && got.Scrap != 10 {
		t.Fatalf("expected refund when component not claimed")
	}

	var success, noActive int
	for err := range errCh {
		if err == nil {
			success++
		} else if err == ErrNoActiveCraft {
			noActive++
		} else {
			t.Fatalf("unexpected error %v", err)
		}
	}
	if success != 1 || noActive != 1 {
		t.Fatalf("expected 1 success and 1 ErrNoActiveCraft got %d success %d no active", success, noActive)
	}
}
