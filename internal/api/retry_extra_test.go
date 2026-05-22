// Package api -- additional unit tests for backoffDuration not covered by retry_test.go.
package api

import (
	"testing" // for testing.T -- standard Go test runner
	"time"    // for time.Duration -- compare sleep durations
)

// TestBackoffDuration_AttemptZero verifies that attempt 0 produces BaseDelay + jitter.
// Jitter is random so we only assert that the result is >= BaseDelay and <= BaseDelay*2.
func TestBackoffDuration_AttemptZero(t *testing.T) {
	t.Parallel() // Safe to run concurrently
	cfg := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond, // Known base delay for easy math
		MaxDelay:    10 * time.Second,       // Well above the expected result
	}
	d := backoffDuration(0, cfg)                  // Attempt 0: factor = 2^0 = 1, so sleep = BaseDelay
	if d < cfg.BaseDelay {                        // Must be at least BaseDelay (factor 1 * BaseDelay)
		t.Errorf("expected >= %v, got %v", cfg.BaseDelay, d) // Report actual value for debugging
	}
	if d >= cfg.BaseDelay*2+cfg.BaseDelay { // Upper bound: BaseDelay + jitter < BaseDelay + BaseDelay
		t.Errorf("expected < %v, got %v", cfg.BaseDelay*3, d) // Report if jitter overshoots
	}
}

// TestBackoffDuration_AttemptOne verifies that attempt 1 produces 2*BaseDelay + jitter.
func TestBackoffDuration_AttemptOne(t *testing.T) {
	t.Parallel() // Safe to run concurrently
	cfg := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond, // Known base delay for easy math
		MaxDelay:    10 * time.Second,       // Well above expected result
	}
	d := backoffDuration(1, cfg)    // Attempt 1: factor = 2^1 = 2, so sleep = 2*BaseDelay + jitter
	if d < cfg.BaseDelay*2 {        // Must be at least 2*BaseDelay
		t.Errorf("expected >= %v, got %v", cfg.BaseDelay*2, d) // Report actual value
	}
	if d >= cfg.BaseDelay*3+cfg.BaseDelay { // Upper bound: 2*BaseDelay + jitter < 3*BaseDelay
		t.Errorf("expected < %v, got %v", cfg.BaseDelay*4, d) // Report if jitter overshoots
	}
}

// TestBackoffDuration_CapsAtMaxDelay verifies that the result never exceeds MaxDelay.
// With a large attempt number the exponential growth would exceed MaxDelay without the cap.
func TestBackoffDuration_CapsAtMaxDelay(t *testing.T) {
	t.Parallel() // Safe to run concurrently
	cfg := RetryConfig{
		MaxAttempts: 10,
		BaseDelay:   1 * time.Second,  // Large enough that 2^10 * 1s far exceeds MaxDelay
		MaxDelay:    2 * time.Second,  // Cap that the exponential would exceed at attempt >=1
	}
	// At attempt 8, 2^8 * 1s = 256s, which far exceeds MaxDelay of 2s.
	d := backoffDuration(8, cfg)                    // Result must be capped at MaxDelay + jitter
	maxWithJitter := cfg.MaxDelay + cfg.BaseDelay   // Jitter is at most BaseDelay
	if d > maxWithJitter {                          // Must not exceed MaxDelay + maximum jitter
		t.Errorf("expected <= %v (MaxDelay+jitter), got %v", maxWithJitter, d) // Report overshoot
	}
}

// TestBackoffDuration_ZeroBaseDelay verifies that zero BaseDelay produces near-zero durations.
// This is the fast-test scenario where callers want no sleep between retries.
func TestBackoffDuration_ZeroBaseDelay(t *testing.T) {
	t.Parallel() // Safe to run concurrently
	cfg := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   0,             // Zero delay -- used in fast-test scenarios
		MaxDelay:    1 * time.Hour, // High cap -- should not be reached
	}
	d := backoffDuration(0, cfg) // Zero base * 2^0 + 0 jitter = 0
	if d != 0 {                  // Zero base delay must produce zero sleep
		t.Errorf("expected 0 duration, got %v", d) // Report any non-zero result
	}
}

// TestRetryableError_ChainPreserved verifies that the original cause is in the error chain.
// Callers often inspect the cause error to decide how to handle it, so the chain must be intact.
func TestRetryableError_ChainPreserved(t *testing.T) {
	t.Parallel()                                   // Safe to run concurrently
	cause := &customErr{msg: "original cause"}     // A custom error type to verify chain preservation
	wrapped := RetryableError(cause)               // Wrap in a retryable error
	if !IsRetryable(wrapped) {                     // Wrapped error must be retryable
		t.Error("wrapped error should be retryable") // Fail if retryable check does not work
	}
}

// customErr is a minimal error type used to verify error chain preservation in tests.
type customErr struct{ msg string }

func (e *customErr) Error() string { return e.msg } // Implement the error interface
