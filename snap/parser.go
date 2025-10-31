package snap

import (
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"time"
	"unsafe"

	"github.com/dzonerzy/go-snap/internal/intern"
	"github.com/dzonerzy/go-snap/internal/pool"
)

// Pre-allocated byte slices for common values to avoid allocations
var (
	trueBoolBytes = []byte("true")
)

// ParseState represents the current state of the parser state machine
type ParseState int

const (
	StateInit ParseState = iota
	StateGlobalFlags
	StateCommand
	StateCommandFlags
	StatePositionalArgs
	StateComplete
	StateError
)

// ParseResult contains the parsed command structure without allocations
type ParseResult struct {
	Command           *Command
	*pool.ParseResult // Embed the pooled ParseResult

	// Slices that need cleanup
	stringSlices []*[]string
	intSlices    []*[]int
}

// Parser implements zero-allocation argument parsing
type Parser struct {
	// Pre-allocated buffers for zero-allocation parsing
	flagBuffer  []byte       // Reusable buffer for flag parsing
	valueBuffer []byte       // Reusable buffer for value parsing
	argsBuffer  []string     // Reusable slice for arguments
	flagsBuffer []ParsedFlag // Reusable slice for parsed flags

	// Parser state
	state      ParseState
	position   int
	currentCmd *Command
	app        *App

	// Error tracking (pre-allocated)
	lastError     error
	suggestions   []string
	currentResult *ParseResult

	// Pre-allocated result for zero allocations
	reusableResult *ParseResult

	// Reusable buffer for levenshtein distance calculation (avoid allocations in error paths)
	levenshteinBuffer []int

	// Pre-allocated error for reuse (avoid allocations in error paths)
	reusableError *ParseError

	// Removed: boxedValues approach doesn't scale
}

// ParsedFlag represents a parsed flag without string allocations
type ParsedFlag struct {
	Name     []byte // Flag name as byte slice
	Value    []byte // Flag value as byte slice
	IsShort  bool   // True if short flag (-v)
	HasValue bool   // True if flag has explicit value
	Position int    // Position in argument list
}

// NewParser creates a new zero-allocation parser
func NewParser(app *App) *Parser {
	p := &Parser{
		app:               app,
		flagBuffer:        make([]byte, 0, 256),      // Pre-allocate 256 bytes
		valueBuffer:       make([]byte, 0, 512),      // Pre-allocate 512 bytes
		argsBuffer:        make([]string, 0, 32),     // Pre-allocate 32 strings
		flagsBuffer:       make([]ParsedFlag, 0, 16), // Pre-allocate 16 flags
		suggestions:       make([]string, 0, 8),      // Pre-allocate suggestions
		levenshteinBuffer: make([]int, 64),           // Pre-allocate buffer for edit distance
		reusableError:     &ParseError{},             // Pre-allocate error for reuse
	}

	// Use pooled result instead of pre-allocated one
	pooledResult := pool.GetParseResult()
	p.reusableResult = &ParseResult{ParseResult: pooledResult}

	// Removed: Pre-allocated boxed values approach
	// Note: String interning is now handled by internal/intern package

	return p
}

// Parse parses command line arguments with zero allocations for hot path
func (p *Parser) Parse(args []string) (*ParseResult, error) {
	// Reset parser state without allocations
	p.reset()

	// Initialize result early
	p.currentResult = p.getResult()

	// Fast path: no arguments
	if len(args) == 0 {
		return p.finalize()
	}

	// Main parsing loop - single pass, left to right
	for p.position < len(args) {
		arg := args[p.position]

		// Fast path: check for common patterns without allocation
		if len(arg) == 0 {
			p.position++
			continue
		}

		// Parse based on current state and argument format
		if err := p.parseArgument(arg, args); err != nil {
			return nil, err
		}

		p.position++
	}

	// Finalize parsing and return result
	return p.finalize()
}

// parseArgument handles a single argument based on parser state
//
//nolint:gocognit,gocyclo,cyclop // This is acceptable because it's a complex function that handles many cases.
func (p *Parser) parseArgument(arg string, allArgs []string) error {
	// Convert to byte slice for zero-allocation operations
	argBytes := stringToBytes(arg)

	// If already in positional mode, treat everything as positional
	if p.state == StatePositionalArgs {
		return p.parsePositionalArg(argBytes)
	}

	// Check if RestArgs is enabled - if so, treat everything as positional
	hasRestArgs := false
	if p.currentCmd != nil {
		hasRestArgs = p.currentCmd.hasRestArgs
	} else if p.app != nil {
		hasRestArgs = p.app.hasRestArgs
	}
	if hasRestArgs {
		return p.parsePositionalArg(argBytes)
	}

	// "--" terminates flag parsing; subsequent tokens are positional args
	// In WrapDynamic mode, we need to preserve "--" as a positional arg because
	// tools like cgo use it to separate tool flags from compiler flags
	if len(argBytes) == 2 && argBytes[0] == '-' && argBytes[1] == '-' {
		p.state = StatePositionalArgs

		// Check if we're in dynamic wrapper mode - if so, preserve "--" as positional arg
		isDynamic := false
		if p.currentCmd != nil && p.currentCmd.wrapper != nil && p.currentCmd.wrapper.Dynamic {
			isDynamic = true
		} else if p.app != nil && p.app.defaultWrapper != nil && p.app.defaultWrapper.Dynamic {
			isDynamic = true
		}

		// In dynamic mode, add "--" as a positional argument instead of consuming it
		if isDynamic {
			return p.parsePositionalArg(argBytes)
		}
		return nil
	}

	// In dynamic wrapper mode, treat all args as positional (don't parse flags)
	// This prevents buildid values like -hoF... from being parsed as -h help flag
	isDynamic := false
	if p.currentCmd != nil && p.currentCmd.wrapper != nil && p.currentCmd.wrapper.Dynamic {
		isDynamic = true
	} else if p.app != nil && p.app.defaultWrapper != nil && p.app.defaultWrapper.Dynamic {
		isDynamic = true
	}
	if isDynamic {
		return p.parsePositionalArg(argBytes)
	}

	switch {
	case len(argBytes) >= 2 && argBytes[0] == '-' && argBytes[1] == '-':
		// Long flag: --flag or --flag=value
		return p.parseLongFlag(argBytes, allArgs)

	case len(argBytes) >= 1 && argBytes[0] == '-':
		// Short flag(s): -f or -abc
		return p.parseShortFlag(argBytes, allArgs)

	case p.state == StateInit || p.state == StateGlobalFlags:
		// Top-level token: treat as a command only if it exists; otherwise,
		// treat as positional arg if positional args are defined, or if wrapper is configured.
		name := intern.InternBytes(argBytes)
		if cmd := p.findCommand(name); cmd != nil {
			return p.parseCommand(argBytes)
		}
		// If app has positional args defined or RestArgs, treat as positional
		if p.app != nil && (len(p.app.args) > 0 || p.app.hasRestArgs) {
			return p.parsePositionalArg(argBytes)
		}
		// If app has a wrapper, treat as positional
		if p.app != nil && p.app.defaultWrapper != nil {
			return p.parsePositionalArg(argBytes)
		}
		// Otherwise, it's an unknown command
		return p.createUnknownCommandError(name)
	case p.state == StateCommandFlags:
		// In a command context: if the current command defines subcommands,
		// an unknown non-flag token should be treated as an unknown subcommand
		// to enable suggestions (rather than silently becoming a positional arg).
		if p.currentCmd != nil && p.currentCmd.subcommands != nil && len(p.currentCmd.subcommands) > 0 {
			name := intern.InternBytes(argBytes)
			if _, ok := p.currentCmd.subcommands[name]; ok {
				return p.parseCommand(argBytes)
			}
			// Unknown token while subcommands exist -> surface an error with suggestion
			return p.createUnknownCommandError(name)
		}
		// No subcommands defined -> treat as positional argument
		return p.parsePositionalArg(argBytes)

	case p.state == StatePositionalArgs:
		return p.parsePositionalArg(argBytes)
	default:
		// Positional argument
		return p.parsePositionalArg(argBytes)
	}
}

// parseLongFlag parses long flags (--flag, --flag=value) with zero allocations
func (p *Parser) parseLongFlag(argBytes []byte, allArgs []string) error {
	// Skip the '--' prefix
	flagBytes := argBytes[2:]

	// Find '=' separator without allocation
	var nameBytes, valueBytes []byte
	var hasValue bool

	if eqPos := findByte(flagBytes, '='); eqPos != -1 {
		nameBytes = flagBytes[:eqPos]
		valueBytes = flagBytes[eqPos+1:]
		hasValue = true
	} else {
		nameBytes = flagBytes
		hasValue = false
	}

	// Intern flag name to avoid string allocation
	flagName := intern.InternBytes(nameBytes)

	// Look up flag definition
	flagDef := p.findFlag(flagName)
	if flagDef == nil {
		// Wrapper support: forward unknown flags as positional args when enabled
		if p.currentCmd != nil && p.currentCmd.wrapper != nil && p.currentCmd.wrapper.ForwardUnknown {
			// Treat the whole token as positional
			return p.parsePositionalArg(argBytes)
		}
		if p.currentCmd == nil && p.app != nil && p.app.defaultWrapper != nil && p.app.defaultWrapper.ForwardUnknown {
			return p.parsePositionalArg(argBytes)
		}
		return p.createUnknownFlagError(flagName)
	}

	// Direct parsing to typed maps to avoid interface{} boxing

	// Store parsed flag without allocation - direct to typed maps
	if hasValue {
		return p.storeFlagValue(flagName, flagDef, valueBytes, flagDef.IsGlobal())
	}
	if flagDef.RequiresValue() {
		// Value should be next argument - get it and parse directly
		if p.position+1 >= len(allArgs) {
			return &ParseError{Type: ErrorTypeInvalidValue, Message: "missing required value"}
		}

		// Advance position and get next argument
		p.position++
		nextArg := allArgs[p.position]
		valueBytes = stringToBytes(nextArg)
		return p.storeFlagValue(flagName, flagDef, valueBytes, flagDef.IsGlobal())
	}
	if flagDef.Type == FlagTypeBool {
		// Boolean flag without value - defaults to true
		return p.storeFlagValue(flagName, flagDef, trueBoolBytes, flagDef.IsGlobal())
	}
	// Non-boolean flag without value - this is an error
	return &ParseError{
		Type:    ErrorTypeMissingValue,
		Message: "flag requires a value: --" + flagName,
		Flag:    flagName,
	}
}

