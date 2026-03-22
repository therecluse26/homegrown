package shared

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
)

// ─── Interfaces ───────────────────────────────────────────────────────────────

// DomainEvent is the interface all domain events must implement.
// Events MUST be defined in the emitting domain's events.go. [CODING §8.4]
type DomainEvent interface {
	EventName() string
}

// DomainEventHandler handles a specific domain event type.
// Handlers MUST be defined in the consuming domain's event_handlers.go. [CODING §8.4]
type DomainEventHandler interface {
	Handle(ctx context.Context, event DomainEvent) error
}

// ─── EventBus ─────────────────────────────────────────────────────────────────

// EventBus dispatches domain events to registered handlers using reflect.Type dispatch.
// Phase 1: synchronous in-process dispatch. No message broker or serialization. [§11.2]
type EventBus struct {
	mu       sync.RWMutex
	handlers map[reflect.Type][]DomainEventHandler
}

// NewEventBus creates a new EventBus.
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[reflect.Type][]DomainEventHandler),
	}
}

// Subscribe registers a handler for a specific event type.
// MUST be called at startup only (in cmd/server/main.go). [CODING §8.4]
func (b *EventBus) Subscribe(eventType reflect.Type, handler DomainEventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Publish dispatches a domain event to all registered handlers.
//
// Handler errors are logged but do NOT fail the publish — the publishing domain's
// operation has already committed successfully. No retry in Phase 1. [§11.3]
func (b *EventBus) Publish(ctx context.Context, event DomainEvent) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	eventType := reflect.TypeOf(event)
	handlers, ok := b.handlers[eventType]
	if !ok {
		return nil
	}

	for _, handler := range handlers {
		if err := handler.Handle(ctx, event); err != nil {
			slog.Error("event handler failed",
				"event_type", event.EventName(),
				"handler", fmt.Sprintf("%T", handler),
				"error", err,
			)
			// Handler errors are logged, not propagated.
			// The domain operation that triggered the event has already completed.
		}
	}

	return nil
}
