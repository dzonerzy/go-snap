# Migration Guide: Cobra → go-snap

This guide helps you migrate from [spf13/cobra](https://github.com/spf13/cobra) to go-snap. While both libraries share similar concepts (commands, flags, subcommands), go-snap offers superior performance and a more fluent API.

## Why Migrate?

- **4-10x faster execution** - Proven by benchmarks
- **3x less memory usage** - More efficient resource utilization
- **3-4x fewer allocations** - Better for performance-critical applications
- **Cleaner API** - Fluent builder pattern with type safety
- **Better testing** - Built-in testing framework
- **Unique features** - Wrapper CLI system, lifecycle hooks, zero-allocation parsing

## Quick Comparison

### Basic App Structure

**Cobra:**
```go
import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
    Use:   "myapp",
    Short: "My application",
    Run: func(cmd *cobra.Command, args []string) {
        // action
    },
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
```

**go-snap:**
```go
import "github.com/dzonerzy/go-snap/snap"

func main() {
    app := snap.New("myapp", "My application")
    
    app.Command("run", "Run the application").
        Action(func(ctx *snap.Context) error {
            // action
            return nil
        })
    
    if err := app.Run(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
```

## Core Concepts Mapping

### 1. Application/Root Command

**Cobra:**
```go
rootCmd := &cobra.Command{
    Use:     "myapp",
    Short:   "Short description",
    Long:    "Long description...",
    Version: "1.0.0",
}
```

**go-snap:**
```go
app := snap.New("myapp", "Short description").
    HelpText("Long description...").
    Version("1.0.0")
```

### 2. Flags

#### String Flags

**Cobra:**
```go
var name string
cmd.Flags().StringVarP(&name, "name", "n", "default", "Your name")
// Access: name variable
```

**go-snap:**
```go
app.StringFlag("name", "Your name").
    Short('n').
    Default("default").
    Back()

// Access in action:
name := ctx.String("name")
```

#### Int Flags

**Cobra:**
```go
var port int
cmd.Flags().IntVarP(&port, "port", "p", 8080, "Server port")
```

**go-snap:**
```go
app.IntFlag("port", "Server port").
    Short('p').
    Default(8080).
    Back()

// Access:
port := ctx.Int("port")
```

#### Bool Flags

**Cobra:**
```go
var verbose bool
cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
```

**go-snap:**
```go
app.BoolFlag("verbose", "Verbose output").
    Short('v').
    Back()

// Access:
verbose := ctx.Bool("verbose")
```

#### Required Flags

**Cobra:**
```go
cmd.Flags().StringP("config", "c", "", "Config file")
cmd.MarkFlagRequired("config")
```

**go-snap:**
```go
app.StringFlag("config", "Config file").
    Short('c').
    Required().
    Back()
```

#### Flag Groups (Mutually Exclusive)

**Cobra:**
```go
cmd.MarkFlagsMutuallyExclusive("json", "yaml")
```

**go-snap:**
```go
app.FlagGroup().
    AddFlag("json").
    AddFlag("yaml").
    ExactlyOne()
```

### 3. Commands and Subcommands

**Cobra:**
```go
var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start the server",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("Server starting...")
    },
}

serveCmd.Flags().IntP("port", "p", 8080, "Port")

rootCmd.AddCommand(serveCmd)
```

**go-snap:**
```go
app.Command("serve", "Start the server").
    IntFlag("port", "Port").Short('p').Default(8080).Back().
    Action(func(ctx *snap.Context) error {
        fmt.Println("Server starting...")
        port := ctx.Int("port")
        return nil
    })
```

#### Nested Subcommands

**Cobra:**
```go
serverCmd := &cobra.Command{Use: "server"}
startCmd := &cobra.Command{
    Use: "start",
    Run: func(cmd *cobra.Command, args []string) {
        // start server
    },
}
stopCmd := &cobra.Command{
    Use: "stop",
    Run: func(cmd *cobra.Command, args []string) {
        // stop server
    },
}

serverCmd.AddCommand(startCmd)
serverCmd.AddCommand(stopCmd)
rootCmd.AddCommand(serverCmd)
```

**go-snap:**
```go
server := app.Command("server", "Server management")

server.Command("start", "Start server").
    Action(func(ctx *snap.Context) error {
        // start server
        return nil
    })

server.Command("stop", "Stop server").
    Action(func(ctx *snap.Context) error {
        // stop server
        return nil
    })
```

### 4. Persistent/Global Flags

**Cobra:**
```go
rootCmd.PersistentFlags().BoolP("debug", "d", false, "Debug mode")

// Available in all subcommands
```

**go-snap:**
```go
// App-level flags are automatically available to all commands
app.BoolFlag("debug", "Debug mode").Short('d').Back()

// Access in any command:
debug := ctx.Bool("debug")
```

### 5. PreRun/PostRun Hooks

**Cobra:**
```go
cmd := &cobra.Command{
    Use: "mycommand",
    PreRun: func(cmd *cobra.Command, args []string) {
        fmt.Println("Before command")
    },
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("Running command")
    },
    PostRun: func(cmd *cobra.Command, args []string) {
        fmt.Println("After command")
    },
}
```

**go-snap:**
```go
app.Command("mycommand", "My command").
    Before(func(ctx *snap.Context) error {
        fmt.Println("Before command")
        return nil
    }).
    Action(func(ctx *snap.Context) error {
        fmt.Println("Running command")
        return nil
    }).
    After(func(ctx *snap.Context) error {
        fmt.Println("After command")
        return nil
    })
```

### 6. Positional Arguments

**Cobra:**
```go
cmd := &cobra.Command{
    Use:  "greet [name]",
    Args: cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        name := args[0]
        fmt.Printf("Hello, %s!\n", name)
    },
}
```

**go-snap:**
```go
app.Command("greet", "Greet someone").
    Action(func(ctx *snap.Context) error {
        args := ctx.Args()
        if len(args) < 1 {
            return errors.New("name required")
        }
        name := args[0]
        fmt.Printf("Hello, %s!\n", name)
        return nil
    })
```

### 7. Help and Version

**Cobra:**
```go
rootCmd.Version = "1.0.0"
// Help is automatic with --help or -h
```

**go-snap:**
```go
app.Version("1.0.0")
// Help is automatic with --help or -h
// Disable with: app.DisableHelp()
```

### 8. Custom Help

**Cobra:**
```go
cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
    fmt.Println("Custom help")
})
```

**go-snap:**
```go
app.HelpText("Custom help text that appears in help output")
```

### 9. Error Handling

**Cobra:**
```go
cmd.SilenceErrors = true
cmd.SilenceUsage = true

if err := rootCmd.Execute(); err != nil {
    // Handle error
}
```

**go-snap:**
```go
app.ErrorHandler().
    ShowHelpOnError(true). // Show help after errors
    Silent(false)          // Don't silence errors

if err := app.Run(); err != nil {
    // Handle error
}
```

## Complete Migration Example

Let's migrate a complete Cobra application to go-snap.

### Cobra Version

```go
package main

import (
    "fmt"
    "os"
    "github.com/spf13/cobra"
)

var (
    configFile string
    verbose    bool
    port       int
)

var rootCmd = &cobra.Command{
    Use:     "myapp",
    Short:   "My application",
    Version: "1.0.0",
}

var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start the server",
    PreRun: func(cmd *cobra.Command, args []string) {
        fmt.Println("Initializing...")
    },
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("Starting server on port %d\n", port)
        if verbose {
            fmt.Println("Verbose mode enabled")
        }
    },
    PostRun: func(cmd *cobra.Command, args []string) {
        fmt.Println("Cleanup...")
    },
}

var dbCmd = &cobra.Command{
    Use:   "db",
    Short: "Database commands",
}

var dbMigrateCmd = &cobra.Command{
    Use:   "migrate",
    Short: "Run migrations",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("Running migrations...")
    },
}

func init() {
    rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Config file")
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
    
    serveCmd.Flags().IntVarP(&port, "port", "p", 8080, "Server port")
    
    dbCmd.AddCommand(dbMigrateCmd)
    rootCmd.AddCommand(serveCmd, dbCmd)
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
```

### go-snap Version

```go
package main

import (
    "fmt"
    "os"
    "github.com/dzonerzy/go-snap/snap"
)

func main() {
    app := snap.New("myapp", "My application").
        Version("1.0.0")
    
    // Global flags
    app.StringFlag("config", "Config file").Short('c').Back()
    app.BoolFlag("verbose", "Verbose output").Short('v').Back()
    
    // Serve command
    app.Command("serve", "Start the server").
        IntFlag("port", "Server port").Short('p').Default(8080).Back().
        Before(func(ctx *snap.Context) error {
            fmt.Println("Initializing...")
            return nil
        }).
        Action(func(ctx *snap.Context) error {
            port := ctx.Int("port")
            verbose := ctx.Bool("verbose")
            
            fmt.Printf("Starting server on port %d\n", port)
            if verbose {
                fmt.Println("Verbose mode enabled")
            }
            return nil
        }).
        After(func(ctx *snap.Context) error {
            fmt.Println("Cleanup...")
            return nil
        })
    
    // Database commands
    db := app.Command("db", "Database commands")
    db.Command("migrate", "Run migrations").
        Action(func(ctx *snap.Context) error {
            fmt.Println("Running migrations...")
            return nil
        })
    
    if err := app.Run(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
```

## Advanced Features in go-snap

### 1. Wrapper CLI System (Unique to go-snap)

Enhance existing CLI tools:

```go
app.Command("docker-build", "Enhanced docker build").
    Wrap("docker").
    Action("build").
    BeforeExec(func(ctx *snap.Context, cmd *exec.Cmd) error {
        // Inject custom build args
        cmd.Args = append(cmd.Args, "--progress=plain")
        return nil
    }).
    AfterExec(func(ctx *snap.Context, result *snap.ExecResult) error {
        fmt.Printf("Build completed in %v\n", result.Duration)
        return nil
    })
```

### 2. Built-in Testing

```go
func TestServeCommand(t *testing.T) {
    app := createApp()
    
    result := snap.Test(app).
        Args("serve", "--port", "9000").
        Run()
    
    result.
        ShouldSucceed().
        ShouldContain("Starting server on port 9000")
}
```

### 3. Configuration Files

```go
app.Config().
    File("config.yaml").
    EnvPrefix("MYAPP").
    AutoBind()
```

### 4. Middleware

```go
app.Use(snap.LoggerMiddleware())
app.Use(snap.RecoveryMiddleware())
app.Use(snap.TimeoutMiddleware(30 * time.Second))
```

## Common Pitfalls

### 1. Variable Access

**Cobra** uses package-level variables:
```go
var port int
cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port")
// Access: port
```

**go-snap** uses context methods:
```go
app.IntFlag("port", "Port").Short('p').Default(8080).Back()
// Access: ctx.Int("port")
```

### 2. Command Chaining

**Cobra** uses separate variable assignments:
```go
serveCmd := &cobra.Command{...}
serveCmd.Flags().IntP("port", "p", 8080, "Port")
rootCmd.AddCommand(serveCmd)
```

**go-snap** uses fluent chaining:
```go
app.Command("serve", "Start server").
    IntFlag("port", "Port").Short('p').Default(8080).Back().
    Action(func(ctx *snap.Context) error { return nil })
```

### 3. Flag Groups

**Cobra** doesn't have built-in flag groups (needs manual validation).

**go-snap** has first-class support:
```go
app.FlagGroup().
    AddFlag("json").
    AddFlag("yaml").
    ExactlyOne() // Exactly one must be provided
```

## Performance Benefits

After migrating to go-snap, you should see:

- **4-5x faster command execution**
- **3x reduction in memory usage**
- **3-4x fewer allocations**
- **Smaller binary size** (fewer dependencies)

Run benchmarks before and after migration to measure improvements.

## Migration Checklist

- [ ] Replace `cobra.Command` with `snap.New()` for root
- [ ] Convert `cmd.Flags()` to app-level or command-level flags
- [ ] Replace `Run` functions with `Action()` methods
- [ ] Convert `PreRun`/`PostRun` to `Before()`/`After()`
- [ ] Change flag access from variables to `ctx.Type("name")`
- [ ] Replace `cmd.AddCommand()` with `app.Command()` or `parent.Command()`
- [ ] Update tests to use `snap.Test()`
- [ ] Update documentation with new API
- [ ] Run benchmarks to verify performance improvements
- [ ] Update imports: `github.com/spf13/cobra` → `github.com/dzonerzy/go-snap/snap`

## Need Help?

- [Examples](../examples/) - Complete working examples
- [API Documentation](./api.md) - Full API reference
- [GitHub Issues](https://github.com/dzonerzy/go-snap/issues) - Report bugs or ask questions

## Summary

Migrating from Cobra to go-snap is straightforward:

1. **Replace command structures** with fluent builders
2. **Convert flag access** from variables to context methods
3. **Update hooks** to use Before/After pattern
4. **Enjoy 4-10x performance improvement**

The migration typically takes 1-2 hours for small applications and pays off immediately with better performance and cleaner code.
