package clock

import "time"

// Clock abstracts time for deterministic tests.
type Clock interface {
	Now() time.Time
}

type RealClock struct{}

// Now returns the current time using the system clock.
func (RealClock) Now() time.Time {
	return time.Now()
}
