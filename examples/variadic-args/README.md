# Variadic Arguments Example

This example demonstrates variadic argument handling and RestArgs pass-through in go-snap. Variadic arguments collect multiple values, while RestArgs passes all arguments directly to wrapped commands.

## Features Demonstrated

- **Variadic string slice arguments** - Collect multiple string values
- **Variadic int slice arguments** - Collect multiple integer values
- **Mixed regular + variadic** - Combine fixed and variadic arguments
- **RestArgs pass-through** - Pass all arguments to wrapped commands
- **Type-safe slice access** - Direct typed slice access without casting
- **Help flag priority** - `--help` works even with required variadic args

## Commands

### rm
Remove one or more files (string slice).

```bash
# Single file
go run main.go rm file1.txt

# Multiple files
go run main.go rm file1.txt file2.txt file3.txt

# Many files
go run main.go rm *.txt

# Help works with required variadic args
go run main.go rm --help
```

### sum
Calculate sum of multiple numbers (int slice).

```bash
# Sum two numbers
go run main.go sum 10 20

# Sum many numbers
go run main.go sum 1 2 3 4 5 6 7 8 9 10

# Help works with required variadic args
go run main.go sum --help
```

### copy-many
Copy multiple source files to a destination directory.

```bash
# Copy multiple files
go run main.go copy-many /tmp/dest file1.txt file2.txt file3.txt

# Copy with pattern
go run main.go copy-many /backup *.log
```

### docker-run
Simulate docker run with RestArgs pass-through.

```bash
# All args passed through
go run main.go docker-run ubuntu:latest /bin/bash

# Flags passed through as-is
go run main.go docker-run -it --rm ubuntu:latest bash

# Complex arguments preserved
go run main.go docker-run -v /host:/container -e "VAR=value" image:tag
```

## Key Implementation Patterns

### Variadic String Slice
```go
app.Command("rm", "Remove one or more files").
    StringSliceArg("files", "Files to remove").Required().Variadic().
    Action(rmAction)
```

### Variadic Int Slice
```go
app.Command("sum", "Calculate sum of numbers").
    IntSliceArg("numbers", "Numbers to sum").Required().Variadic().
    Action(sumAction)
```

### Mixed Regular + Variadic
```go
app.Command("copy-many", "Copy multiple files to destination directory").
    StringArg("dest", "Destination directory").Required().Back().
    StringSliceArg("sources", "Source files").Required().Variadic().
    Action(copyManyAction)
```

### RestArgs Pass-Through
```go
app.Command("docker-run", "Simulate docker run with pass-through args").
    RestArgs().
    Action(dockerRunAction)
```

### Accessing Variadic Arguments
```go
func rmAction(c *snap.Context) error {
    files := c.MustGetArgStringSlice("files", nil)
    
    fmt.Printf("Removing %d files:\n", len(files))
    for _, file := range files {
        fmt.Printf("  rm %s\n", file)
    }
    
    return nil
}
```

### Accessing RestArgs
```go
func dockerRunAction(c *snap.Context) error {
    restArgs := c.RestArgs()
    
    fmt.Println("Simulating: docker run", strings.Join(restArgs, " "))
    fmt.Printf("Would execute docker with %d arguments:\n", len(restArgs))
    for i, arg := range restArgs {
        fmt.Printf("  [%d] %s\n", i, arg)
    }
    
    return nil
}
```

## Help Output

Run any command with `--help` to see the full help:

```bash
go run main.go --help
go run main.go rm --help
go run main.go sum --help
go run main.go copy-many --help
go run main.go docker-run --help
```

The help output shows:
- Variadic arguments with `...` notation: `<files>...`
- RestArgs with `[args...]` notation
- Arguments section with full descriptions
- Help works even when required variadic args are missing

## Variadic vs RestArgs

### Variadic Arguments
- **Type-safe**: Defined with specific types (StringSlice, IntSlice)
- **Validated**: Can mark as required, apply validators
- **Last position**: Must be the last argument defined
- **Parsed**: Values are parsed according to their type
- **Use case**: Collecting multiple typed values (files, numbers, etc.)

### RestArgs
- **Untyped**: All arguments passed as string slice
- **No validation**: Everything is passed through as-is
- **All arguments**: Captures everything after command name
- **No parsing**: Flags like `-it` are preserved exactly
- **Use case**: Wrapper CLIs that enhance existing commands (docker, git, kubectl)

## Zero Allocations

All variadic argument parsing maintains zero allocations (0 B/op, 0 allocs/op) through pooled slice management for optimal performance.
