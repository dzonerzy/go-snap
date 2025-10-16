# Changelog

## [Unreleased] - v0.2.0

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
  * Existing `ctx.Args()` now returns only unparsed positional args (after named args consumed)

### Changed
- Help output now displays positional arguments in usage line:
  * Named args: `myapp <filename> [count]`
  * Variadic args: `myapp rm <files>...`
  * Rest args: `myapp run <script> [args...]`
- Parser validates positional argument count against declared requirements
- `ctx.Args()` behavior: returns remaining positional args after named args are consumed

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

