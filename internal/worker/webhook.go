package worker

import (
	"context"
	"sync"
	"time"

	"github.com/malwarebo/conductor/models"
)

type EventClaimer interface {
	ClaimPendingEvents(ctx context.Context, limit int, staleAfter time.Duration) ([]*models.WebhookEvent, error)
}

type EventProcessor interface {
	ProcessClaimedEvent(ctx context.Context, event *models.WebhookEvent) error
}

type Config struct {
	Workers        int
	BatchSize      int
	PollInterval   time.Duration
	StaleAfter     time.Duration
	ProcessTimeout time.Duration
}

func DefaultConfig() Config {
	return Config{
		Workers:        4,
		BatchSize:      20,
		PollInterval:   2 * time.Second,
		StaleAfter:     5 * time.Minute,
		ProcessTimeout: 30 * time.Second,
	}
}

func (c Config) withDefaults() Config {
	d := DefaultConfig()
	if c.Workers <= 0 {
		c.Workers = d.Workers
	}
	if c.BatchSize <= 0 {
		c.BatchSize = d.BatchSize
	}
	if c.PollInterval <= 0 {
		c.PollInterval = d.PollInterval
	}
	if c.StaleAfter <= 0 {
		c.StaleAfter = d.StaleAfter
	}
	if c.ProcessTimeout <= 0 {
		c.ProcessTimeout = d.ProcessTimeout
	}
	return c
}

type WebhookPool struct {
	claimer   EventClaimer
	processor EventProcessor
	cfg       Config

	OnError func(error)

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewWebhookPool(claimer EventClaimer, processor EventProcessor, cfg Config) *WebhookPool {
	return &WebhookPool{
		claimer:   claimer,
		processor: processor,
		cfg:       cfg.withDefaults(),
	}
}

func (p *WebhookPool) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	p.cancel = cancel

	events := make(chan *models.WebhookEvent)

	for i := 0; i < p.cfg.Workers; i++ {
		p.wg.Add(1)
		go p.worker(events)
	}

	p.wg.Add(1)
	go p.dispatch(ctx, events)
}

func (p *WebhookPool) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}

func (p *WebhookPool) dispatch(ctx context.Context, events chan<- *models.WebhookEvent) {
	defer p.wg.Done()
	defer close(events)

	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()

	for {
		if ctx.Err() != nil {
			return
		}

		claimed, err := p.claimer.ClaimPendingEvents(ctx, p.cfg.BatchSize, p.cfg.StaleAfter)
		if err != nil {
			p.reportError(err)
		}

		for _, ev := range claimed {
			select {
			case events <- ev:
			case <-ctx.Done():
				return
			}
		}

		if err == nil && len(claimed) >= p.cfg.BatchSize {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *WebhookPool) worker(events <-chan *models.WebhookEvent) {
	defer p.wg.Done()
	for ev := range events {
		procCtx, cancel := context.WithTimeout(context.Background(), p.cfg.ProcessTimeout)
		if err := p.processor.ProcessClaimedEvent(procCtx, ev); err != nil {
			p.reportError(err)
		}
		cancel()
	}
}

func (p *WebhookPool) reportError(err error) {
	if p.OnError != nil {
		p.OnError(err)
	}
}
