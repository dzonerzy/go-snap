# Positional Arguments Example

This example demonstrates basic positional argument handling in go-snap, including required arguments, optional arguments with defaults, and various argument types.

## Features Demonstrated

- **Required string arguments** - Must be provided by the user
- **Optional arguments with defaults** - Fall back to default values if not provided
- **Multiple argument types** - String, Int, Bool, Float, Duration
- **Fluent API chaining** - Using `.Back()` to chain multiple arguments
- **Type-safe access** - Direct typed access without casting

## Commands

### greet (App-level action)
Greet a person with optional age.

```bash
# Required name only
go run main.go John

# With optional age
go run main.go John 30
```

### copy
Copy a file from source to destination.

```bash
# Required source only (uses default destination)
go run main.go copy input.txt

# Both source and destination
go run main.go copy input.txt output.txt
```

### convert
Convert a file with quality and verbose options.

```bash
# Required input and output only (uses defaults)
go run main.go convert input.jpg output.png

# With custom quality
go run main.go convert input.jpg output.png 95

# With quality and verbose flag
go run main.go convert input.jpg output.png 95 true
```

### process
Process a file with timeout and threshold.

```bash
# Required file only (uses defaults)
go run main.go process data.txt

# With custom timeout
go run main.go process data.txt 1m30s

# With timeout and threshold
go run main.go process data.txt 1m30s 0.75
```

## Key Implementation Patterns

### Required Arguments
```go
app.StringArg("name", "Name argument").Required().Back()
```

### Optional Arguments with Defaults
```go
app.IntArg("age", "Age argument").Default(25)
```

### Fluent Chaining
```go
app.Command("copy", "Copy a file from source to destination").
    StringArg("source", "Source file path").Required().Back().
    StringArg("dest", "Destination file path").Default("output.txt").
    Action(copyAction)
```

### Accessing Arguments in Actions
```go
func greetAction(c *snap.Context) error {
    name := c.MustGetArg("name", "")     // Required arg
    age := c.MustGetArg("age", 25)       // Optional with default
    
    fmt.Printf("Hello %s", name)
    if age > 0 {
        fmt.Printf(", you are %d years old", age)
    }
    fmt.Println("!")
    
    return nil
}
```

## Help Output

Run any command with `--help` to see the full help:

```bash
go run main.go --help
go run main.go copy --help
go run main.go convert --help
```

The help output shows:
- Usage line with positional arguments: `<required> [optional]`
- Arguments section with full descriptions
- All available flags and commands

## Zero Allocations

All positional argument parsing maintains zero allocations (0 B/op, 0 allocs/op) for optimal performance.