// parseShortFlag parses short flags (-f, -abc) with zero allocations
//
//nolint:gocognit // Handles many short-flag shapes and error paths compactly.
func (p *Parser) parseShortFlag(argBytes []byte, allArgs []string) error {
	// Skip the '-' prefix
	flagBytes := argBytes[1:]

	// Handle combined short flags (-abc = -a -b -c)
parseShort:
	for i, flagRune := range flagBytes {
		// Convert rune to flag name
		flagName := intern.InternByte(flagRune)

		// Look up flag definition
		flagDef := p.findFlag(flagName)
		if flagDef == nil {
			// Wrapper support: forward unknown short flags when enabled
			if p.currentCmd != nil && p.currentCmd.wrapper != nil && p.currentCmd.wrapper.ForwardUnknown {
				return p.parsePositionalArg(argBytes)
			}
			if p.currentCmd == nil && p.app != nil && p.app.defaultWrapper != nil &&
				p.app.defaultWrapper.ForwardUnknown {
				return p.parsePositionalArg(argBytes)
			}
			return p.createUnknownFlagError(flagName)
		}

		// Variables removed since we're using direct typed parsing

		// Direct typed parsing - moved to storeFlagValue calls below

		// Store parsed flag - use different approach based on flag type
		switch {
		case flagDef.RequiresValue():
			if i == len(flagBytes)-1 {
				// Value is next argument - get it and parse directly
				if p.position+1 >= len(allArgs) {
					return &ParseError{Type: ErrorTypeInvalidValue, Message: "missing required value"}
				}

				// Advance position and get next argument
				p.position++
				nextArg := allArgs[p.position]
				valueBytes := stringToBytes(nextArg)
				err := p.storeFlagValue(flagDef.Name, flagDef, valueBytes, flagDef.IsGlobal())
				if err != nil {
					return err
				}
				break parseShort
			}
			// Value is rest of current argument
			valueBytes := flagBytes[i+1:]
			err := p.storeFlagValue(flagDef.Name, flagDef, valueBytes, flagDef.IsGlobal())
			if err != nil {
				return err
			}
			break parseShort
		case flagDef.Type == FlagTypeBool:
			// Boolean flag without value - defaults to true
			err := p.storeFlagValue(flagDef.Name, flagDef, trueBoolBytes, flagDef.IsGlobal())
			if err != nil {
				return err
			}
		default:
			// Non-boolean flag without value - this is an error
			return &ParseError{
				Type:    ErrorTypeMissingValue,
				Message: "flag requires a value: -" + flagName,
				Flag:    flagDef.Name,
			}
		}
	}

	return nil
}

// parseCommand identifies and sets the current command
func (p *Parser) parseCommand(argBytes []byte) error {
	cmdName := intern.InternBytes(argBytes)

	// Find command in current context
	cmd := p.findCommand(cmdName)
	if cmd == nil {
		return p.createUnknownCommandError(cmdName)
	}

	p.currentCmd = cmd
	p.currentResult.Command = cmd // Update result to point to most nested command
	p.state = StateCommandFlags

	return nil
}

// parsePositionalArg handles positional arguments
func (p *Parser) parsePositionalArg(argBytes []byte) error {
	// Convert to string and store (this is where we allocate for final result)
	arg := bytesToString(argBytes)
	p.argsBuffer = append(p.argsBuffer, arg)

	return nil
}

// Utility methods for zero-allocation operations

// stringToBytes converts string to byte slice without allocation
func stringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&struct {
		string
		int
	}{s, len(s)}))
}

// bytesToString converts byte slice to string without allocation
func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// findByte finds byte in slice, returns -1 if not found
func findByte(b []byte, target byte) int {
	for i, c := range b {
		if c == target {
			return i
		}
	}
	return -1
}

// reset resets parser state for reuse without allocations
func (p *Parser) reset() {
	p.state = StateInit
	p.position = 0
	p.currentCmd = nil
	p.lastError = nil
	p.currentResult = nil

	// Reset slices without allocation
	p.flagBuffer = p.flagBuffer[:0]
	p.valueBuffer = p.valueBuffer[:0]
	p.argsBuffer = p.argsBuffer[:0]
	p.flagsBuffer = p.flagsBuffer[:0]
	p.suggestions = p.suggestions[:0]
}

// findFlag performs O(1) flag lookup in the application's flag registry.
// Uses interned strings from internal/intern package for key lookup to avoid allocations.
// Also supports O(1) short flag lookup using dedicated maps.
func (p *Parser) findFlag(name string) *Flag {
	// First check current command's flags if we're in a command context
	if p.currentCmd != nil && p.currentCmd.flags != nil {
		// Check by name first (O(1))
		if flag := p.currentCmd.flags[name]; flag != nil {
			return flag
		}

		// If name is single character, check short flag map (O(1))
		if len(name) == 1 {
			if flag := p.currentCmd.shortFlags[rune(name[0])]; flag != nil {
				return flag
			}
		}
	}

	// Then check global flags
	if p.app == nil || p.app.flags == nil {
		return nil
	}

	// Check by name first (O(1))
	if flag := p.app.flags[name]; flag != nil {
		return flag
	}

	// If name is single character, check short flag map (O(1))
	if len(name) == 1 {
		if flag := p.app.shortFlags[rune(name[0])]; flag != nil {
			return flag
		}
	}

	return nil
}

// findCommand performs O(1) command lookup in the application's command registry.
// Uses interned strings from internal/intern package for key lookup to avoid allocations.
func (p *Parser) findCommand(name string) *Command {
	// First check for subcommands if we're already in a command context
	if p.currentCmd != nil && p.currentCmd.subcommands != nil {
		if subCmd := p.currentCmd.subcommands[name]; subCmd != nil {
			return subCmd
		}
	}

	// Then check top-level commands
	if p.app == nil || p.app.commands == nil {
		return nil
	}
	return p.app.commands[name]
}

// storeFlag stores a parsed flag value in the appropriate result map.
// Global flags are stored separately from command-specific flags.
//
//nolint:gocognit,funlen // Parsing and storing across types in one place for performance.
func (p *Parser) storeFlagValue(name string, flag *Flag, valueBytes []byte, isGlobal bool) error {
	result := p.currentResult
	if result == nil {
		return &ParseError{Type: ErrorTypeInternal, Message: "no result context"}
	}

	// Parse and store directly in typed maps to avoid interface{} boxing
	switch flag.Type {
	case FlagTypeInt:
		value, err := p.parseIntBytes(valueBytes)
		if err != nil {
			return &ParseError{
				Type:    ErrorTypeInvalidValue,
				Message: "invalid integer value",
				Flag:    flag.Name,
			}
		}
		if isGlobal {
			result.GlobalIntFlags[name] = value
		} else {
			result.IntFlags[name] = value
		}

	case FlagTypeString:
		value := bytesToString(valueBytes)
		if isGlobal {
			result.GlobalStringFlags[name] = value
		} else {
			result.StringFlags[name] = value
		}

	case FlagTypeBool:
		value := p.parseBoolBytes(valueBytes)
		if isGlobal {
			result.GlobalBoolFlags[name] = value
		} else {
			result.BoolFlags[name] = value
		}

	case FlagTypeDuration:
		value, err := p.parseDurationBytes(valueBytes)
		if err != nil {
			return &ParseError{
				Type:    ErrorTypeInvalidValue,
				Message: "invalid duration value",
				Flag:    flag.Name,
			}
		}
		if isGlobal {
			result.GlobalDurationFlags[name] = value
		} else {
			result.DurationFlags[name] = value
		}

	case FlagTypeFloat:
		value, err := p.parseFloatBytes(valueBytes)
		if err != nil {
			return &ParseError{
				Type:    ErrorTypeInvalidValue,
				Message: "invalid float value",
				Flag:    flag.Name,
			}
		}
		if isGlobal {
			result.GlobalFloatFlags[name] = value
		} else {
			result.FloatFlags[name] = value
		}

	case FlagTypeEnum:
		// Parse enum value with validation
		value := bytesToString(valueBytes)
		if !p.isValidEnumValue(flag, value) {
			return &ParseError{
				Type:    ErrorTypeInvalidValue,
				Message: "invalid enum value: " + value + ", valid values: " + p.enumValuesString(flag),
				Flag:    flag.Name,
			}
		}
		if isGlobal {
			result.GlobalEnumFlags[name] = value
		} else {
			result.EnumFlags[name] = value
		}

	case FlagTypeStringSlice:
		// Parse comma-separated strings using pooled slice
		slice := p.parseStringSlice(valueBytes)
		// Store slice for cleanup and create offset
		result.stringSlices = append(result.stringSlices, slice)
		offset := pool.SliceOffset{Start: len(result.stringSlices) - 1, End: len(result.stringSlices)}
		if isGlobal {
			result.GlobalStringSliceOffsets[name] = offset
		} else {
			result.StringSliceOffsets[name] = offset
		}

	case FlagTypeIntSlice:
		// Parse comma-separated integers using pooled slice
		slice, err := p.parseIntSlice(valueBytes)
		if err != nil {
			return &ParseError{
				Type:    ErrorTypeInvalidValue,
				Message: "invalid int slice value",
				Flag:    flag.Name,
			}
		}
		// Store slice for cleanup and create offset
		result.intSlices = append(result.intSlices, slice)
		offset := pool.SliceOffset{Start: len(result.intSlices) - 1, End: len(result.intSlices)}
		if isGlobal {
			result.GlobalIntSliceOffsets[name] = offset
		} else {
			result.IntSliceOffsets[name] = offset
		}

	default:
		// Unknown flag type - return error
		return &ParseError{
			Type:    ErrorTypeInvalidFlag,
			Message: "unsupported flag type",
			Flag:    flag.Name,
		}
	}

	return nil
}

