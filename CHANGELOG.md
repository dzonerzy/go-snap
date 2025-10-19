# Changelog

## [0.2.2] - 2025-01-20

### Fixed
- **Logger color handling**: `LogFormatPlain` now correctly applies semantic colors while omitting prefixes
  * Previously returned uncolored text in plain format
  * Plain format now means "no prefix" not "no color"
  * Debug messages still purple, info blue, success green, warning yellow, error red
- **Code quality improvements**: Resolved all golangci-lint issues
  * Extracted duplicate argument printing logic into `printArgumentsSection()` helper method
  * Fixed variable shadowing in `io_unix.go` color detection
  * Fixed built-in `cap` redefinition in color capability detection
  * Simplified argument width calculation to eliminate duplicate branches
  * Reduced cyclomatic complexity in help rendering functions

## [0.2.1] - 2025-01-19

### Added
- **Structured logging system** with semantic log levels and customizable formatting
  * Context methods: `ctx.LogDebug()`, `ctx.LogInfo()`, `ctx.LogSuccess()`, `ctx.LogWarning()`, `ctx.LogError()`
  * Default format: Colored circle emoji (🟣 debug, 🔵 info, 🟢 success, 🟡 warning, 🔴 error)
  * Built-in formats: `LogFormatCircles` (default), `LogFormatSymbols` (● ◆ ✓ ▲ ✗), `LogFormatTagged` ([INFO] [WARN]), `LogFormatPlain`
  * Custom templates with `WithTemplate()` supporting `{{.Level}}`, `{{.Time}}`, `{{.Message}}`, `{{.Prefix}}`
  * Per-level prefix customization with `SetPrefix(level, prefix)`
  * Optional timestamps with `WithTimestamp(true)` and custom time formats
  * Automatic color coding: light purple (debug), cyan (info), green (success), yellow (warning), red (error)
  * **Smart theme selection**: automatically selects optimal theme based on terminal color capability
    - ColorLevel 1 (16 colors): `DefaultTheme16()` with BrightMagenta for debug
    - ColorLevel 2 (256 colors): `DefaultTheme256()` with Indexed(141) light purple for debug
    - ColorLevel 3 (truecolor): `DefaultThemeTruecolor()` with RGB(189, 147, 249) light purple for debug
  * Smart output routing: errors/warnings to stderr, info/success/debug to stdout
  * Full integration with existing `IOManager` and `Theme` system
  * Theme functions: `DefaultTheme(io)`, `DefaultTheme16()`, `DefaultTheme256()`, `DefaultThemeTruecolor()`
  * New example: `examples/logging-demo` demonstrating all formats and customization options
  * Color test utility: `examples/logging-demo/color-test.go` to preview color options
  * App-level access via `app.Logger()` for advanced configuration

- README.md documentation for previously undocumented examples:
  * `examples/version-command` - Custom version commands with Context metadata accessors
  * `examples/raw-args-demo` - RawArgs() usage for audit logging and debugging
  * `examples/smart_errors` - Smart error handling with fuzzy matching suggestions
  * `examples/lifecycle-hooks` - Before/After and BeforeExec/AfterExec hooks
  * `examples/demo` - Comprehensive help system demonstration

### Added
- `IOManager.ForceColorLevel(level)` method to manually override color level detection (0=none, 1=16, 2=256, 3=truecolor)
- **Platform-specific color capability detection** via terminfo queries (`tput colors`, `tput RGB`) on Unix/Linux/WSL for accurate terminal capability detection beyond environment variables
- **Comprehensive color palette** with normal/bright variants across all color modes:
  * 16-color: `Black`, `Red`, `Green`, etc. with `Bright*` variants
  * 256-color: Extended palette including `LightPurple`, `Orange`, `SkyBlue`, `Gray1-6`, etc.
  * Truecolor: RGB colors with `True*` prefix including `TrueLightPurple`, `TrueBrightRed`, `TrueSkyBlue`, etc.

### Changed
- **Logger symbol format**: Replaced emoji with pure Unicode symbols in `LogFormatSymbols` for consistency
  * Info: Changed from ℹ️ (emoji) to ◆ (diamond symbol)
  * Warning: Changed from ⚠️ (emoji) to ▲ (triangle symbol)
  * All symbols are now geometric Unicode characters (●◆✓▲✗) with no emoji

