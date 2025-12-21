package domain

import "time"

type State struct {
	Scrap            uint64
	Components       uint64
	CraftingUnlocked bool
	LastSettledAt    time.Time
	ActiveCraft      *CraftJob
}

type CraftJob struct {
	StartedAt  time.Time
	FinishesAt time.Time
	ScrapCost  uint64
}
