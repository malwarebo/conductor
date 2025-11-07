package utils

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry_Success(t *testing.T) {
	config := CreateDefaultRetryConfig()
	ctx := context.Background()
	attempts := 0

	err := CreateRetry(ctx, config, func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("CreateRetry() error = %v, want nil", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestRetry_EventualSuccess(t *testing.T) {
	config := CreateDefaultRetryConfig()
	ctx := context.Background()
	attempts := 0

	err := CreateRetry(ctx, config, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("CreateRetry() error = %v, want nil", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetry_MaxAttempts(t *testing.T) {
	config := CreateDefaultRetryConfig()
	config.MaxAttempts = 3
	ctx := context.Background()
	attempts := 0

	err := CreateRetry(ctx, config, func() error {
		attempts++
		return errors.New("persistent error")
	})

	if err == nil {
		t.Error("CreateRetry() expected error")
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetry_WithResult(t *testing.T) {
	config := CreateDefaultRetryConfig()
	ctx := context.Background()
	attempts := 0

	result, err := CreateRetryWithResult(ctx, config, func() (interface{}, error) {
		attempts++
		if attempts < 2 {
			return nil, errors.New("temporary error")
		}
		return "success", nil
	})

	if err != nil {
		t.Errorf("CreateRetryWithResult() error = %v, want nil", err)
	}
	if result != "success" {
		t.Errorf("result = %v, want success", result)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	config := CreateDefaultRetryConfig()
	config.MaxAttempts = 10
	config.BaseDelay = 100 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	attempts := 0
	start := time.Now()

	err := CreateRetry(ctx, config, func() error {
		attempts++
		return errors.New("error")
	})

	duration := time.Since(start)
	if err == nil {
		t.Error("CreateRetry() expected error")
	}
	if duration >= 200*time.Millisecond {
		t.Errorf("duration = %v, want < 200ms", duration)
	}
}

func TestRetry_ExponentialBackoff(t *testing.T) {
	config := CreateDefaultRetryConfig()
	config.MaxAttempts = 3
	config.BaseDelay = 10 * time.Millisecond
	config.Jitter = false
	config.BackoffType = Exponential

	ctx := context.Background()
	start := time.Now()
	attempts := 0

	CreateRetry(ctx, config, func() error {
		attempts++
		return errors.New("error")
	})

	duration := time.Since(start)
	expectedMin := 10*time.Millisecond + 20*time.Millisecond
	if duration < expectedMin {
		t.Errorf("duration = %v, want >= %v", duration, expectedMin)
	}
}

func TestRetry_RetryableError(t *testing.T) {
	config := CreateDefaultRetryConfig()
	config.MaxAttempts = 3
	ctx := context.Background()
	attempts := 0

	retryableErr := CreateRetryableError(errors.New("temporary failure"))

	err := CreateRetry(ctx, config, func() error {
		attempts++
		if attempts < 2 {
			return retryableErr
		}
		return nil
	})

	if err != nil {
		t.Errorf("CreateRetry() error = %v, want nil", err)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}