// createUnknownFlagError creates an error with smart suggestions for unknown flags.
// Uses Levenshtein distance to find the closest matching flag name.
func (p *Parser) createUnknownFlagError(name string) error {
	suggestion := p.findClosestFlag(name)

	// Pre-allocate error message to avoid allocations
	// Note: Don't embed suggestion in message - error handler will add it
	p.resetStringBuilder()
	p.appendString("unknown flag: --")
	p.appendString(name)

	// Reuse pre-allocated error to avoid allocations, but copy string to avoid aliasing
	p.reusableError.Type = ErrorTypeUnknownFlag
	p.reusableError.Message = string(append([]byte(nil), p.valueBuffer...))
	p.reusableError.Flag = name
	p.reusableError.Suggestion = suggestion
	p.reusableError.CurrentCommand = p.currentCmd
	return p.reusableError
}

// createUnknownCommandError creates an error with smart suggestions for unknown commands.
// Uses Levenshtein distance to find the closest matching command name.
func (p *Parser) createUnknownCommandError(name string) error {
	suggestion := p.findClosestCommand(name)

	// Pre-allocate error message to avoid allocations
	// Note: Don't embed suggestion in message - error handler will add it
	p.resetStringBuilder()
	p.appendString("unknown command: ")
	p.appendString(name)

	// Reuse pre-allocated error to avoid allocations, but copy string to avoid aliasing
	p.reusableError.Type = ErrorTypeUnknownCommand
	p.reusableError.Message = string(append([]byte(nil), p.valueBuffer...))
	p.reusableError.Command = name
	p.reusableError.Suggestion = suggestion
	p.reusableError.CurrentCommand = p.currentCmd
	return p.reusableError
}

// getResult returns a ParseResult from the pre-allocated reusable result.
// Uses the same result object to achieve zero allocations.
func (p *Parser) getResult() *ParseResult {
	p.clearResult(p.reusableResult)
	return p.reusableResult
}

// finalize completes parsing and returns the final result.
// Copies collected arguments and sets the current command.
func (p *Parser) finalize() (*ParseResult, error) {
	if p.currentResult == nil {
		p.currentResult = p.getResult()
	}

	result := p.currentResult
	result.Command = p.currentCmd

	// Process positional arguments
	if err := p.processPositionalArgs(result); err != nil {
		return nil, err
	}

	// Apply default values for flags that weren't provided
	p.applyDefaults(result)

	// Validate flag groups
	if err := p.validateFlagGroups(result); err != nil {
		return nil, err
	}

	return result, nil
}

// processPositionalArgs processes positional arguments after flag parsing is complete.
// This handles: type conversion, required validation, variadic args, RestArgs, and defaults.
// Zero-allocation: Uses existing p.argsBuffer and stores directly in typed maps.
//
//nolint:gocognit // Handles all arg types and validation in one place for performance
func (p *Parser) processPositionalArgs(result *ParseResult) error {
	// Get the argument definitions for the current context
	var args []*Arg
	var hasRestArgs bool

	if result.Command != nil {
		args = result.Command.args
		hasRestArgs = result.Command.hasRestArgs
	} else if p.app != nil {
		args = p.app.args
		hasRestArgs = p.app.hasRestArgs
	}

	// Fast path: no args defined and no RestArgs
	if len(args) == 0 && !hasRestArgs {
		// Store raw args for ctx.Arg(index) access
		result.Args = append(result.Args[:0], p.argsBuffer...)
		return nil
	}

	// Check if help flag is set - skip required validation if help is requested
	helpRequested := result.MustGetBool("help", false)
	if !helpRequested {
		// Also check global help flag
		helpRequested = result.MustGetGlobalBool("help", false)
	}

	// Check for RestArgs mode: collect all remaining args
	if hasRestArgs {
		result.RestArgs = append(result.RestArgs[:0], p.argsBuffer...)
		result.Args = append(result.Args[:0], p.argsBuffer...)
		return nil
	}

	// Find if we have a variadic arg (must be last)
	var variadicArg *Arg
	var variadicPosition int
	for i, arg := range args {
		if arg.Variadic {
			variadicArg = arg
			variadicPosition = i
			break
		}
	}

	// Validate: variadic must be last if present
	if variadicArg != nil && variadicPosition != len(args)-1 {
		return &ParseError{
			Type:    ErrorTypeInvalidArgument,
			Message: "variadic argument must be the last positional argument",
		}
	}

	// Process each declared positional argument
	numProvidedArgs := len(p.argsBuffer)
	argIndex := 0 // Index into p.argsBuffer

	for _, argDef := range args {
		// Check if we're at the variadic arg
		if argDef.Variadic {
			// Collect all remaining args into variadic slice
			remaining := p.argsBuffer[argIndex:]

			if len(remaining) == 0 && argDef.Required && !helpRequested {
				return &ParseError{
					Type:    ErrorTypeInvalidArgument,
					Message: "missing required variadic argument: " + argDef.Name,
				}
			}

			// Process variadic based on type
			if err := p.processVariadicArg(result, argDef, remaining); err != nil {
				return err
			}

			// All args consumed
			break
		}

		// Regular (non-variadic) arg
		if argIndex >= numProvidedArgs {
			// No more args provided
			if argDef.Required && !helpRequested {
				return &ParseError{
					Type:    ErrorTypeInvalidArgument,
					Message: "missing required argument: " + argDef.Name,
				}
			}
			// Apply default for optional arg
			if err := p.applyArgDefault(result, argDef); err != nil {
				return err
			}
			continue
		}

		// Get the arg value from buffer
		argValue := p.argsBuffer[argIndex]
		argIndex++

		// Parse and store based on type
		if err := p.storeArgValue(result, argDef, argValue); err != nil {
			return err
		}
	}

	// Store raw args for ctx.Arg(index) access (zero-alloc copy)
	result.Args = append(result.Args[:0], p.argsBuffer...)

	return nil
}

// storeArgValue parses and stores a single positional argument value
// Zero-allocation: Stores directly in typed maps without interface{} boxing
func (p *Parser) storeArgValue(result *ParseResult, argDef *Arg, value string) error {
	switch argDef.Type {
	case ArgTypeString:
		result.ArgStrings[argDef.Name] = value

	case ArgTypeInt:
		intValue, err := p.parseIntBytes(stringToBytes(value))
		if err != nil {
			return &ParseError{
				Type:    ErrorTypeInvalidArgument,
				Message: "invalid integer value for argument '" + argDef.Name + "': " + value,
			}
		}
		result.ArgInts[argDef.Name] = intValue

	case ArgTypeBool:
		boolValue := p.parseBoolBytes(stringToBytes(value))
		result.ArgBools[argDef.Name] = boolValue

	case ArgTypeDuration:
		durationValue, err := p.parseDurationBytes(stringToBytes(value))
		if err != nil {
			return &ParseError{
				Type:    ErrorTypeInvalidArgument,
				Message: "invalid duration value for argument '" + argDef.Name + "': " + value,
			}
		}
		result.ArgDurations[argDef.Name] = durationValue

	case ArgTypeFloat:
		floatValue, err := p.parseFloatBytes(stringToBytes(value))
		if err != nil {
			return &ParseError{
				Type:    ErrorTypeInvalidArgument,
				Message: "invalid float value for argument '" + argDef.Name + "': " + value,
			}
		}
		result.ArgFloats[argDef.Name] = floatValue

	case ArgTypeStringSlice, ArgTypeIntSlice:
		// Slice types should be handled by processVariadicArg, not storeArgValue
		return &ParseError{
			Type:    ErrorTypeInvalidArgument,
			Message: "slice argument types must be variadic: " + argDef.Name,
		}

	default:
		return &ParseError{
			Type:    ErrorTypeInvalidArgument,
			Message: "unsupported argument type: " + string(argDef.Type),
		}
	}

	return nil
}

// processVariadicArg processes a variadic argument (StringSlice or IntSlice)
// Zero-allocation: Uses pooled slices
func (p *Parser) processVariadicArg(result *ParseResult, argDef *Arg, values []string) error {
	switch argDef.Type {
	case ArgTypeStringSlice:
		// Use pooled slice
		slice := pool.GetStringSlice()
		*slice = append(*slice, values...)

		// Store slice and create offset
		result.stringSlices = append(result.stringSlices, slice)
		offset := pool.SliceOffset{Start: len(result.stringSlices) - 1, End: len(result.stringSlices)}
		result.ArgStringSlices[argDef.Name] = offset

	case ArgTypeIntSlice:
		// Parse each value as int
		slice := pool.GetIntSlice()
		for _, valueStr := range values {
			intValue, err := p.parseIntBytes(stringToBytes(valueStr))
			if err != nil {
				return &ParseError{
					Type:    ErrorTypeInvalidArgument,
					Message: "invalid integer value in variadic argument '" + argDef.Name + "': " + valueStr,
				}
			}
			*slice = append(*slice, intValue)
		}

		// Store slice and create offset
		result.intSlices = append(result.intSlices, slice)
		offset := pool.SliceOffset{Start: len(result.intSlices) - 1, End: len(result.intSlices)}
		result.ArgIntSlices[argDef.Name] = offset

	case ArgTypeString, ArgTypeBool, ArgTypeInt, ArgTypeDuration, ArgTypeFloat:
		// Non-slice types should not be processed as variadic
		return &ParseError{
			Type:    ErrorTypeInvalidArgument,
			Message: "non-slice argument type cannot be variadic: " + argDef.Name,
		}

	default:
		return &ParseError{
			Type:    ErrorTypeInvalidArgument,
			Message: "variadic arguments only support StringSlice and IntSlice types",
		}
	}

	return nil
}

