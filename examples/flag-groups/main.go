package main

import (
    "fmt"
    "github.com/dzonerzy/go-snap/snap"
)

// Showcases all flag-group constraints and the contextual help on violations.
func main() {
    app := snap.New("groups", "Flag groups constraints demo")

    // Output group: exactly one format
    app.FlagGroup("output").
        ExactlyOne().
        Description("Choose exactly one output format").
        BoolFlag("json", "JSON output").Short('j').Back().
        BoolFlag("yaml", "YAML output").Short('y').Back().
        BoolFlag("table", "Table output").Short('t').Back().
        EndGroup()

    // SSL group: all or none
    app.FlagGroup("ssl").
        AllOrNone().
        StringFlag("cert", "Path to certificate").Back().
        StringFlag("key", "Path to private key").Back().
        EndGroup()

    app.Command("run", "Execute with chosen options").Action(func(ctx *snap.Context) error {
        // Decide on format
        switch {
        case ctx.MustBool("json", false):
            fmt.Fprintln(ctx.Stdout(), `{"status":"ok"}`)
        case ctx.MustBool("yaml", false):
            fmt.Fprintln(ctx.Stdout(), "status: ok")
        case ctx.MustBool("table", false):
            fmt.Fprintln(ctx.Stdout(), "STATUS\n ok")
        }
        if c, ok := ctx.String("cert"); ok {
            k, _ := ctx.String("key")
            fmt.Fprintf(ctx.Stdout(), "using SSL cert=%s key=%s\n", c, k)
        }
        return nil
    })

    app.RunAndExit()
}
