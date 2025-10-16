package snap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	snapio "github.com/dzonerzy/go-snap/io"
	"github.com/dzonerzy/go-snap/middleware"
)

// Special error types for graceful exits
var (
	ErrHelpShown    = errors.New("help shown")
	ErrVersionShown = errors.New("version shown")
)

// ActionFunc defines the command execution function
type ActionFunc func(*Context) error

// Author represents an application author
type Author struct {
	Name  string
	Email string
}

// App represents the main CLI application
type App struct {
	name        string
	description string
	helpText    string
	version     string
	authors     []Author

	// Internal storage
	flags      map[string]*Flag
	shortFlags map[rune]*Flag // O(1) lookup for short flags
	commands   map[string]*Command
	flagGroups []*FlagGroup // Flag groups for validation

	// Global configuration
	helpFlag    bool
	versionFlag bool

	// Execution context
	beforeAction ActionFunc
	afterAction  ActionFunc

	// Error handling
	errorHandler *ErrorHandler

	// Middleware
	middleware []middleware.Middleware

	// Current parse result for flag access (available after parsing)
	currentResult *ParseResult

	// Configuration builder for automatic config population during Run()
	configBuilder *ConfigBuilder

	// IO management
	ioManager *snapio.IOManager

	// Exit code management
	exitCodes *ExitCodeManager

	// Wrapper at app level (optional)
	defaultWrapper *WrapperSpec

	// Raw arguments as passed to RunWithArgs (before parsing)
	rawArgs []string
}

// New creates a new CLI application with fluent API
func New(name, description string) *App {
	return &App{
		name:         name,
		description:  description,
		authors:      make([]Author, 0),
		flags:        make(map[string]*Flag),
		shortFlags:   make(map[rune]*Flag),
		commands:     make(map[string]*Command),
		flagGroups:   make([]*FlagGroup, 0),
		helpFlag:     true,              // Enable help by default
		versionFlag:  false,             // Disable version by default
		errorHandler: NewErrorHandler(), // Initialize with default error handler
		middleware:   make([]middleware.Middleware, 0),
		ioManager:    snapio.New(),
	}
}

// App configuration methods

// Version sets the application version
func (a *App) Version(version string) *App {
	a.version = version
	a.versionFlag = true
	return a
}

// Author adds an application author
func (a *App) Author(name, email string) *App {
	a.authors = append(a.authors, Author{Name: name, Email: email})
	return a
}

// Authors sets multiple application authors
func (a *App) Authors(authors ...Author) *App {
	a.authors = append(a.authors, authors...)
	return a
}

// HelpText sets detailed help text for the application
func (a *App) HelpText(help string) *App {
	a.helpText = help
	return a
}

// Use adds middleware to the application
func (a *App) Use(middleware ...middleware.Middleware) *App {
	a.middleware = append(a.middleware, middleware...)
	return a
}

// DisableHelp disables automatic help flag generation
func (a *App) DisableHelp() *App {
	a.helpFlag = false
	return a
}

// Before sets a function to run before any command action
func (a *App) Before(fn ActionFunc) *App {
	a.beforeAction = fn
	return a
}

// After sets a function to run after any command action
func (a *App) After(fn ActionFunc) *App {
	a.afterAction = fn
	return a
}

// IO returns the application's IOManager for fluent configuration.
func (a *App) IO() *snapio.IOManager {
	if a.ioManager == nil {
		a.ioManager = snapio.New()
	}
	return a.ioManager
}

// Flag builders - Type-safe flag definitions

// StringFlag adds a string flag to the application
func (a *App) StringFlag(name, description string) *FlagBuilder[string, *App] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeString,
	}
	a.flags[name] = flag
	return &FlagBuilder[string, *App]{flag: flag, parent: a}
}

// IntFlag adds an integer flag to the application
func (a *App) IntFlag(name, description string) *FlagBuilder[int, *App] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeInt,
	}
	a.flags[name] = flag
	return &FlagBuilder[int, *App]{flag: flag, parent: a}
}