// applyArgDefault applies the default value for an optional positional argument
// Zero-allocation: Stores directly in typed maps
func (p *Parser) applyArgDefault(result *ParseResult, argDef *Arg) error {
	switch argDef.Type {
	case ArgTypeString:
		if argDef.DefaultString != "" {
			result.ArgStrings[argDef.Name] = argDef.DefaultString
		}

	case ArgTypeInt:
		if argDef.DefaultInt != 0 {
			result.ArgInts[argDef.Name] = argDef.DefaultInt
		}

	case ArgTypeBool:
		result.ArgBools[argDef.Name] = argDef.DefaultBool

	case ArgTypeDuration:
		if argDef.DefaultDuration != 0 {
			result.ArgDurations[argDef.Name] = argDef.DefaultDuration
		}

	case ArgTypeFloat:
		if argDef.DefaultFloat != 0.0 {
			result.ArgFloats[argDef.Name] = argDef.DefaultFloat
		}

	case ArgTypeStringSlice:
		if len(argDef.DefaultStringSlice) > 0 {
			slice := pool.GetStringSlice()
			*slice = append(*slice, argDef.DefaultStringSlice...)
			result.stringSlices = append(result.stringSlices, slice)
			offset := pool.SliceOffset{Start: len(result.stringSlices) - 1, End: len(result.stringSlices)}
			result.ArgStringSlices[argDef.Name] = offset
		}

	case ArgTypeIntSlice:
		if len(argDef.DefaultIntSlice) > 0 {
			slice := pool.GetIntSlice()
			*slice = append(*slice, argDef.DefaultIntSlice...)
			result.intSlices = append(result.intSlices, slice)
			offset := pool.SliceOffset{Start: len(result.intSlices) - 1, End: len(result.intSlices)}
			result.ArgIntSlices[argDef.Name] = offset
		}
	}

	return nil
}

// applyDefaults applies default values for flags that weren't explicitly provided
func (p *Parser) applyDefaults(result *ParseResult) {
	// Apply defaults for app-level flags
	for name, flag := range p.app.flags {
		if flag.Global {
			p.applyGlobalDefault(result, name, flag)
		} else {
			p.applyFlagDefault(result, name, flag)
		}
	}

	// Apply defaults for command-specific flags if we have a command
	if result.Command != nil {
		for name, flag := range result.Command.flags {
			if !flag.Global {
				p.applyFlagDefault(result, name, flag)
			}
		}
	}
}

// applyFlagDefault applies environment variable or default value for a regular flag if not already set
//
//nolint:dupl,gocognit,gocyclo,cyclop // Similar to applyGlobalDefault but for non-global flags
func (p *Parser) applyFlagDefault(result *ParseResult, name string, flag *Flag) {
	switch flag.Type {
	case FlagTypeString:
		if _, exists := result.StringFlags[name]; !exists {
			// Check environment variables first (precedence order)
			if envValue := p.getEnvValue(flag.EnvVars); envValue != "" {
				result.StringFlags[name] = envValue
			} else if flag.DefaultString != "" {
				result.StringFlags[name] = flag.DefaultString
			}
		}
	case FlagTypeInt:
		if _, exists := result.IntFlags[name]; !exists {
			// Check environment variables first (precedence order)
			if envValue := p.getEnvValue(flag.EnvVars); envValue != "" {
				if intValue, err := p.parseIntValue(envValue); err == nil {
					result.IntFlags[name] = intValue
				}
			} else if flag.DefaultInt != 0 {
				result.IntFlags[name] = flag.DefaultInt
			}
		}
	case FlagTypeBool:
		if _, exists := result.BoolFlags[name]; !exists {
			// Check environment variables first (precedence order)
			if envValue := p.getEnvValue(flag.EnvVars); envValue != "" {
				boolValue := p.parseBoolValue(envValue)
				result.BoolFlags[name] = boolValue
			} else {
				result.BoolFlags[name] = flag.DefaultBool
			}
		}
	case FlagTypeDuration:
		if _, exists := result.DurationFlags[name]; !exists {
			// Check environment variables first (precedence order)
			if envValue := p.getEnvValue(flag.EnvVars); envValue != "" {
				if durationValue, err := p.parseDurationValue(envValue); err == nil {
					result.DurationFlags[name] = durationValue
				}
			} else if flag.DefaultDuration != 0 {
				result.DurationFlags[name] = flag.DefaultDuration
			}
		}
	case FlagTypeFloat:
		if _, exists := result.FloatFlags[name]; !exists {
			// Check environment variables first (precedence order)
			if envValue := p.getEnvValue(flag.EnvVars); envValue != "" {
				if floatValue, err := p.parseFloatValue(envValue); err == nil {
					result.FloatFlags[name] = floatValue
				}
			} else if flag.DefaultFloat != 0.0 {
				result.FloatFlags[name] = flag.DefaultFloat
			}
		}
	case FlagTypeEnum:
		if _, exists := result.EnumFlags[name]; !exists {
			// Check environment variables first (precedence order)
			if envValue := p.getEnvValue(flag.EnvVars); envValue != "" {
				// Validate enum value
				if p.isValidEnumValue(flag, envValue) {
					result.EnumFlags[name] = envValue
				}
			} else if flag.DefaultEnum != "" {
				result.EnumFlags[name] = flag.DefaultEnum
			}
		}
	case FlagTypeStringSlice:
		if _, exists := result.StringSliceOffsets[name]; !exists {
			if envValue := p.getEnvValue(flag.EnvVars); envValue != "" {
				slice := p.parseStringSlice([]byte(envValue))
				result.stringSlices = append(result.stringSlices, slice)
				offset := pool.SliceOffset{Start: len(result.stringSlices) - 1, End: len(result.stringSlices)}
				result.StringSliceOffsets[name] = offset
			} else if len(flag.DefaultStringSlice) > 0 {
				slice := pool.GetStringSlice()
				*slice = append(*slice, flag.DefaultStringSlice...)
				result.stringSlices = append(result.stringSlices, slice)
				offset := pool.SliceOffset{Start: len(result.stringSlices) - 1, End: len(result.stringSlices)}
				result.StringSliceOffsets[name] = offset
			}
		}
	case FlagTypeIntSlice:
		if _, exists := result.IntSliceOffsets[name]; !exists {
			if envValue := p.getEnvValue(flag.EnvVars); envValue != "" {
				slice, err := p.parseIntSlice([]byte(envValue))
				if err == nil {
					result.intSlices = append(result.intSlices, slice)
					offset := pool.SliceOffset{Start: len(result.intSlices) - 1, End: len(result.intSlices)}
					result.IntSliceOffsets[name] = offset
				}
			} else if len(flag.DefaultIntSlice) > 0 {
				slice := pool.GetIntSlice()
				*slice = append(*slice, flag.DefaultIntSlice...)
				result.intSlices = append(result.intSlices, slice)
				offset := pool.SliceOffset{Start: len(result.intSlices) - 1, End: len(result.intSlices)}
				result.IntSliceOffsets[name] = offset
			}
		}
	}
}

// applyGlobalDefault applies environment variable or default value for a global flag if not already set
//
//nolint:dupl,gocognit,gocyclo,cyclop // Similar to applyFlagDefault but for global flags
func (p *Parser) applyGlobalDefault(result *ParseResult, name string, flag *Flag) {
	switch flag.Type {
	case FlagTypeString:
		if _, exists := result.GlobalStringFlags[name]; !exists {
			// Check environment variables first (precedence order)
			if envValue := p.getEnvValue(flag.EnvVars); envValue != "" {
				result.GlobalStringFlags[name] = envValue
			} else if flag.DefaultString != "" {
				result.GlobalStringFlags[name] = flag.DefaultString
			}
		}
	case FlagTypeInt:
		if _, exists := result.GlobalIntFlags[name]; !exists {
			// Check environment variables first (precedence order)
			if envValue := p.getEnvValue(flag.EnvVars); envValue != "" {
				if intValue, err := p.parseIntValue(envValue); err == nil {
					result.GlobalIntFlags[name] = intValue
				}
			} else if flag.DefaultInt != 0 {
				result.GlobalIntFlags[name] = flag.DefaultInt
			}
		}
	case FlagTypeBool:
		if _, exists := result.GlobalBoolFlags[name]; !exists {
			// Check environment variables first (precedence order)
			if envValue := p.getEnvValue(flag.EnvVars); envValue != "" {
				boolValue := p.parseBoolValue(envValue)
				result.GlobalBoolFlags[name] = boolValue
			} else {
				result.GlobalBoolFlags[name] = flag.DefaultBool
			}
		}
	case FlagTypeDuration:
		if _, exists := result.GlobalDurationFlags[name]; !exists {
			// Check environment variables first (precedence order)
			if envValue := p.getEnvValue(flag.EnvVars); envValue != "" {
				if durationValue, err := p.parseDurationValue(envValue); err == nil {
					result.GlobalDurationFlags[name] = durationValue
				}
			} else if flag.DefaultDuration != 0 {
				result.GlobalDurationFlags[name] = flag.DefaultDuration
			}
		}
	case FlagTypeFloat:
		if _, exists := result.GlobalFloatFlags[name]; !exists {
			// Check environment variables first (precedence order)
			if envValue := p.getEnvValue(flag.EnvVars); envValue != "" {
				if floatValue, err := p.parseFloatValue(envValue); err == nil {
					result.GlobalFloatFlags[name] = floatValue
				}
			} else if flag.DefaultFloat != 0.0 {
				result.GlobalFloatFlags[name] = flag.DefaultFloat
			}
		}
	case FlagTypeEnum:
		if _, exists := result.GlobalEnumFlags[name]; !exists {
			// Check environment variables first (precedence order)
			if envValue := p.getEnvValue(flag.EnvVars); envValue != "" {
				// Validate enum value
				if p.isValidEnumValue(flag, envValue) {
					result.GlobalEnumFlags[name] = envValue
				}
			} else if flag.DefaultEnum != "" {
				result.GlobalEnumFlags[name] = flag.DefaultEnum
			}
		}
	case FlagTypeStringSlice:
		if _, exists := result.GlobalStringSliceOffsets[name]; !exists {
			if envValue := p.getEnvValue(flag.EnvVars); envValue != "" {
				slice := p.parseStringSlice([]byte(envValue))
				result.stringSlices = append(result.stringSlices, slice)
				offset := pool.SliceOffset{Start: len(result.stringSlices) - 1, End: len(result.stringSlices)}
				result.GlobalStringSliceOffsets[name] = offset
			} else if len(flag.DefaultStringSlice) > 0 {
				slice := pool.GetStringSlice()
				*slice = append(*slice, flag.DefaultStringSlice...)
				result.stringSlices = append(result.stringSlices, slice)
				offset := pool.SliceOffset{Start: len(result.stringSlices) - 1, End: len(result.stringSlices)}
				result.GlobalStringSliceOffsets[name] = offset
			}
		}
	case FlagTypeIntSlice:
		if _, exists := result.GlobalIntSliceOffsets[name]; !exists {
			if envValue := p.getEnvValue(flag.EnvVars); envValue != "" {
				slice, err := p.parseIntSlice([]byte(envValue))
				if err == nil {
					result.intSlices = append(result.intSlices, slice)
					offset := pool.SliceOffset{Start: len(result.intSlices) - 1, End: len(result.intSlices)}
					result.GlobalIntSliceOffsets[name] = offset
				}
			} else if len(flag.DefaultIntSlice) > 0 {
				slice := pool.GetIntSlice()
				*slice = append(*slice, flag.DefaultIntSlice...)
				result.intSlices = append(result.intSlices, slice)
				offset := pool.SliceOffset{Start: len(result.intSlices) - 1, End: len(result.intSlices)}
				result.GlobalIntSliceOffsets[name] = offset
			}
		}
	}
}

