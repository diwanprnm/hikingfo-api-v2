package domain

import "hikingfo/backend/internal/shared/kerr"

// BadgeNotFound is returned when a badge config lookup misses.
func BadgeNotFound() *kerr.Error { return kerr.NotFound("badge not found") }

// BadgeConfigError wraps an internal failure during badge config retrieval.
func BadgeConfigError(cause error) *kerr.Error {
	return kerr.WrapInternal("could not load badge configs", cause)
}

// JourneyQueryError wraps an internal failure during journey data retrieval.
func JourneyQueryError(cause error) *kerr.Error {
	return kerr.WrapInternal("could not load journey data", cause)
}
