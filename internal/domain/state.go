package domain

import "time"

// State holds the current in-memory game state.
type State struct {
	Scrap            uint64
	Components       uint64
	CraftingUnlocked bool
	LastSettledAt    time.Time
	ActiveCraft      *CraftJob
}

// CraftJob represents an in-progress component craft.
type CraftJob struct {
	StartedAt  time.Time
	FinishesAt time.Time
	ScrapCost  uint64
}

func cloneCraftJob(job *CraftJob) *CraftJob {
	if job == nil {
		return nil
	}
	clone := *job
	return &clone
}
