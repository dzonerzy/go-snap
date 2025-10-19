package snap

import (
	"context"
	stdio "io"
	"time"

	snapio "github.com/dzonerzy/go-snap/io"
	"github.com/dzonerzy/go-snap/middleware"
)

// Context provides execution context and lifecycle management
type Context struct {
	App           *App
	Result        *ParseResult
	ctx           context.Context
	parent        *Context
	cancel        context.CancelFunc
	metadata      map[string]any
	currentBinary string   // Current binary being executed (for WrapMany)
	binaries      []string // All binaries in WrapMany execution
}

// Context methods for accessing the underlying Go context

// Context returns the underlying Go context for cancellation/timeouts
func (c *Context) Context() context.Context {
	return c.ctx
}

// WithContext creates a new Context with a different underlying context
func (c *Context) WithContext(ctx context.Context) *Context {
	return &Context{
		App:    c.App,
		Result: c.Result,
		ctx:    ctx,
	}
}

// Deadline returns the time when work done on behalf of this context should be canceled
func (c *Context) Deadline() (time.Time, bool) {
	return c.ctx.Deadline()
}

// Done returns a channel that's closed when work done on behalf of this context should be canceled
func (c *Context) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Err returns a non-nil error value after Done is closed
func (c *Context) Err() error {
	return c.ctx.Err()
}

// Value returns the value associated with this context for key
func (c *Context) Value(key interface{}) interface{} {
	return c.ctx.Value(key)
}

// Context management methods

// Set stores a key-value pair in the context metadata
func (c *Context) Set(key string, value any) {
	if c.metadata == nil {
		c.metadata = make(map[string]any)
	}
	c.metadata[key] = value
}

// Get retrieves a value from the context metadata
func (c *Context) Get(key string) any {
	if c.metadata == nil {
		return nil
	}
	return c.metadata[key]
}

// Exit helpers integrate with ExitCodeManager. They store an exit request
// in context metadata and cancel the context; App handles mapping at the end.
func (c *Context) Exit(code int) {
	if c.metadata == nil {
		c.metadata = make(map[string]any)
	}
	c.metadata["__exit_error__"] = &ExitError{Code: code}
	c.Cancel()
}

func (c *Context) ExitWithError(err error, code int) {
	if c.metadata == nil {
		c.metadata = make(map[string]any)
	}
	c.metadata["__exit_error__"] = &ExitError{Code: code, Err: err}
	c.Cancel()
}

func (c *Context) ExitOnError(err error) {
	if err == nil {
		return
	}
	mgr := c.App.ExitCodes()
	code := mgr.resolve(err)
	c.ExitWithError(err, code)
}

// Cancel cancels the context
func (c *Context) Cancel() {
	if c.cancel != nil {
		c.cancel()
	}
}

// Parent returns the parent context
func (c *Context) Parent() *Context {
	return c.parent
}

// IO accessors
func (c *Context) IO() *snapio.IOManager { return c.App.IO() }
func (c *Context) Stdout() stdio.Writer  { return c.App.IO().Out() }
func (c *Context) Stderr() stdio.Writer  { return c.App.IO().Err() }
func (c *Context) Stdin() stdio.Reader   { return c.App.IO().In() }

// Convenience methods for flag access - delegates to ParseResult

// String retrieves a string flag value (safe access)
func (c *Context) String(name string) (string, bool) {
	return c.Result.GetString(name)
}

// MustString retrieves a string flag value with default fallback
func (c *Context) MustString(name, defaultValue string) string {
	return c.Result.MustGetString(name, defaultValue)
}

// Int retrieves an int flag value (safe access)
func (c *Context) Int(name string) (int, bool) {
	return c.Result.GetInt(name)
}

// MustInt retrieves an int flag value with default fallback
func (c *Context) MustInt(name string, defaultValue int) int {
	return c.Result.MustGetInt(name, defaultValue)
}

// Bool retrieves a bool flag value (safe access)
func (c *Context) Bool(name string) (bool, bool) {
	return c.Result.GetBool(name)
}

// MustBool retrieves a bool flag value with default fallback
func (c *Context) MustBool(name string, defaultValue bool) bool {
	return c.Result.MustGetBool(name, defaultValue)
}

// Duration retrieves a duration flag value (safe access)
func (c *Context) Duration(name string) (time.Duration, bool) {
	return c.Result.GetDuration(name)
}

// MustDuration retrieves a duration flag value with default fallback
func (c *Context) MustDuration(name string, defaultValue time.Duration) time.Duration {
	return c.Result.MustGetDuration(name, defaultValue)
}

// Float retrieves a float64 flag value (safe access)
func (c *Context) Float(name string) (float64, bool) {
	return c.Result.GetFloat(name)
}

// MustFloat retrieves a float64 flag value with default fallback
func (c *Context) MustFloat(name string, defaultValue float64) float64 {
	return c.Result.MustGetFloat(name, defaultValue)
}

