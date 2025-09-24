# Middleware

The `middleware` package provides a small, focused set of built-ins and a simple interface:

Types
- `type Context` (implemented by `*snap.Context`)
- `type ActionFunc func(ctx Context) error`
- `type Middleware func(next ActionFunc) ActionFunc`
- `type MiddlewareChain []Middleware` with `Apply`/`Use`

Applying middleware
```go
app := snap.New("svc", "demo").
    Use(middleware.Logger(), middleware.Recovery(), middleware.Timeout(30*time.Second))

app.Command("serve", "Start").
    Use(middleware.Validate(middleware.Custom("port_range", validatePort))).
    Action(run)
```

Logger
- `Logger(options ...)`
- Helpers: `DebugLogger()`, `InfoLogger()`, `ErrorLogger()`, `JSONLogger()`, `SilentLogger()`
- Options (`WithLogLevel`, `WithTimeout` etc. via config where applicable)

Recovery
- `Recovery(options ...)`
- `RecoveryWithHandler(handler)`
- `RecoveryToError()` (no stack), `RecoveryWithStack()` (print stack), `MustRecover()`, `SafeRecovery()`
- `RecoveryWithStats(stats, options ...)`

Timeout
- `Timeout(duration)`
- `TimeoutWithDefault(options ...)`
- `TimeoutWithGracefulShutdown(timeout, gracePeriod)`
- `TimeoutPerCommand(map[string]time.Duration, default)`
- `TimeoutWithCallback(duration, onTimeout)`
- `TimeoutWithRetry(duration, maxRetries)`
- `NoTimeout()`
- `DynamicTimeout(func(ctx Context) time.Duration)`
 - `TimeoutFromFlag(flagName string, default time.Duration)`
 - `TimeoutWithStats(duration time.Duration, stats *TimeoutStats)`

Validator
- `Validator(options ...)`
- `ValidatorWithCustom(map[string]ValidatorFunc)`
- Shorthand: `Validate(NamedValidator...)`
- Helpers: `Custom(name, fn)`, `File(flagNames...)`, `Dir(flagNames...)`,
  `FileExists`, `DirectoryExists`, `ConditionalRequired`, `FileSystemValidator`, `NoopValidator`

Example
```go
app := snap.New("server", "demo").
    Use(
        middleware.Logger(middleware.WithLogLevel(middleware.LogLevelInfo)),
        middleware.Recovery(middleware.WithStackTrace(true)),
        middleware.Timeout(30*time.Second),
        middleware.Validate(
            middleware.Custom("port_range", validatePortRange),
        ),
    )
```

Related
- [App & Commands](./app-and-commands.md)
- [Errors & Exit Codes](./errors-and-exit-codes.md)
