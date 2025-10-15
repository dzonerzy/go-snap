# Parsing and Context

Zero-allocation parser
- The parser in `snap/parser.go` uses pooled structures, byte math and string interning to avoid allocations on the hot path.
- Supports `--flag=value`, `--flag value`, short flags (`-v`, combined `-abc`), `--` terminator for positional args.
- Unknown flag/command errors include edit-distance suggestions.

ParseResult accessors (implemented)
- Per-type getters: `GetString`, `GetInt`, `GetBool`, `GetDuration`, `GetFloat`, `GetEnum`, `GetStringSlice`, `GetIntSlice`
- Global variants: `GetGlobalString`, `GetGlobalInt`, `GetGlobalBool`, `GetGlobalDuration`, `GetGlobalFloat`, `GetGlobalEnum`, `GetGlobalStringSlice`, `GetGlobalIntSlice`
- Must* with default: `MustGetString`, `MustGetInt`, `MustGetBool`, `MustGetDuration`, `MustGetFloat`, `MustGetEnum`, `MustGetStringSlice`, `MustGetIntSlice` and global counterparts
- `HasFlag`, `HasGlobalFlag`
- `Args []string`, `Command *Command`

Context API (`snap/context.go`)
- `Context()` / `Done()` / `Cancel()` / `Err()` – propagation and cancellation
- `Set(key, val)`, `Get(key)` – metadata
- Flag helpers mirror ParseResult: `String/Int/Bool/Duration/Float/Enum`, `StringSlice/IntSlice`, global variants
- Positional args: `Args()`, `NArgs()`, `Arg(i)`
- IO: `IO()`, `Stdout()`, `Stderr()`, `Stdin()`
- Exit helpers: `Exit(code)`, `ExitWithError(err, code)`, `ExitOnError(err)`
- Wrapper result: `WrapperResult() (*ExecResult, bool)`
- App metadata: `AppName()`, `AppVersion()`, `AppDescription()`, `AppAuthors()`

App metadata access

Use `AppName()`, `AppVersion()`, `AppDescription()`, and `AppAuthors()` to access application metadata from within actions:

```go
app := snap.New("myapp", "My awesome CLI tool").
    Version("1.2.3").
    Author("Alice", "alice@example.com")

app.Command("version", "Show version information").
    Action(func(ctx *snap.Context) error {
        fmt.Fprintf(ctx.Stdout(), "%s v%s\n", ctx.AppName(), ctx.AppVersion())
        
        authors := ctx.AppAuthors()
        if len(authors) > 0 {
            fmt.Fprintf(ctx.Stdout(), "Authors:\n")
            for _, author := range authors {
                fmt.Fprintf(ctx.Stdout(), "  %s <%s>\n", author.Name, author.Email)
            }
        }
        return nil
    })
```

Output:
```
myapp v1.2.3
Authors:
  Alice <alice@example.com>
```

Help/version handling
- Global `--help/--version` handled if enabled on app.
- Command-level `--help` always available.
- Short alias `-h` is provided by default for help at both app and command level if not already taken by another flag. If you bind `-h` yourself, your flag wins and help remains available via `--help`.

Notes
- Float parsing in CLI path is implemented for common cases; env parsing of floats uses `strconv.ParseFloat`.

Related
- [Errors & Exit Codes](./errors-and-exit-codes.md)
- [Flags & Groups](./flags-and-groups.md)
