# Changelog

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

