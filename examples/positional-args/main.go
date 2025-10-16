package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dzonerzy/go-snap/snap"
)

func main() {
	app := snap.New("fileutil", "File utility with positional arguments").
		Version("1.0.0").
		Author("Example Author", "author@example.com")

	// Example 1: Copy command with required source and optional destination
	app.Command("copy", "Copy a file from source to destination").
		StringArg("source", "Source file path").Required().Back().
		StringArg("dest", "Destination file path").Default("output.txt").
		Action(copyAction)

	// Example 2: Convert command with type-safe arguments
	app.Command("convert", "Convert file with various options").
		StringArg("input", "Input file").Required().Back().
		StringArg("output", "Output file").Required().Back().
		IntArg("quality", "Quality level (1-100)").Default(80).
		BoolArg("verbose", "Verbose output").Default(false).
		Action(convertAction)

	// Example 3: Process command with different types
	app.Command("process", "Process file with timeout and threshold").
		StringArg("file", "File to process").Required().Back().
		DurationArg("timeout", "Processing timeout").Default(30*time.Second).
		FloatArg("threshold", "Threshold value").Default(0.5).
		Action(processAction)

	if err := app.Run(); err != nil {
		if err != snap.ErrHelpShown && err != snap.ErrVersionShown {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func copyAction(ctx *snap.Context) error {
	source := ctx.MustArgString("source", "")
	dest := ctx.MustArgString("dest", "output.txt")
	overwrite := ctx.MustBool("overwrite", false)

	fmt.Printf("Copying file:\n")
	fmt.Printf("  Source: %s\n", source)
	fmt.Printf("  Destination: %s\n", dest)
	fmt.Printf("  Overwrite: %v\n", overwrite)

	// Actual copy logic would go here
	fmt.Println("\n✓ File copied successfully!")
	return nil
}

func convertAction(ctx *snap.Context) error {
	input := ctx.MustArgString("input", "")
	output := ctx.MustArgString("output", "")
	quality := ctx.MustArgInt("quality", 80)
	verbose := ctx.MustArgBool("verbose", false)

	fmt.Printf("Converting file:\n")
	fmt.Printf("  Input: %s\n", input)
	fmt.Printf("  Output: %s\n", output)
	fmt.Printf("  Quality: %d\n", quality)
	fmt.Printf("  Verbose: %v\n", verbose)

	if verbose {
		fmt.Println("\n[VERBOSE] Processing conversion...")
		fmt.Println("[VERBOSE] Applying quality settings...")
	}

	fmt.Println("\n✓ Conversion completed!")
	return nil
}

func processAction(ctx *snap.Context) error {
	file := ctx.MustArgString("file", "")
	timeout := ctx.MustArgDuration("timeout", 0)
	threshold := ctx.MustArgFloat("threshold", 0.5)

	fmt.Printf("Processing file:\n")
	fmt.Printf("  File: %s\n", file)
	fmt.Printf("  Timeout: %v\n", timeout)
	fmt.Printf("  Threshold: %.2f\n", threshold)

	fmt.Println("\n✓ Processing completed!")
	return nil
}
