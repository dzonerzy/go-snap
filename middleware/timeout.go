package middleware

import (
	"context"
	"errors"
	"time"
)

// Timeout creates a middleware that enforces a timeout on command execution
func Timeout(duration time.Duration) Middleware {
	return func(next ActionFunc) ActionFunc {
		return func(ctx Context) error {
			// Create a context with timeout derived from the current context when possible
			parent := context.Background()
			if c, ok := any(ctx).(interface{ Context() context.Context }); ok {
				parent = c.Context()
			}
			timeoutCtx, cancel := context.WithTimeout(parent, duration)
			defer cancel()

			// Channel to capture the result of the action
			resultChan := make(chan error, 1)

			// Run the action in a goroutine
			go func() {
				defer func() {
					// Recover from panic and send it as an error
					if r := recover(); r != nil {
						resultChan <- &RecoveryError{
							Panic:   r,
							Command: getCommandName(ctx),
						}
					}
				}()
				resultChan <- next(ctx)
			}()

			// Wait for either completion or timeout
			select {
			case err := <-resultChan:
				return err
			case <-timeoutCtx.Done():
				// Cancel the original context to signal timeout
				ctx.Cancel()
				return &TimeoutError{
					Duration: duration,
					Command:  getCommandName(ctx),
				}
			case <-ctx.Done():
				// Context was cancelled externally
				return context.Canceled
			}
		}
	}
}

// TimeoutWithDefault creates a timeout middleware with the default timeout from config
func TimeoutWithDefault(options ...MiddlewareOption) Middleware {
	config := DefaultConfig()
	for _, option := range options {
		option(config)
	}
	return Timeout(config.DefaultTimeout)
}

// TimeoutWithGracefulShutdown creates a timeout middleware that attempts graceful shutdown
func TimeoutWithGracefulShutdown(timeout, gracePeriod time.Duration) Middleware {
	return func(next ActionFunc) ActionFunc {
		return func(ctx Context) error {
			// Create contexts for timeout and grace period derived from parent
			parent := context.Background()
			if c, ok := any(ctx).(interface{ Context() context.Context }); ok {
				parent = c.Context()
			}
			timeoutCtx, timeoutCancel := context.WithTimeout(parent, timeout)
			graceCtx, graceCancel := context.WithTimeout(parent, timeout+gracePeriod)
			defer timeoutCancel()
			defer graceCancel()

			resultChan := make(chan error, 1)
			done := make(chan struct{})

			// Run the action in a goroutine
			go func() {
				defer close(done)
				defer func() {
					if r := recover(); r != nil {
						resultChan <- &RecoveryError{
							Panic:   r,
							Command: getCommandName(ctx),
						}
					}
				}()
				resultChan <- next(ctx)
			}()

			select {
			case err := <-resultChan:
				return err
			case <-timeoutCtx.Done():
				// Signal cancellation to the context
				ctx.Cancel()

				// Wait for graceful shutdown or force termination
				select {
				case err := <-resultChan:
					return err
				case <-done:
					// Action completed during grace period
					select {
					case err := <-resultChan:
						return err
					default:
						return &TimeoutError{
							Duration: timeout,
							Command:  getCommandName(ctx),
						}
					}
				case <-graceCtx.Done():
					// Grace period expired
					return &TimeoutError{
						Duration: timeout + gracePeriod,
						Command:  getCommandName(ctx),
					}
				}
			case <-ctx.Done():
				// Context was cancelled externally
				return context.Canceled
			}
		}
	}
}

// TimeoutPerCommand creates a timeout middleware with different timeouts per
// command. If a command name is not present in commandTimeouts, defaultTimeout
// is used.
func TimeoutPerCommand(commandTimeouts map[string]time.Duration, defaultTimeout time.Duration) Middleware {
	return func(next ActionFunc) ActionFunc {
		return func(ctx Context) error {
			command := getCommandName(ctx)

			// Get timeout for this specific command
			timeout, exists := commandTimeouts[command]
			if !exists {
				timeout = defaultTimeout
			}

			return Timeout(timeout)(next)(ctx)
		}
	}
}