// BoolFlag adds a boolean flag to the application
func (a *App) BoolFlag(name, description string) *FlagBuilder[bool, *App] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeBool,
	}
	a.flags[name] = flag
	return &FlagBuilder[bool, *App]{flag: flag, parent: a}
}

// DurationFlag adds a duration flag to the application
func (a *App) DurationFlag(name, description string) *FlagBuilder[time.Duration, *App] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeDuration,
	}
	a.flags[name] = flag
	return &FlagBuilder[time.Duration, *App]{flag: flag, parent: a}
}

// FloatFlag adds a float64 flag to the application
func (a *App) FloatFlag(name, description string) *FlagBuilder[float64, *App] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeFloat,
	}
	a.flags[name] = flag
	return &FlagBuilder[float64, *App]{flag: flag, parent: a}
}

// EnumFlag adds an enum flag to the application
func (a *App) EnumFlag(name, description string, values ...string) *FlagBuilder[string, *App] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeEnum,
		EnumValues:  values,
	}
	a.flags[name] = flag
	return &FlagBuilder[string, *App]{flag: flag, parent: a}
}

// StringSliceFlag adds a string slice flag to the application
func (a *App) StringSliceFlag(name, description string) *FlagBuilder[[]string, *App] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeStringSlice,
	}
	a.flags[name] = flag
	return &FlagBuilder[[]string, *App]{flag: flag, parent: a}
}

// IntSliceFlag adds an int slice flag to the application
func (a *App) IntSliceFlag(name, description string) *FlagBuilder[[]int, *App] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeIntSlice,
	}
	a.flags[name] = flag
	return &FlagBuilder[[]int, *App]{flag: flag, parent: a}
}

// Command builder

// Command adds a command to the application
func (a *App) Command(name, description string) *CommandBuilder {
	cmd := &Command{
		name:        name,
		description: description,
		Aliases:     make([]string, 0),
		Hidden:      false,
		flags:       make(map[string]*Flag),
		shortFlags:  make(map[rune]*Flag),
		subcommands: make(map[string]*Command),
		flagGroups:  make([]*FlagGroup, 0),
		middleware:  make([]middleware.Middleware, 0),
	}
	a.addCommandHelpFlag(cmd)
	a.commands[name] = cmd
	return &CommandBuilder{
		command: cmd,
		app:     a,
	}
}

// Execution methods

// Run parses command line arguments and executes the appropriate action
func (a *App) Run() error {
	return a.RunContext(context.Background())
}

// RunContext runs the application with a context for cancellation
func (a *App) RunContext(ctx context.Context) error {
	return a.RunWithArgs(ctx, os.Args[1:])
}

