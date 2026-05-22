package api

import (
	"context"   // for cancellation propagation through retry loops
	"errors"    // for sentinel error comparison
	"fmt"       // for wrapping errors with context
	"log/slog"  // for structured info/debug logging on retry events
	"math/rand" // for jitter calculation to spread retry load
	"time"      // for sleep durations and backoff calculation
)

// RetryConfig holds parameters that govern exponential back-off retry behaviour.
// Callers pass this by value so each operation can tune retries independently.
type RetryConfig struct {
	MaxAttempts int           // Total number of attempts including the first (minimum 1)
	BaseDelay   time.Duration // Starting sleep duration before first retry
	MaxDelay    time.Duration // Upper cap on sleep duration (prevents runaway backoff)
}

// DefaultRetryConfig is the standard retry policy used for all Mist API calls.
// 3 attempts with 1s base delay capped at 30s matches the Python reference implementation.
var DefaultRetryConfig = RetryConfig{
	MaxAttempts: 3,              // Attempt the call up to 3 times total
	BaseDelay:   1 * time.Second, // Start with a 1-second pause before retry 1
	MaxDelay:    30 * time.Second, // Never wait longer than 30 seconds between attempts
}

// errRetryable is a sentinel that op functions should wrap to signal retryable failure.
// Non-wrapped errors are treated as permanent and short-circuit the retry loop.
var errRetryable = errors.New("retryable error")

// IsRetryable returns true if err signals that the operation should be retried.
// Callers should wrap their errors with fmt.Errorf("msg: %w", errRetryable) to opt in.
func IsRetryable(err error) bool {
	return errors.Is(err, errRetryable) // Check the error chain for the sentinel
}

// RetryableError wraps a cause error so the retry loop will attempt again.
// Use this when an API call returns HTTP 429 or 5xx to signal a transient failure.
func RetryableError(cause error) error {
	return fmt.Errorf("%w: %w", errRetryable, cause) // Wrap both sentinel and cause for chain inspection
}

// withRetry executes op up to cfg.MaxAttempts times with exponential back-off and jitter.
// It stops early on context cancellation or when op returns a non-retryable error.
// Retry sleep durations are: BaseDelay * 2^attempt + jitter (capped at MaxDelay).
func withRetry(ctx context.Context, op func() error, cfg RetryConfig) error {
	var lastErr error // Track the most recent error for the final return
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ { // Iterate up to MaxAttempts times
		if err := ctx.Err(); err != nil { // Honour context cancellation before each attempt
			return fmt.Errorf("retry cancelled: %w", err) // Propagate cancellation reason
		}

		lastErr = op() // Execute the operation and capture any error
		if lastErr == nil { // Success -- no retry needed
			return nil
		}

		if !IsRetryable(lastErr) { // Non-retryable error -- fail immediately without sleeping
			return lastErr
		}

		if attempt == cfg.MaxAttempts-1 { // Last attempt -- no point computing sleep duration
			break // Exit loop; lastErr will be returned below
		}

		sleep := backoffDuration(attempt, cfg) // Compute sleep with exponential backoff + jitter
		slog.Info("API call failed, retrying", // Log retry event at info level for operator visibility
			"attempt", attempt+1,          // 1-based attempt number for readability
			"max_attempts", cfg.MaxAttempts, // Total attempts so context is clear
			"sleep_ms", sleep.Milliseconds(), // Sleep duration to help diagnose throttling
		)

		select {
		case <-ctx.Done(): // Context cancelled while sleeping -- abort cleanly
			return fmt.Errorf("retry cancelled during sleep: %w", ctx.Err())
		case <-time.After(sleep): // Sleep the full backoff duration before next attempt
		}
	}
	return fmt.Errorf("all %d attempts failed: %w", cfg.MaxAttempts, lastErr) // All attempts exhausted
}

// backoffDuration computes the sleep duration for a given attempt index.
// Formula: min(BaseDelay * 2^attempt, MaxDelay) + random jitter up to BaseDelay.
// Jitter prevents thundering herd when many clients retry simultaneously.
func backoffDuration(attempt int, cfg RetryConfig) time.Duration {
	exp := time.Duration(1 << attempt) // 2^attempt multiplier (1, 2, 4, 8...)
	sleep := cfg.BaseDelay * exp       // Exponential growth from base delay
	if sleep > cfg.MaxDelay {          // Cap at MaxDelay to prevent very long waits
		sleep = cfg.MaxDelay
	}
	jitterNs := rand.Int63n(int64(cfg.BaseDelay)) // #nosec G404 -- math/rand jitter for retry backoff is not security-sensitive; it prevents thundering herd, not cryptographic operations
	return sleep + time.Duration(jitterNs)         // Add jitter to spread concurrent retries
}
