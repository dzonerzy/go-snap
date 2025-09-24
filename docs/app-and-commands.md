# App & Commands

The `snap` package exposes a fluent, type-safe builder for defining your CLI.

Constructing an app
```go
app := snap.New("myapp", "Short description").
    Version("1.2.3").
    Author("Acme", "dev@acme.test")
```

Core App methods (implemented)
- `Version(string) *App`
- `Author(name, email string) *App`
- `Authors(authors ...Author) *App`
- `HelpText(string) *App`
- `Use(middleware ...middleware.Middleware) *App`
- `DisableHelp() *App` (disables built-in `--help`)
- `Before(fn ActionFunc) *App`
- `After(fn ActionFunc) *App`
- `IO() *snapio.IOManager`
- `Run() error`
- `RunContext(ctx context.Context) error`
- `RunWithArgs(ctx context.Context, args []string) error`
- `RunAndGetExitCode() int`
- `RunAndExit()`
- `ExitCodes() *ExitCodeManager`

Commands
```go
app.Command("serve", "Start HTTP server").
    Alias("run").
    HelpText("Starts the server with defaults").
    Use(middleware.Logger()).
    IntFlag("port", "Port").Default(8080).Back().
    Action(func(ctx *snap.Context) error { /* ... */ return nil })
```

CommandBuilder methods (implemented)
- `Alias(...string) *CommandBuilder`
- `Action(fn ActionFunc) *CommandBuilder`
- `Hidden() *CommandBuilder`
- `HelpText(string) *CommandBuilder`
- `Use(middleware ...middleware.Middleware) *CommandBuilder`
- `Command(name, description string) *CommandBuilder` (subcommands)
- Flag methods (typed) – see Flags & Groups

Help & Version
- App automatically provides `--help` unless `DisableHelp()` is used
- If `Version()` is set, `--version` is handled at all levels
- Command-specific `--help` is injected for every command

Execution lifecycle
1) Parse args (smart errors, suggestions, grouping validation)
2) Build `*snap.Context` with cancellation
3) Run `App.Before`
4) Run action (app/command middleware applied)
5) Apply `Context` exit semantics if set
6) Run `App.After`

Notes
- When no command is provided, the app shows help unless an app-level wrapper is configured (see Wrapper DSL).
- Help output is deterministic and grouped when flag groups are present.

Related
- [Flags & Groups](./flags-and-groups.md)
- [Middleware](./middleware.md)
- [Wrapper DSL](./wrapper.md)
- [Errors & Exit Codes](./errors-and-exit-codes.md)
