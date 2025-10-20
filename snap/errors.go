package snap

import (
	"fmt"
	"strings"

	"github.com/dzonerzy/go-snap/internal/fuzzy"
)

// ErrorType represents error categories for CLI operations.
// These categories drive suggestion logic and exit-code mapping (via ExitCodeManager).
type ErrorType string

const (
	ErrorTypeUnknownFlag        ErrorType = "unknown_flag"
	ErrorTypeUnknownCommand     ErrorType = "unknown_command"
	ErrorTypeInvalidFlag        ErrorType = "invalid_flag"
	ErrorTypeInvalidValue       ErrorType = "invalid_value"
	ErrorTypeMissingValue       ErrorType = "missing_value"
	ErrorTypeInternal           ErrorType = "internal_error"
	ErrorTypeFlagGroupViolation ErrorType = "flag_group_violation"
	ErrorTypeMissingRequired    ErrorType = "missing_required"
	ErrorTypePermission         ErrorType = "permission"
	ErrorTypeValidation         ErrorType = "validation"
	ErrorTypeInvalidArgument    ErrorType = "invalid_argument"
)

// ParseError represents parsing-specific errors (used by parser.go)
type ParseError struct {
	Type           ErrorType
	Message        string
	Flag           string
	Command        string
	GroupName      string // For flag group errors - enables contextual help
	Suggestion     string
	CurrentCommand *Command // The command context where error occurred (for flag suggestions)
}

func (e *ParseError) Error() string {
	return e.Message
}

// NewParseError creates a new ParseError with the given type and message
func NewParseError(errType ErrorType, message string) *ParseError {
	return &ParseError{
		Type:    errType,
		Message: message,
	}
}

// CLIError is an enhanced error type with smart suggestions (see SPECS.md).
type CLIError struct {
	Type           ErrorType
	Message        string
	Suggestions    []string
	Cause          error
	Context        map[string]any
	formattedError string // Full formatted error message including suggestions and help
}

// Error implements the error interface
func (e *CLIError) Error() string {
	// Return the fully formatted error if available, otherwise just the message
	if e.formattedError != "" {
		return e.formattedError
	}
	return e.Message
}

// Error builders for fluent API

// NewError creates a new CLIError with the given type and message
func NewError(typ ErrorType, message string) *CLIError {
	return &CLIError{
		Type:        typ,
		Message:     message,
		Suggestions: make([]string, 0),
		Context:     make(map[string]any),
	}
}

// WithSuggestion adds a suggestion to the error
func (e *CLIError) WithSuggestion(suggestion string) *CLIError {
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// WithCause adds an underlying cause to the error
func (e *CLIError) WithCause(cause error) *CLIError {
	e.Cause = cause
	return e
}

// WithContext adds context information to the error
func (e *CLIError) WithContext(key string, value any) *CLIError {
	e.Context[key] = value
	return e
}

// ErrorHandler provides smart error handling with fuzzy matching suggestions.
type ErrorHandler struct {
	suggestCommands bool
	suggestFlags    bool
	maxDistance     int
	customHandlers  map[ErrorType]func(*CLIError) *CLIError
	showHelpOnError bool
}

// NewErrorHandler creates a new error handler with defaults
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		suggestCommands: false, // Disabled by default - user must opt-in
		suggestFlags:    false, // Disabled by default - user must opt-in
		maxDistance:     2,
		customHandlers:  make(map[ErrorType]func(*CLIError) *CLIError),
	}
}

// SuggestCommands enables/disables command suggestions
func (eh *ErrorHandler) SuggestCommands(enabled bool) *ErrorHandler {
	eh.suggestCommands = enabled
	return eh
}

// SuggestFlags enables/disables flag suggestions
func (eh *ErrorHandler) SuggestFlags(enabled bool) *ErrorHandler {
	eh.suggestFlags = enabled
	return eh
}

