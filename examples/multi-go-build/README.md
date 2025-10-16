# Multi-Go-Build Example

This example demonstrates the `WrapMany()` feature for executing multiple binaries with the same arguments.

## Use Case

When you have multiple versions of a tool installed (e.g., Go 1.21, 1.22, 1.23) and want to test your code against all of them.

## Features Demonstrated

### 1. Sequential Execution (Default)

Executes binaries one after another. If StopOnError is true (default), stops on first failure.

### 2. Parallel Execution

All binaries execute concurrently, significantly faster for independent operations.

### 3. Context Accessors

Inside AfterExec, you can access:
- ctx.CurrentBinary() - which binary is currently executing
- ctx.Binaries() - all binaries in the list

## Running the Example

```bash
# Sequential build
go run main.go build-seq -o myapp

# Parallel build  
go run main.go build-parallel -o myapp

# Check versions
go run main.go versions
```

## Key Options

- **WrapMany(binaries...)** - Execute multiple binaries
- **.Parallel()** - Run concurrently (default: sequential)
- **.StopOnError(bool)** - Stop on first error (default: true)
- **AfterExec()** - Called once per binary execution