// RunWithArgs runs the application with provided arguments
//
//nolint:gocognit,nestif,funlen // Main execution flow is inherently complex
func (a *App) RunWithArgs(ctx context.Context, args []string) error {
	// Store raw arguments before parsing for later access via Context.RawArgs()
	a.rawArgs = args

	// Windows: auto-enable Virtual Terminal (ANSI) when writing to a TTY, unless disabled
	if runtime.GOOS == "windows" && a.IO().IsTTY() && os.Getenv("SNAP_DISABLE_VT") == "" {
		_ = a.IO().EnableVirtualTerminal() // best-effort; ignore failure
	}
	// Add default help and version flags if enabled
	if a.helpFlag {
		a.addHelpFlag()
	}
	if a.versionFlag {
		a.addVersionFlag()
	}

	// Create parser and parse arguments
	parser := NewParser(a)
	result, err := parser.Parse(args)
	if err != nil {
		// Handle parsing errors with smart suggestions and contextual help
		parseErr := &ParseError{}
		if errors.As(err, &parseErr) {
			return a.handleParseError(parseErr)
		}
		return err
	}

	// Store parse result for flag access
	a.currentResult = result

	// Handle built-in flags BEFORE populating configuration
	if helpErr := a.handleHelpAndVersion(result); helpErr != nil {
		return helpErr
	}

	// Populate configuration if config builder is attached
	if a.configBuilder != nil {
		cfgErr := a.populateConfiguration()
		if cfgErr != nil {
			return fmt.Errorf("configuration error: %w", cfgErr)
		}
	}

	// Create execution context with cancellation support
	ctxWithCancel, cancel := context.WithCancel(ctx)
	execCtx := &Context{
		App:      a,
		Result:   result,
		ctx:      ctxWithCancel,
		cancel:   cancel,
		metadata: make(map[string]any),
	}

	// Execute before action
	if a.beforeAction != nil {
		if beforeErr := a.beforeAction(execCtx); beforeErr != nil {
			return beforeErr
		}
	}

	// Execute command action
	var actionErr error
	if result.Command != nil {
		// Execute command-level Before hook
		if result.Command.beforeAction != nil {
			if beforeErr := result.Command.beforeAction(execCtx); beforeErr != nil {
				return beforeErr
			}
		}

		// Check command context: help vs action vs wrapper
		switch {
		case result.MustGetBool("help", false):
			actionErr = a.showCommandHelp(result.Command)
		case result.Command.Action != nil:
			// Apply middleware and execute action
			wrappedAction := a.wrapActionWithMiddleware(result.Command.Action, result.Command)
			actionErr = wrappedAction(execCtx)
		case result.Command.wrapper != nil:
			// Command-level wrapper (no explicit action)
			actionErr = result.Command.wrapper.run(execCtx, args)
		default:
			// No explicit action or wrapper: show the command help (especially when it has subcommands)
			actionErr = a.showCommandHelp(result.Command)
		}

		// Execute command-level After hook
		if result.Command.afterAction != nil {
			if afterErr := result.Command.afterAction(execCtx); afterErr != nil {
				// If action succeeded but after hook failed, return after error
				if actionErr == nil {
					actionErr = afterErr
				}
			}
		}
	} else {
		// No command specified, check if app has a default wrapper
		if a.defaultWrapper != nil {
			actionErr = a.defaultWrapper.run(execCtx, args)
		} else {
			// Default to help
			actionErr = a.showHelp()
		}
	}

	// If the action requested exit via context, prefer that
	if ee, ok := execCtx.Get("__exit_error__").(*ExitError); ok && ee != nil {
		actionErr = ee
	}

	// Execute after action
	if a.afterAction != nil {
		if afterErr := a.afterAction(execCtx); afterErr != nil {
			return afterErr
		}
	}

	return actionErr
}

// ExitCodes returns the exit-code manager for this app. Use it to override
// defaults or register custom mappings. Resolution precedence is:
// ExitError > CLI category (DefineCLI) > concrete error type (DefineError) > defaults.
func (a *App) ExitCodes() *ExitCodeManager {
	if a.exitCodes == nil {
		a.exitCodes = newExitCodeManager()
	}
	return a.exitCodes
}

// RunAndGetExitCode executes the app and returns the mapped exit code according
// to ExitCodes(). Useful for embedding in your own main() without os.Exit.
func (a *App) RunAndGetExitCode() int {
	err := a.Run()
	if err == nil {
		return a.ExitCodes().defaults.Success
	}
	return a.ExitCodes().resolve(err)
}

// RunAndExit executes the app and terminates the process with the mapped exit
// code. Equivalent to os.Exit(a.RunAndGetExitCode()).
func (a *App) RunAndExit() {
	os.Exit(a.RunAndGetExitCode())
}

// FlagParent interface implementation

// addShortFlag adds a short flag mapping for O(1) lookup
func (a *App) addShortFlag(short rune, flag *Flag) {
	a.shortFlags[short] = flag
}

// addFlagGroup adds a flag group to the app (implements FlagGroupParent interface)
func (a *App) addFlagGroup(group *FlagGroup) {
	// Check if group already exists to prevent duplicates
	for _, existingGroup := range a.flagGroups {
		if existingGroup.Name == group.Name {
			return // Group already added, skip
		}
	}

	a.flagGroups = append(a.flagGroups, group)

	// Also add all flags in the group to the app's flag map for parsing
	for _, flag := range group.Flags {
		a.flags[flag.Name] = flag
		if flag.Short != 0 {
			a.shortFlags[flag.Short] = flag
		}
	}
}

