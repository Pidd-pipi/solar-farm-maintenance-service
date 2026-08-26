package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Notification is one outbound alert queued for a maintenance operator.
type Notification struct {
	ID       string
	RecordID string
	Kind     string
	Message  string
	At       string
}

// Notifier queues notifications and dispatches them through a caller supplied
// handler. Dispatch honours the context: cancellation stops further attempts.
type Notifier struct {
	mu       sync.Mutex
	queue    []Notification
	maxRetry int
}

func newNotifier(maxRetry int) *Notifier {
	if maxRetry < 0 {
		maxRetry = 0
	}
	return &Notifier{queue: []Notification{}, maxRetry: maxRetry}
}

// Enqueue adds a notification to the outbox.
func (n *Notifier) Enqueue(recordID, kind, message string) Notification {
	n.mu.Lock()
	defer n.mu.Unlock()
	item := Notification{
		ID:       fmt.Sprintf("ntf-%06d", len(n.queue)+1),
		RecordID: recordID,
		Kind:     kind,
		Message:  message,
		At:       time.Now().UTC().Format(time.RFC3339Nano),
	}
	n.queue = append(n.queue, item)
	return item
}

// Dispatch drains the outbox by sending each notification through handler.
// A failed send is retried up to maxRetry times with backoff; context
// cancellation aborts the whole drain immediately.
func (n *Notifier) Dispatch(ctx context.Context, handler func(Notification) error) error {
	for {
		n.mu.Lock()
		if len(n.queue) == 0 {
			n.mu.Unlock()
			return nil
		}
		item := n.queue[0]
		n.queue = n.queue[1:]
		n.mu.Unlock()

		attempts := 0
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			err := handler(item)
			if err == nil {
				break
			}
			attempts++
			if attempts > n.maxRetry {
				return fmt.Errorf("%w: notify %s after %d attempts", ErrOpsPolicy, item.ID, attempts)
			}
			if err := opsDelay(ctx, opsBackoff(attempts)); err != nil {
				return err
			}
		}
	}
}

// Pending returns the number of notifications still queued.
func (n *Notifier) Pending() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.queue)
}

// Reset drops all queued notifications and reclaims storage.
func (n *Notifier) Reset() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.queue = []Notification{}
}
