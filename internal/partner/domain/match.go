package domain

import "time"

// HasDateOverlap reports whether two date ranges overlap (inclusive).
func HasDateOverlap(aStart, aEnd, bStart, bEnd time.Time) bool {
	return !aEnd.Before(bStart) && !bEnd.Before(aStart)
}

// ComputeExpiry returns the expiry timestamp for a notice or request.
// Per spec: expires_at = trip_end (same day, end of day in UTC).
func ComputeExpiry(tripEnd time.Time) time.Time {
	return time.Date(tripEnd.Year(), tripEnd.Month(), tripEnd.Day(), 23, 59, 59, 0, time.UTC)
}
