package main

import (
	"context"
	"fmt"

	"github.com/dzonerzy/go-snap/snap"
)

// Example CLI showing nested subcommands: myapp server up
//
// Usage:
//
//	go run ./examples/nested-subcommands server up
//	go run ./examples/nested-subcommands server down --force
func main() {
	app := snap.New("myapp", "Nested subcommands demo").
		Version("1.0.0")

	// Show contextual help automatically when users make mistakes (e.g., unknown flags)
	app.ErrorHandler().ShowHelpOnError(true)

	// Top-level command: server
	srv := app.Command("server", "Server management")

	// server up
	srv.Command("up", "Start the server").
		BoolFlag("dry-run", "Print the action without executing").Short('n').Back().
		Action(func(ctx *snap.Context) error {
			if dry, _ := ctx.Bool("dry-run"); dry {
				fmt.Println("[dry-run] server up")
				return nil
			}
			fmt.Println("server up")
			return nil
		})

	// server down
	srv.Command("down", "Stop the server").
		BoolFlag("force", "Force stop").Short('f').Back().
		Action(func(ctx *snap.Context) error {
			if force, _ := ctx.Bool("force"); force {
				fmt.Println("server down (forced)")
				return nil
			}
			fmt.Println("server down")
			return nil
		})

	// Run with process args
	_ = app.RunContext(context.Background())
}