// FlagGroup creates a new flag group builder
func (a *App) FlagGroup(name string) *FlagGroupBuilder[*App] {
	group := &FlagGroup{
		Name:  name,
		Flags: make([]*Flag, 0),
	}
	return &FlagGroupBuilder[*App]{
		group:  group,
		parent: a,
	}
}

// ErrorHandler returns the app's error handler for configuration
func (a *App) ErrorHandler() *ErrorHandler {
	return a.errorHandler
}

// wrapActionWithMiddleware wraps the action with app-level and command-level middleware
func (a *App) wrapActionWithMiddleware(action ActionFunc, cmd *Command) ActionFunc {
	// Combine app-level and command-level middleware
	allMiddleware := make([]middleware.Middleware, 0, len(a.middleware)+len(cmd.middleware))
	allMiddleware = append(allMiddleware, a.middleware...)
	allMiddleware = append(allMiddleware, cmd.middleware...)

	if len(allMiddleware) == 0 {
		return action
	}

	// Create middleware chain
	chain := middleware.Chain(allMiddleware...)

	// Convert snap.ActionFunc to middleware.ActionFunc using an adapter
	middlewareAction := func(ctx middleware.Context) error {
		// The context passed to middleware is a snap.Context that implements middleware.Context
		snapCtx, ok := ctx.(*Context)
		if !ok {
			return NewError(ErrorTypeInternal, "invalid middleware context type")
		}
		return action(snapCtx)
	}

	// Apply middleware chain
	wrappedMiddlewareAction := chain.Apply(middlewareAction)

	// Convert back to snap.ActionFunc
	return func(ctx *Context) error {
		return wrappedMiddlewareAction(ctx)
	}
}

// handleParseError converts ParseError to CLIError and displays it with context
func (a *App) handleParseError(parseErr *ParseError) error {
	// Convert ParseError to CLIError for enhanced handling
	cliErr := NewError(parseErr.Type, parseErr.Message)

	// Add context based on error type
	switch parseErr.Type { // exhaustive over ErrorType for context enrichment
	case ErrorTypeUnknownFlag:
		if parseErr.Flag != "" {
			cliErr = cliErr.WithContext("flag", parseErr.Flag)
		}
	case ErrorTypeUnknownCommand:
		if parseErr.Command != "" {
			cliErr = cliErr.WithContext("command", parseErr.Command)
		}
	case ErrorTypeFlagGroupViolation:
		if parseErr.GroupName != "" {
			cliErr = cliErr.WithContext("group", parseErr.GroupName)
		}
	case ErrorTypeInvalidFlag, ErrorTypeInvalidValue, ErrorTypeMissingValue,
		ErrorTypeInternal, ErrorTypeMissingRequired, ErrorTypePermission, ErrorTypeValidation:
		// No additional context for these types here.
	}

	// Process error with smart suggestions
	cliErr = a.errorHandler.ProcessError(cliErr, a)

	// Display the error with contextual help
	a.errorHandler.DisplayError(cliErr, a)

	return cliErr
}

// Helper methods

// addHelpFlag adds the built-in help flag
func (a *App) addHelpFlag() {
	if _, exists := a.flags["help"]; !exists {
		flag := &Flag{
			Name:        "help",
			Description: "Show help",
			Type:        FlagTypeBool,
			Global:      true,
		}
		a.flags["help"] = flag
		// Provide -h by default if not already in use
		if _, taken := a.shortFlags['h']; !taken {
			flag.Short = 'h'
			a.shortFlags['h'] = flag
		}
	}
}

