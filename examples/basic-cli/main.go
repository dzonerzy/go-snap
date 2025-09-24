package main

import (
    "fmt"
    "strings"

    "github.com/dzonerzy/go-snap/snap"
)

// A minimal CLI showing commands, global vs command flags, args, and help/version.
func main() {
    app := snap.New("basic", "Basic CLI showcasing go-snap").
        Version("1.0.0").
        Author("Go Snap", "support@example.com").
        // Global flags
        StringFlag("name", "Your name").Default("world").Global().Back().
        BoolFlag("verbose", "Verbose output").Short('v').Global().Back()

    // greet command: prints a greeting N times
    app.Command("greet", "Print a friendly greeting").
        IntFlag("times", "How many times").Default(1).Back().
        Action(func(ctx *snap.Context) error {
            name := ctx.MustGlobalString("name", "world")
            times := ctx.MustInt("times", 1)
            verbose := ctx.MustGlobalBool("verbose", false)
            msg := fmt.Sprintf("Hello, %s!", name)
            if verbose {
                fmt.Fprintln(ctx.Stdout(), "[verbose] repeating:", times)
            }
            for i := 0; i < times; i++ {
                fmt.Fprintln(ctx.Stdout(), msg)
            }
            return nil
        })

    // echo command: shows positional args usage
    app.Command("echo", "Echo all positional args").
        Action(func(ctx *snap.Context) error {
            fmt.Fprintln(ctx.Stdout(), strings.Join(ctx.Args(), " "))
            return nil
        })

    // Default to help when no command is given
    app.RunAndExit()
}