// MaxDistance sets the maximum edit distance for suggestions
func (eh *ErrorHandler) MaxDistance(distance int) *ErrorHandler {
	eh.maxDistance = distance
	return eh
}

// ShowHelpOnError controls whether contextual help is printed after an error.
// When enabled, app-level or command-level help is displayed based on the
// current parse context.
func (eh *ErrorHandler) ShowHelpOnError(enabled bool) *ErrorHandler {
	eh.showHelpOnError = enabled
	return eh
}

// Handle registers a custom handler for a specific error type
func (eh *ErrorHandler) Handle(typ ErrorType, handler func(*CLIError) *CLIError) *ErrorHandler {
	eh.customHandlers[typ] = handler
	return eh
}

// ProcessError handles a CLIError and potentially modifies it with suggestions
func (eh *ErrorHandler) ProcessError(err *CLIError, app *App) *CLIError {
	// Apply custom handler if exists
	if handler, exists := eh.customHandlers[err.Type]; exists {
		err = handler(err)
	}

	// Add smart suggestions based on error type
	switch err.Type { // exhaustive over ErrorType
	case ErrorTypeUnknownFlag:
		if eh.suggestFlags {
			eh.addFlagSuggestions(err, app)
		}
	case ErrorTypeUnknownCommand:
		if eh.suggestCommands {
			eh.addCommandSuggestions(err, app)
		}
	case ErrorTypeFlagGroupViolation:
		// Flag group errors get contextual help
		eh.addGroupContext(err, app)
	case ErrorTypeInvalidFlag, ErrorTypeInvalidValue, ErrorTypeMissingValue,
		ErrorTypeInternal, ErrorTypeMissingRequired, ErrorTypePermission, ErrorTypeValidation,
		ErrorTypeInvalidArgument:
		// No suggestions for these by default.
	}

	return err
}

// addFlagSuggestions adds fuzzy-matched flag suggestions using internal/fuzzy.
func (eh *ErrorHandler) addFlagSuggestions(err *CLIError, app *App) {
	if flagName, ok := err.Context["flag"].(string); ok {
		// Get command context if available
		var currentCmd *Command
		if cmd, okCmd := err.Context["current_command"].(*Command); okCmd {
			currentCmd = cmd
		}

		// Find similar flag names using fuzzy matching
		bestMatch := eh.findBestFlagMatch(flagName, app, currentCmd)
		if bestMatch != "" {
			_ = err.WithSuggestion(fmt.Sprintf("Did you mean '--%s'?", bestMatch))
		}
	}
}

// addCommandSuggestions adds fuzzy-matched command suggestions using internal/fuzzy.
func (eh *ErrorHandler) addCommandSuggestions(err *CLIError, app *App) {
	if cmdName, ok := err.Context["command"].(string); ok {
		// Find similar command names
		bestMatch := eh.findBestCommandMatch(cmdName, app)
		if bestMatch != "" {
			_ = err.WithSuggestion(fmt.Sprintf("Did you mean '%s'?", bestMatch))
		}
	}
}

// addGroupContext adds context for flag group violations
func (eh *ErrorHandler) addGroupContext(err *CLIError, app *App) {
	// This will be enhanced when we integrate with help system
	if groupName, ok := err.Context["group"].(string); ok {
		_ = err.WithSuggestion(
			fmt.Sprintf(
				"Run '%s --help' to see valid flag combinations for group '%s'",
				app.name, groupName,
			),
		)
	}
}

// Efficient fuzzy matching using internal/fuzzy package
func (eh *ErrorHandler) findBestFlagMatch(input string, app *App, currentCmd *Command) string {
	// Collect app-level flags
	flagNames := make([]string, 0, len(app.flags))
	for flagName := range app.flags {
		flagNames = append(flagNames, flagName)
	}

	// If we're in a command context, also include command-level flags
	if currentCmd != nil {
		for flagName := range currentCmd.flags {
			flagNames = append(flagNames, flagName)
		}
	}

	return fuzzy.FindBestFlag(input, flagNames, eh.maxDistance)
}