// addVersionFlag adds the built-in version flag
func (a *App) addVersionFlag() {
	if _, exists := a.flags["version"]; !exists {
		flag := &Flag{
			Name:        "version",
			Description: "Show version",
			Type:        FlagTypeBool,
			Global:      true,
		}
		a.flags["version"] = flag
	}
}

// addCommandHelpFlag adds the built-in help flag to a command
func (a *App) addCommandHelpFlag(cmd *Command) {
	if _, exists := cmd.flags["help"]; !exists {
		flag := &Flag{
			Name:        "help",
			Description: "Show command help",
			Type:        FlagTypeBool,
			Global:      false,
		}
		cmd.flags["help"] = flag
		// Provide -h by default at command level if not already in use
		if _, taken := cmd.shortFlags['h']; !taken {
			flag.Short = 'h'
			cmd.shortFlags['h'] = flag
		}
	}
}

// showHelp displays comprehensive application help
//
//nolint:gocognit,funlen // Help rendering involves many small branches; splitting would harm readability.
func (a *App) showHelp() error {
	// Application name and description
	if a.description != "" {
		println(a.description)
		println()
	}

	// Detailed help text if available
	if a.helpText != "" {
		println(a.helpText)
		println()
	}

	// Usage line
	println("Usage:")
	print("  ", a.name)
	if len(a.flags) > 0 {
		print(" [GLOBAL FLAGS]")
	}

	if len(a.commands) > 0 {
		print(" COMMAND [COMMAND FLAGS]")
	}
	println()

	// Version information
	if a.version != "" {
		println()
		println("Version:", a.version)
	}

	// Authors information
	if len(a.authors) > 0 {
		println()
		if len(a.authors) == 1 {
			println("Author:", a.authors[0].Name, "<"+a.authors[0].Email+">")
		} else {
			println("Authors:")
			for _, author := range a.authors {
				println("  ", author.Name, "<"+author.Email+">")
			}
		}
	}

	// Show flags organized by groups
	a.showOrganizedFlags()

	// Commands (deterministic order)
	if len(a.commands) > 0 { //nolint:nestif // help rendering uses explicit nested branches for clarity
		println()
		println("Commands:")
		names := make([]string, 0, len(a.commands))
		for name := range a.commands {
			if !a.commands[name].Hidden {
				names = append(names, name)
			}
		}
		for i := 0; i < len(names); i++ {
			for j := i + 1; j < len(names); j++ {
				if names[j] < names[i] {
					names[i], names[j] = names[j], names[i]
				}
			}
		}

		// Calculate max command name length for alignment
		maxNameLen := 0
		for _, name := range names {
			if len(name) > maxNameLen {
				maxNameLen = len(name)
			}
		}

		for _, name := range names {
			cmd := a.commands[name]
			print("  ", name)
			if cmd.Description() != "" {
				// Add padding to align descriptions
				padding := maxNameLen - len(name)
				for range padding {
					print(" ")
				}
				print("\t", cmd.Description())
			}
			if len(cmd.Aliases) > 0 {
				print(" (aliases: ")
				for i, alias := range cmd.Aliases {
					if i > 0 {
						print(", ")
					}
					print(alias)
				}
				print(")")
			}
			println()
		}
	}

	// Footer
	println()
	println("Use \"" + a.name + " COMMAND --help\" for more information about a command.")

	return nil
}

// flagDisplayWidth calculates the width of the flag display string (before description)
func flagDisplayWidth(flag *Flag) int {
	width := 2 + len(flag.Name) // "  --" + name
	if flag.Short != 0 {
		width += 4 // ", -X"
	}
	if flag.Type != FlagTypeBool {
		width += 6 // " value"
	}
	return width
}

