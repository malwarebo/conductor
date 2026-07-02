package worker

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/malwarebo/conductor/models"
)

type fakeClaimer struct {
	mu     sync.Mutex
	events []*models.WebhookEvent
}

func (f *fakeClaimer) ClaimPendingEvents(_ context.Context, limit int, _ time.Duration) ([]*models.WebhookEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.events) == 0 {
		return nil, nil
	}
	if limit > len(f.events) {
		limit = len(f.events)
	}
	batch := f.events[:limit]
	f.events = f.events[limit:]
	return batch, nil
}

type fakeProcessor struct {
	mu   sync.Mutex
	seen map[string]int
}

func (f *fakeProcessor) ProcessClaimedEvent(_ context.Context, e *models.WebhookEvent) error {
	f.mu.Lock()
	f.seen[e.ID]++
	f.mu.Unlock()
	return nil
}

func (f *fakeProcessor) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.seen)
}

func (f *fakeProcessor) maxSeen() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	max := 0
	for _, c := range f.seen {
		if c > max {
			max = c
		}
	}
	return max
}

func TestWebhookPoolProcessesEachEventOnce(t *testing.T) {
	const total = 50
	events := make([]*models.WebhookEvent, total)
	for i := range events {
		events[i] = &models.WebhookEvent{ID: fmt.Sprintf("evt-%d", i)}
	}

	claimer := &fakeClaimer{events: events}
	proc := &fakeProcessor{seen: make(map[string]int)}

	pool := NewWebhookPool(claimer, proc, Config{
		Workers:      5,
		BatchSize:    8,
		PollInterval: 5 * time.Millisecond,
	})
	pool.Start(context.Background())

	deadline := time.After(5 * time.Second)
	for proc.count() < total {
		select {
		case <-deadline:
			pool.Stop()
			t.Fatalf("timed out: processed %d/%d events", proc.count(), total)
		case <-time.After(10 * time.Millisecond):
		}
	}
	pool.Stop()

	if got := proc.count(); got != total {
		t.Fatalf("expected %d distinct events, got %d", total, got)
	}
	if m := proc.maxSeen(); m != 1 {
		t.Fatalf("expected each event processed exactly once, but one was processed %d times", m)
	}
}

func TestWebhookPoolStopIsGracefulWhenIdle(t *testing.T) {
	claimer := &fakeClaimer{}
	proc := &fakeProcessor{seen: make(map[string]int)}

	pool := NewWebhookPool(claimer, proc, Config{Workers: 3, PollInterval: 5 * time.Millisecond})
	pool.Start(context.Background())

	done := make(chan struct{})
	go func() {
		pool.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("pool.Stop did not return promptly when idle")
	}
}
