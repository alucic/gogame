package clock

import (
	"testing"
	"time"
)

type FakeClock struct {
	now time.Time
}

func NewFakeClock(start time.Time) *FakeClock {
	return &FakeClock{now: start}
}

func (f *FakeClock) Now() time.Time {
	return f.now
}

func (f *FakeClock) Advance(d time.Duration) {
	f.now = f.now.Add(d)
}

func TestRealClockNow(t *testing.T) {
	clk := RealClock{}
	if clk.Now().IsZero() {
		t.Fatalf("expected non-zero time")
	}
}

func TestFakeClockAdvance(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := NewFakeClock(start)

	if !clk.Now().Equal(start) {
		t.Fatalf("expected start time")
	}

	clk.Advance(1500 * time.Millisecond)
	want := start.Add(1500 * time.Millisecond)
	if !clk.Now().Equal(want) {
		t.Fatalf("expected %v got %v", want, clk.Now())
	}
}
