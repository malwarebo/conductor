package providers

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrFuseOpen    = errors.New("fuse is open")
	ErrFuseTimeout = errors.New("fuse timeout")
)

type FuseState int

const (
	FuseClosed FuseState = iota
	FuseOpen
	FuseHalfOpen
)

type Fuse struct {
	name          string
	maxFailures   int
	timeout       time.Duration
	halfOpenMax   int
	state         FuseState
	failures      int
	successes     int
	lastFailure   time.Time
	mu            sync.RWMutex
	onStateChange func(name string, from, to FuseState)
}

type FuseConfig struct {
	Name          string
	MaxFailures   int
	Timeout       time.Duration
	HalfOpenMax   int
	OnStateChange func(name string, from, to FuseState)
}

func CreateFuse(cfg FuseConfig) *Fuse {
	if cfg.MaxFailures <= 0 {
		cfg.MaxFailures = 5
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.HalfOpenMax <= 0 {
		cfg.HalfOpenMax = 3
	}

	return &Fuse{
		name:          cfg.Name,
		maxFailures:   cfg.MaxFailures,
		timeout:       cfg.Timeout,
		halfOpenMax:   cfg.HalfOpenMax,
		state:         FuseClosed,
		onStateChange: cfg.OnStateChange,
	}
}

func (f *Fuse) Execute(ctx context.Context, fn func() error) error {
	if !f.allowRequest() {
		return ErrFuseOpen
	}

	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	select {
	case err := <-done:
		f.recordResult(err)
		return err
	case <-ctx.Done():
		f.recordResult(ctx.Err())
		return ErrFuseTimeout
	}
}

func (f *Fuse) allowRequest() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch f.state {
	case FuseClosed:
		return true
	case FuseOpen:
		if time.Since(f.lastFailure) > f.timeout {
			f.transitionTo(FuseHalfOpen)
			return true
		}
		return false
	case FuseHalfOpen:
		return f.successes < f.halfOpenMax
	}
	return false
}

func (f *Fuse) recordResult(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err != nil {
		f.failures++
		f.lastFailure = time.Now()

		switch f.state {
		case FuseClosed:
			if f.failures >= f.maxFailures {
				f.transitionTo(FuseOpen)
			}
		case FuseHalfOpen:
			f.transitionTo(FuseOpen)
		}
	} else {
		switch f.state {
		case FuseClosed:
			f.failures = 0
		case FuseHalfOpen:
			f.successes++
			if f.successes >= f.halfOpenMax {
				f.transitionTo(FuseClosed)
			}
		}
	}
}

func (f *Fuse) transitionTo(newState FuseState) {
	if f.state == newState {
		return
	}

	oldState := f.state
	f.state = newState
	f.failures = 0
	f.successes = 0

	if f.onStateChange != nil {
		go f.onStateChange(f.name, oldState, newState)
	}
}

func (f *Fuse) State() FuseState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state
}

func (f *Fuse) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state = FuseClosed
	f.failures = 0
	f.successes = 0
}

func (s FuseState) String() string {
	switch s {
	case FuseClosed:
		return "closed"
	case FuseOpen:
		return "open"
	case FuseHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

