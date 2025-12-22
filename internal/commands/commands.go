package commands

// Command represents a typed command for the GameService executor.
type Command interface {
	CommandID() string
	Name() string
}

// SyncState requests a state snapshot without changing game state.
type SyncState struct {
	CommandIDValue string
}

func (c SyncState) CommandID() string {
	return c.CommandIDValue
}

func (c SyncState) Name() string {
	return "SyncState"
}

// Settle requests scrap settlement and exposes the minted amount.
type Settle struct {
	CommandIDValue string
	MintedScrap    uint64
}

func (c *Settle) CommandID() string {
	return c.CommandIDValue
}

func (c *Settle) Name() string {
	return "Settle"
}

// UnlockComponentCrafting attempts to unlock component crafting.
type UnlockComponentCrafting struct {
	CommandIDValue string
}

func (c UnlockComponentCrafting) CommandID() string {
	return c.CommandIDValue
}

func (c UnlockComponentCrafting) Name() string {
	return "UnlockComponentCrafting"
}

// CraftComponent starts a component crafting job.
type CraftComponent struct {
	CommandIDValue string
}

func (c CraftComponent) CommandID() string {
	return c.CommandIDValue
}

func (c CraftComponent) Name() string {
	return "CraftComponent"
}

// ClaimCraftedComponent claims a completed craft and exposes components gained.
type ClaimCraftedComponent struct {
	CommandIDValue   string
	ComponentsGained uint64
}

func (c *ClaimCraftedComponent) CommandID() string {
	return c.CommandIDValue
}

func (c *ClaimCraftedComponent) Name() string {
	return "ClaimCraftedComponent"
}

// CancelCraft cancels a craft job and refunds scrap.
type CancelCraft struct {
	CommandIDValue string
}

func (c CancelCraft) CommandID() string {
	return c.CommandIDValue
}

func (c CancelCraft) Name() string {
	return "CancelCraft"
}
