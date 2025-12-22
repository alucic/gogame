package domain

import (
	"testing"
	"time"
)

func TestStateZeroValues(t *testing.T) {
	var st State
	if st.Scrap != 0 || st.Components != 0 || st.CraftingUnlocked {
		t.Fatalf("expected zero values in state")
	}
	if !st.LastSettledAt.Equal(time.Time{}) {
		t.Fatalf("expected zero LastSettledAt")
	}
	if st.ActiveCraft != nil {
		t.Fatalf("expected nil ActiveCraft")
	}
}

func TestCraftJobFields(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	job := CraftJob{
		StartedAt:  start,
		FinishesAt: start.Add(10 * time.Second),
		ScrapCost:  10,
	}

	if !job.StartedAt.Equal(start) || !job.FinishesAt.Equal(start.Add(10*time.Second)) {
		t.Fatalf("unexpected CraftJob timestamps")
	}
	if job.ScrapCost != 10 {
		t.Fatalf("expected ScrapCost 10 got %d", job.ScrapCost)
	}
}
