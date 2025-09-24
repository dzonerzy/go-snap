# Middleware Example

This example demonstrates how to use go-snap's middleware system to add cross-cutting concerns like logging, error recovery, timeouts, and validation to your CLI applications.

## Features Demonstrated

- **Global Middleware**: Applied to all commands
  - `Logger`: Logs command execution with configurable levels
  - `Recovery`: Catches panics and provides stack traces
  - `Timeout`: Prevents commands from running too long

- **Command-Specific Middleware**: Applied only to specific commands
  - `Validator`: Custom business logic validation
  - `JSONLogger`: Alternative logging format for specific operations

- **Custom Validators**: User-defined validation functions
  - Port range validation
  - File existence validation

- **Environment Variable Support**: Demonstrates `FromEnv()` functionality
  - Configuration precedence: command line > environment variables > defaults
  - Multiple environment variable fallbacks
  - Global and command-specific flag environment binding

## Run

```bash
# Basic server start (uses global middleware + friendly validation DSL)
go run main.go serve --port 8080 --host localhost

# Verbose mode (global flag affects logging)
go run main.go --verbose serve --port 3000

# Status check (only global middleware)
go run main.go status

# Config validation (uses JSON logging + file exists rule)
# A sample file is provided at ./config/config.json
go run main.go config validate --config ./config/config.json

# Invalid port (triggers validation error)
go run main.go serve --port 70000

# Privileged port warning
go run main.go serve --port 80

# Using environment variables (demonstrates FromEnv functionality)
export PORT=9000
export HOST=0.0.0.0
export VERBOSE=true
go run main.go serve

# Environment variable precedence (multiple variables checked in order)
export SERVER_PORT=8000  # Will be used if PORT is not set
export DEBUG=true        # Will be used if VERBOSE is not set
go run main.go serve

# Command line flags override environment variables
export PORT=9000
go run main.go serve --port 3000  # Uses 3000, not 9000
```

## Middleware Chain Execution Order

1. **Global Middleware** (applied to all commands):
   - Logger (logs request start)
   - Recovery (panic protection)
   - Timeout (30 second limit)

2. **Command-Specific Middleware** (if defined):
   - Validator (business logic validation)
   - Custom middleware (e.g., JSONLogger)

3. **Command Action** (your actual command logic)

4. **Response Middleware** (in reverse order):
   - Timeout completion
   - Recovery cleanup
   - Logger (logs request completion)

## Customizing Middleware

### Logger Options
```go
middleware.Logger(
    middleware.WithLogLevel(middleware.LogLevelDebug),  // Log everything
    middleware.WithLogFormat(middleware.LogFormatJSON), // JSON output
)
```

### Custom Validators
```go
middleware.Validator(middleware.WithCustomValidators(map[string]middleware.ValidatorFunc{
    "my_validator": func(ctx middleware.Context) error {
        // Your validation logic here
        return nil
    },
}))
```

### Recovery Options
```go
middleware.Recovery(
    middleware.WithStackTrace(true),  // Include stack traces
)
```

### Timeout Configuration
```go
middleware.Timeout(
    middleware.WithTimeout(60 * time.Second),  // 1 minute timeout
)
```

### Environment Variable Binding
```go
// Single environment variable
StringFlag("host", "Server host").
    Default("localhost").
    FromEnv("HOST").
    Back()

// Multiple environment variables (precedence order)
IntFlag("port", "Server port").
    Default(8080).
    FromEnv("PORT", "SERVER_PORT", "APP_PORT").  // Checks in order
    Back()

// Global flags with environment support
BoolFlag("debug", "Debug mode").
    Global().
    FromEnv("DEBUG", "VERBOSE").
    Back()
```

## Key Benefits

1. **Separation of Concerns**: Business logic stays in actions, cross-cutting concerns in middleware
2. **Reusability**: Same middleware can be applied to multiple commands
3. **Composability**: Mix and match middleware as needed
4. **Zero Allocation**: Middleware system preserves go-snap's performance characteristics
5. **Type Safety**: Full type safety with middleware context interface
