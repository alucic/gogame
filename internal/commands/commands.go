package commands

// Command represents a typed command for the GameService executor.
type Command interface {
	CommandID() string
	Name() string
}

// SyncState requests a state snapshot without changing game state.
type SyncState struct {
	ID string
}

func (c SyncState) CommandID() string {
	return c.ID
}

func (c SyncState) Name() string {
	return "SyncState"
}

// Settle requests scrap settlement and exposes the minted amount.
type Settle struct {
	ID          string
	MintedScrap    uint64
}

func (c *Settle) CommandID() string {
	return c.ID
}

func (c *Settle) Name() string {
	return "Settle"
}

// UnlockComponentCrafting attempts to unlock component crafting.
type UnlockComponentCrafting struct {
	ID string
}

func (c UnlockComponentCrafting) CommandID() string {
	return c.ID
}

func (c UnlockComponentCrafting) Name() string {
	return "UnlockComponentCrafting"
}

// CraftComponent starts a component crafting job.
type CraftComponent struct {
	ID string
}

func (c CraftComponent) CommandID() string {
	return c.ID
}

func (c CraftComponent) Name() string {
	return "CraftComponent"
}

// StartCraftComponent starts a component crafting job.
type StartCraftComponent struct {
	ID string
}

func (c StartCraftComponent) CommandID() string {
	return c.ID
}

func (c StartCraftComponent) Name() string {
	return "StartCraftComponent"
}

// ClaimCraftedComponent claims a completed craft and exposes components gained.
type ClaimCraftedComponent struct {
	ID               string
	ComponentsGained uint64
}

func (c *ClaimCraftedComponent) CommandID() string {
	return c.ID
}

func (c *ClaimCraftedComponent) Name() string {
	return "ClaimCraftedComponent"
}

// CancelCraft cancels a craft job and refunds scrap.
type CancelCraft struct {
	ID string
}

func (c CancelCraft) CommandID() string {
	return c.ID
}

func (c CancelCraft) Name() string {
	return "CancelCraft"
}
