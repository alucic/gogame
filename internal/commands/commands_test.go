package commands

import "testing"

func TestSyncStateCommand(t *testing.T) {
	cmd := SyncState{CommandIDValue: "sync-1"}
	if cmd.CommandID() != "sync-1" {
		t.Fatalf("expected CommandID sync-1 got %s", cmd.CommandID())
	}
	if cmd.Name() != "SyncState" {
		t.Fatalf("expected name SyncState got %s", cmd.Name())
	}
}

func TestSettleCommand(t *testing.T) {
	cmd := &Settle{CommandIDValue: "settle-1"}
	if cmd.CommandID() != "settle-1" {
		t.Fatalf("expected CommandID settle-1 got %s", cmd.CommandID())
	}
	if cmd.Name() != "Settle" {
		t.Fatalf("expected name Settle got %s", cmd.Name())
	}
}

func TestUnlockComponentCraftingCommand(t *testing.T) {
	cmd := UnlockComponentCrafting{CommandIDValue: "unlock-1"}
	if cmd.CommandID() != "unlock-1" {
		t.Fatalf("expected CommandID unlock-1 got %s", cmd.CommandID())
	}
	if cmd.Name() != "UnlockComponentCrafting" {
		t.Fatalf("expected name UnlockComponentCrafting got %s", cmd.Name())
	}
}

func TestCraftComponentCommand(t *testing.T) {
	cmd := CraftComponent{CommandIDValue: "craft-1"}
	if cmd.CommandID() != "craft-1" {
		t.Fatalf("expected CommandID craft-1 got %s", cmd.CommandID())
	}
	if cmd.Name() != "CraftComponent" {
		t.Fatalf("expected name CraftComponent got %s", cmd.Name())
	}
}

func TestClaimCraftedComponentCommand(t *testing.T) {
	cmd := &ClaimCraftedComponent{CommandIDValue: "claim-1"}
	if cmd.CommandID() != "claim-1" {
		t.Fatalf("expected CommandID claim-1 got %s", cmd.CommandID())
	}
	if cmd.Name() != "ClaimCraftedComponent" {
		t.Fatalf("expected name ClaimCraftedComponent got %s", cmd.Name())
	}
}

func TestCancelCraftCommand(t *testing.T) {
	cmd := CancelCraft{CommandIDValue: "cancel-1"}
	if cmd.CommandID() != "cancel-1" {
		t.Fatalf("expected CommandID cancel-1 got %s", cmd.CommandID())
	}
	if cmd.Name() != "CancelCraft" {
		t.Fatalf("expected name CancelCraft got %s", cmd.Name())
	}
}
