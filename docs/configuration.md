# Configuration (Struct Tags + Precedence + Auto-Flags)

The configuration builder in `snap/config.go` turns a struct definition into:
1) A schema (fields, types, defaults, enum options, groups)
2) A merged configuration from multiple sources with explicit precedence
3) Optional CLI flags generated automatically from the schema

Core API
```go
// Standalone builder (backed by a lightweight App)
snap.Config(name, description string) *ConfigBuilder

// Fluent sources
(cb *ConfigBuilder) Bind(target any) *ConfigBuilder
(cb *ConfigBuilder) FromDefaults(snap.D) *ConfigBuilder
(cb *ConfigBuilder) FromFile(filename string) *ConfigBuilder // JSON only
(cb *ConfigBuilder) FromEnv() *ConfigBuilder
(cb *ConfigBuilder) FromFlags() *ConfigBuilder               // generate CLI flags
(cb *ConfigBuilder) Build() (*snap.App, error)
```

Precedence (highest → lowest)
1) Flags
2) Environment
3) File (JSON)
4) Defaults

Struct tags
- `flag:"name[,required][,ignore]"`
- `env:"ENV_VAR"`
- `default:"value"`
- `description:"..."`
- `enum:"a,b,c"`
- `group:"groupName"` (affects help grouping and constraints)
- `group_constraint:"mutually|all_or_none|exactly_one|at_least_one"` (on nested struct field)
- `group_description:"..."` (on nested struct field)
- `ignore:"true"` (skip flag generation)

Auto flag generation (FromFlags)
- For each field, a typed flag is created on the app (or within a group) with description/default/enum.
- Groupings are created from nested structs or explicit `group` tag.

Binding
```go
type ServerConfig struct {
    Host string        `flag:"host" env:"HOST" default:"localhost" description:"Hostname"`
    Port int           `flag:"port" env:"PORT" default:"8080" description:"Port"`
    Debug bool         `flag:"debug" env:"DEBUG"`
    Timeout time.Duration `flag:"timeout" env:"TIMEOUT" default:"30s"`
    LogLevel string    `flag:"log-level" enum:"debug,info,warn,error" default:"info"`

    Database struct {
        URL string    `flag:"db-url" env:"DATABASE_URL" default:"sqlite://app.db"`
        MaxConns int  `flag:"db-max-conns" env:"DB_MAX_CONNS" default:"20"`
    } `group:"database"`
}

var cfg ServerConfig
app, err := snap.Config("webserver", "Prod server").
    FromDefaults(snap.D{"workers": 8}).
    FromFile("config.json").
    FromEnv().
    FromFlags().
    Bind(&cfg).
    Build() // returns *snap.App when FromFlags was used
```

Running
```go
if err != nil { panic(err) }
if err := app.Run(); err != nil {
    // handle error (help/version return nil, not an error)
    log.Fatalf("Error: %v", err)
}
// cfg is now fully populated with precedence applied
```

File format
- Only JSON is supported by `FromFile` in the current code.

Examples
- `examples/config-precedence/main.go`

Related
- [Flags & Groups](./flags-and-groups.md)
- [App & Commands](./app-and-commands.md)