// Utility methods for zero-allocation operations

// clearResult resets a ParseResult for reuse without allocating new maps.
func (p *Parser) clearResult(result *ParseResult) {
	// Return pooled slices before clearing
	for _, slice := range result.stringSlices {
		if slice != nil {
			pool.PutStringSlice(slice)
		}
	}
	for _, slice := range result.intSlices {
		if slice != nil {
			pool.PutIntSlice(slice)
		}
	}
	result.stringSlices = result.stringSlices[:0]
	result.intSlices = result.intSlices[:0]

	// Use the pool's reset functionality
	if result.ParseResult != nil {
		pool.PutParseResult(result.ParseResult)
		result.ParseResult = pool.GetParseResult()
	}

	result.Args = result.Args[:0]
	result.Command = nil
}

// parseBoolBytes parses boolean value from byte slice without allocation.
func (p *Parser) parseBoolBytes(b []byte) bool {
	if len(b) == 0 {
		return false
	}

	// Check common true values
	if len(b) == 1 && (b[0] == '1' || b[0] == 't' || b[0] == 'T') {
		return true
	}

	if len(b) == 4 &&
		(b[0] == 't' || b[0] == 'T') &&
		(b[1] == 'r' || b[1] == 'R') &&
		(b[2] == 'u' || b[2] == 'U') &&
		(b[3] == 'e' || b[3] == 'E') {
		return true
	}

	return false
}

// parseIntBytes transparently parses decimal and hex integers using ASCII math.
// Supports: 123, -456, 0xFF, 0x1A2B, etc. Zero allocations.
func (p *Parser) parseIntBytes(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "empty integer"}
	}

	negative := false
	start := 0

	// Handle sign
	switch b[0] {
	case '-':
		negative = true
		start = 1
		if len(b) == 1 {
			return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "invalid integer"}
		}
	case '+':
		start = 1
		if len(b) == 1 {
			return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "invalid integer"}
		}
	}

	// Check for hex prefix (0x or 0X) - transparent to user
	remaining := b[start:]
	if len(remaining) > 2 && remaining[0] == '0' && (remaining[1] == 'x' || remaining[1] == 'X') {
		result, err := p.parseHexBytes(remaining[2:])
		if err != nil {
			return 0, err
		}
		if negative {
			result = -result
		}
		return result, nil
	}

	// Default to decimal parsing with ASCII math
	result, err := p.parseDecimalBytes(remaining)
	if err != nil {
		return 0, err
	}

	if negative {
		result = -result
	}

	return result, nil
}

// parseDecimalBytes parses decimal using direct ASCII math: '8' - '0' = 8
func (p *Parser) parseDecimalBytes(b []byte) (int, error) {
	result := 0

	for i := 0; i < len(b); i++ {
		c := b[i]

		// Validate it's a digit
		if c < '0' || c > '9' {
			return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "invalid decimal character"}
		}

		// ASCII math: '8' - '0' = 8
		digit := int(c - '0')

		// Check for overflow before multiplication (platform-agnostic)
		if result > (math.MaxInt-digit)/10 {
			return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "integer overflow"}
		}

		// Build number: "123" -> 1*10 + 2 -> 12*10 + 3 = 123
		result = result*10 + digit
	}

	return result, nil
}

// parseHexBytes parses hexadecimal using ASCII math: 'A' - 'A' + 10 = 10
func (p *Parser) parseHexBytes(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "empty hex value"}
	}

	result := 0

	for i := 0; i < len(b); i++ {
		c := b[i]
		var digit int

		// ASCII math for hex digits
		switch {
		case c >= '0' && c <= '9':
			digit = int(c - '0') // '0' = 0, '9' = 9
		case c >= 'A' && c <= 'F':
			digit = int(c - 'A' + 10) // 'A' = 10, 'F' = 15
		case c >= 'a' && c <= 'f':
			digit = int(c - 'a' + 10) // 'a' = 10, 'f' = 15
		default:
			return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "invalid hex character"}
		}

		// Check for overflow (hex can get large quickly)
		if result > (math.MaxInt-digit)/16 {
			return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "hex integer overflow"}
		}

		// Build hex number: "A1" -> 10*16 + 1 = 161
		result = result*16 + digit
	}

	return result, nil
}

// parseDurationBytes parses time.Duration from bytes using zero allocations.
// Supports: "00:30" (30s), "01:30:15" (1h30m15s), "3s", "1h30m", "3 sec", "1d", "1w", "1M", "1Y"
func (p *Parser) parseDurationBytes(b []byte) (time.Duration, error) {
	if len(b) == 0 {
		return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "empty duration"}
	}

	// 1. Check for colon format first: "MM:SS" or "HH:MM:SS"
	if colonCount := countByte(b, ':'); colonCount > 0 {
		return p.parseColonDuration(b, colonCount)
	}

	// 2. Check for extended units: "1d", "1w", "1M", "1Y"
	if duration, ok := p.parseExtendedDuration(b); ok {
		return duration, nil
	}

	// 3. Parse standard Go duration format manually: "1h30m15s"
	return p.parseStandardDuration(b)
}

// parseFloatBytes parses float64 from bytes using zero allocations
func (p *Parser) parseFloatBytes(b []byte) (float64, error) {
	// Simple implementation for common cases like "3.14"
	result := 0.0
	decimal := 0.0
	decimalPlace := 1.0
	negative := false
	afterDecimal := false

	for i := 0; i < len(b); i++ {
		c := b[i]

		if c == '-' && i == 0 {
			negative = true
			continue
		}

		if c == '.' {
			if afterDecimal {
				return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "multiple decimal points"}
			}
			afterDecimal = true
			continue
		}

		if c < '0' || c > '9' {
			return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "invalid float character"}
		}

		digit := float64(c - '0')

		if afterDecimal {
			decimalPlace *= 10.0
			decimal += digit / decimalPlace
		} else {
			result = result*10.0 + digit
		}
	}

	result += decimal
	if negative {
		result = -result
	}

	return result, nil
}

// countByte counts occurrences of a byte in a slice
func countByte(b []byte, target byte) int {
	count := 0
	for i := 0; i < len(b); i++ {
		if b[i] == target {
			count++
		}
	}
	return count
}

// parseColonDuration parses "MM:SS" or "HH:MM:SS" format
func (p *Parser) parseColonDuration(b []byte, colonCount int) (time.Duration, error) {
	switch colonCount {
	case 1:
		// Format: "MM:SS" - minutes:seconds
		colonPos := findByte(b, ':')
		if colonPos == -1 {
			return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "invalid colon duration"}
		}

		minutes, err := p.parseDecimalBytes(b[:colonPos])
		if err != nil {
			return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "invalid minutes"}
		}

		seconds, err := p.parseDecimalBytes(b[colonPos+1:])
		if err != nil {
			return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "invalid seconds"}
		}

		return time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second, nil
	case 2:
		// Format: "HH:MM:SS" - hours:minutes:seconds
		firstColon := findByte(b, ':')
		if firstColon == -1 {
			return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "invalid colon duration"}
		}

		secondColon := findByte(b[firstColon+1:], ':')
		if secondColon == -1 {
			return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "invalid colon duration"}
		}
		secondColon += firstColon + 1 // Adjust for offset

		hours, err := p.parseDecimalBytes(b[:firstColon])
		if err != nil {
			return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "invalid hours"}
		}

		minutes, err := p.parseDecimalBytes(b[firstColon+1 : secondColon])
		if err != nil {
			return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "invalid minutes"}
		}

		seconds, err := p.parseDecimalBytes(b[secondColon+1:])
		if err != nil {
			return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "invalid seconds"}
		}

		return time.Duration(
			hours,
		)*time.Hour + time.Duration(
			minutes,
		)*time.Minute + time.Duration(
			seconds,
		)*time.Second, nil
	}

	return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "too many colons"}
}

