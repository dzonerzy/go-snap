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
- `Before(fn ActionFunc) *CommandBuilder` (runs before command action)
- `After(fn ActionFunc) *CommandBuilder` (runs after command action)
- `Hidden() *CommandBuilder`
- `HelpText(string) *CommandBuilder`
- `Use(middleware ...middleware.Middleware) *CommandBuilder`
- `Command(name, description string) *CommandBuilder` (subcommands)
- Flag methods (typed) – see Flags & Groups

Nested subcommands
```go
app := snap.New("myapp", "demo")

srv := app.Command("server", "Server management")
srv.Command("up", "Start the server").
    BoolFlag("dry-run", "Print the action only").Short('n').Back().
    Action(func(ctx *snap.Context) error { /* ... */ return nil })

srv.Command("down", "Stop the server").
    BoolFlag("force", "Force stop").Short('f').Back().
    Action(func(ctx *snap.Context) error { /* ... */ return nil })

// Invocations:
//   myapp server up --dry-run
//   myapp server down --force
```

Tip: returning to the parent builder
- The fluent builders use `Back()` to return to the parent context after finishing a flag definition. This makes chaining explicit and predictable.
- Example: `BoolFlag("force", "").Short('f').Back()` defines the flag, sets a short alias, then returns to the command builder for more methods.

Help & Version
- App automatically provides `--help` unless `DisableHelp()` is used
- If `Version()` is set, `--version` is handled at all levels
- Command-specific `--help` is injected for every command

Execution lifecycle
1) Parse args (smart errors, suggestions, grouping validation)
2) Build `*snap.Context` with cancellation
3) Run `App.Before`
4) Run `Command.Before` (if set on the active command)
5) Run action (app/command middleware applied)
6) Run `Command.After` (if set on the active command)
7) Apply `Context` exit semantics if set
8) Run `App.After`

Command lifecycle hooks (Before/After)

Commands support `Before` and `After` hooks for setup and cleanup:

```go
app.Command("deploy", "Deploy application").
    StringFlag("env", "Target environment").Default("staging").Back().
    Before(func(ctx *snap.Context) error {
        env, _ := ctx.String("env")
        fmt.Printf("Validating credentials for %s...\n", env)
        // Setup: validate credentials, check prerequisites
        return nil
    }).
    Action(func(ctx *snap.Context) error {
        env, _ := ctx.String("env")
        fmt.Printf("Deploying to %s...\n", env)
        return nil
    }).
    After(func(ctx *snap.Context) error {
        env, _ := ctx.String("env")
        fmt.Printf("Deployment to %s complete, sending notifications...\n", env)
        // Cleanup: send notifications, log metrics
        return nil
    })
```

Hook behavior:
- `Before` runs before the action. If it returns an error, the action and `After` are skipped.
- `After` runs after the action completes, even if the action returns an error.
- If `After` returns an error and the action succeeded, the `After` error is returned.
- Hooks combine with app-level `Before`/`After`: `App.Before` → `Command.Before` → Action → `Command.After` → `App.After`

Notes
- When no command is provided, the app shows help unless an app-level wrapper is configured (see Wrapper DSL).
- Help output is deterministic and grouped when flag groups are present.
- Command-level hooks are only called for the active command (deepest in the command chain).

Related
- [Flags & Groups](./flags-and-groups.md)
- [Middleware](./middleware.md)
- [Wrapper DSL](./wrapper.md)
- [Errors & Exit Codes](./errors-and-exit-codes.md)
