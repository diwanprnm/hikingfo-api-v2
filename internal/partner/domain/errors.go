package domain

// NoticeNotFound is returned when a notice lookup misses.
type NoticeNotFound struct{}

func (NoticeNotFound) Error() string { return "partner: notice not found" }

// RequestNotFound is returned when a request lookup misses.
type RequestNotFound struct{}

func (RequestNotFound) Error() string { return "partner: request not found" }

// ErrAlreadyMatched is returned when attempting to send a request to a user with whom
// an accepted match already exists on the same mountain and overlapping dates.
type ErrAlreadyMatched struct{}

func (ErrAlreadyMatched) Error() string { return "partner: already matched" }
