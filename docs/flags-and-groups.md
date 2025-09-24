# Flags & Groups

Flag types
- `string`, `int`, `bool`, `duration` (time.Duration), `float64`
- `enum` (string with allowed set)
- `[]string`, `[]int`

Defining flags (app-level)
```go
app.StringFlag("config", "Path to config").Global().Short('c').Back()
app.BoolFlag("verbose", "Verbose").Global().Short('v').Back()
```

Defining flags (command-level)
```go
app.Command("serve", "Run server").
    IntFlag("port", "Port").Default(8080).Back().
    EnumFlag("log", "Level", "debug","info","warn","error").Default("info").Back()
```

FlagBuilder modifiers (implemented)
- `Default(value)` – typed default
- `Required()` – mark as required
- `Short(rune)` – single-letter alias, O(1) lookup
- `Global()` – available to all commands
- `Hidden()` – hide from help
- `FromEnv(...string)` – precedence-aware env vars
- `Usage(string)` – extra description
- `Validate(func(T) error)` – typed validator
- `Back()` – return to parent builder

Convenience validators (from `snap/flag.go`)
- `Range(fb, min, max)` for `int`/`float64`
- `OneOf(fb, values...)` for `string`
- `File(fb, mustExist)` / `Dir(fb, mustExist)`
- `Regex(fb, pattern)`

Flag groups
```go
app.FlagGroup("output").
    ExactlyOne().
    Description("Choose one format").
    BoolFlag("json", "JSON").Short('j').Back().
    BoolFlag("yaml", "YAML").Short('y').Back().
    BoolFlag("table", "Table").Short('t').Back().
    EndGroup()
```

Constraints (implemented)
- `MutuallyExclusive()`
- `AllOrNone()`
- `ExactlyOne()`
- `AtLeastOne()` (alias: `RequiredGroup()`)

Grouping behavior
- Group definitions are attached to app or command and validated after parsing.
- Grouped flags are shown together in help, with a human-readable constraint note.

Environment + defaults
- Parser applies env vars and defaults for missing flags per type.
- For slice flags, comma-separated env values are supported.

Examples
- Full groups demo: `examples/flag-groups/main.go`

Related
- [Parsing & Context](./parsing-and-context.md)
- [App & Commands](./app-and-commands.md)
