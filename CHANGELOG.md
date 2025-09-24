# Changelog

All notable changes to this project will be documented in this file.

The format is inspired by Keep a Changelog. This project follows semantic versioning as the API stabilizes.

## [0.1.0] - 2025-09-24
### Added
- Core `snap` package with fluent `App`, `CommandBuilder`, `FlagBuilder`, and `FlagGroupBuilder`.
- Zero-allocation parser with typed storage, env/default application, and group validation.
- Configuration builder with struct tags (`flag`, `env`, `default`, `description`, `enum`, `group`, `ignore`, `group_constraint`, `group_description`), precedence (flags > env > file > defaults), and `FromFlags()` auto flag generation.
- Middleware package: Logger (text/JSON), Recovery (variants + stats), Timeout (variants + dynamic/from flag + stats), Validator (helpers and composition).
- IO package (`snapio`): TTY/size detection, ANSI helpers, styles, Windows VT support.
- Wrapper DSL: app/command-level `Wrap`, `WrapDynamic` (toolexec shim), transforms, forwarding, tee/capture, allow-list.
- Error handling: `CLIError` + `ErrorHandler` with fuzzy suggestions and contextual group help.
- Exit code management with `ExitCodeManager` and `Context` exit helpers.
- Comprehensive docs in `docs/`, plus runnable examples in `examples/`.

### Notes
- Stable early beta focused on the core features. Please use [GitHub Discussions](https://github.com/dzonerzy/go-snap/discussions) to propose new features or share ideas.

[0.1.0]: https://github.com/dzonerzy/go-snap/releases/tag/v0.1.0