// parseExtendedDuration parses "1d", "1w", "1M", "1Y" format
func (p *Parser) parseExtendedDuration(b []byte) (time.Duration, bool) {
	if len(b) < 2 {
		return 0, false
	}

	// Find the last character (unit)
	lastChar := b[len(b)-1]
	unit := lastChar
	if unit >= 'A' && unit <= 'Z' {
		unit += 32 // Convert to lowercase
	}

	var multiplier time.Duration
	switch unit {
	case 'd':
		multiplier = 24 * time.Hour // 1 day = 24 hours
	case 'w':
		multiplier = 7 * 24 * time.Hour // 1 week = 7 days
	case 'm':
		// Check if it's 'M' (month) vs 'm' (minute)
		if lastChar == 'M' {
			multiplier = 30 * 24 * time.Hour // 1 month = 30 days (assumption)
		} else {
			return 0, false // Regular minute - handled by standard parsing
		}
	case 'y':
		multiplier = 365 * 24 * time.Hour // 1 year = 365 days (assumption)
	default:
		return 0, false
	}

	// Parse the number part
	numberBytes := b[:len(b)-1]
	number, err := p.parseDecimalBytes(numberBytes)
	if err != nil {
		return 0, false
	}

	return time.Duration(number) * multiplier, true
}

// parseStandardDuration parses "1h30m15s" and "3 sec" formats manually
func (p *Parser) parseStandardDuration(b []byte) (time.Duration, error) {
	var result time.Duration
	var currentNumber int
	var hasNumber bool
	i := 0

	for i < len(b) {
		// Skip whitespace
		if b[i] == ' ' || b[i] == '\t' {
			i++
			continue
		}

		// Parse number
		if b[i] >= '0' && b[i] <= '9' {
			currentNumber = 0
			hasNumber = true
			for i < len(b) && b[i] >= '0' && b[i] <= '9' {
				currentNumber = currentNumber*10 + int(b[i]-'0')
				i++
			}
			continue
		}

		// Parse unit
		if hasNumber {
			unit, consumed := p.parseTimeUnit(b[i:])
			if consumed == 0 {
				return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "invalid duration unit"}
			}

			result += time.Duration(currentNumber) * unit
			i += consumed
			hasNumber = false
			currentNumber = 0
		} else {
			return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "number expected before unit"}
		}
	}

	if hasNumber {
		return 0, &ParseError{Type: ErrorTypeInvalidValue, Message: "missing unit after number"}
	}

	return result, nil
}

// parseTimeUnit parses time unit from bytes and returns the duration and bytes consumed
//
//nolint:gocognit,gocyclo,cyclop // Compact table-driven-ish branching for unit parsing.
func (p *Parser) parseTimeUnit(b []byte) (time.Duration, int) {
	if len(b) == 0 {
		return 0, 0
	}

	// Convert first char to lowercase for comparison
	firstChar := b[0]
	if firstChar >= 'A' && firstChar <= 'Z' {
		firstChar += 32
	}

	switch firstChar {
	case 'n':
		if len(b) >= 2 && (b[1] == 's' || b[1] == 'S') {
			return time.Nanosecond, 2
		}
	case 'u':
		if len(b) >= 2 && (b[1] == 's' || b[1] == 'S') {
			return time.Microsecond, 2
		}
	case 0xce: // μ in UTF-8 starts with 0xce
		if len(b) >= 3 && b[1] == 0xbc && (b[2] == 's' || b[2] == 'S') {
			return time.Microsecond, 3 // "μs"
		}
	case 'm':
		if len(b) >= 2 && (b[1] == 's' || b[1] == 'S') {
			return time.Millisecond, 2
		}
		// Check for "min", "minute", "minutes"
		if len(b) >= 3 && (b[1] == 'i' || b[1] == 'I') && (b[2] == 'n' || b[2] == 'N') {
			if len(b) >= 6 && matchesWord(b[3:], "ute") {
				if len(b) >= 7 && (b[6] == 's' || b[6] == 'S') {
					return time.Minute, 7 // "minutes"
				}
				return time.Minute, 6 // "minute"
			}
			return time.Minute, 3 // "min"
		}
		return time.Minute, 1 // "m"
	case 's':
		// Check for "sec", "second", "seconds"
		if len(b) >= 3 && (b[1] == 'e' || b[1] == 'E') && (b[2] == 'c' || b[2] == 'C') {
			if len(b) >= 6 && matchesWord(b[3:], "ond") {
				if len(b) >= 7 && (b[6] == 's' || b[6] == 'S') {
					return time.Second, 7 // "seconds"
				}
				return time.Second, 6 // "second"
			}
			return time.Second, 3 // "sec"
		}
		return time.Second, 1 // "s"
	case 'h':
		// Check for "hour", "hours"
		if len(b) >= 4 && matchesWord(b[1:], "our") {
			if len(b) >= 5 && (b[4] == 's' || b[4] == 'S') {
				return time.Hour, 5 // "hours"
			}
			return time.Hour, 4 // "hour"
		}
		return time.Hour, 1 // "h"
	}

	return 0, 0
}

// matchesWord checks if bytes match a word (case insensitive)
func matchesWord(b []byte, word string) bool {
	if len(b) < len(word) {
		return false
	}
	for i := 0; i < len(word); i++ {
		char := b[i]
		if char >= 'A' && char <= 'Z' {
			char += 32
		}
		if char != word[i] {
			return false
		}
	}
	return true
}

// trimSpaceBytes trims leading and trailing whitespace from bytes
func trimSpaceBytes(b []byte) []byte {
	// Trim leading whitespace
	start := 0
	for start < len(b) && (b[start] == ' ' || b[start] == '\t' || b[start] == '\n' || b[start] == '\r') {
		start++
	}

	// Trim trailing whitespace
	end := len(b)
	for end > start && (b[end-1] == ' ' || b[end-1] == '\t' || b[end-1] == '\n' || b[end-1] == '\r') {
		end--
	}

	return b[start:end]
}

// parseStringSlice parses comma-separated strings using pooled slice
// Note: No error conditions for strings; signature returns only the slice.
func (p *Parser) parseStringSlice(b []byte) *[]string {
	slice := pool.GetStringSlice()

	if len(b) == 0 {
		return slice
	}

	start := 0
	for i := 0; i <= len(b); i++ {
		if i == len(b) || b[i] == ',' {
			// Extract substring from start to i
			segment := b[start:i]

			// Trim whitespace
			segment = trimSpaceBytes(segment)

			// Convert to string and append to slice
			if len(segment) > 0 {
				*slice = append(*slice, bytesToString(segment))
			}
			start = i + 1
		}
	}

	return slice
}

// parseIntSlice parses comma-separated integers using pooled slice
func (p *Parser) parseIntSlice(b []byte) (*[]int, error) {
	slice := pool.GetIntSlice()

	if len(b) == 0 {
		return slice, nil
	}

	start := 0
	for i := 0; i <= len(b); i++ {
		if i == len(b) || b[i] == ',' {
			// Extract substring from start to i
			segment := b[start:i]

			// Trim whitespace
			segment = trimSpaceBytes(segment)

			if len(segment) > 0 {
				// Parse using our existing zero-allocation int parser
				value, err := p.parseIntBytes(segment)
				if err != nil {
					return nil, err
				}
				*slice = append(*slice, value)
			}
			start = i + 1
		}
	}

	return slice, nil
}

// isValidEnumValue checks if a value is valid for an enum flag
func (p *Parser) isValidEnumValue(flag *Flag, value string) bool {
	if flag == nil || flag.Type != FlagTypeEnum {
		return false
	}

	return slices.Contains(flag.EnumValues, value)
}

// enumValuesString returns a comma-separated string of valid enum values
func (p *Parser) enumValuesString(flag *Flag) string {
	if flag == nil || flag.Type != FlagTypeEnum || len(flag.EnumValues) == 0 {
		return ""
	}

	// Use string builder for efficiency
	p.resetStringBuilder()
	for i, value := range flag.EnumValues {
		if i > 0 {
			p.appendString(", ")
		}
		p.appendString(value)
	}
	return p.getBuiltString()
}

// findClosestFlag finds the closest matching flag name using Levenshtein distance.
func (p *Parser) findClosestFlag(name string) string {
	if p.app == nil || p.app.flags == nil {
		return ""
	}

	bestMatch := ""
	bestDistance := 3 // Only suggest if distance <= 2

	for flagName := range p.app.flags {
		distance := p.levenshteinDistance(name, flagName)
		if distance < bestDistance {
			bestDistance = distance
			bestMatch = flagName
		}
	}

	return bestMatch
}

// findClosestCommand finds the closest matching command name using Levenshtein distance.
func (p *Parser) findClosestCommand(name string) string {
	if p.app == nil {
		return ""
	}

	bestMatch := ""
	bestDistance := 3 // Only suggest if distance <= 2
	// Prefer subcommands of the current command
	if p.currentCmd != nil && p.currentCmd.subcommands != nil {
		for cmdName := range p.currentCmd.subcommands {
			distance := p.levenshteinDistance(name, cmdName)
			if distance < bestDistance {
				bestDistance = distance
				bestMatch = cmdName
			}
		}
	}
	// Fall back to top-level commands
	for cmdName := range p.app.commands {
		distance := p.levenshteinDistance(name, cmdName)
		if distance < bestDistance {
			bestDistance = distance
			bestMatch = cmdName
		}
	}

	return bestMatch
}

// levenshteinDistance calculates edit distance between two strings.
// Uses a space-optimized algorithm with O(min(m,n)) space complexity.
func (p *Parser) levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Ensure a is the shorter string for space efficiency
	if len(a) > len(b) {
		a, b = b, a
	}

	// Use reusable buffer to avoid allocations
	needed := len(a) + 1
	if len(p.levenshteinBuffer) < needed {
		p.levenshteinBuffer = make([]int, needed*2) // Grow with some headroom
	}
	row := p.levenshteinBuffer[:needed]

	for i := range row {
		row[i] = i
	}

	for i := 1; i <= len(b); i++ {
		prev := row[0]
		row[0] = i

		for j := 1; j <= len(a); j++ {
			current := row[j]
			cost := 0
			if a[j-1] != b[i-1] {
				cost = 1
			}

			row[j] = min3(
				row[j-1]+1, // insertion
				row[j]+1,   // deletion
				prev+cost,  // substitution
			)
			prev = current
		}
	}

	return row[len(a)]
}

