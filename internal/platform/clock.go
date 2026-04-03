package platform

import "time"

// Clock abstracts time for testability.
type Clock interface {
	Now() time.Time
}

// RealClock uses the system clock.
type RealClock struct{}

// NewClock creates a production clock.
func NewClock() Clock {
	return RealClock{}
}

// Now returns the current system time.
func (RealClock) Now() time.Time {
	return time.Now()
}

// FakeClock is a controllable test clock.
type FakeClock struct {
	current time.Time
}

// NewFakeClock creates a fake clock pinned to the given instant.
func NewFakeClock(now time.Time) *FakeClock {
	return &FakeClock{current: now}
}

// Now returns the current fake instant.
func (c *FakeClock) Now() time.Time {
	return c.current
}

// Set updates the fake instant.
func (c *FakeClock) Set(now time.Time) {
	c.current = now
}
