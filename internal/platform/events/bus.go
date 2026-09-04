// Package events is the in-process domain-event bus (plan.md → platform;
// research note: notification is the classic domain-event recipient). Contexts
// publish typed domain events; the bus fans them out to registered handlers.
// It is synchronous and in-process: a handler returning an error is logged by
// the caller but does not roll back the transaction that produced the event
// (outbox/persistence for events is out of v1 scope).
package events

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Event is the minimal marker a domain event must satisfy.
type Event interface {
	// EventName returns a stable, lowercase dotted name, e.g. "badge.earned".
	EventName() string
}

// Handler processes one published event.
type Handler interface {
	HandleEvent(ctx context.Context, e Event) error
}

// HandlerFunc adapts a func to Handler.
type HandlerFunc func(ctx context.Context, e Event) error

// HandleEvent implements Handler.
func (f HandlerFunc) HandleEvent(ctx context.Context, e Event) error { return f(ctx, e) }

// Bus fans out events to handlers registered by event name.
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// New returns an empty Bus.
func New() *Bus {
	return &Bus{handlers: make(map[string][]Handler)}
}

// Subscribe registers h for every event named name. It panics on a nil
// handler; registering the same handler twice for one name runs it twice.
func (b *Bus) Subscribe(name string, h Handler) {
	if h == nil {
		panic("events: nil handler")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[name] = append(b.handlers[name], h)
}

// Publish dispatches e to every handler registered for its event name.
// Errors are aggregated and returned; callers should log and continue, not
// fail their transaction (outbox is future scope).
func (b *Bus) Publish(ctx context.Context, e Event) error {
	b.mu.RLock()
	hs := b.handlers[e.EventName()]
	b.mu.RUnlock()

	var errs []error
	for _, h := range hs {
		if err := h.HandleEvent(ctx, e); err != nil {
			slog.Error("events: handler failed", "event", e.EventName(), "error", err)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("events: %d handler(s) failed for %s", len(errs), e.EventName())
	}
	return nil
}
