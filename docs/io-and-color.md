# IO & Color

The `snapio.IOManager` centralizes process IO and terminal capabilities.

Access from App/Context
- `app.IO()` returns `*snapio.IOManager` (fluent setters available)
- From `*snap.Context`: `Stdout()`, `Stderr()`, `Stdin()`, `IO()`

Capabilities
- `IsTTY()`, `IsInteractive()`, `IsPiped()`, `IsRedirected()`
- `Width()`, `Height()`
- Color detection: `SupportsColor()`, `ColorLevel()` (0/1/2/3)
- Windows: `EnableVirtualTerminal()` is called automatically when appropriate in `App.RunWithArgs`

Color helpers
- Simple: `IOManager.Colorize(s, code)`, `Bold`, `Faint`, `Italic`, `Underline`
- Styles (in `io/color.go`):
  - `ColorSpec` (basic/indexed/truecolor)
  - `NewStyle().Fg(...).Bg(...).Bold().Underline()...`
  - `style.Sprintf(io, "format %s", x)` / `style.Sprint(io, text)`

Example
```go
io := app.IO()
title := io.Bold("Welcome")
fmt.Fprintln(io.Out(), title)

style := snapio.NewStyle().Fg(snapio.BrightBlue).Bold()
fmt.Fprintln(io.Out(), style.Sprint(io, "styled line"))
```

Related
- [App & Commands](./app-and-commands.md)
- [Wrapper DSL](./wrapper.md) (passthrough/capture uses IO)
