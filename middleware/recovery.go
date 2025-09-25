package middleware

import (
	"fmt"
	"os"
	"runtime"
)

// Recovery creates a middleware that recovers from panics during command execution
func Recovery(options ...MiddlewareOption) Middleware {
	config := DefaultConfig()
	for _, option := range options {
		option(config)
	}

	return func(next ActionFunc) ActionFunc {
		return func(ctx Context) (err error) {
			// Set up panic recovery
			defer func() {
				if r := recover(); r != nil {
					// Capture stack trace if enabled
					var stack []byte
					if config.PrintStack {
						stack = make([]byte, config.StackSize)
						length := runtime.Stack(stack, false)
						stack = stack[:length]
					}

					// Create recovery error
					recoveryErr := &RecoveryError{
						Panic:   r,
						Command: getCommandName(ctx),
						Stack:   stack,
					}

					// Print stack trace to stderr if enabled
					if config.PrintStack && len(stack) > 0 {
						fmt.Fprintf(os.Stderr, "PANIC in command '%s': %v\n", recoveryErr.Command, r)
						fmt.Fprintf(os.Stderr, "Stack trace:\n%s\n", stack)
					}

					// Set the error to be returned
					err = recoveryErr
				}
			}()

			// Execute the action
			return next(ctx)
		}
	}
}

// RecoveryWithHandler creates a recovery middleware with a custom panic handler
func RecoveryWithHandler(
	handler func(panicVal any, command string, stack []byte) error,
	options ...MiddlewareOption,
) Middleware {
	config := DefaultConfig()
	for _, option := range options {
		option(config)
	}

	return func(next ActionFunc) ActionFunc {
		return func(ctx Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					// Capture stack trace if enabled
					var stack []byte
					if config.PrintStack {
						stack = make([]byte, config.StackSize)
						length := runtime.Stack(stack, false)
						stack = stack[:length]
					}

					// Call custom handler
					err = handler(r, getCommandName(ctx), stack)
				}
			}()

			return next(ctx)
		}
	}
}

// RecoveryToError creates a recovery middleware that converts panics to regular errors
// without printing stack traces (useful for production)
func RecoveryToError() Middleware {
	return Recovery(WithStackTrace(false))
}

// RecoveryWithStack creates a recovery middleware that always prints stack traces
// (useful for development)
func RecoveryWithStack() Middleware {
	return Recovery(WithStackTrace(true))
}

// NoopRecovery creates a recovery middleware that doesn't actually recover
// (useful for development when you want panics to crash the program)
func NoopRecovery() Middleware {
	return func(next ActionFunc) ActionFunc {
		return next // No recovery, let panics bubble up
	}
}

// MustRecover creates a recovery middleware that converts panics to errors
// and logs them appropriately based on the environment
func MustRecover() Middleware {
	// Check if we're in development mode (basic heuristic)
	isDev := os.Getenv("GO_ENV") == "development" ||
		os.Getenv("ENV") == "dev" ||
		os.Getenv("ENVIRONMENT") == "development"

	if isDev {
		return RecoveryWithStack()
	}
	return RecoveryToError()
}

// SafeRecovery creates a recovery middleware with safe defaults
// - Captures stack traces but doesn't print them to stderr
// - Returns structured error information
// - Suitable for production use
func SafeRecovery() Middleware {
	return func(next ActionFunc) ActionFunc {
		return func(ctx Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					// Always capture stack for debugging, but don't print
					stack := make([]byte, 4096)
					length := runtime.Stack(stack, false)
					stack = stack[:length]

					// Create structured error
					err = &RecoveryError{
						Panic:   r,
						Command: getCommandName(ctx),
						Stack:   stack,
					}

					// Store stack in context metadata for potential logging
					ctx.Set("panic_stack", string(stack))
					ctx.Set("panic_value", r)
				}
			}()

			return next(ctx)
		}
	}
}

// RecoveryStats tracks recovery statistics
type RecoveryStats struct {
	TotalPanics   int
	CommandPanics map[string]int
	LastPanic     *RecoveryError
}

// NewRecoveryStats creates a new recovery statistics tracker
func NewRecoveryStats() *RecoveryStats {
	return &RecoveryStats{
		CommandPanics: make(map[string]int),
	}
}

// RecoveryWithStats creates a recovery middleware that tracks statistics
func RecoveryWithStats(stats *RecoveryStats, options ...MiddlewareOption) Middleware {
	config := DefaultConfig()
	for _, option := range options {
		option(config)
	}

	return func(next ActionFunc) ActionFunc {
		return func(ctx Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					command := getCommandName(ctx)

					// Capture stack trace
					var stack []byte
					if config.PrintStack {
						stack = make([]byte, config.StackSize)
						length := runtime.Stack(stack, false)
						stack = stack[:length]
					}

					// Update statistics
					stats.TotalPanics++
					stats.CommandPanics[command]++
					stats.LastPanic = &RecoveryError{
						Panic:   r,
						Command: command,
						Stack:   stack,
					}

					// Print stack if enabled
					if config.PrintStack && len(stack) > 0 {
						fmt.Fprintf(os.Stderr, "PANIC in command '%s': %v\n", command, r)
						fmt.Fprintf(os.Stderr, "Stack trace:\n%s\n", stack)
					}

					err = stats.LastPanic
				}
			}()

			return next(ctx)
		}
	}
}
