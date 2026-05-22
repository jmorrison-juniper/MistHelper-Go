package api

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// TestWithRetry_SuccessFirstAttempt verifies that op is called once and returns nil on first success.
func TestWithRetry_SuccessFirstAttempt(t *testing.T) {
	calls := 0                                   // Count how many times op is invoked
	op := func() error { calls++; return nil }  // Op succeeds immediately
	err := withRetry(context.Background(), op, DefaultRetryConfig)
	if err != nil { // No error expected on immediate success
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 { // Must call op exactly once -- no spurious retries
		t.Errorf("calls = %d, want 1", calls)
	}
}

// TestWithRetry_SuccessAfterRetry verifies that op is retried after a retryable failure and eventual success is returned.
func TestWithRetry_SuccessAfterRetry(t *testing.T) {
	calls := 0 // Track invocation count
	op := func() error {
		calls++
		if calls < 2 { // Fail once then succeed
			return RetryableError(fmt.Errorf("transient error")) // Signal retryable failure
		}
		return nil // Succeed on second attempt
	}
	cfg := RetryConfig{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond, MaxDelay: 10 * time.Millisecond} // Fast delays for test
	err := withRetry(context.Background(), op, cfg)
	if err != nil { // Should succeed after one retry
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 { // Exactly 2 calls: 1 failure + 1 success
		t.Errorf("calls = %d, want 2", calls)
	}
}

// TestWithRetry_ExhaustsRetries verifies that all attempts are made and the final error is returned.
func TestWithRetry_ExhaustsRetries(t *testing.T) {
	calls := 0 // Track invocation count
	op := func() error {
		calls++ // Count each attempt
		return RetryableError(fmt.Errorf("always fails")) // Always retryable failure
	}
	cfg := RetryConfig{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond, MaxDelay: 10 * time.Millisecond}
	err := withRetry(context.Background(), op, cfg)
	if err == nil { // Must return an error after exhausting all attempts
		t.Fatal("expected error after exhausting retries, got nil")
	}
	if calls != 3 { // Exactly 3 attempts per MaxAttempts
		t.Errorf("calls = %d, want 3", calls)
	}
}

// TestWithRetry_NonRetryableError verifies that non-retryable errors short-circuit immediately.
func TestWithRetry_NonRetryableError(t *testing.T) {
	calls := 0 // Track invocation count
	op := func() error {
		calls++
		return errors.New("permanent error") // Non-retryable -- plain error, not wrapped with errRetryable
	}
	cfg := RetryConfig{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond, MaxDelay: 10 * time.Millisecond}
	err := withRetry(context.Background(), op, cfg)
	if err == nil { // Must return the permanent error
		t.Fatal("expected error for non-retryable failure, got nil")
	}
	if calls != 1 { // Must NOT retry -- only 1 call expected for permanent errors
		t.Errorf("calls = %d, want 1 (non-retryable should not retry)", calls)
	}
}

// TestWithRetry_ContextCancelled verifies that a cancelled context aborts the retry loop.
func TestWithRetry_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background()) // Create cancellable context
	cancel()                                                // Cancel immediately before first attempt

	op := func() error { return RetryableError(fmt.Errorf("transient")) } // Would retry if not cancelled
	err := withRetry(ctx, op, DefaultRetryConfig)
	if err == nil { // Must return an error due to cancellation
		t.Fatal("expected error for cancelled context, got nil")
	}
}

// TestIsRetryable verifies that errRetryable chain detection works correctly.
func TestIsRetryable(t *testing.T) {
	retryable := RetryableError(fmt.Errorf("transient")) // Wrapped retryable error
	plain := fmt.Errorf("permanent")                     // Plain non-retryable error

	if !IsRetryable(retryable) { // Wrapped error must be recognised as retryable
		t.Error("RetryableError should be retryable")
	}
	if IsRetryable(plain) { // Plain error must NOT be retryable
		t.Error("plain error should not be retryable")
	}
}
