package domain

// MountainNotFound is returned when a mountain lookup misses.
type MountainNotFound struct{}

func (MountainNotFound) Error() string { return "catalogue: mountain not found" }
