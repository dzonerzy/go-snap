# Parsing and Context

Zero-allocation parser
- The parser in `snap/parser.go` uses pooled structures, byte math and string interning to avoid allocations on the hot path.
- Supports `--flag=value`, `--flag value`, short flags (`-v`, combined `-abc`), `--` terminator for positional args.
- Unknown flag/command errors include edit-distance suggestions.

ParseResult accessors (implemented)
- Per-type flag getters: `GetString`, `GetInt`, `GetBool`, `GetDuration`, `GetFloat`, `GetEnum`, `GetStringSlice`, `GetIntSlice`
- Global flag variants: `GetGlobalString`, `GetGlobalInt`, `GetGlobalBool`, `GetGlobalDuration`, `GetGlobalFloat`, `GetGlobalEnum`, `GetGlobalStringSlice`, `GetGlobalIntSlice`
- Must* with default: `MustGetString`, `MustGetInt`, `MustGetBool`, `MustGetDuration`, `MustGetFloat`, `MustGetEnum`, `MustGetStringSlice`, `MustGetIntSlice` and global counterparts
- Positional argument getters: `GetArg`, `GetArgInt`, `GetArgBool`, `GetArgDuration`, `GetArgFloat`, `GetArgStringSlice`, `GetArgIntSlice`
- Must* for args: `MustGetArg`, `MustGetArgInt`, `MustGetArgBool`, `MustGetArgDuration`, `MustGetArgFloat`, `MustGetArgStringSlice`, `MustGetArgIntSlice`
- `HasFlag`, `HasGlobalFlag`, `HasArg`
- `Args []string`, `Command *Command`, `RestArgs []string`

Context API (`snap/context.go`)
- `Context()` / `Done()` / `Cancel()` / `Err()` – propagation and cancellation
- `Set(key, val)`, `Get(key)` – metadata
- Flag helpers mirror ParseResult: `String/Int/Bool/Duration/Float/Enum`, `StringSlice/IntSlice`, global variants
- Positional argument helpers: `StringArg/IntArg/BoolArg/DurationArg/FloatArg`, `StringSliceArg/IntSliceArg`
- Positional args: `Args()`, `RawArgs()`, `NArgs()`, `Arg(i)`, `RestArgs()`
- IO: `IO()`, `Stdout()`, `Stderr()`, `Stdin()`
- Exit helpers: `Exit(code)`, `ExitWithError(err, code)`, `ExitOnError(err)`
- Wrapper result: `WrapperResult() (*ExecResult, bool)`
- App metadata: `AppName()`, `AppVersion()`, `AppDescription()`, `AppAuthors()`

Positional arguments

Positional arguments are defined by their position in the command line, not by flag names. They support all the same types as flags: string, int, bool, float, duration, and slices.

```go
// Define positional arguments
app.Command("copy", "Copy a file").
    StringArg("source", "Source file path").Required().Back().
    StringArg("dest", "Destination file path").Default("output.txt").
    Action(func(ctx *snap.Context) error {
        source := ctx.StringArg("source", "")
        dest := ctx.StringArg("dest", "output.txt")
        fmt.Printf("Copying %s to %s\n", source, dest)
        return nil
    })
```

Example invocations:
- `myapp copy input.txt` – uses default destination "output.txt"
- `myapp copy input.txt /tmp/output.txt` – both source and destination provided

Key features:
- **Type-safe**: Each argument type has its own builder and accessor
- **Required or optional**: Mark arguments as required or provide defaults
- **Fluent API**: Chain multiple arguments using `.Back()`
- **Zero allocations**: All parsing maintains 0 B/op, 0 allocs/op
- **Help integration**: Arguments shown in usage line and Arguments section

Variadic arguments

The last positional argument can be marked as variadic to collect multiple values:

```go
app.Command("rm", "Remove files").
    StringSliceArg("files", "Files to remove").Required().Variadic().
    Action(func(ctx *snap.Context) error {
        files := ctx.StringSliceArg("files", nil)
        fmt.Printf("Removing %d files\n", len(files))
        for _, file := range files {
            fmt.Printf("  rm %s\n", file)
        }
        return nil
    })
```

Example invocations:
- `myapp rm file1.txt` – removes one file
- `myapp rm file1.txt file2.txt file3.txt` – removes three files
- `myapp rm *.txt` – shell expands to multiple files

Variadic arguments:
- Must be the **last** positional argument defined
- Collect all remaining values into a slice
- Support both `StringSlice` and `IntSlice` types
- Can be marked as required (at least one value needed)
- Shown in help with `...` notation: `<files>...`

RestArgs pass-through

For wrapper CLIs that enhance existing commands (like docker, git, kubectl), use `RestArgs()` to pass all arguments through without parsing:

```go
app.Command("docker-run", "Enhanced docker run").
    RestArgs().
    Action(func(ctx *snap.Context) error {
        args := ctx.RestArgs()
        fmt.Println("Running: docker run", strings.Join(args, " "))
        // Execute: docker run <args>
        return nil
    })
```

Example invocation: `myapp docker-run -it --rm ubuntu:latest bash`
- `RestArgs()` returns: `["-it", "--rm", "ubuntu:latest", "bash"]`
- All arguments preserved exactly as typed, including flags

RestArgs behavior:
- **No flag parsing**: Everything after command name is passed through
- **Preserved flags**: `-it`, `--rm` treated as regular arguments, not parsed
- **No validation**: All arguments accepted as-is
- **Use case**: Wrapper CLIs that add behavior around existing tools

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
- `Args()` returns: `["file1.txt", "file2.txt"]` (positional args only)

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
