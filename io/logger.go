package snapio

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelSuccess
	LevelWarning
	LevelError
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelSuccess:
		return "SUCCESS"
	case LevelWarning:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// LogFormat defines the output format for log messages
type LogFormat int

const (
	LogFormatCircles LogFormat = iota // Default: 🔵 🟢 🟡 🔴 🟣
	LogFormatSymbols                  // ℹ ✓ ⚠ ✗ •
	LogFormatTagged                   // [INFO] [SUCCESS] [WARN] [ERROR] [DEBUG]
	LogFormatPlain                    // No prefix
	LogFormatCustom                   // User-defined template
)

// Logger provides structured logging with semantic levels and customizable formatting
type Logger struct {
	io           *IOManager
	format       LogFormat
	template     string
	prefixes     map[LogLevel]string
	withTime     bool
	timeFormat   string
	errorsStderr bool
	theme        Theme
}

// NewLogger creates a new logger bound to the given IOManager
func NewLogger(io *IOManager) *Logger {
	// Auto-select theme based on terminal color support
	// DefaultTheme(io) automatically selects the best theme for the color level:
	// - ColorLevel 1 (16 colors): DefaultTheme16()
	// - ColorLevel 2 (256 colors): DefaultTheme256()
	// - ColorLevel 3 (truecolor): DefaultThemeTruecolor()
	theme := DefaultTheme(io)

	return &Logger{
		io:           io,
		format:       LogFormatCircles,
		errorsStderr: true,
		timeFormat:   "15:04:05",
		theme:        theme,
		prefixes:     defaultCirclePrefixes(),
	}
}

// defaultCirclePrefixes returns the default colored circle emoji prefixes
func defaultCirclePrefixes() map[LogLevel]string {
	return map[LogLevel]string{
		LevelDebug:   "🟣",
		LevelInfo:    "🔵",
		LevelSuccess: "🟢",
		LevelWarning: "🟡",
		LevelError:   "🔴",
	}
}

// defaultSymbolPrefixes returns Unicode symbol prefixes (no emoji)
func defaultSymbolPrefixes() map[LogLevel]string {
	return map[LogLevel]string{
		LevelDebug:   "●", // U+25CF Black Circle
		LevelInfo:    "◆", // U+25C6 Black Diamond
		LevelSuccess: "✓", // U+2713 Check Mark
		LevelWarning: "▲", // U+25B2 Black Up-Pointing Triangle
		LevelError:   "✗", // U+2717 Ballot X
	}
}

// defaultTaggedPrefixes returns bracketed tag prefixes
func defaultTaggedPrefixes() map[LogLevel]string {
	return map[LogLevel]string{
		LevelDebug:   "[DEBUG]",
		LevelInfo:    "[INFO]",
		LevelSuccess: "[SUCCESS]",
		LevelWarning: "[WARN]",
		LevelError:   "[ERROR]",
	}
}

// WithFormat sets the log format and returns the logger for chaining
func (l *Logger) WithFormat(format LogFormat) *Logger {
	l.format = format
	switch format {
	case LogFormatCircles:
		l.prefixes = defaultCirclePrefixes()
	case LogFormatSymbols:
		l.prefixes = defaultSymbolPrefixes()
	case LogFormatTagged:
		l.prefixes = defaultTaggedPrefixes()
	case LogFormatPlain:
		l.prefixes = make(map[LogLevel]string)
	case LogFormatCustom:
		// Custom template will be used, prefixes may be customized separately
	}
	return l
}

// WithTemplate sets a custom template for LogFormatCustom
// Template variables: {{.Level}}, {{.Time}}, {{.Message}}, {{.Prefix}}
func (l *Logger) WithTemplate(template string) *Logger {
	l.template = template
	l.format = LogFormatCustom
	return l
}

// SetPrefix sets a custom prefix for a specific log level
func (l *Logger) SetPrefix(level LogLevel, prefix string) *Logger {
	if l.prefixes == nil {
		l.prefixes = make(map[LogLevel]string)
	}
	l.prefixes[level] = prefix
	return l
}

// WithTimestamp enables or disables timestamp in log output
func (l *Logger) WithTimestamp(enabled bool) *Logger {
	l.withTime = enabled
	return l
}