// showOrganizedFlags displays flags organized by groups
//
//nolint:gocognit // Structured flag rendering across groups/types is intentionally verbose.
func (a *App) showOrganizedFlags() {
	// Collect ungrouped flags (flags not in any group)
	ungroupedFlags := make(map[string]*Flag)
	groupedFlags := make(map[string]bool) // Track which flags are in groups

	// Mark flags that are in groups
	for _, group := range a.flagGroups {
		for _, flag := range group.Flags {
			groupedFlags[flag.Name] = true
		}
	}

	// Collect ungrouped flags
	for name, flag := range a.flags {
		if !groupedFlags[name] && !flag.Hidden {
			ungroupedFlags[name] = flag
		}
	}

	// Calculate max flag display width across all visible flags
	maxWidth := 0
	for _, flag := range a.flags {
		if !flag.Hidden {
			width := flagDisplayWidth(flag)
			if width > maxWidth {
				maxWidth = width
			}
		}
	}

	// Sort groups by name for deterministic output
	groups := append(make([]*FlagGroup, 0, len(a.flagGroups)), a.flagGroups...)
	for i := 0; i < len(groups); i++ {
		for j := i + 1; j < len(groups); j++ {
			if groups[j].Name < groups[i].Name {
				groups[i], groups[j] = groups[j], groups[i]
			}
		}
	}

	// Show flag groups first (sorted)
	//nolint:dupl // Similar to command flag rendering but operates on app-level flags
	for _, group := range groups {
		println()
		if group.Description != "" {
			println(group.Name + " - " + group.Description + ":")
		} else {
			println(group.Name + ":")
		}

		// sort flags by name
		names := make([]string, 0, len(group.Flags))
		for _, flag := range group.Flags {
			if !flag.Hidden {
				names = append(names, flag.Name)
			}
		}
		for i := 0; i < len(names); i++ {
			for j := i + 1; j < len(names); j++ {
				if names[j] < names[i] {
					names[i], names[j] = names[j], names[i]
				}
			}
		}
		for _, name := range names {
			a.showFlag(a.flags[name], maxWidth)
		}

		// Show constraint info
		constraintDesc := a.formatGroupConstraint(group.Constraint)
		if constraintDesc != "" {
			println("  Note:", constraintDesc)
		}
	}

	// Show ungrouped flags
	if len(ungroupedFlags) > 0 {
		println()
		if len(a.flagGroups) > 0 {
			println("Global Flags:")
		} else {
			println("Flags:")
		}

		// sort names
		names := make([]string, 0, len(ungroupedFlags))
		for n := range ungroupedFlags {
			names = append(names, n)
		}
		for i := 0; i < len(names); i++ {
			for j := i + 1; j < len(names); j++ {
				if names[j] < names[i] {
					names[i], names[j] = names[j], names[i]
				}
			}
		}
		for _, n := range names {
			a.showFlag(ungroupedFlags[n], maxWidth)
		}
	}
}

// showFlag displays a single flag with both long and short forms
func (a *App) showFlag(flag *Flag, maxWidth int) {
	print("  --", flag.Name)

	// Show short form if available
	if flag.Short != 0 {
		print(", -", string(flag.Short))
	}

	// Show value type for non-boolean flags
	if flag.Type != FlagTypeBool {
		print(" value")
	}

	// Add padding to align descriptions
	currentWidth := flagDisplayWidth(flag)
	padding := maxWidth - currentWidth
	for range padding {
		print(" ")
	}

	// Show description
	if flag.Description != "" {
		print("\t", flag.Description)
	}

	// Show default value if present
	defaultValue := a.getDefaultValue(flag)
	if defaultValue != "" {
		print(" (default: ", defaultValue, ")")
	}

	println()
}

