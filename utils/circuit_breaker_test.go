package utils

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_Success(t *testing.T) {
	cb := CreateCircuitBreaker(3, 100*time.Millisecond)
	ctx := context.Background()

	err := cb.Execute(ctx, func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
	if cb.GetState() != StateClosed {
		t.Errorf("GetState() = %v, want StateClosed", cb.GetState())
	}
}

func TestCircuitBreaker_OpenAfterFailures(t *testing.T) {
	cb := CreateCircuitBreaker(3, 100*time.Millisecond)
	ctx := context.Background()

	testError := errors.New("test error")

	for i := 0; i < 3; i++ {
		err := cb.Execute(ctx, func() error {
			return testError
		})
		if err == nil {
			t.Error("Execute() expected error")
		}
	}

	if cb.GetState() != StateOpen {
		t.Errorf("GetState() = %v, want StateOpen", cb.GetState())
	}

	err := cb.Execute(ctx, func() error {
		return nil
	})
	if err == nil {
		t.Error("Execute() expected error when circuit is open")
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cb := CreateCircuitBreaker(2, 50*time.Millisecond)
	ctx := context.Background()

	testError := errors.New("test error")
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func() error {
			return testError
		})
	}

	if cb.GetState() != StateOpen {
		t.Errorf("GetState() = %v, want StateOpen", cb.GetState())
	}

	time.Sleep(60 * time.Millisecond)

	err := cb.Execute(ctx, func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
	if cb.GetState() != StateClosed {
		t.Errorf("GetState() = %v, want StateClosed", cb.GetState())
	}
}

func TestCircuitBreaker_Recovery(t *testing.T) {
	cb := CreateCircuitBreaker(2, 50*time.Millisecond)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func() error {
			return errors.New("fail")
		})
	}

	if cb.GetState() != StateOpen {
		t.Errorf("GetState() = %v, want StateOpen", cb.GetState())
	}

	time.Sleep(60 * time.Millisecond)

	err := cb.Execute(ctx, func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
	if cb.GetState() != StateClosed {
		t.Errorf("GetState() = %v, want StateClosed", cb.GetState())
	}
}

func TestCircuitBreaker_ContextCancellation(t *testing.T) {
	cb := CreateCircuitBreaker(5, 100*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := cb.Execute(ctx, func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	})

	if err == nil {
		t.Error("Execute() expected context cancellation error")
	}
}
