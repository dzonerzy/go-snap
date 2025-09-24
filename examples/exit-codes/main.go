package main

import (
    "errors"
    "fmt"
    "os"
    "github.com/dzonerzy/go-snap/snap"
)

var ErrNotFound = errors.New("resource not found")

func main() {
    app := snap.New("exit-demo", "Exit code management demo")

    // Customize exit code mapping
    app.ExitCodes().
        Define("not_found", 127). // conventional: 127 for not found
        DefineError(ErrNotFound, 127)

    app.Command("success", "Return success").Action(func(ctx *snap.Context) error {
        fmt.Fprintln(ctx.Stdout(), "everything ok")
        return nil
    })

    app.Command("not-found", "Return specific not-found error").Action(func(ctx *snap.Context) error {
        return ErrNotFound
    })

    app.Command("custom-exit", "Request exit programmatically").Action(func(ctx *snap.Context) error {
        fmt.Fprintln(ctx.Stdout(), "exiting with code 42…")
        ctx.Exit(42)
        return nil
    })

    // Default: if no subcommand, show help and exit with success
    if len(os.Args) < 2 {
        _ = app.Run() // prints help
        os.Exit(0)
    }
    app.RunAndExit()
}
