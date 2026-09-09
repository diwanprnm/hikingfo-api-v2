package domain

// MountainNotFound is returned when a mountain lookup misses.
type MountainNotFound struct{}

func (MountainNotFound) Error() string { return "catalogue: mountain not found" }

// SlugConflict is returned when a create/update hits the unique slug constraint.
type SlugConflict struct{}

func (SlugConflict) Error() string { return "catalogue: slug already in use" }
