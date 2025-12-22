package events

import (
	"testing"
	"time"
)

func TestEventFields(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	ev := Event{
		ID:        1,
		At:        now,
		CommandID: "cmd-1",
		Type:      EventType("SyncState"),
		Data:      "payload",
	}

	if ev.ID != 1 {
		t.Fatalf("expected ID 1 got %d", ev.ID)
	}
	if !ev.At.Equal(now) {
		t.Fatalf("expected At %v got %v", now, ev.At)
	}
	if ev.CommandID != "cmd-1" {
		t.Fatalf("expected CommandID cmd-1 got %s", ev.CommandID)
	}
	if ev.Type != EventType("SyncState") {
		t.Fatalf("expected Type SyncState got %s", ev.Type)
	}
	if ev.Data != "payload" {
		t.Fatalf("expected Data payload got %v", ev.Data)
	}
}

func TestNewEvent(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	ev := New(2, now, "cmd-2", EventType("Test"), 42)
	if ev.ID != 2 || ev.CommandID != "cmd-2" || ev.Type != EventType("Test") || ev.Data != 42 {
		t.Fatalf("unexpected event from New: %+v", ev)
	}
	if !ev.At.Equal(now) {
		t.Fatalf("expected At %v got %v", now, ev.At)
	}
}
