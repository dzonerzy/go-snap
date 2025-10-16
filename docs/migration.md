# Migration Guide

This guide helps migrate common patterns from Cobra and urfave/cli to go-snap.

## Comprehensive Migration Guides

For detailed, step-by-step migration instructions with complete examples:

- **[Migration Guide: Cobra → go-snap](./migration-from-cobra.md)** - Complete guide with side-by-side comparisons, performance benefits (4-10x faster), and migration checklist
- **[Migration Guide: urfave/cli → go-snap](./migration-from-urfave-cli.md)** - Detailed urfave/cli v2 migration with examples and common pitfalls

## Key Differences

- Typed, fluent builders (no stringly-typed flag values)
- Native flag grouping and constraint validation
- Smart errors and edit-distance suggestions
- Config precedence (defaults/file/env/flags) and auto-flag generation
- Minimal, focused middleware set
- **4-10x faster performance** with 3x less memory usage

From Cobra
```go
// Cobra: persistent flag (global)
rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "verbose")

// go-snap equivalent
app.BoolFlag("verbose", "Verbose").Global().Short('v').Back()

// Cobra: command + RunE
var serveCmd = &cobra.Command{ Use: "serve", RunE: func(cmd *cobra.Command, args []string) error { ... } }

// go-snap
app.Command("serve", "Start server").
    IntFlag("port", "Port").Default(8080).Back().
    Action(func(ctx *snap.Context) error { ... })

// Cobra: mutually exclusive flags → manual validation
// go-snap: use FlagGroup with constraints
app.FlagGroup("output").ExactlyOne().BoolFlag("json","JSON").Back().BoolFlag("yaml","YAML").Back().EndGroup()
```

From urfave/cli
```go
// urfave: app + flags
app := &cli.App{ Name: "myapp", Flags: []cli.Flag{ &cli.BoolFlag{Name: "verbose"} } }

// go-snap
app := snap.New("myapp", "...").BoolFlag("verbose","Verbose").Global().Back()

// urfave: subcommand
&cli.Command{Name: "serve", Action: func(c *cli.Context) error { ... }}

// go-snap
app.Command("serve","Start").Action(func(ctx *snap.Context) error { ... })
```

Config patterns
- Instead of manual env/file/flag merging, define struct tags and use `Config(...).From...().FromFlags().Bind(&cfg).Build()`.
- Enums map naturally via `enum:"val1,val2"`.

Exit codes
- Map domain errors with `ExitCodes().DefineError(err, code)`.
- Use `Context.Exit(code)` for explicit exits rather than `os.Exit`.

Wrappers
- Replace shell scripts or custom arg munging with `Wrap/WrapDynamic` and `TransformArgs`.

Tips
- Global flags in Cobra ≈ `.Global()` in go-snap.
- PersistentPreRun/PreRun ≈ `App.Before` and command middleware.

See also
- [Best Practices](./best-practices.md)
- [Examples](./examples.md)