// formatGroupConstraint returns a human-readable constraint description
func (a *App) formatGroupConstraint(constraint GroupConstraintType) string {
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

// getDefaultValue returns the default value of a flag as a string
func (a *App) getDefaultValue(flag *Flag) string {
	switch flag.Type {
	case FlagTypeString, FlagTypeEnum:
		if flag.DefaultString != "" {
			return flag.DefaultString
		}
	case FlagTypeInt:
		if flag.DefaultInt != 0 {
			return strconv.Itoa(flag.DefaultInt)
		}
	case FlagTypeBool:
		if flag.DefaultBool {
			return "true"
		}
	case FlagTypeDuration:
		if flag.DefaultDuration != 0 {
			return flag.DefaultDuration.String()
		}
	case FlagTypeFloat:
		if flag.DefaultFloat != 0 {
			return fmt.Sprintf("%g", flag.DefaultFloat)
		}
	case FlagTypeStringSlice:
		if len(flag.DefaultStringSlice) > 0 {
			// join with comma
			s := ""
			for i, v := range flag.DefaultStringSlice {
				if i > 0 {
					s += ","
				}
				s += v
			}
			return s
		}
	case FlagTypeIntSlice:
		if len(flag.DefaultIntSlice) > 0 {
			s := ""
			for i, v := range flag.DefaultIntSlice {
				if i > 0 {
					s += ","
				}
				s += strconv.Itoa(v)
			}
			return s
		}
	}
	return ""
}

// showVersion displays application version
func (a *App) showVersion() error {
	println(a.name, a.version)
	return nil
}

// showCommandHelp displays detailed help for a specific command
//
//nolint:gocognit // Command help rendering prioritizes clarity over reduced nesting.
func (a *App) showCommandHelp(cmd *Command) error {
	// Command name and description
	println(cmd.Description())
	println()

	// Usage line
	println("Usage:")
	print("  ", a.name, " ", cmd.Name())
	if len(cmd.flags) > 0 {
		print(" [FLAGS]")
	}

	if len(cmd.subcommands) > 0 {
		print(" SUBCOMMAND")
	}
	println()

	// Long help text if available
	if cmd.HelpText != "" {
		println()
		println(cmd.HelpText)
	}

	// Command-specific flags (organized by groups, deterministic order)
	a.showOrganizedCommandFlags(cmd)

	// Subcommands (sorted)
	if len(cmd.subcommands) > 0 { //nolint:nestif // help rendering uses explicit nested branches for clarity
		println()
		println("Subcommands:")
		names := make([]string, 0, len(cmd.subcommands))
		for name, sc := range cmd.subcommands {
			if !sc.Hidden {
				names = append(names, name)
			}
		}
		for i := 0; i < len(names); i++ {
			for j := i + 1; j < len(names); j++ {
				if names[j] < names[i] {
					names[i], names[j] = names[j], names[i]
				}
			}
		}
		for _, name := range names {
			subcmd := cmd.subcommands[name]
			print("  ", name)
			if subcmd.Description() != "" {
				print("\t", subcmd.Description())
			}
			if len(subcmd.Aliases) > 0 {
				print(" (aliases: ")
				for i, alias := range subcmd.Aliases {
					if i > 0 {
						print(", ")
					}
					print(alias)
				}
				print(")")
			}
			println()
		}
	}

	// Footer
	println()
	println("Use \"" + a.name + " " + cmd.Name() + " SUBCOMMAND --help\" for more information about a subcommand.")

	return nil
}

// showOrganizedCommandFlags displays command flags with grouping and deterministic order
//
//nolint:gocognit // Command flag organization mirrors app-level logic; acceptable complexity.
func (a *App) showOrganizedCommandFlags(cmd *Command) {
	if cmd == nil {
		return
	}

	// Calculate max flag display width across all visible command flags
	maxWidth := 0
	for _, flag := range cmd.flags {
		if !flag.Hidden {
			width := flagDisplayWidth(flag)
			if width > maxWidth {
				maxWidth = width
			}
		}
	}

	// Track flags that are in groups
	grouped := make(map[string]bool)
	for _, g := range cmd.flagGroups {
		for _, f := range g.Flags {
			grouped[f.Name] = true
		}
	}

	// Print groups
	//nolint:dupl // Similar to app flag rendering but operates on command-level flags
	for _, g := range cmd.flagGroups {
		println()
		if g.Description != "" {
			println(g.Name + " - " + g.Description + ":")
		} else {
			println(g.Name + ":")
		}
		// deterministic order
		names := make([]string, 0, len(g.Flags))
		for _, f := range g.Flags {
			if !f.Hidden {
				names = append(names, f.Name)
			}
		}
		// simple sort (no import to avoid clutter)
		for i := 0; i < len(names); i++ {
			for j := i + 1; j < len(names); j++ {
				if names[j] < names[i] {
					names[i], names[j] = names[j], names[i]
				}
			}
		}
		for _, name := range names {
			a.showFlag(cmd.flags[name], maxWidth)
		}
		constraintDesc := a.formatGroupConstraint(g.Constraint)
		if constraintDesc != "" {
			println("  Note:", constraintDesc)
		}
	}

	// Ungrouped flags
	ungrouped := make([]string, 0)
	for name, f := range cmd.flags {
		if !f.Hidden && !grouped[name] {
			ungrouped = append(ungrouped, name)
		}
	}
	if len(ungrouped) > 0 {
		// sort
		for i := 0; i < len(ungrouped); i++ {
			for j := i + 1; j < len(ungrouped); j++ {
				if ungrouped[j] < ungrouped[i] {
					ungrouped[i], ungrouped[j] = ungrouped[j], ungrouped[i]
				}
			}
		}
		println()
		println("Flags:")
		for _, name := range ungrouped {
			a.showFlag(cmd.flags[name], maxWidth)
		}
	}
}

// populateConfiguration handles configuration population during App.Run()
func (a *App) populateConfiguration() error {
	if a.configBuilder == nil {
		return nil
	}

	// Execute any pending source additions
	for _, addSource := range a.configBuilder.pendingSources {
		addSource()
	}

	// Collect flag values now that we have parsed results
	a.configBuilder.collectFlagValues()

	// Refresh environment source to ensure current process env is honored in CLI mode
	if a.configBuilder.schema != nil {
		envData := a.configBuilder.loadFromEnv()
		if len(envData) > 0 {
			a.configBuilder.precedenceManager.AddSource(SourceTypeEnv, envData)
		}
	}

	// Resolve configuration with precedence using the precedence manager
	resolved, err := a.configBuilder.precedenceManager.ResolveWithSchema(a.configBuilder.schema)
	if err != nil {
		return err
	}

	// Apply resolved configuration to target struct
	return a.configBuilder.applyToStruct(resolved)
}

// handleHelpAndVersion provides comprehensive help and version handling for all command levels
func (a *App) handleHelpAndVersion(result *ParseResult) error {
	// Handle help flag across all command levels
	if a.helpFlag && a.isHelpRequested(result) {
		if err := a.showContextualHelp(result); err != nil {
			return err
		}
		return ErrHelpShown
	}

	// Handle version flag across all command levels
	if a.versionFlag && a.isVersionRequested(result) {
		if err := a.showContextualVersion(result); err != nil {
			return err
		}
		return ErrVersionShown
	}

	return nil
}

// isHelpRequested checks if help was requested at any command level
func (a *App) isHelpRequested(result *ParseResult) bool {
	// Check global help first: myapp --help
	if result.Command == nil {
		return result.MustGetGlobalBool("help", false)
	}

	// Check command-level help: myapp command --help, myapp cmd subcmd --help, etc.
	return result.MustGetBool("help", false)
}

// isVersionRequested checks if version was requested at any command level
func (a *App) isVersionRequested(result *ParseResult) bool {
	// Check global version first: myapp --version
	if result.Command == nil {
		return result.MustGetGlobalBool("version", false)
	}

	// Check command-level version: myapp command --version, myapp cmd subcmd --version, etc.
	return result.MustGetBool("version", false)
}

// showContextualHelp displays help appropriate for the current command context
func (a *App) showContextualHelp(result *ParseResult) error {
	if result.Command == nil {
		// Global context: show main application help
		return a.showHelp()
	}
	// Command context: show command-specific help
	return a.showCommandHelp(result.Command)
}

// showContextualVersion displays version appropriate for the current command context
func (a *App) showContextualVersion(_ *ParseResult) error {
	// Currently both global and command contexts share the same version display.
	// Kept as a separate method for symmetry with showContextualHelp.
	return a.showVersion()
}
