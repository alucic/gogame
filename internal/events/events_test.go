package events

import "testing"

func TestNewEvent(t *testing.T) {
	ev := New("test")
	if ev.Name != "test" {
		t.Fatalf("expected name test got %s", ev.Name)
	}
}
