// Package clock abstracts time to keep domain logic and tests deterministic.
package clock

import "time"

// Clock is the interface used by all domain code that needs the current time.
type Clock interface {
	Now() time.Time
}

// Real returns the actual system time. Use in production wiring.
type Real struct{}

func (Real) Now() time.Time { return time.Now() }

// Fixed returns a constant time. Use in tests.
type Fixed struct {
	T time.Time
}

func (f Fixed) Now() time.Time { return f.T }
