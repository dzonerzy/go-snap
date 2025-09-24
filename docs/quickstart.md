# Quick Start

This guide gets you up and running with go-snap in minutes.

Install
```bash
go get github.com/dzonerzy/go-snap
```

Hello, world
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

Try it
```bash
hello greet --times 3 --name Ada
hello greet -v
hello --help
hello --version
```

Where next?
- [App & Commands](./app-and-commands.md)
- [Flags & Groups](./flags-and-groups.md)
- [Configuration](./configuration.md)
- [Middleware](./middleware.md)
