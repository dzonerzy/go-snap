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

Notes
- When an app-level wrapper is present, unknown top-level tokens are treated as positional args and forwarded if `ForwardUnknownFlags()` is enabled.
- Unknown flags/short flags inside a wrapped command can be forwarded similarly.

Related
- [Parsing & Context](./parsing-and-context.md)
- [IO & Color](./io-and-color.md)
