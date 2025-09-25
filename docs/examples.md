# Examples Tour

This repository ships several runnable examples under `examples/`.

Basic CLI
- `examples/basic-cli/main.go`: App, global vs command flags, help/version, args.

Configuration precedence
- `examples/config-precedence/main.go`: Struct tags, FromDefaults/FromEnv/FromFile/FromFlags, precedence, nested groups, enums.

Flag groups
- `examples/flag-groups/main.go`: Mutually exclusive and all-or-none groups with contextual help.

Middleware
- `examples/middleware/main.go`: Logger, Recovery, Timeout, Validator (including custom validators).

Exit codes
- `examples/exit-codes/main.go`: ExitCodeManager mappings and explicit exits.

IO demo
- `examples/io-demo/main.go`: IO/TTY detection and color helpers.

Wrapper DSL
- `examples/wrapper-echo/main.go`: Echo wrapper with preserved leading flags.
- `examples/wrapper-go-build/main.go`: `go build` wrapper with dynamic toolexec shim.
- `examples/nested-subcommands/main.go`: Nested commands (e.g., `myapp server up`).

Smart errors
- `examples/smart_errors/error_demo.go`: Error suggestions and handler tuning.

Run (Smart Errors)
```bash
go run ./examples/smart_errors --jsno
```

Tip: you can run examples with `go run ./examples/<name>`.

Related
- [Quick Start](./quickstart.md)
- [Wrapper DSL](./wrapper.md)
- [Middleware](./middleware.md)
