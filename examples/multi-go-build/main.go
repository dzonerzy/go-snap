package main

import (
	"fmt"
	"os"

	"github.com/dzonerzy/go-snap/snap"
)

func main() {
	app := snap.New("multi-go-build", "Build with multiple Go versions")

	// Example 1: Sequential execution (default)
	app.Command("build-seq", "Build sequentially with multiple Go versions").
		WrapMany("go1.21.0", "go1.22.0", "go1.23.0").
		InjectArgsPre("build").
		ForwardArgs().
		StopOnError(false). // continue even if one fails
		AfterExec(func(ctx *snap.Context, result *snap.ExecResult) error {
			binary := ctx.CurrentBinary()
			if result.ExitCode == 0 {
				fmt.Fprintf(os.Stderr, "✓ %s: build succeeded\n", binary)
			} else {
				fmt.Fprintf(os.Stderr, "✗ %s: build failed (exit %d)\n", binary, result.ExitCode)
			}
			return nil
		}).
		Back()

	// Example 2: Parallel execution
	app.Command("build-parallel", "Build in parallel with multiple Go versions").
		WrapMany("go1.21.0", "go1.22.0", "go1.23.0").
		Parallel().
		InjectArgsPre("build").
		ForwardArgs().
		StopOnError(false).
		AfterExec(func(ctx *snap.Context, result *snap.ExecResult) error {
			binary := ctx.CurrentBinary()
			if result.ExitCode == 0 {
				fmt.Fprintf(os.Stderr, "✓ %s: build succeeded\n", binary)
			} else {
				fmt.Fprintf(os.Stderr, "✗ %s: build failed (exit %d)\n", binary, result.ExitCode)
			}
			return nil
		}).
		Back()

	// Example 3: With version reporting
	app.Command("versions", "Report Go versions").
		WrapMany("go1.21.0", "go1.22.0", "go1.23.0").
		InjectArgsPre("version").
		Capture().
		AfterExec(func(ctx *snap.Context, result *snap.ExecResult) error {
			binary := ctx.CurrentBinary()
			version := string(result.Stdout)
			fmt.Printf("%s: %s\n", binary, version)
			return nil
		}).
		Back()

	app.RunAndExit()
}
