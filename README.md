<p align="center">
  <img src="assets/logo/logo.svg" width="140" alt="go-snap logo" />
</p>

# go-snap

[![CI](https://github.com/dzonerzy/go-snap/actions/workflows/ci.yml/badge.svg)](https://github.com/dzonerzy/go-snap/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/dzonerzy/go-snap?include_prereleases&sort=semver)](https://github.com/dzonerzy/go-snap/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/dzonerzy/go-snap.svg)](https://pkg.go.dev/github.com/dzonerzy/go-snap)
[![Go Version](https://img.shields.io/badge/go-1.22%2B-blue.svg)](https://go.dev/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/dzonerzy/go-snap)](https://goreportcard.com/report/github.com/dzonerzy/go-snap)

A lean, high-performance Go library for building command-line tools. go-snap focuses on zero-allocation parsing, a fluent and type-safe API, and pragmatic features that matter in real CLIs: smart errors, configuration precedence, first-class middleware, powerful wrappers for existing tools, and terminal/ANSI helpers.

- Zero-allocation parser with typed storage and string interning
- Fluent builders for apps, commands, flags, and flag groups
- Type-safe positional arguments (required, optional, variadic, pass-through)
- Smart suggestions for unknown flags/commands and contextual group help
- Config from struct tags (defaults/file/env/flags) with strict precedence
- Built-in middleware: logger, recovery, timeout, validator
- Wrapper DSL for enhancing existing CLIs (incl. dynamic toolexec shims)
- IO manager for TTY/size/color (Windows VT supported)

Project documentation lives in `docs/`.

- [Quick Start](docs/quickstart.md)
- [Full API and guides](docs/README.md)

## Performance

go-snap is **4-10x faster** than popular Go CLI libraries with **3x less memory** usage:

| Library | Speed | Memory | Allocations |
|---------|-------|--------|-------------|
| **go-snap** | **1.7-2.0 μs** | **5-6 KB** | **33-35** |
| Cobra | 7-9 μs | 17-20 KB | 119-149 |
| urfave/cli | 8-21 μs | 8-16 KB | 254-595 |

*Benchmarked on AMD Ryzen 9 9950X3D. See [benchmark/COMPETITIVE_BENCHMARKS.md](benchmark/COMPETITIVE_BENCHMARKS.md) for details.*

**Migrating from another library?**
- [Migration Guide: Cobra → go-snap](docs/migration-from-cobra.md)
- [Migration Guide: urfave/cli → go-snap](docs/migration-from-urfave-cli.md)

## Install

```bash
go get github.com/dzonerzy/go-snap
```

## Quick taste

Hello CLI with commands, global flags, and an action:
```go
package main

import (
    "fmt"
    "github.com/dzonerzy/go-snap/snap"
)

func main() {
    app := snap.New("hello", "A tiny demo").
        Version("0.1.0").
        Author("you", "you@example.com").
        // global flags
        StringFlag("name", "Who to greet").Default("world").Global().Back().
        BoolFlag("verbose", "Verbose mode").Short('v').Global().Back()

    app.Command("greet", "Print a greeting").
        IntFlag("times", "How many times").Default(1).Back().
        Action(func(ctx *snap.Context) error {
            name := ctx.MustGlobalString("name", "world")
            times := ctx.MustInt("times", 1)
            if ctx.MustGlobalBool("verbose", false) {
                fmt.Fprintln(ctx.Stdout(), "[verbose] repeating:", times)
            }
            for i := 0; i < times; i++ {
                fmt.Fprintf(ctx.Stdout(), "Hello, %s!\n", name)
            }
            return nil
        })

    app.RunAndExit()
}
```

Configuration from struct tags (defaults → file → env → flags), plus auto-generated CLI with `FromFlags()`:
```go
package main

import (
    "log"
    "time"
    "github.com/dzonerzy/go-snap/snap"
)

type ServerConfig struct {
    Host string        `flag:"host" env:"HOST" default:"localhost" description:"Hostname"`
    Port int           `flag:"port" env:"PORT"  default:"8080"     description:"Port"`
    Debug bool         `flag:"debug" env:"DEBUG"`
    Timeout time.Duration `flag:"timeout" env:"TIMEOUT" default:"30s"`
    LogLevel string    `flag:"log-level" enum:"debug,info,warn,error" default:"info"`
}

func main() {
    var cfg ServerConfig
    app, err := snap.Config("server", "Production-ready server").
        FromEnv().
        FromFlags().
        Bind(&cfg).
        Build()
    if err != nil { log.Fatal(err) }
    if err := app.Run(); err != nil { log.Fatal(err) }
    // cfg is now populated with precedence applied
}
```

Wrapping an existing tool (preserve leading flags and inject tokens):
```go
app := snap.New("echo-wrap", "prefix echo output")
app.BoolFlag("n", "suppress trailing newline").Back()
app.Wrap("/bin/echo").
    ForwardUnknownFlags().
    ForwardArgs().
    LeadingFlags("-n","-e","-E").
    MapBoolFlag("n", "-n").
    InsertAfterLeadingFlags("[prefix]").
    Passthrough().
    Back()
```

For more examples, see the `examples/` folder and the guides in `docs/`.

## Status

Stable early beta focused on the core features. Please use
[GitHub Discussions](https://github.com/dzonerzy/go-snap/discussions)
to propose new features or share ideas.

## License

MIT License — see [LICENSE](./LICENSE).

See the [Changelog](./CHANGELOG.md) for release notes.

## Contributing

- Read `docs/README.md` (especially the Contributing to Docs section) to keep documentation accurate and consistent.
- Keep examples runnable and aligned with `examples/`.
- PRs with benchmarks or perf notes are welcome; please attach `go test -bench=. -benchmem` output for context.
