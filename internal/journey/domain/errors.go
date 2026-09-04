package domain

// HikeLogNotFound is returned when a hike log lookup misses.
type HikeLogNotFound struct{}

func (HikeLogNotFound) Error() string { return "journey: hike log not found" }

// JourneyPostNotFound is returned when a journey post lookup misses.
type JourneyPostNotFound struct{}

func (JourneyPostNotFound) Error() string { return "journey: post not found" }
