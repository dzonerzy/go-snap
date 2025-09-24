// Package middleware provides built-in middleware for go-snap CLI applications
// Focused on 4 essential middleware: Logger, Recovery, Timeout, and Validator
package middleware

import (
	"time"
)

// This package defines middleware using interfaces to avoid import cycles.
// The snap package will import this middleware package and its types will satisfy these interfaces.
// Users will use concrete types from snap package: snap.Context, snap.ActionFunc, etc.

// Context describes the runtime information and lifecycle controls that
// middleware can rely on. It is implemented by *snap.Context.
type Context interface {
    // Core methods needed by middleware

    // Done returns a channel that is closed when the command's context is
    // canceled or times out. Use this to abort long‑running work.
    Done() <-chan struct{}

    // Cancel requests cancellation of the current command's context. It is
    // idempotent and safe to call from middleware to stop downstream work.
    Cancel()

    // Args returns the positional (non‑flag) arguments for the current
    // command. The returned slice should be treated as read‑only.
    Args() []string

    // Set stores a key/value pair in the context metadata. This is useful for
    // passing information between middleware. Keys should be namespaced to
    // avoid collisions (e.g., "logger.request_id").
    Set(key string, value any)

    // Get retrieves a value previously stored via Set. It returns nil when no
    // value is present for the given key.
    Get(key string) any

    // Flag access methods that exist in snap.Context

    // String returns the string value of a flag and a boolean indicating
    // presence. Presence is true when the flag was set by CLI/env/defaults.
    String(name string) (string, bool)

    // Int returns the int value of a flag and whether it is present.
    Int(name string) (int, bool)

    // Bool returns the bool value of a flag and whether it is present. For
    // boolean flags, presence typically implies a value of true unless set via
    // explicit false.
    Bool(name string) (bool, bool)

    // Duration returns the time.Duration value of a flag and whether it is
    // present. Supports extended duration formats handled by the parser.
    Duration(name string) (time.Duration, bool)

    // Float returns the float64 value of a flag and whether it is present.
    Float(name string) (float64, bool)

    // Enum returns the selected enum value (string) and whether it is present.
    Enum(name string) (string, bool)

    // StringSlice returns the []string value of a slice flag and whether it is
    // present. The returned slice should be treated as read‑only.
    StringSlice(name string) ([]string, bool)

    // IntSlice returns the []int value of a slice flag and whether it is
    // present. The returned slice should be treated as read‑only.
    IntSlice(name string) ([]int, bool)

    // Global flag access

    // GlobalString returns the string value of a global flag and presence.
    GlobalString(name string) (string, bool)

    // GlobalInt returns the int value of a global flag and presence.
    GlobalInt(name string) (int, bool)

    // GlobalBool returns the bool value of a global flag and presence.
    GlobalBool(name string) (bool, bool)

    // GlobalDuration returns the time.Duration value of a global flag and
    // presence.
    GlobalDuration(name string) (time.Duration, bool)

    // GlobalFloat returns the float64 value of a global flag and presence.
    GlobalFloat(name string) (float64, bool)

    // GlobalEnum returns the enum string value of a global flag and presence.
    GlobalEnum(name string) (string, bool)

    // GlobalStringSlice returns the []string value of a global slice flag and
    // presence. The returned slice should be treated as read‑only.
    GlobalStringSlice(name string) ([]string, bool)

    // GlobalIntSlice returns the []int value of a global slice flag and
    // presence. The returned slice should be treated as read‑only.
    GlobalIntSlice(name string) ([]int, bool)

    // Command access

    // Command returns the current command descriptor (name/description). It
    // can be used by middleware for logging and error messages.
    Command() Command
}

// Command interface will be satisfied by *snap.Command
type Command interface {
	Name() string
	Description() string
}

// ActionFunc represents command action function signature
type ActionFunc func(ctx Context) error

// Middleware defines the middleware function signature
type Middleware func(next ActionFunc) ActionFunc

// MiddlewareChain represents a chain of middleware functions
type MiddlewareChain []Middleware

// Apply applies the middleware chain to an ActionFunc. Middleware are wrapped
// in the order they appear in the chain.
func (chain MiddlewareChain) Apply(action ActionFunc) ActionFunc {
	for i := len(chain) - 1; i >= 0; i-- {
		action = chain[i](action)
	}
	return action
}

// Use returns a new chain with the provided middleware appended.
func (chain MiddlewareChain) Use(middleware ...Middleware) MiddlewareChain {
    return append(chain, middleware...)
}

// Chain creates a new middleware chain from the provided middleware, preserving
// order.
func Chain(middleware ...Middleware) MiddlewareChain {
    return MiddlewareChain(middleware)
}

// Error types for middleware

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   any
	Message string
	Cause   error
}

func (e *ValidationError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// TimeoutError represents a timeout error
type TimeoutError struct {
	Duration time.Duration
	Command  string
}

func (e *TimeoutError) Error() string {
	return "command '" + e.Command + "' timed out after " + e.Duration.String()
}

// RecoveryError represents a panic recovery
type RecoveryError struct {
	Panic   any
	Command string
	Stack   []byte
}

func (e *RecoveryError) Error() string {
	return "command '" + e.Command + "' panicked: " + toString(e.Panic)
}

// Configuration types

// MiddlewareConfig contains configuration for middleware behavior
type MiddlewareConfig struct {
	LogLevel         LogLevel
	LogOutput        LogOutput
	LogFormat        LogFormat
	IncludeArgs      bool
	PrintStack       bool
	StackSize        int
	DefaultTimeout   time.Duration
	CustomValidators map[string]ValidatorFunc
}

// LogLevel represents logging levels
type LogLevel int

const (
	LogLevelNone LogLevel = iota
	LogLevelError
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

// LogOutput represents log output destinations
type LogOutput int

const (
	LogOutputStderr LogOutput = iota
	LogOutputStdout
	LogOutputNone
)

// LogFormat represents log formats
type LogFormat int

const (
	LogFormatText LogFormat = iota
	LogFormatJSON
)

// RequestInfo contains information about command execution
type RequestInfo struct {
	Command   string
	Args      []string
	StartTime time.Time
	Duration  time.Duration
	Error     error
	Metadata  map[string]any
}

// Configuration options

type MiddlewareOption func(config *MiddlewareConfig)

func DefaultConfig() *MiddlewareConfig {
	return &MiddlewareConfig{
		LogLevel:         LogLevelInfo,
		LogOutput:        LogOutputStderr,
		LogFormat:        LogFormatText,
		IncludeArgs:      true,
		PrintStack:       true,
		StackSize:        4096,
		DefaultTimeout:   30 * time.Second,
		CustomValidators: make(map[string]ValidatorFunc),
	}
}

func WithLogLevel(level LogLevel) MiddlewareOption {
	return func(config *MiddlewareConfig) {
		config.LogLevel = level
	}
}

func WithTimeout(timeout time.Duration) MiddlewareOption {
	return func(config *MiddlewareConfig) {
		config.DefaultTimeout = timeout
	}
}

func WithStackTrace(enabled bool) MiddlewareOption {
	return func(config *MiddlewareConfig) {
		config.PrintStack = enabled
	}
}

// Utility functions

func toString(v any) string {
	if v == nil {
		return "<nil>"
	}
	if s, ok := v.(string); ok {
		return s
	}
	if err, ok := v.(error); ok {
		return err.Error()
	}
	return "<unknown>"
}

func getCommandName(ctx Context) string {
	cmd := ctx.Command()
	if cmd == nil {
		return "unknown"
	}
	return cmd.Name()
}
