//go:build integration

package stores_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/malwarebo/conductor/internal/worker"
	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/stores"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("conductor_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(90*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	db, err := gorm.Open(pgdriver.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm: %v", err)
	}
	if err := db.AutoMigrate(&models.WebhookEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func seedPending(t *testing.T, store *stores.WebhookStore, n int) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < n; i++ {
		ev := &models.WebhookEvent{
			Provider:    "stripe",
			EventType:   "payment_intent.succeeded",
			EventID:     fmt.Sprintf("evt_%d", i),
			Payload:     models.JSON{"i": i},
			Status:      models.WebhookEventStatusPending,
			MaxAttempts: 5,
		}
		if err := store.Create(ctx, ev); err != nil {
			t.Fatalf("seed event %d: %v", i, err)
		}
	}
}

func TestClaimNoDoubleProcessing(t *testing.T) {
	db := newTestDB(t)
	store := stores.CreateWebhookStore(db)

	const total = 200
	seedPending(t, store, total)

	var (
		mu        sync.Mutex
		claimedBy = make(map[string]int)
		wg        sync.WaitGroup
	)
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				events, err := store.ClaimPendingEvents(context.Background(), 5, time.Minute)
				if err != nil {
					t.Errorf("claim: %v", err)
					return
				}
				if len(events) == 0 {
					return
				}
				mu.Lock()
				for _, e := range events {
					claimedBy[e.ID]++
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if len(claimedBy) != total {
		t.Fatalf("expected %d distinct events claimed, got %d", total, len(claimedBy))
	}
	for id, c := range claimedBy {
		if c != 1 {
			t.Fatalf("event %s was claimed %d times (double processing)", id, c)
		}
	}
}

func TestRetryNotClaimedUntilDue(t *testing.T) {
	db := newTestDB(t)
	store := stores.CreateWebhookStore(db)
	ctx := context.Background()
	seedPending(t, store, 1)

	claimed, err := store.ClaimPendingEvents(ctx, 10, time.Minute)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("initial claim: err=%v n=%d", err, len(claimed))
	}
	if err := store.MarkFailed(ctx, claimed[0].ID, "boom", true); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	again, err := store.ClaimPendingEvents(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("event scheduled for future retry should not be claimed yet, got %d", len(again))
	}
}

func TestStaleProcessingReclaimed(t *testing.T) {
	db := newTestDB(t)
	store := stores.CreateWebhookStore(db)
	ctx := context.Background()
	seedPending(t, store, 1)

	claimed, err := store.ClaimPendingEvents(ctx, 10, time.Minute)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: err=%v n=%d", err, len(claimed))
	}
	id := claimed[0].ID

	if again, _ := store.ClaimPendingEvents(ctx, 10, time.Hour); len(again) != 0 {
		t.Fatalf("in-flight event should not be reclaimed, got %d", len(again))
	}

	if err := db.Model(&models.WebhookEvent{}).Where("id = ?", id).
		Update("last_attempt_at", time.Now().Add(-2*time.Hour)).Error; err != nil {
		t.Fatalf("age event: %v", err)
	}

	reclaimed, err := store.ClaimPendingEvents(ctx, 10, time.Hour)
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if len(reclaimed) != 1 || reclaimed[0].ID != id {
		t.Fatalf("expected stale event %s reclaimed, got %d", id, len(reclaimed))
	}
	if reclaimed[0].Attempts != 2 {
		t.Fatalf("expected attempts=2 after reclaim, got %d", reclaimed[0].Attempts)
	}
}

func TestMaxAttemptsExhaustedNotClaimed(t *testing.T) {
	db := newTestDB(t)
	store := stores.CreateWebhookStore(db)
	ctx := context.Background()

	past := time.Now().Add(-time.Hour)
	ev := &models.WebhookEvent{
		Provider:      "stripe",
		EventType:     "payment_intent.succeeded",
		EventID:       "evt_exhausted",
		Payload:       models.JSON{},
		Status:        models.WebhookEventStatusRetrying,
		Attempts:      5,
		MaxAttempts:   5,
		NextAttemptAt: &past,
	}
	if err := store.Create(ctx, ev); err != nil {
		t.Fatalf("create: %v", err)
	}

	claimed, err := store.ClaimPendingEvents(ctx, 10, time.Minute)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) != 0 {
		t.Fatalf("exhausted event must not be claimed, got %d", len(claimed))
	}
}

type markCompletedProcessor struct {
	store *stores.WebhookStore
	mu    sync.Mutex
	seen  map[string]int
}

func (p *markCompletedProcessor) ProcessClaimedEvent(ctx context.Context, e *models.WebhookEvent) error {
	p.mu.Lock()
	p.seen[e.ID]++
	p.mu.Unlock()
	return p.store.MarkCompleted(ctx, e.ID)
}

func (p *markCompletedProcessor) distinct() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.seen)
}

func TestWorkerPoolDrainsOutbox(t *testing.T) {
	db := newTestDB(t)
	store := stores.CreateWebhookStore(db)

	const total = 60
	seedPending(t, store, total)

	proc := &markCompletedProcessor{store: store, seen: make(map[string]int)}
	pool := worker.NewWebhookPool(store, proc, worker.Config{
		Workers:      6,
		BatchSize:    10,
		PollInterval: 20 * time.Millisecond,
	})
	pool.Start(context.Background())

	deadline := time.After(30 * time.Second)
	for proc.distinct() < total {
		select {
		case <-deadline:
			pool.Stop()
			t.Fatalf("timed out: %d/%d processed", proc.distinct(), total)
		case <-time.After(50 * time.Millisecond):
		}
	}
	pool.Stop()

	var completed int64
	if err := db.Model(&models.WebhookEvent{}).
		Where("status = ?", models.WebhookEventStatusCompleted).
		Count(&completed).Error; err != nil {
		t.Fatalf("count completed: %v", err)
	}
	if completed != total {
		t.Fatalf("expected %d completed events, got %d", total, completed)
	}
}
