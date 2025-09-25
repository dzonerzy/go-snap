package middleware

import (
	"encoding/json"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/dzonerzy/go-snap/internal/pool"
)

// requestInfoPool is a global pool for RequestInfo objects to reduce allocations
var requestInfoPool = pool.NewPoolWithReset(
	func() *RequestInfo {
		return &RequestInfo{
			Metadata: make(map[string]any, 4), // Small initial capacity
		}
	},
	func(info *RequestInfo) {
		// Reset all fields for reuse
		info.Command = ""
		info.Args = info.Args[:0]
		info.StartTime = time.Time{}
		info.Duration = 0
		info.Error = nil
		// Clear metadata map without reallocating
		for k := range info.Metadata {
			delete(info.Metadata, k)
		}
	},
)

// Logger creates a middleware that logs command execution requests
func Logger(options ...MiddlewareOption) Middleware {
	config := DefaultConfig()
	for _, option := range options {
		option(config)
	}

	return func(next ActionFunc) ActionFunc {
		return func(ctx Context) error {
			// Skip logging if level is None
			if config.LogLevel == LogLevelNone {
				return next(ctx)
			}

			// Get RequestInfo from pool
			info := requestInfoPool.Get()
			defer requestInfoPool.Put(info) // Return to pool when done

			// Initialize request info
			info.Command = getCommandName(ctx)
			info.Args = append(info.Args, ctx.Args()...) // Reuse slice capacity
			info.StartTime = time.Now()

			// Log request start (debug level only)
			if config.LogLevel >= LogLevelDebug {
				logRequest(config, info, "START")
			}

			// Execute the action
			err := next(ctx)

			// Update request info with results
			info.Duration = time.Since(info.StartTime)
			info.Error = err

			// Log request completion
			logRequest(config, info, getLogLevel(err))

			return err
		}
	}
}

// getLogLevel determines log level based on error status
func getLogLevel(err error) string {
	if err != nil {
		return "ERROR"
	}
	return "SUCCESS"
}

// logRequest writes the log entry based on configuration
func logRequest(config *MiddlewareConfig, info *RequestInfo, level string) {
	// Determine if we should log this level
	if !shouldLog(config.LogLevel, level) {
		return
	}

	// Get output writer
	writer := getLogWriter(config.LogOutput)
	if writer == nil {
		return
	}

	// Format and write log entry
	switch config.LogFormat { // exhaustive over LogFormat
	case LogFormatJSON:
		writeJSONLog(writer, info, level, config)
	case LogFormatText:
		writeTextLog(writer, info, level, config)
	default:
		writeTextLog(writer, info, level, config)
	}
}

// shouldLog determines if the log level warrants logging
func shouldLog(configLevel LogLevel, messageLevel string) bool {
	switch messageLevel {
	case "ERROR":
		return configLevel >= LogLevelError
	case "SUCCESS":
		return configLevel >= LogLevelInfo
	case "START":
		return configLevel >= LogLevelDebug
	default:
		return configLevel >= LogLevelInfo
	}
}

// getLogWriter returns the appropriate writer based on configuration
func getLogWriter(output LogOutput) io.Writer {
	switch output {
	case LogOutputStdout:
		return os.Stdout
	case LogOutputStderr:
		return os.Stderr
	case LogOutputNone:
		return nil
	default:
		return os.Stderr
	}
}

// writeTextLog writes a human-readable text log entry with minimal allocations
func writeTextLog(writer io.Writer, info *RequestInfo, level string, config *MiddlewareConfig) {
	// Get buffer from pool instead of strings.Builder
	buf := pool.GetBuffer(256)
	defer pool.PutBuffer(buf)

	// Build log entry efficiently
	*buf = append(*buf, '[')
	*buf = append(*buf, info.StartTime.Format("2006-01-02 15:04:05")...)
	*buf = append(*buf, "] "...)
	*buf = append(*buf, level...)
	*buf = append(*buf, " command="...)
	*buf = append(*buf, info.Command...)

	if info.Duration > 0 {
		*buf = append(*buf, " duration="...)
		*buf = append(*buf, info.Duration.String()...)
	}

	if config.IncludeArgs && len(info.Args) > 0 {
		*buf = append(*buf, " args="...)
		// Avoid strings.Join allocation for small arg counts
		for i, arg := range info.Args {
			if i > 0 {
				*buf = append(*buf, ' ')
			}
			*buf = append(*buf, arg...)
		}
	}

	if info.Error != nil {
		*buf = append(*buf, " error=\""...)
		*buf = append(*buf, info.Error.Error()...)
		*buf = append(*buf, '"')
	}

	*buf = append(*buf, '\n')

	// Write directly from buffer; ignore write errors (logging best-effort)
	//nolint:errcheck,gosec // Logging is best-effort; ignore write errors.
	writer.Write(*buf)
}

