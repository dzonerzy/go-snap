# Errors and Exit Codes

Smart errors
- Parser produces `*ParseError` with type (unknown flag/command, invalid/missing value, group violation).
- `App` converts parse errors into `*CLIError` and uses `ErrorHandler` to add suggestions/context.
- Suggestions use internal fuzzy matching over known flags/commands.

ErrorHandler configuration
```go
app.ErrorHandler().
    SuggestFlags(true).
    SuggestCommands(true).
    MaxDistance(2).
    Handle(snap.ErrorTypeValidation, func(e *snap.CLIError) *snap.CLIError { return e })
```

Group violations
- Errors of type `flag_group_violation` include contextual help rendering for the offending group.

Exit codes
- `ExitError{Code int, Err error}` for explicit exit from actions (via `Context.Exit*`).
- `ExitCodeManager` precedence:
  1) `ExitError` requested code
  2) `*CLIError` category mapping (`DefineCLI`)
  3) Concrete error type mapping (`DefineError`)
  4) Defaults (`ExitCodeDefaults`)

Defaults
- Success: 0
- GeneralError: 1
- MisusageError: 2 (unknown flag/command, group violations, missing required)
- ValidationError: 3
- PermissionError: 126
- NotFoundError: 127

API (implemented)
- `App.ExitCodes() *ExitCodeManager`
- `(*ExitCodeManager) Define(name string, code int)`
- `(*ExitCodeManager) DefineError(err error, code int)`
- `(*ExitCodeManager) DefineCLI(typ ErrorType, code int)`
- `(*ExitCodeManager) Default(ExitCodeDefaults)`
- `Context.Exit(code)`, `ExitWithError(err, code)`, `ExitOnError(err)`
- `App.RunAndGetExitCode()`, `App.RunAndExit()`

Example
```go
var ErrNotFound = errors.New("resource not found")

app.ExitCodes().
    Define("not_found", 127).
    DefineError(ErrNotFound, 127)

app.Command("custom-exit", "exit code 42").Action(func(ctx *snap.Context) error {
    ctx.Exit(42)
    return nil
})
```

Related
- [Parsing & Context](./parsing-and-context.md)
- [App & Commands](./app-and-commands.md)
