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
- Positional args: `Args()`, `RawArgs()`, `NArgs()`, `Arg(i)`
- IO: `IO()`, `Stdout()`, `Stderr()`, `Stdin()`
- Exit helpers: `Exit(code)`, `ExitWithError(err, code)`, `ExitOnError(err)`
- Wrapper result: `WrapperResult() (*ExecResult, bool)`
- App metadata: `AppName()`, `AppVersion()`, `AppDescription()`, `AppAuthors()`

Raw arguments access

Use `RawArgs()` to access the original unparsed arguments as passed to the application, before any parsing occurs:

```go
app.Command("proxy", "Proxy command to another tool").
    Action(func(ctx *snap.Context) error {
        // RawArgs() returns all arguments before parsing
        raw := ctx.RawArgs()
        fmt.Printf("Original invocation: %s %s\n", ctx.AppName(), strings.Join(raw, " "))
        
        // Args() returns only positional arguments after parsing
        positional := ctx.Args()
        fmt.Printf("Positional args: %v\n", positional)
        
        return nil
    })
```

Example invocation: `myapp --verbose proxy --port 8080 file1.txt file2.txt`
- `RawArgs()` returns: `["--verbose", "proxy", "--port", "8080", "file1.txt", "file2.txt"]`
- `Args()` returns: `["file1.txt", "file2.txt"]`

Use cases:
- **Audit logging**: Record the exact command as typed by the user
- **Debugging**: See what was passed before parsing
- **Proxying**: Forward the complete invocation to another tool
- **Custom parsing**: Implement special syntax handling

Note: `RawArgs()` does NOT include the binary name (`os.Args[0]`), only the arguments passed to `RunWithArgs()`.

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
