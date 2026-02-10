package globaltime

import "time"

// Time is a wrapper for time.Time
type Time interface {
	Now() time.Time
}

// RealTime implements Time using the real system clock
type RealTime struct{}

func (RealTime) Now() time.Time {
	return time.Now()
}