// WithTimeFormat sets the time format (Go time format string)
func (l *Logger) WithTimeFormat(format string) *Logger {
	l.timeFormat = format
	return l
}

// ErrorsToStderr controls whether errors and warnings go to stderr
func (l *Logger) ErrorsToStderr(enabled bool) *Logger {
	l.errorsStderr = enabled
	return l
}

// WithTheme sets a custom theme for semantic colors
func (l *Logger) WithTheme(theme Theme) *Logger {
	l.theme = theme
	return l
}

// Log outputs a log message at the specified level
func (l *Logger) Log(level LogLevel, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	output := l.formatMessage(level, msg)

	writer := l.selectWriter(level)
	fmt.Fprintln(writer, output)
}

// formatMessage formats the log message according to the configured format
func (l *Logger) formatMessage(level LogLevel, msg string) string {
	if l.format == LogFormatCustom && l.template != "" {
		return l.formatCustomTemplate(level, msg)
	}

	// Check if message is empty or only whitespace
	trimmedMsg := strings.TrimSpace(msg)
	isEmpty := len(trimmedMsg) == 0

	prefix := l.prefixes[level]
	timeStr := ""

	if l.withTime {
		timeStr = " [" + time.Now().Format(l.timeFormat) + "]"
	}

	// Build the formatted message
	var formatted string

	// If message is empty/whitespace, don't add prefix - just return the original message
	if isEmpty {
		return msg
	}

	// For plain format, no prefix but still apply color
	if l.format == LogFormatPlain {
		if l.withTime {
			formatted = timeStr[1:] + " " + msg // Remove leading space from timeStr
		} else {
			formatted = msg
		}
		return l.colorizeByLevel(level, formatted)
	}

	// Build formatted message with prefix
	if prefix != "" {
		formatted = prefix + timeStr + " " + msg
	} else {
		formatted = strings.TrimPrefix(timeStr+" "+msg, " ")
	}

	// Apply semantic color based on level
	return l.colorizeByLevel(level, formatted)
}

// formatCustomTemplate formats using a custom template
func (l *Logger) formatCustomTemplate(level LogLevel, msg string) string {
	output := l.template
	output = strings.ReplaceAll(output, "{{.Level}}", level.String())
	output = strings.ReplaceAll(output, "{{.Message}}", msg)
	output = strings.ReplaceAll(output, "{{.Prefix}}", l.prefixes[level])

	if strings.Contains(output, "{{.Time}}") {
		output = strings.ReplaceAll(output, "{{.Time}}", time.Now().Format(l.timeFormat))
	}

	return l.colorizeByLevel(level, output)
}

// colorizeByLevel applies semantic color based on log level
func (l *Logger) colorizeByLevel(level LogLevel, text string) string {
	if !l.io.SupportsColor() {
		return text
	}

	var color ColorSpec
	switch level {
	case LevelDebug:
		color = l.theme.Debug
	case LevelInfo:
		color = l.theme.Info
	case LevelSuccess:
		color = l.theme.Success
	case LevelWarning:
		color = l.theme.Warning
	case LevelError:
		color = l.theme.Error
	default:
		return text
	}

	style := NewStyle().Fg(color)
	return style.Sprint(l.io, text)
}

// selectWriter chooses stdout or stderr based on log level and configuration
func (l *Logger) selectWriter(level LogLevel) io.Writer {
	if l.errorsStderr && (level == LevelError || level == LevelWarning) {
		return l.io.Err()
	}
	return l.io.Out()
}

// Convenience methods for each log level

// Debug logs a debug message (purple circle by default)
func (l *Logger) Debug(format string, args ...any) {
	l.Log(LevelDebug, format, args...)
}

// Info logs an informational message (blue circle by default)
func (l *Logger) Info(format string, args ...any) {
	l.Log(LevelInfo, format, args...)
}

// Success logs a success message (green circle by default)
func (l *Logger) Success(format string, args ...any) {
	l.Log(LevelSuccess, format, args...)
}

// Warning logs a warning message (yellow circle by default)
func (l *Logger) Warning(format string, args ...any) {
	l.Log(LevelWarning, format, args...)
}

// Error logs an error message (red circle by default)
func (l *Logger) Error(format string, args ...any) {
	l.Log(LevelError, format, args...)
}