func (eh *ErrorHandler) findBestCommandMatch(input string, app *App) string {
	// Collect app-level commands
	cmdNames := make([]string, 0, len(app.commands))
	for cmdName := range app.commands {
		cmdNames = append(cmdNames, cmdName)
	}

	// If we're in a command context, also include subcommands
	if app.currentResult != nil && app.currentResult.Command != nil {
		for cmdName := range app.currentResult.Command.subcommands {
			cmdNames = append(cmdNames, cmdName)
		}
	}

	return fuzzy.FindBestCommand(input, cmdNames, eh.maxDistance)
}

// formatError builds the error message with suggestions.
// The formatted message is stored in the CLIError and returned by Error().
// Note: This does NOT include help text - help should be printed separately if ShowHelpOnError is enabled.
func (eh *ErrorHandler) formatError(err *CLIError, app *App) *CLIError {
	var builder strings.Builder

	// Build the main error message
	builder.WriteString(fmt.Sprintf("Error: %s\n", err.Message))

	// Add suggestions if any
	for _, suggestion := range err.Suggestions {
		builder.WriteString(fmt.Sprintf("  %s\n", suggestion))
	}

	// For flag group violations, add group help
	if err.Type == ErrorTypeFlagGroupViolation {
		if groupName, ok := err.Context["group"].(string); ok {
			builder.WriteString("\n")
			builder.WriteString(eh.formatFlagGroupHelp(groupName, app))
		}
	}

	// Store the formatted error (without trailing newline for cleaner output)
	err.formattedError = strings.TrimRight(builder.String(), "\n")
	return err
}

// formatFlagGroupHelp builds help text for a specific flag group
func (eh *ErrorHandler) formatFlagGroupHelp(groupName string, app *App) string {
	var builder strings.Builder

	for _, group := range app.flagGroups {
		if group.Name == groupName {
			builder.WriteString(fmt.Sprintf("Flag group '%s':\n", groupName))
			if group.Description != "" {
				builder.WriteString(fmt.Sprintf("  %s\n", group.Description))
			}

			for _, flag := range group.Flags {
				builder.WriteString(fmt.Sprintf("  --%s    %s\n", flag.Name, flag.Description))
			}

			builder.WriteString(fmt.Sprintf("\nConstraint: %s\n", eh.formatConstraint(group.Constraint)))
			return builder.String()
		}
	}
	// Also check current command's groups if not found at app level
	if app.currentResult != nil && app.currentResult.Command != nil {
		for _, group := range app.currentResult.Command.flagGroups {
			if group.Name == groupName {
				builder.WriteString(fmt.Sprintf("Flag group '%s':\n", groupName))
				if group.Description != "" {
					builder.WriteString(fmt.Sprintf("  %s\n", group.Description))
				}
				for _, flag := range group.Flags {
					builder.WriteString(fmt.Sprintf("  --%s    %s\n", flag.Name, flag.Description))
				}
				builder.WriteString(fmt.Sprintf("\nConstraint: %s\n", eh.formatConstraint(group.Constraint)))
				return builder.String()
			}
		}
	}
	return ""
}

// formatConstraint returns a human-readable description of the constraint
func (eh *ErrorHandler) formatConstraint(constraint GroupConstraintType) string {
	switch constraint { // exhaustive over GroupConstraintType
	case GroupMutuallyExclusive:
		return "Only one of these flags can be used at a time"
	case GroupRequiredGroup:
		return "At least one of these flags is required"
	case GroupAllOrNone:
		return "Either all of these flags must be provided, or none"
	case GroupExactlyOne:
		return "Exactly one of these flags must be provided"
	case GroupNoConstraint:
		return ""
	case GroupAtLeastOne:
		return "At least one of these flags is required"
	default:
		return ""
	}
}