// Enum retrieves an enum flag value (safe access)
func (c *Context) Enum(name string) (string, bool) {
	return c.Result.GetEnum(name)
}

// MustEnum retrieves an enum flag value with default fallback
func (c *Context) MustEnum(name, defaultValue string) string {
	return c.Result.MustGetEnum(name, defaultValue)
}

// StringSlice retrieves a string slice flag value (safe access)
func (c *Context) StringSlice(name string) ([]string, bool) {
	return c.Result.GetStringSlice(name)
}

// MustStringSlice retrieves a string slice flag value with default fallback
func (c *Context) MustStringSlice(name string, defaultValue []string) []string {
	return c.Result.MustGetStringSlice(name, defaultValue)
}

// IntSlice retrieves an int slice flag value (safe access)
func (c *Context) IntSlice(name string) ([]int, bool) {
	return c.Result.GetIntSlice(name)
}

// MustIntSlice retrieves an int slice flag value with default fallback
func (c *Context) MustIntSlice(name string, defaultValue []int) []int {
	return c.Result.MustGetIntSlice(name, defaultValue)
}

// Global flag access methods

// GlobalString retrieves a global string flag value (safe access)
func (c *Context) GlobalString(name string) (string, bool) {
	return c.Result.GetGlobalString(name)
}

// MustGlobalString retrieves a global string flag value with default fallback
func (c *Context) MustGlobalString(name, defaultValue string) string {
	return c.Result.MustGetGlobalString(name, defaultValue)
}

// GlobalInt retrieves a global int flag value (safe access)
func (c *Context) GlobalInt(name string) (int, bool) {
	return c.Result.GetGlobalInt(name)
}

// MustGlobalInt retrieves a global int flag value with default fallback
func (c *Context) MustGlobalInt(name string, defaultValue int) int {
	return c.Result.MustGetGlobalInt(name, defaultValue)
}

// GlobalBool retrieves a global bool flag value (safe access)
func (c *Context) GlobalBool(name string) (bool, bool) {
	return c.Result.GetGlobalBool(name)
}

// MustGlobalBool retrieves a global bool flag value with default fallback
func (c *Context) MustGlobalBool(name string, defaultValue bool) bool {
	return c.Result.MustGetGlobalBool(name, defaultValue)
}

// GlobalDuration retrieves a global duration flag value (safe access)
func (c *Context) GlobalDuration(name string) (time.Duration, bool) {
	return c.Result.GetGlobalDuration(name)
}

// GlobalFloat retrieves a global float flag value (safe access)
func (c *Context) GlobalFloat(name string) (float64, bool) {
	return c.Result.GetGlobalFloat(name)
}

// GlobalEnum retrieves a global enum flag value (safe access)
func (c *Context) GlobalEnum(name string) (string, bool) {
	return c.Result.GetGlobalEnum(name)
}

// GlobalStringSlice retrieves a global string slice flag value (safe access)
func (c *Context) GlobalStringSlice(name string) ([]string, bool) {
	return c.Result.GetGlobalStringSlice(name)
}

// GlobalIntSlice retrieves a global int slice flag value (safe access)
func (c *Context) GlobalIntSlice(name string) ([]int, bool) {
	return c.Result.GetGlobalIntSlice(name)
}

// Positional argument access methods

// ArgString retrieves a string positional argument value (safe access)
func (c *Context) ArgString(name string) (string, bool) {
	return c.Result.GetArgString(name)
}

// MustArgString retrieves a string positional argument value with default fallback
func (c *Context) MustArgString(name, defaultValue string) string {
	return c.Result.MustGetArgString(name, defaultValue)
}

// ArgInt retrieves an int positional argument value (safe access)
func (c *Context) ArgInt(name string) (int, bool) {
	return c.Result.GetArgInt(name)
}

// MustArgInt retrieves an int positional argument value with default fallback
func (c *Context) MustArgInt(name string, defaultValue int) int {
	return c.Result.MustGetArgInt(name, defaultValue)
}

// ArgBool retrieves a bool positional argument value (safe access)
func (c *Context) ArgBool(name string) (bool, bool) {
	return c.Result.GetArgBool(name)
}

// MustArgBool retrieves a bool positional argument value with default fallback
func (c *Context) MustArgBool(name string, defaultValue bool) bool {
	return c.Result.MustGetArgBool(name, defaultValue)
}

// ArgDuration retrieves a duration positional argument value (safe access)
func (c *Context) ArgDuration(name string) (time.Duration, bool) {
	return c.Result.GetArgDuration(name)
}

// MustArgDuration retrieves a duration positional argument value with default fallback
func (c *Context) MustArgDuration(name string, defaultValue time.Duration) time.Duration {
	return c.Result.MustGetArgDuration(name, defaultValue)
}

// ArgFloat retrieves a float64 positional argument value (safe access)
func (c *Context) ArgFloat(name string) (float64, bool) {
	return c.Result.GetArgFloat(name)
}

