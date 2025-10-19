# Logging Demo

Demonstrates the structured logging system with colored circles, multiple formats, and customization options.

Shows how to use `ctx.LogDebug()`, `ctx.LogInfo()`, `ctx.LogSuccess()`, `ctx.LogWarning()`, and `ctx.LogError()` for semantic logging with automatic color coding.

## Features

- **Default Format**: Colored circles (🟣 🔵 🟢 🟡 🔴)
- **Built-in Formats**: Symbols (ℹ ✓ ⚠ ✗ •), Tags ([INFO] [WARN]), Plain
- **Custom Templates**: Create your own format with `{{.Level}}`, `{{.Time}}`, `{{.Message}}`
- **Timestamps**: Optional timestamps with customizable format
- **Semantic Colors**: Auto-colored based on log level using theme

## Run

```bash
# Default format (colored circles)
go run ./examples/logging-demo deploy

# Symbol format
go run ./examples/logging-demo deploy --format symbols

# Tagged format (traditional)
go run ./examples/logging-demo deploy --format tagged

# With timestamps
go run ./examples/logging-demo deploy --timestamp

# Show all formats
go run ./examples/logging-demo showcase

# Timestamp demo with varying durations
go run ./examples/logging-demo with-timestamps

# Database migration example
go run ./examples/logging-demo db-migrate
go run ./examples/logging-demo db-migrate --dry-run
```

## Log Levels

- `LogDebug()` - 🟣 Purple - Verbose debugging information
- `LogInfo()` - 🔵 Blue - Informational messages
- `LogSuccess()` - 🟢 Green - Success confirmations
- `LogWarning()` - 🟡 Yellow - Warnings (goes to stderr)
- `LogError()` - 🔴 Red - Error messages (goes to stderr)