### Fixed
- **Logger prefix skipping**: Prefixes (emoji/symbols/tags) are now skipped for empty or whitespace-only messages, allowing clean visual spacing in logs
- **Truecolor detection on Windows**: `ColorLevel()` now correctly returns 3 (truecolor) on Windows terminals with VT support enabled, instead of limiting to 2 (256 colors). Modern Windows terminals (Windows Terminal, VS Code, PowerShell) now automatically use TruecolorTheme
- **Enhanced truecolor detection**: Added support for `COLORTERM=24bit` and `TERM` variants containing "truecolor" or "24bit"
- **IDE terminal detection**: Added support for Zed and VS Code terminals via `TERM_PROGRAM` environment variable (cross-platform)
- **Unix/WSL terminal detection**: Added `tput` queries to detect actual terminal color capability, improving accuracy over environment variables alone
- **Help output alignment** now uses spaces-only instead of tabs for consistent formatting across all terminals
  * Fixed `flagDisplayWidth()` calculation bug (was using 2 instead of 4 for "  --" prefix)
  * Flag descriptions now properly aligned at both app-level and command-level
  * Subcommand descriptions now properly aligned with consistent spacing
  * Positional argument descriptions now properly aligned
  * All help sections (flags, commands, subcommands, arguments) use uniform space-based alignment
  * Minimum 2-space separation ensures visual clarity even for longest names
  * Eliminates tab stop issues that caused inconsistent spacing in different terminal environments

## [0.2.0] - 2025-10-16

### Added
- **Type-safe positional arguments** with `StringArg()`, `IntArg()`, `BoolArg()`, `FloatArg()`, `DurationArg()`, `StringSliceArg()`, `IntSliceArg()`
  * Incremental declaration: first arg is position 0, second is position 1, etc.
  * Support at both app-level and command-level
  * Required vs optional arguments with `.Required()` and `.Default(value)`
  * Access via `ctx.String("argname")`, `ctx.MustString("argname", default)` (same pattern as flags)
  * Automatic type conversion and validation
  * Help text integration: displays `<required>` and `[optional]` args in usage
  * Validation: enforces required arg count before action execution
  * Example: `app.StringArg("filename", "Input file").Required().IntArg("count", "Number of items").Default(10)`

- **Variadic positional arguments** with `StringSliceArg().Variadic()` and `IntSliceArg().Variadic()`
  * Collect multiple values for the last positional argument
  * Type-safe: `ctx.StringSlice("files")` returns `[]string`, `ctx.IntSlice("ports")` returns `[]int`
  * Can be required (1+ items) or optional with default (0+ items)
  * Help shows as `<files>...` for required or `[files]...` for optional
  * Only last argument can be variadic
  * Example: `app.Command("rm").StringSliceArg("files", "Files to remove").Required().Variadic()`
  * Use cases: `rm file1 file2 file3`, `tar -czf out.tar file1 file2 file3`, `listen 8080 8081 8082`

- **Rest arguments** with `RestArgs()` for pass-through scenarios
  * Captures all remaining arguments after declared positional args as raw strings
  * Access via `ctx.RestArgs()` or `ctx.Args()` (returns `[]string`)
  * No validation or type conversion - raw pass-through
  * Ideal for docker-style commands or forwarding to wrapped binaries
  * Cannot combine with `.Variadic()` - choose one approach
  * Example: `app.Command("run").StringArg("script", "Script").Required().RestArgs()`
  * Use cases: `docker run image cmd args...`, `go run main.go args...`

- Context methods for positional arguments:
  * `ctx.Arg(index)` - Get raw positional arg by index
  * `ctx.RestArgs()` - Get all remaining args when using `RestArgs()`
  * `ctx.StringArg()`, `ctx.IntArg()`, `ctx.BoolArg()`, `ctx.FloatArg()`, `ctx.DurationArg()` - Type-safe accessors
  * `ctx.StringSliceArg()`, `ctx.IntSliceArg()` - Slice accessors for variadic arguments
  * Existing `ctx.Args()` now returns only unparsed positional args (after named args consumed)