// TimeoutWithCallback creates a timeout middleware that calls onTimeout when the
// command exceeds duration. The callback runs after the timeout is reached.
func TimeoutWithCallback(duration time.Duration, onTimeout func(command string, duration time.Duration)) Middleware {
	return func(next ActionFunc) ActionFunc {
		return func(ctx Context) error {
			parent := context.Background()
			if c, ok := any(ctx).(interface{ Context() context.Context }); ok {
				parent = c.Context()
			}
			timeoutCtx, cancel := context.WithTimeout(parent, duration)
			defer cancel()

			resultChan := make(chan error, 1)

			go func() {
				defer func() {
					if r := recover(); r != nil {
						resultChan <- &RecoveryError{
							Panic:   r,
							Command: getCommandName(ctx),
						}
					}
				}()
				resultChan <- next(ctx)
			}()

			select {
			case err := <-resultChan:
				return err
			case <-timeoutCtx.Done():
				command := getCommandName(ctx)

				// Call the timeout callback
				if onTimeout != nil {
					onTimeout(command, duration)
				}

				ctx.Cancel()
				return &TimeoutError{
					Duration: duration,
					Command:  command,
				}
			case <-ctx.Done():
				return context.Canceled
			}
		}
	}
}

// TimeoutWithRetry creates a timeout middleware that retries on timeout
func TimeoutWithRetry(duration time.Duration, maxRetries int) Middleware {
	return func(next ActionFunc) ActionFunc {
		return func(ctx Context) error {
			var lastErr error

			for attempt := 0; attempt <= maxRetries; attempt++ {
				// Apply a per-attempt timeout
				err := Timeout(duration)(next)(ctx)

				// Success: return immediately
				if err == nil {
					return nil
				}

				// If it is a timeout, retry up to maxRetries
				var tErr *TimeoutError
				if errors.As(err, &tErr) {
					lastErr = err
					if attempt < maxRetries {
						continue
					}
					// Out of retries: return the timeout error
					return err
				}

				// Non-timeout error: do not retry
				return err
			}

			// Should not reach here normally; return the last seen error
			return lastErr
		}
	}
}

// NoTimeout creates a middleware that doesn't enforce any timeout
// (useful for development or long-running operations)
func NoTimeout() Middleware {
	return func(next ActionFunc) ActionFunc {
		return next // No timeout enforcement
	}
}

// DynamicTimeout creates a timeout middleware where duration is computed at
// runtime from the Context. If the computed duration is <= 0, the action runs
// without a timeout.
func DynamicTimeout(timeoutFunc func(ctx Context) time.Duration) Middleware {
	return func(next ActionFunc) ActionFunc {
		return func(ctx Context) error {
			duration := timeoutFunc(ctx)
			if duration <= 0 {
				// No timeout if duration is zero or negative
				return next(ctx)
			}
			return Timeout(duration)(next)(ctx)
		}
	}
}

// TimeoutFromFlag creates a timeout middleware that reads duration from a flag.
// It checks the command-local flag first and then the global flag of the same
// name. If neither is set, defaultTimeout is used.
func TimeoutFromFlag(flagName string, defaultTimeout time.Duration) Middleware {
	return DynamicTimeout(func(ctx Context) time.Duration {
		if duration, exists := ctx.Duration(flagName); exists {
			return duration
		}
		if duration, exists := ctx.GlobalDuration(flagName); exists {
			return duration
		}
		return defaultTimeout
	})
}

// TimeoutStats tracks timeout statistics
type TimeoutStats struct {
	TotalTimeouts   int
	CommandTimeouts map[string]int
	TotalDuration   time.Duration
	LastTimeout     *TimeoutError
}

// NewTimeoutStats creates a new timeout statistics tracker
func NewTimeoutStats() *TimeoutStats {
	return &TimeoutStats{
		CommandTimeouts: make(map[string]int),
	}
}

// TimeoutWithStats creates a timeout middleware that tracks statistics
func TimeoutWithStats(duration time.Duration, stats *TimeoutStats) Middleware {
	return func(next ActionFunc) ActionFunc {
		return func(ctx Context) error {
			start := time.Now()

			parent := context.Background()
			if c, ok := any(ctx).(interface{ Context() context.Context }); ok {
				parent = c.Context()
			}
			timeoutCtx, cancel := context.WithTimeout(parent, duration)
			defer cancel()

			resultChan := make(chan error, 1)

			go func() {
				defer func() {
					if r := recover(); r != nil {
						resultChan <- &RecoveryError{
							Panic:   r,
							Command: getCommandName(ctx),
						}
					}
				}()
				resultChan <- next(ctx)
			}()

			select {
			case err := <-resultChan:
				return err
			case <-timeoutCtx.Done():
				command := getCommandName(ctx)

				// Update statistics
				stats.TotalTimeouts++
				stats.CommandTimeouts[command]++
				stats.TotalDuration += time.Since(start)
				stats.LastTimeout = &TimeoutError{
					Duration: duration,
					Command:  command,
				}

				ctx.Cancel()
				return stats.LastTimeout
			case <-ctx.Done():
				return context.Canceled
			}
		}
	}
}