// MustArgFloat retrieves a float64 positional argument value with default fallback
func (c *Context) MustArgFloat(name string, defaultValue float64) float64 {
	return c.Result.MustGetArgFloat(name, defaultValue)
}

// ArgStringSlice retrieves a string slice positional argument value (variadic args)
func (c *Context) ArgStringSlice(name string) ([]string, bool) {
	return c.Result.GetArgStringSlice(name)
}

// MustArgStringSlice retrieves a string slice positional argument value with default fallback
func (c *Context) MustArgStringSlice(name string, defaultValue []string) []string {
	return c.Result.MustGetArgStringSlice(name, defaultValue)
}

// ArgIntSlice retrieves an int slice positional argument value (variadic args)
func (c *Context) ArgIntSlice(name string) ([]int, bool) {
	return c.Result.GetArgIntSlice(name)
}

// MustArgIntSlice retrieves an int slice positional argument value with default fallback
func (c *Context) MustArgIntSlice(name string, defaultValue []int) []int {
	return c.Result.MustGetArgIntSlice(name, defaultValue)
}

// Arg retrieves a raw positional argument by index (0-based)
// Returns empty string if index is out of bounds
func (c *Context) Arg(index int) string {
	if index >= 0 && index < len(c.Result.Args) {
		return c.Result.Args[index]
	}
	return ""
}

// RestArgs returns remaining positional arguments when using RestArgs()
// Returns empty slice if RestArgs() was not configured
func (c *Context) RestArgs() []string {
	return c.Result.RestArgs
}

// Command and argument access

// Command returns the executed command (implements middleware.Context interface)
func (c *Context) Command() middleware.Command {
	return c.Result.Command
}

// Args returns positional arguments (non-flag arguments after parsing)
func (c *Context) Args() []string {
	return c.Result.Args
}

// RawArgs returns the original unparsed arguments as passed to RunWithArgs.
// This includes all flags, commands, and arguments before any parsing.
// The binary name (os.Args[0]) is NOT included.
//
// Example: if invoked as "myapp --verbose serve --port 8080 file.txt"
// RawArgs() returns ["--verbose", "serve", "--port", "8080", "file.txt"]
// Args() returns ["file.txt"] (only positional args after parsing)
func (c *Context) RawArgs() []string {
	return c.App.rawArgs
}

// CurrentBinary returns the binary currently being executed in a WrapMany() scenario.
// Returns empty string for single Wrap() or if not in a wrapper execution.
//
// Example:
//
//	app.Command("build").WrapMany("go1.21", "go1.22").
//	    AfterExec(func(ctx *Context, result *ExecResult) error {
//	        binary := ctx.CurrentBinary() // "go1.21" or "go1.22"
//	        return nil
//	    })
func (c *Context) CurrentBinary() string {
	return c.currentBinary
}

// Binaries returns all binaries configured in WrapMany().
// Returns nil for single Wrap() configurations.
//
// Example:
//
//	app.Command("build").WrapMany("go1.21", "go1.22", "go1.23").
//	    AfterExec(func(ctx *Context, result *ExecResult) error {
//	        binaries := ctx.Binaries() // ["go1.21", "go1.22", "go1.23"]
//	        return nil
//	    })
func (c *Context) Binaries() []string {
	return c.binaries
}

// NArgs returns the number of positional arguments
func (c *Context) NArgs() int {
	return len(c.Result.Args)
}

// WrapperResult returns the last ExecResult produced by a wrapper when running
// with Capture() or CaptureTo(). It returns (nil, false) if no result is
// available.
func (c *Context) WrapperResult() (*ExecResult, bool) {
	v := c.Get("__wrapper_result__")
	if r, ok := v.(*ExecResult); ok {
		return r, true
	}
	return nil, false
}

// App metadata accessors

// AppName returns the application name
func (c *Context) AppName() string {
	return c.App.name
}

// AppVersion returns the application version (empty string if not set)
func (c *Context) AppVersion() string {
	return c.App.version
}

// AppDescription returns the application description
func (c *Context) AppDescription() string {
	return c.App.description
}

// AppAuthors returns the list of application authors
func (c *Context) AppAuthors() []Author {
	return c.App.authors
}

// Logging methods - convenient access to structured logging

// LogDebug logs a debug message (purple circle by default)
func (c *Context) LogDebug(format string, args ...any) {
	c.App.Logger().Debug(format, args...)
}

// LogInfo logs an informational message (blue circle by default)
func (c *Context) LogInfo(format string, args ...any) {
	c.App.Logger().Info(format, args...)
}

// LogSuccess logs a success message (green circle by default)
func (c *Context) LogSuccess(format string, args ...any) {
	c.App.Logger().Success(format, args...)
}

// LogWarning logs a warning message (yellow circle by default)
func (c *Context) LogWarning(format string, args ...any) {
	c.App.Logger().Warning(format, args...)
}

// LogError logs an error message (red circle by default)
func (c *Context) LogError(format string, args ...any) {
	c.App.Logger().Error(format, args...)
}