// intMin returns the minimum of two integers.
func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// min3 returns the minimum of three integers.
func min3(a, b, c int) int {
	return intMin(intMin(a, b), c)
}

// String builder methods for zero-allocation error message construction

// resetStringBuilder resets the value buffer for string building.
func (p *Parser) resetStringBuilder() {
	p.valueBuffer = p.valueBuffer[:0]
}

// appendString appends a string to the value buffer.
func (p *Parser) appendString(s string) {
	p.valueBuffer = append(p.valueBuffer, stringToBytes(s)...)
}

// getBuiltString returns the built string from the value buffer.
func (p *Parser) getBuiltString() string {
	return bytesToString(p.valueBuffer)
}

// Method-based API for ParseResult - consistent access patterns for all flag types

// GetString retrieves a string flag value
func (r *ParseResult) GetString(name string) (string, bool) {
	if value, exists := r.StringFlags[name]; exists {
		return value, true
	}
	return "", false
}

// GetInt retrieves an integer flag value
func (r *ParseResult) GetInt(name string) (int, bool) {
	if value, exists := r.IntFlags[name]; exists {
		return value, true
	}
	return 0, false
}

// GetBool retrieves a boolean flag value
func (r *ParseResult) GetBool(name string) (bool, bool) {
	if value, exists := r.BoolFlags[name]; exists {
		return value, true
	}
	return false, false
}

// GetDuration retrieves a duration flag value
func (r *ParseResult) GetDuration(name string) (time.Duration, bool) {
	if value, exists := r.DurationFlags[name]; exists {
		return value, true
	}
	return 0, false
}

// GetFloat retrieves a float64 flag value
func (r *ParseResult) GetFloat(name string) (float64, bool) {
	if value, exists := r.FloatFlags[name]; exists {
		return value, true
	}
	return 0.0, false
}

// GetEnum retrieves an enum flag value
func (r *ParseResult) GetEnum(name string) (string, bool) {
	if value, exists := r.EnumFlags[name]; exists {
		return value, true
	}
	return "", false
}

// GetStringSlice retrieves a string slice flag value using stored slice
func (r *ParseResult) GetStringSlice(name string) ([]string, bool) {
	if offset, exists := r.StringSliceOffsets[name]; exists {
		if offset.Start >= 0 && offset.Start < len(r.stringSlices) {
			slice := r.stringSlices[offset.Start]
			if slice != nil {
				return *slice, true
			}
		}
	}
	return []string{}, false
}

// GetIntSlice retrieves an int slice flag value using stored slice
func (r *ParseResult) GetIntSlice(name string) ([]int, bool) {
	if offset, exists := r.IntSliceOffsets[name]; exists {
		if offset.Start >= 0 && offset.Start < len(r.intSlices) {
			slice := r.intSlices[offset.Start]
			if slice != nil {
				return *slice, true
			}
		}
	}
	return []int{}, false
}

// Global flag access methods

// GetGlobalString retrieves a global string flag value
func (r *ParseResult) GetGlobalString(name string) (string, bool) {
	if value, exists := r.GlobalStringFlags[name]; exists {
		return value, true
	}
	return "", false
}

// GetGlobalInt retrieves a global integer flag value
func (r *ParseResult) GetGlobalInt(name string) (int, bool) {
	if value, exists := r.GlobalIntFlags[name]; exists {
		return value, true
	}
	return 0, false
}

// GetGlobalBool retrieves a global boolean flag value
func (r *ParseResult) GetGlobalBool(name string) (bool, bool) {
	if value, exists := r.GlobalBoolFlags[name]; exists {
		return value, true
	}
	return false, false
}

// GetGlobalDuration retrieves a global duration flag value
func (r *ParseResult) GetGlobalDuration(name string) (time.Duration, bool) {
	if value, exists := r.GlobalDurationFlags[name]; exists {
		return value, true
	}
	return 0, false
}

// GetGlobalFloat retrieves a global float64 flag value
func (r *ParseResult) GetGlobalFloat(name string) (float64, bool) {
	if value, exists := r.GlobalFloatFlags[name]; exists {
		return value, true
	}
	return 0.0, false
}

// GetGlobalEnum retrieves a global enum flag value
func (r *ParseResult) GetGlobalEnum(name string) (string, bool) {
	if value, exists := r.GlobalEnumFlags[name]; exists {
		return value, true
	}
	return "", false
}

// GetGlobalStringSlice retrieves a global string slice flag value using stored slice
func (r *ParseResult) GetGlobalStringSlice(name string) ([]string, bool) {
	if offset, exists := r.GlobalStringSliceOffsets[name]; exists {
		if offset.Start >= 0 && offset.Start < len(r.stringSlices) {
			slice := r.stringSlices[offset.Start]
			if slice != nil {
				return *slice, true
			}
		}
	}
	return []string{}, false
}

// GetGlobalIntSlice retrieves a global int slice flag value using stored slice
func (r *ParseResult) GetGlobalIntSlice(name string) ([]int, bool) {
	if offset, exists := r.GlobalIntSliceOffsets[name]; exists {
		if offset.Start >= 0 && offset.Start < len(r.intSlices) {
			slice := r.intSlices[offset.Start]
			if slice != nil {
				return *slice, true
			}
		}
	}
	return []int{}, false
}

// Convenience methods with defaults (Must pattern) - return value or default

// MustGetString retrieves a string flag value or returns the default
func (r *ParseResult) MustGetString(name, defaultValue string) string {
	if value, exists := r.GetString(name); exists {
		return value
	}
	return defaultValue
}

// MustGetInt retrieves an int flag value or returns the default
func (r *ParseResult) MustGetInt(name string, defaultValue int) int {
	if value, exists := r.GetInt(name); exists {
		return value
	}
	return defaultValue
}

// MustGetBool retrieves a bool flag value or returns the default
func (r *ParseResult) MustGetBool(name string, defaultValue bool) bool {
	if value, exists := r.GetBool(name); exists {
		return value
	}
	return defaultValue
}

// MustGetDuration retrieves a duration flag value or returns the default
func (r *ParseResult) MustGetDuration(name string, defaultValue time.Duration) time.Duration {
	if value, exists := r.GetDuration(name); exists {
		return value
	}
	return defaultValue
}

// MustGetFloat retrieves a float flag value or returns the default
func (r *ParseResult) MustGetFloat(name string, defaultValue float64) float64 {
	if value, exists := r.GetFloat(name); exists {
		return value
	}
	return defaultValue
}

// MustGetEnum retrieves an enum flag value or returns the default
func (r *ParseResult) MustGetEnum(name, defaultValue string) string {
	if value, exists := r.GetEnum(name); exists {
		return value
	}
	return defaultValue
}

// MustGetStringSlice retrieves a string slice flag value or returns the default
func (r *ParseResult) MustGetStringSlice(name string, defaultValue []string) []string {
	if value, exists := r.GetStringSlice(name); exists {
		return value
	}
	return defaultValue
}

// MustGetIntSlice retrieves an int slice flag value or returns the default
func (r *ParseResult) MustGetIntSlice(name string, defaultValue []int) []int {
	if value, exists := r.GetIntSlice(name); exists {
		return value
	}
	return defaultValue
}

// Global convenience methods with defaults (Must pattern)

// MustGetGlobalString retrieves a global string flag value or returns the default
func (r *ParseResult) MustGetGlobalString(name, defaultValue string) string {
	if value, exists := r.GetGlobalString(name); exists {
		return value
	}
	return defaultValue
}

// MustGetGlobalInt retrieves a global int flag value or returns the default
func (r *ParseResult) MustGetGlobalInt(name string, defaultValue int) int {
	if value, exists := r.GetGlobalInt(name); exists {
		return value
	}
	return defaultValue
}

// MustGetGlobalBool retrieves a global bool flag value or returns the default
func (r *ParseResult) MustGetGlobalBool(name string, defaultValue bool) bool {
	if value, exists := r.GetGlobalBool(name); exists {
		return value
	}
	return defaultValue
}

// MustGetGlobalDuration retrieves a global duration flag value or returns the default
func (r *ParseResult) MustGetGlobalDuration(name string, defaultValue time.Duration) time.Duration {
	if value, exists := r.GetGlobalDuration(name); exists {
		return value
	}
	return defaultValue
}

// MustGetGlobalFloat retrieves a global float flag value or returns the default
func (r *ParseResult) MustGetGlobalFloat(name string, defaultValue float64) float64 {
	if value, exists := r.GetGlobalFloat(name); exists {
		return value
	}
	return defaultValue
}

// MustGetGlobalEnum retrieves a global enum flag value or returns the default
func (r *ParseResult) MustGetGlobalEnum(name, defaultValue string) string {
	if value, exists := r.GetGlobalEnum(name); exists {
		return value
	}
	return defaultValue
}

// MustGetGlobalStringSlice retrieves a global string slice flag value or returns the default
func (r *ParseResult) MustGetGlobalStringSlice(name string, defaultValue []string) []string {
	if value, exists := r.GetGlobalStringSlice(name); exists {
		return value
	}
	return defaultValue
}

// MustGetGlobalIntSlice retrieves a global int slice flag value or returns the default
func (r *ParseResult) MustGetGlobalIntSlice(name string, defaultValue []int) []int {
	if value, exists := r.GetGlobalIntSlice(name); exists {
		return value
	}
	return defaultValue
}

// Positional argument access methods (zero-allocation)

// GetArgString retrieves a string positional argument value
func (r *ParseResult) GetArgString(name string) (string, bool) {
	if value, exists := r.ArgStrings[name]; exists {
		return value, true
	}
	return "", false
}

// MustGetArgString retrieves a string positional argument value or returns the default
func (r *ParseResult) MustGetArgString(name, defaultValue string) string {
	if value, exists := r.GetArgString(name); exists {
		return value
	}
	return defaultValue
}

// GetArgInt retrieves an integer positional argument value
func (r *ParseResult) GetArgInt(name string) (int, bool) {
	if value, exists := r.ArgInts[name]; exists {
		return value, true
	}
	return 0, false
}