// writeJSONLog writes a structured JSON log entry with minimal allocations
func writeJSONLog(writer io.Writer, info *RequestInfo, level string, config *MiddlewareConfig) {
	// Get buffer from pool for JSON construction
	buf := pool.GetBuffer(512)
	defer pool.PutBuffer(buf)

	// Build JSON manually to reduce allocations
	*buf = append(*buf, '{')
	*buf = append(*buf, `"timestamp":"`...)
	*buf = append(*buf, info.StartTime.Format(time.RFC3339)...)
	*buf = append(*buf, `","level":"`...)
	*buf = append(*buf, level...)
	*buf = append(*buf, `","command":"`...)
	*buf = append(*buf, info.Command...)
	*buf = append(*buf, '"')

	if info.Duration > 0 {
		*buf = append(*buf, `,"duration_ms":`...)
		*buf = append(*buf, strconv.FormatInt(info.Duration.Milliseconds(), 10)...)
	}

	if config.IncludeArgs && len(info.Args) > 0 {
		*buf = append(*buf, `,"args":[`...)
		for i, arg := range info.Args {
			if i > 0 {
				*buf = append(*buf, ',')
			}
			enc, _ := json.Marshal(arg)
			*buf = append(*buf, enc...)
		}
		*buf = append(*buf, ']')
	}

	if info.Error != nil {
		*buf = append(*buf, `,"error":`...)
		enc, _ := json.Marshal(info.Error.Error())
		*buf = append(*buf, enc...)
	}

	// For metadata, fall back to json.Marshal since it's complex and rarely used
	if len(info.Metadata) > 0 {
		metadataJSON, err := json.Marshal(info.Metadata)
		if err == nil {
			*buf = append(*buf, `,"metadata":`...)
			*buf = append(*buf, metadataJSON...)
		}
	}

	*buf = append(*buf, '}')
	*buf = append(*buf, '\n')

	//nolint:errcheck,gosec // Logging is best-effort; ignore write errors.
	writer.Write(*buf)
}

// LoggerWithWriter creates a logger middleware that writes to a specific writer
func LoggerWithWriter(writer io.Writer, options ...MiddlewareOption) Middleware {
	config := DefaultConfig()
	for _, option := range options {
		option(config)
	}

	return func(next ActionFunc) ActionFunc {
		return func(ctx Context) error {
			if config.LogLevel == LogLevelNone {
				return next(ctx)
			}

			// Get RequestInfo from pool
			info := requestInfoPool.Get()
			defer requestInfoPool.Put(info)

			// Initialize request info
			info.Command = getCommandName(ctx)
			info.Args = append(info.Args, ctx.Args()...)
			info.StartTime = time.Now()

			if config.LogLevel >= LogLevelDebug {
				logRequestToWriter(writer, config, info, "START")
			}

			err := next(ctx)

			info.Duration = time.Since(info.StartTime)
			info.Error = err

			logRequestToWriter(writer, config, info, getLogLevel(err))

			return err
		}
	}
}

// logRequestToWriter writes log entry to specific writer
func logRequestToWriter(writer io.Writer, config *MiddlewareConfig, info *RequestInfo, level string) {
	if !shouldLog(config.LogLevel, level) {
		return
	}

	switch config.LogFormat { // exhaustive over LogFormat
	case LogFormatJSON:
		writeJSONLog(writer, info, level, config)
	case LogFormatText:
		writeTextLog(writer, info, level, config)
	default:
		writeTextLog(writer, info, level, config)
	}
}

// Convenience constructors for common logging scenarios

// DebugLogger creates a logger with debug level (logs everything)
func DebugLogger() Middleware {
	return Logger(WithLogLevel(LogLevelDebug))
}

// InfoLogger creates a logger with info level (logs success and errors)
func InfoLogger() Middleware {
	return Logger(WithLogLevel(LogLevelInfo))
}

// ErrorLogger creates a logger with error level (logs only errors)
func ErrorLogger() Middleware {
	return Logger(WithLogLevel(LogLevelError))
}

// JSONLogger creates a logger that outputs JSON format
func JSONLogger() Middleware {
	return Logger(func(config *MiddlewareConfig) {
		config.LogFormat = LogFormatJSON
	})
}

// SilentLogger creates a logger that doesn't output anything (useful for testing)
func SilentLogger() Middleware {
	return Logger(func(config *MiddlewareConfig) {
		config.LogOutput = LogOutputNone
	})
}
