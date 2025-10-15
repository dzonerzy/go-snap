package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/dzonerzy/go-snap/snap"
)

// Example demonstrating RawArgs() usage for audit logging, debugging, and proxying
//
// Usage:
//   go run ./examples/raw-args-demo audit --verbose serve --port 8080 file1.txt
//   go run ./examples/raw-args-demo debug -abc test
//   go run ./examples/raw-args-demo compare --verbose process --workers 4 data.txt

func main() {
	app := snap.New("rawargs-demo", "Demonstrates RawArgs() usage").
		Version("1.0.0")

	// Global flags
	app.BoolFlag("verbose", "Verbose output").Short('v').Global().Back()

	// Audit logging example - log exact command as typed
	app.Command("audit", "Command with audit logging").
		IntFlag("port", "Port number").Default(8080).Back().
		Before(func(ctx *snap.Context) error {
			// Log the complete invocation for audit trail
			timestamp := time.Now().Format(time.RFC3339)
			rawCmd := strings.Join(ctx.RawArgs(), " ")

			fmt.Printf("[AUDIT] %s | User invoked: %s %s\n",
				timestamp, ctx.AppName(), rawCmd)

			return nil
		}).
		Action(func(ctx *snap.Context) error {
			port := ctx.MustInt("port", 8080)
			files := ctx.Args()

			fmt.Printf("\n[ACTION] Processing with port=%d, files=%v\n", port, files)
			return nil
		}).
		After(func(ctx *snap.Context) error {
			timestamp := time.Now().Format(time.RFC3339)
			fmt.Printf("[AUDIT] %s | Command completed\n", timestamp)
			return nil
		})

	// Debug example - show difference between raw and parsed args
	app.Command("debug", "Show raw vs parsed arguments").
		BoolFlag("a", "Flag A").Back().
		BoolFlag("b", "Flag B").Back().
		BoolFlag("c", "Flag C").Back().
		Action(func(ctx *snap.Context) error {
			raw := ctx.RawArgs()
			parsed := ctx.Args()

			fmt.Println("=== Argument Analysis ===")
			fmt.Printf("\nRaw arguments (as typed):\n")
			for i, arg := range raw {
				fmt.Printf("  [%d] %q\n", i, arg)
			}

			fmt.Printf("\nParsed positional arguments:\n")
			if len(parsed) == 0 {
				fmt.Println("  (none)")
			} else {
				for i, arg := range parsed {
					fmt.Printf("  [%d] %q\n", i, arg)
				}
			}

			fmt.Printf("\nTotal: %d raw args, %d positional args\n", len(raw), len(parsed))
			return nil
		})

	// Comparison example - show both raw and parsed side by side
	app.Command("compare", "Compare raw and parsed arguments").
		StringFlag("output", "Output format").Default("text").Back().
		IntFlag("workers", "Number of workers").Default(1).Back().
		Action(func(ctx *snap.Context) error {
			fmt.Println("╔═══════════════════════════════════════╗")
			fmt.Println("║     Raw vs Parsed Arguments           ║")
			fmt.Println("╚═══════════════════════════════════════╝")

			fmt.Println("\n📝 Raw Arguments (before parsing):")
			fmt.Println("   What the user actually typed:")
			raw := ctx.RawArgs()
			for i, arg := range raw {
				fmt.Printf("   %d. %q\n", i+1, arg)
			}

			fmt.Println("\n🔍 Parsed Arguments (after snap processing):")

			// Show flags
			fmt.Println("\n   Flags:")
			if v, ok := ctx.GlobalBool("verbose"); ok {
				fmt.Printf("   - verbose: %v (global)\n", v)
			}
			if output, ok := ctx.String("output"); ok {
				fmt.Printf("   - output: %q\n", output)
			}
			if workers, ok := ctx.Int("workers"); ok {
				fmt.Printf("   - workers: %d\n", workers)
			}

			// Show positional args
			fmt.Println("\n   Positional arguments:")
			parsed := ctx.Args()
			if len(parsed) == 0 {
				fmt.Println("   (none)")
			} else {
				for i, arg := range parsed {
					fmt.Printf("   %d. %q\n", i+1, arg)
				}
			}

			return nil
		})

	// Proxy example - forward complete command to another tool
	app.Command("proxy", "Proxy to another tool (simulation)").
		Action(func(ctx *snap.Context) error {
			raw := ctx.RawArgs()

			fmt.Println("🔄 Proxy Mode: Forwarding to external tool")
			fmt.Printf("\nWould execute: external-tool %s\n", strings.Join(raw, " "))
			fmt.Printf("\n(In real usage, you would exec.Command(\"external-tool\", raw...))\n")

			return nil
		})

	app.RunAndExit()
}
