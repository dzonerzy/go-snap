package main

import (
	"fmt"
	"runtime"

	"github.com/dzonerzy/go-snap/snap"
)

// Example demonstrating how to create a custom version command
// using Context app metadata accessors.
//
// Usage:
//   go run ./examples/version-command version
//   go run ./examples/version-command version --verbose

func main() {
	app := snap.New("myapp", "A CLI tool demonstrating version command").
		Version("1.2.3").
		Author("Alice Smith", "alice@example.com").
		Author("Bob Johnson", "bob@example.com")

	app.Command("version", "Display version information").
		BoolFlag("verbose", "Show detailed version information").Short('v').Back().
		Action(func(ctx *snap.Context) error {
			verbose, _ := ctx.Bool("verbose")

			if verbose {
				// Detailed version output
				fmt.Fprintf(ctx.Stdout(), "%s version %s\n\n", ctx.AppName(), ctx.AppVersion())
				fmt.Fprintf(ctx.Stdout(), "Description: %s\n", ctx.AppDescription())

				authors := ctx.AppAuthors()
				if len(authors) > 0 {
					fmt.Fprintf(ctx.Stdout(), "\nAuthors:\n")
					for _, author := range authors {
						fmt.Fprintf(ctx.Stdout(), "  • %s <%s>\n", author.Name, author.Email)
					}
				}

				fmt.Fprintf(ctx.Stdout(), "\nBuild Information:\n")
				fmt.Fprintf(ctx.Stdout(), "  Go Version: %s\n", runtime.Version())
				fmt.Fprintf(ctx.Stdout(), "  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
			} else {
				// Simple version output
				fmt.Fprintf(ctx.Stdout(), "%s v%s\n", ctx.AppName(), ctx.AppVersion())
			}

			return nil
		})

	app.Command("serve", "Start the server").
		IntFlag("port", "Server port").Default(8080).Back().
		Action(func(ctx *snap.Context) error {
			port := ctx.MustInt("port", 8080)
			fmt.Fprintf(ctx.Stdout(), "[%s v%s] Starting server on port %d...\n",
				ctx.AppName(), ctx.AppVersion(), port)
			// Server logic here
			return nil
		})

	app.RunAndExit()
}
