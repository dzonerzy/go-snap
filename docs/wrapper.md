# Wrapper DSL

go-snap includes a powerful wrapper DSL for enhancing existing tools, including a dynamic toolexec shim mode.

Where to define wrappers
- App-level: `app.Wrap(binary)` – runs when no explicit command is provided
- Command-level: `app.Command("foo").Wrap(binary)` – runs when the command is invoked
- Dynamic shim: `CommandBuilder.WrapDynamic()` – for `go build --toolexec` style tools

Common options (from `snap/wrapper.go`)
- Process: `Binary`, `DiscoverOnPATH(bool)`, `WorkingDir`, `Env(k,v)`, `EnvMap(map)`, `InheritEnv(bool)`
- argv shaping: `InjectArgsPre(...)`, `InjectArgsPost(...)`, `ForwardArgs()`
- unknown flags: `ForwardUnknownFlags()` – forward unknown CLI flags as positional tokens
- transform: `TransformArgs(func(*Context, []string) ([]string,error))`
- lifecycle hooks: `BeforeExec(func(*Context, []string) ([]string,error))`, `AfterExec(func(*Context, *ExecResult) error)`
- dynamic tool rewrite: `TransformTool(fn(tool string, args []string) (string, []string, error))`
- I/O modes: `Passthrough()`, `Capture()`, `CaptureTo(out,err io.Writer)`, `TeeTo(out,err)`
- policy: `AllowTools(names...)` (dynamic shim)
- visibility: `HideFromHelp()` / `Visible()` (command-level only)
- DSL helpers: `LeadingFlags(...)`, `InsertAfterLeadingFlags(...)`, `MapBoolFlag(wrapperFlag, childTokens...)`

Result capture
- `Capture()` returns data in `*ExecResult` exposed via `ctx.WrapperResult()`
- In passthrough mode you can also `CaptureTo(...)` to stream and capture

Echo wrapper example
```go
app := snap.New("echo-wrap", "prefix echo output")
app.BoolFlag("n", "suppress trailing newline").Back()
app.Wrap("/bin/echo").
    ForwardUnknownFlags().
    ForwardArgs().
    LeadingFlags("-n","-e","-E").
    MapBoolFlag("n", "-n").
    InsertAfterLeadingFlags("[prefix]").
    Passthrough().
    Back()
```

toolexec logger example
```go
app.Command("build", "wrap go build").
    Wrap("go").
    InjectArgsPre("build", "--toolexec", "${SELF} log").
    ForwardUnknownFlags().
    ForwardArgs().
    Passthrough().
    Back()

app.Command("log", "toolexec logger").
    WrapDynamic().
    ForwardUnknownFlags().
    TransformArgs(func(ctx *snap.Context, in []string) ([]string, error) {
        if len(ctx.Args()) > 0 {
            base := filepath.Base(ctx.Args()[0])
            q := make([]string, len(in))
            for i, a := range in { q[i] = strconv.Quote(a) }
            fmt.Fprintln(ctx.Stderr(), "[toolexec]", base, strings.Join(q, " "))
        }
        return in, nil
    }).
    Passthrough().
    HideFromHelp().
    Back()
```

Wrapper lifecycle hooks (BeforeExec/AfterExec)

Wrappers support `BeforeExec` and `AfterExec` hooks for advanced argument transformation and result processing:

```go
app.Command("docker-build", "Enhanced Docker build").
    StringFlag("tag", "Image tag").Required().Back().
    Wrap("docker").
    BeforeExec(func(ctx *snap.Context, args []string) ([]string, error) {
        tag, _ := ctx.String("tag")
        fmt.Printf("Starting Docker build for tag: %s\n", tag)
        
        // Modify arguments before execution
        return append([]string{"build", "--tag", tag, "."}, args...), nil
    }).
    Passthrough().
    AfterExec(func(ctx *snap.Context, result *snap.ExecResult) error {
        fmt.Printf("Docker build completed with exit code: %d\n", result.ExitCode)
        
        // Process result: log metrics, send notifications, etc.
        if result.ExitCode == 0 {
            fmt.Println("Build succeeded! Pushing to registry...")
            // Custom logic here
        }
        return nil
    }).
    Back()
```

Hook behavior:
- `BeforeExec` runs after all argument transformations (`PreArgs`, `PostArgs`, `Transform`, etc.) and receives the final arguments. It can modify them one last time before execution.
- `AfterExec` runs after the wrapped command completes. It receives an `*ExecResult` with `ExitCode`, `Stdout`, `Stderr`, and `Error`.
- If `BeforeExec` returns an error, the wrapped command is not executed.
- `AfterExec` runs even if the wrapped command fails, allowing for cleanup and logging.
- If `AfterExec` returns an error, it overrides a successful execution result.

Result structure:
```go
type ExecResult struct {
    ExitCode int       // Exit code from wrapped command
    Stdout   []byte    // Captured stdout (if Capture() or CaptureTo() used)
    Stderr   []byte    // Captured stderr (if Capture() or CaptureTo() used)
    Error    error     // Error from execution (nil on success)
}
```

Example with error handling:
```go
app.Wrap("flaky-tool").
    ForwardArgs().
    Capture().
    AfterExec(func(ctx *snap.Context, result *snap.ExecResult) error {
        if result.ExitCode != 0 {
            // Retry logic, logging, alerting
            fmt.Printf("Command failed with: %s\n", result.Stderr)
            return fmt.Errorf("wrapped command failed: exit code %d", result.ExitCode)
        }
        return nil
    }).
    Back()
```

Notes
- When an app-level wrapper is present, unknown top-level tokens are treated as positional args and forwarded if `ForwardUnknownFlags()` is enabled.
- Unknown flags/short flags inside a wrapped command can be forwarded similarly.
- `BeforeExec` is called after `Transform` but is more explicit about its purpose (final pre-execution hook).
- In `Passthrough` mode without `CaptureTo`, `AfterExec` still receives a minimal `ExecResult` with `ExitCode` and `Error`.

Related
- [Parsing & Context](./parsing-and-context.md)
- [IO & Color](./io-and-color.md)
- [App & Commands](./app-and-commands.md) (for Command-level Before/After hooks)