- **Fluent API chaining** with `.Back()` method for ArgBuilder
  * Consistent with FlagBuilder pattern
  * Example: `app.StringArg("source", "Source file").Required().Back().StringArg("dest", "Destination").Default("out.txt")`

- **App-level action** with `app.Action(fn)` method
  * Execute action when no command is matched
  * Falls back to help if no action defined
  * Enables standalone app behavior without requiring commands

- **Zero-allocation parsing** for positional arguments
  * Uses typed maps (ArgStrings, ArgInts, etc.) to avoid interface{} boxing
  * Pooled slices for variadic arguments
  * Slice offset pattern for zero-alloc slice access
  * Benchmarks: 0 B/op, 0 allocs/op maintained

- New examples demonstrating positional arguments:
  * `examples/positional-args` - Basic positional argument usage with all types
  * `examples/variadic-args` - Variadic arguments and RestArgs pass-through

### Changed
- Help output now displays positional arguments in usage line:
  * Named args: `myapp <filename> [count]`
  * Variadic args: `myapp rm <files>...`
  * Rest args: `myapp run <script> [args...]`
- Help output includes "Arguments:" section with descriptions
- Help flag (`--help`) now has priority over required argument validation
  * `--help` works even when required args are missing
  * Allows users to see help before providing all required arguments
- Parser validates positional argument count against declared requirements
- `ctx.Args()` behavior: returns remaining positional args after named args are consumed
- ArgBuilder now uses two type parameters `ArgBuilder[T, P]` matching FlagBuilder pattern
- `.Required()` and `.Validate()` return `*ArgBuilder[T, P]` for chaining
- `.Default()` and `.Variadic()` return parent type `P` to complete chain

## [0.1.3] - 2025-10-16

### Added
- `WrapMany()` method for executing multiple binaries with same arguments
- `Parallel()` method for concurrent execution of multiple binaries (default: sequential)
- `StopOnError()` method to control error handling (default: stop on first error)
- `CurrentBinary()` and `Binaries()` Context methods for accessing binary information in WrapMany
- New example: `examples/multi-go-build` demonstrating multi-version builds
- Competitive benchmarks comparing go-snap vs Cobra vs urfave/cli
  * 4-10x faster execution (4.9x vs Cobra, 6.7x vs urfave/cli average)
  * 3-3.4x less memory usage
  * 3.6-17x fewer allocations
- Benchmark documentation: `benchmark/COMPETITIVE_BENCHMARKS.md` with detailed analysis

### Fixed
- Flag descriptions now properly aligned in help output regardless of flag name and option lengths

### Changed
- Modernized for loops to use `range` over int syntax (Go 1.22+)

## [0.1.2] - 2025-10-16

### Added
- Command lifecycle hooks: `Before()` and `After()` methods for command-level setup/teardown logic
- Wrapper lifecycle hooks: `BeforeExec()` and `AfterExec()` for enhanced wrapper control and result processing
- Context app metadata accessors: `AppName()`, `AppVersion()`, `AppDescription()`, `AppAuthors()`
- Context `RawArgs()` method to access original unparsed arguments before parsing
- New example: `examples/lifecycle-hooks` demonstrating Before/After and BeforeExec/AfterExec usage
- New example: `examples/version-command` showing custom version commands with app metadata
- New example: `examples/raw-args-demo` demonstrating audit logging, debugging, and proxying use cases

### Changed
- Command execution order now includes Before/After hooks: App.Before → Command.Before → Action → Command.After → App.After
- Wrapper execution includes BeforeExec (after transformations) and AfterExec (with ExecResult)
- Documentation updated with comprehensive lifecycle hook examples and execution diagrams

### Fixed
- Command descriptions now properly aligned in help output regardless of command name length

## [0.1.1] - 2025-09-25

### Added
- Default short alias `-h` for help at app and command levels when not already taken.
- ErrorHandler option `ShowHelpOnError(true)` to print contextual help (app/command) after errors.
- New example: `examples/nested-subcommands` (server up/down) with README.
- CI: integrate golangci-lint and add `.golangci.yml` configuration.

### Changed
- Show command help when a command has no action or wrapper (e.g., a container for subcommands).
- Documentation: nested subcommands, single-letter aliases, parsing notes; linked the new example.

### Fixed
- Unknown subcommand surfaced and suggestions improved (prefers child suggestions).
