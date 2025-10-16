# Changelog

## [0.1.3] - 2025-10-16

### Added
- `WrapMany()` method for executing multiple binaries with same arguments
- `Parallel()` method for concurrent execution of multiple binaries (default: sequential)
- `StopOnError()` method to control error handling (default: stop on first error)
- `CurrentBinary()` and `Binaries()` Context methods for accessing binary information in WrapMany
- New example: `examples/multi-go-build` demonstrating multi-version builds

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

