# IO, Color & Logging

The `snapio.IOManager` centralizes process IO, terminal capabilities, and structured logging.

## IO Manager

Access from App/Context
- `app.IO()` returns `*snapio.IOManager` (fluent setters available)
- From `*snap.Context`: `Stdout()`, `Stderr()`, `Stdin()`, `IO()`

Capabilities
- `IsTTY()`, `IsInteractive()`, `IsPiped()`, `IsRedirected()`
- `Width()`, `Height()`
- Color detection: `SupportsColor()`, `ColorLevel()` (0=none, 1=16, 2=256, 3=truecolor)
- `ForceColorLevel(level)` - manually override color detection
- Windows: `EnableVirtualTerminal()` is called automatically when appropriate in `App.RunWithArgs`

## Color System

### Color Levels
go-snap automatically detects terminal color capabilities:
- **Level 0**: No color support
- **Level 1**: Basic 16 colors (ANSI)
- **Level 2**: 256-color palette
- **Level 3**: Truecolor (16 million colors, 24-bit RGB)

Detection uses multiple methods:
- Environment variables (`COLORTERM`, `TERM`, `TERM_PROGRAM`)
- Platform-specific queries (`tput colors`, `tput RGB` on Unix/WSL)
- Windows VT support detection

### Color Palette

**16-Color (Basic)**
```go
snapio.Black, snapio.Red, snapio.Green, snapio.Yellow
snapio.Blue, snapio.Magenta, snapio.Cyan, snapio.White
snapio.BrightBlack, snapio.BrightRed, snapio.BrightGreen, etc.
```

**256-Color (Extended)**
```go
snapio.LightPurple, snapio.Orange, snapio.SkyBlue
snapio.Gray1, snapio.Gray2, ..., snapio.Gray6
snapio.DarkRed, snapio.DarkGreen, snapio.DarkBlue, etc.
```

**Truecolor (RGB)**
```go
snapio.TrueLightPurple, snapio.TrueBrightRed, snapio.TrueSkyBlue
snapio.Truecolor(189, 147, 249) // Custom RGB colors
```

### Styling

Color helpers
- Simple: `IOManager.Colorize(s, code)`, `Bold`, `Faint`, `Italic`, `Underline`
- Advanced styles:
  ```go
  style := snapio.NewStyle().
      Fg(snapio.BrightBlue).
      Bg(snapio.Black).
      Bold().
      Underline()
  
  fmt.Fprintln(io.Out(), style.Sprint(io, "styled text"))
  ```

### Themes

Themes provide semantic colors that adapt to terminal capabilities:

```go
// Auto-selected based on ColorLevel
theme := snapio.DefaultTheme(io)

// Specific theme variants
theme16 := snapio.DefaultTheme16()        // 16 colors
theme256 := snapio.DefaultTheme256()      // 256 colors
themeTruecolor := snapio.DefaultThemeTruecolor() // RGB

// Custom theme
customTheme := snapio.Theme{
    Primary: snapio.TrueBrightBlue,
    Success: snapio.TrueBrightGreen,
    Warning: snapio.TrueBrightYellow,
    Error:   snapio.TrueBrightRed,
    Info:    snapio.TrueBrightCyan,
    Debug:   snapio.TrueLightPurple,
    Muted:   snapio.TrueGray,
}
```

## Structured Logging

go-snap includes a built-in structured logging system with semantic levels.

### Quick Start

```go
app.Command("deploy", "Deploy application").
    Action(func(ctx *snap.Context) error {
        ctx.LogInfo("Starting deployment...")
        ctx.LogSuccess("Deployment complete!")
        ctx.LogWarning("Cache invalidation may take a few minutes")
        return nil
    })
```

### Log Levels

Five semantic log levels with automatic color coding:

```go
ctx.LogDebug("Verbose debug information")    // 🟣 Purple
ctx.LogInfo("Informational message")         // 🔵 Cyan
ctx.LogSuccess("Operation succeeded")        // 🟢 Green
ctx.LogWarning("Warning message")            // 🟡 Yellow
ctx.LogError("Error occurred")               // 🔴 Red
```

### Log Formats

Four built-in formats:

**Circles (Default)**: Colored emoji circles
```
🟣 Debug message
🔵 Info message
🟢 Success message
🟡 Warning message
🔴 Error message
```

**Symbols**: Pure Unicode geometric symbols
```
● Debug message
◆ Info message
✓ Success message
▲ Warning message
✗ Error message
```

**Tagged**: Traditional bracketed tags
```
[DEBUG] Debug message
[INFO] Info message
[SUCCESS] Success message
[WARN] Warning message
[ERROR] Error message
```

**Plain**: No prefix, just the message

### Configuration

```go
// Change format
app.Logger().WithFormat(snapio.LogFormatSymbols)

// Add timestamps
app.Logger().WithTimestamp(true)

// Custom time format
app.Logger().WithTimeFormat("15:04:05.000")

// Custom template
app.Logger().WithFormat(snapio.LogFormatCustom).
    WithTemplate("[{{.Level}}] {{.Time}} - {{.Message}}")

// Custom prefix per level
app.Logger().
    SetPrefix(snapio.LevelInfo, "ℹ️").
    SetPrefix(snapio.LevelError, "❌")

// Custom theme
app.Logger().WithTheme(customTheme)
```

### Output Routing

By default:
- Errors and warnings → `stderr`
- Info, success, debug → `stdout`

Control routing:
```go
app.Logger().ErrorsToStderr(false) // Send everything to stdout
```

### Advanced Features

**Empty Message Handling**
Prefixes are automatically skipped for empty/whitespace-only messages, allowing clean visual spacing:
```go
ctx.LogInfo("Phase 1 complete")
ctx.LogInfo("")  // Clean separator, no emoji
ctx.LogInfo("Phase 2 starting")
```

**Direct Logger Access**
```go
logger := app.Logger()
logger.Debug("Debug message")
logger.Info("Info message")
logger.Success("Success message")
logger.Warning("Warning message")
logger.Error("Error message")
```

## Examples

### Basic Styling
```go
io := app.IO()
title := io.Bold("Welcome")
fmt.Fprintln(io.Out(), title)

style := snapio.NewStyle().Fg(snapio.BrightBlue).Bold()
fmt.Fprintln(io.Out(), style.Sprint(io, "styled line"))
```

### Logging with Formatting
```go
app.Command("serve", "Start server").
    IntFlag("port", "Port number").Default(8080).Back().
    Action(func(ctx *snap.Context) error {
        port := ctx.MustInt("port", 8080)
        
        ctx.LogInfo("Starting server on port %d...", port)
        // Server startup logic
        ctx.LogSuccess("Server is running!")
        
        return nil
    })
```

### Multiple Formats
```go
// Show deployment progress with different formats
app.Logger().WithFormat(snapio.LogFormatCircles)
ctx.LogInfo("🚀 Starting deployment...")

app.Logger().WithFormat(snapio.LogFormatSymbols)
ctx.LogSuccess("✓ Build completed")

app.Logger().WithFormat(snapio.LogFormatTagged)
ctx.LogInfo("[INFO] Running tests...")
```

## Related
- [App & Commands](./app-and-commands.md)
- [Wrapper DSL](./wrapper.md) (passthrough/capture uses IO)
- [Examples](./examples.md) - See `examples/logging-demo` for comprehensive logging examples