// MustGetArgInt retrieves an integer positional argument value or returns the default
func (r *ParseResult) MustGetArgInt(name string, defaultValue int) int {
	if value, exists := r.GetArgInt(name); exists {
		return value
	}
	return defaultValue
}

// GetArgBool retrieves a boolean positional argument value
func (r *ParseResult) GetArgBool(name string) (bool, bool) {
	if value, exists := r.ArgBools[name]; exists {
		return value, true
	}
	return false, false
}

// MustGetArgBool retrieves a boolean positional argument value or returns the default
func (r *ParseResult) MustGetArgBool(name string, defaultValue bool) bool {
	if value, exists := r.GetArgBool(name); exists {
		return value
	}
	return defaultValue
}

// GetArgDuration retrieves a duration positional argument value
func (r *ParseResult) GetArgDuration(name string) (time.Duration, bool) {
	if value, exists := r.ArgDurations[name]; exists {
		return value, true
	}
	return 0, false
}

// MustGetArgDuration retrieves a duration positional argument value or returns the default
func (r *ParseResult) MustGetArgDuration(name string, defaultValue time.Duration) time.Duration {
	if value, exists := r.GetArgDuration(name); exists {
		return value
	}
	return defaultValue
}

// GetArgFloat retrieves a float64 positional argument value
func (r *ParseResult) GetArgFloat(name string) (float64, bool) {
	if value, exists := r.ArgFloats[name]; exists {
		return value, true
	}
	return 0.0, false
}

// MustGetArgFloat retrieves a float64 positional argument value or returns the default
func (r *ParseResult) MustGetArgFloat(name string, defaultValue float64) float64 {
	if value, exists := r.GetArgFloat(name); exists {
		return value
	}
	return defaultValue
}

// GetArgStringSlice retrieves a string slice positional argument value (variadic args)
// Uses zero-allocation slice storage pattern
func (r *ParseResult) GetArgStringSlice(name string) ([]string, bool) {
	if offset, exists := r.ArgStringSlices[name]; exists {
		if offset.Start >= 0 && offset.Start < len(r.stringSlices) {
			slice := r.stringSlices[offset.Start]
			if slice != nil {
				return *slice, true
			}
		}
	}
	return []string{}, false
}

// MustGetArgStringSlice retrieves a string slice positional argument value or returns the default
func (r *ParseResult) MustGetArgStringSlice(name string, defaultValue []string) []string {
	if value, exists := r.GetArgStringSlice(name); exists {
		return value
	}
	return defaultValue
}

// GetArgIntSlice retrieves an int slice positional argument value (variadic args)
// Uses zero-allocation slice storage pattern
func (r *ParseResult) GetArgIntSlice(name string) ([]int, bool) {
	if offset, exists := r.ArgIntSlices[name]; exists {
		if offset.Start >= 0 && offset.Start < len(r.intSlices) {
			slice := r.intSlices[offset.Start]
			if slice != nil {
				return *slice, true
			}
		}
	}
	return []int{}, false
}

// MustGetArgIntSlice retrieves an int slice positional argument value or returns the default
func (r *ParseResult) MustGetArgIntSlice(name string, defaultValue []int) []int {
	if value, exists := r.GetArgIntSlice(name); exists {
		return value
	}
	return defaultValue
}

// HasFlag returns true if the flag exists (was provided or has a default)
func (r *ParseResult) HasFlag(name string) bool {
	_, exists := r.StringFlags[name]
	if exists {
		return true
	}
	_, exists = r.IntFlags[name]
	if exists {
		return true
	}
	_, exists = r.BoolFlags[name]
	if exists {
		return true
	}
	_, exists = r.DurationFlags[name]
	if exists {
		return true
	}
	_, exists = r.FloatFlags[name]
	if exists {
		return true
	}
	_, exists = r.EnumFlags[name]
	if exists {
		return true
	}
	_, exists = r.StringSliceOffsets[name]
	if exists {
		return true
	}
	_, exists = r.IntSliceOffsets[name]
	return exists
}

// HasGlobalFlag returns true if the global flag exists (was provided or has a default)
func (r *ParseResult) HasGlobalFlag(name string) bool {
	_, exists := r.GlobalStringFlags[name]
	if exists {
		return true
	}
	_, exists = r.GlobalIntFlags[name]
	if exists {
		return true
	}
	_, exists = r.GlobalBoolFlags[name]
	if exists {
		return true
	}
	_, exists = r.GlobalDurationFlags[name]
	if exists {
		return true
	}
	_, exists = r.GlobalFloatFlags[name]
	if exists {
		return true
	}
	_, exists = r.GlobalEnumFlags[name]
	if exists {
		return true
	}
	_, exists = r.GlobalStringSliceOffsets[name]
	if exists {
		return true
	}
	_, exists = r.GlobalIntSliceOffsets[name]
	return exists
}

// Flag group validation

// validateFlagGroups validates all flag group constraints
func (p *Parser) validateFlagGroups(result *ParseResult) error {
	// Validate app-level flag groups
	for _, group := range p.app.flagGroups {
		if err := p.validateSingleGroup(group, result); err != nil {
			return err
		}
	}

	// Validate command-level flag groups if we have a command
	if result.Command != nil {
		for _, group := range result.Command.flagGroups {
			if err := p.validateSingleGroup(group, result); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateSingleGroup validates a single flag group constraint
func (p *Parser) validateSingleGroup(group *FlagGroup, result *ParseResult) error {
	// First pass: count how many flags in the group are set without allocating
	setCount := 0
	for _, flag := range group.Flags {
		if p.isFlagSet(flag, result) {
			setCount++
		}
	}

	// Validate based on constraint type
	switch group.Constraint { // exhaustive over GroupConstraintType
	case GroupMutuallyExclusive:
		if setCount > 1 {
			// Slow path (error): collect names only when needed
			setFlags := make([]string, 0, setCount)
			for _, flag := range group.Flags {
				if p.isFlagSet(flag, result) {
					setFlags = append(setFlags, flag.Name)
				}
			}
			err := NewParseError(
				ErrorTypeFlagGroupViolation,
				fmt.Sprintf("flags in group '%s' are mutually exclusive, but multiple were provided: %v",
					group.Name, setFlags),
			)
			err.GroupName = group.Name
			return err
		}

	case GroupRequiredGroup, GroupAtLeastOne:
		if setCount == 0 {
			err := NewParseError(
				ErrorTypeFlagGroupViolation,
				fmt.Sprintf("group '%s' requires at least one flag to be set", group.Name),
			)
			err.GroupName = group.Name
			return err
		}

	case GroupAllOrNone:
		if setCount > 0 && setCount < len(group.Flags) {
			err := NewParseError(
				ErrorTypeFlagGroupViolation,
				fmt.Sprintf("group '%s' requires either all flags or no flags to be set", group.Name),
			)
			err.GroupName = group.Name
			return err
		}

	case GroupExactlyOne:
		if setCount != 1 {
			err := NewParseError(
				ErrorTypeFlagGroupViolation,
				fmt.Sprintf("group '%s' requires exactly one flag to be set, but %d were provided",
					group.Name, setCount),
			)
			err.GroupName = group.Name
			return err
		}
	case GroupNoConstraint:
		// No validation needed
	}

	return nil
}

// isFlagSet checks if a flag is set in the parse result
//
//nolint:funlen // Compact switch over flag types
func (p *Parser) isFlagSet(flag *Flag, result *ParseResult) bool {
	switch flag.Type {
	case FlagTypeString:
		if flag.Global {
			_, exists := result.GlobalStringFlags[flag.Name]
			return exists
		}
		_, exists := result.StringFlags[flag.Name]
		return exists

	case FlagTypeInt:
		if flag.Global {
			_, exists := result.GlobalIntFlags[flag.Name]
			return exists
		}
		_, exists := result.IntFlags[flag.Name]
		return exists

	case FlagTypeBool:
		if flag.Global {
			value, exists := result.GlobalBoolFlags[flag.Name]
			return exists && value
		}
		value, exists := result.BoolFlags[flag.Name]
		return exists && value

	case FlagTypeDuration:
		if flag.Global {
			_, exists := result.GlobalDurationFlags[flag.Name]
			return exists
		}
		_, exists := result.DurationFlags[flag.Name]
		return exists

	case FlagTypeFloat:
		if flag.Global {
			_, exists := result.GlobalFloatFlags[flag.Name]
			return exists
		}
		_, exists := result.FloatFlags[flag.Name]
		return exists

	case FlagTypeEnum:
		if flag.Global {
			_, exists := result.GlobalEnumFlags[flag.Name]
			return exists
		}
		_, exists := result.EnumFlags[flag.Name]
		return exists

	case FlagTypeStringSlice:
		if flag.Global {
			_, exists := result.GlobalStringSliceOffsets[flag.Name]
			return exists
		}
		_, exists := result.StringSliceOffsets[flag.Name]
		return exists

	case FlagTypeIntSlice:
		if flag.Global {
			_, exists := result.GlobalIntSliceOffsets[flag.Name]
			return exists
		}
		_, exists := result.IntSliceOffsets[flag.Name]
		return exists

	default:
		return false
	}
}

// getEnvValue checks environment variables in precedence order and returns the first non-empty value
func (p *Parser) getEnvValue(envVars []string) string {
	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			return value
		}
	}
	return ""
}

// parseIntValue parses a string value as an integer
func (p *Parser) parseIntValue(value string) (int, error) {
	return p.parseIntBytes([]byte(value))
}

// parseBoolValue parses a string value as a boolean without error.
func (p *Parser) parseBoolValue(value string) bool {
	return p.parseBoolBytes([]byte(value))
}

// parseFloatValue parses a string value as a float64
func (p *Parser) parseFloatValue(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}

// parseDurationValue parses a string value as a time.Duration
func (p *Parser) parseDurationValue(value string) (time.Duration, error) {
	// Support the same extended formats as CLI parsing
	return p.parseDurationBytes([]byte(value))
}
