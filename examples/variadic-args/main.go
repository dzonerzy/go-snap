package main

import (
	"fmt"
	"os"

	"github.com/dzonerzy/go-snap/snap"
)

func main() {
	app := snap.New("fileops", "File operations with variadic arguments").
		Version("1.0.0")

	// Example 1: rm-style command - remove multiple files
	app.Command("rm", "Remove one or more files").
		StringSliceArg("files", "Files to remove").Required().Variadic().
		Action(rmAction)

	// Example 2: cat-style command - concatenate multiple files
	app.Command("cat", "Concatenate and display file contents").
		StringSliceArg("files", "Files to display").Required().Variadic().
		Action(catAction)

	// Example 3: sum command - sum multiple numbers
	app.Command("sum", "Calculate sum of numbers").
		IntSliceArg("numbers", "Numbers to sum").Required().Variadic().
		Action(sumAction)

	// Example 4: copy-many - copy multiple source files to a destination
	app.Command("copy-many", "Copy multiple files to destination directory").
		StringArg("dest", "Destination directory").Required().Back().
		StringSliceArg("sources", "Source files").Required().Variadic().
		Action(copyManyAction)

	// Example 5: RestArgs - pass-through style (like docker run)
	app.Command("docker-run", "Simulate docker run with pass-through args").
		RestArgs().
		Action(dockerRunAction)

	if err := app.Run(); err != nil {
		if err != snap.ErrHelpShown && err != snap.ErrVersionShown {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func rmAction(ctx *snap.Context) error {
	files := ctx.MustArgStringSlice("files", []string{})

	fmt.Printf("Removing %d file(s):\n", len(files))
	for i, file := range files {
		fmt.Printf("  [%d] %s\n", i+1, file)
	}
	fmt.Println("\n✓ All files removed successfully!")
	return nil
}

func catAction(ctx *snap.Context) error {
	files := ctx.MustArgStringSlice("files", []string{})

	fmt.Printf("Displaying contents of %d file(s):\n\n", len(files))
	for _, file := range files {
		fmt.Printf("=== %s ===\n", file)
		fmt.Printf("(content of %s would appear here)\n\n", file)
	}
	return nil
}

func sumAction(ctx *snap.Context) error {
	numbers := ctx.MustArgIntSlice("numbers", []int{})

	sum := 0
	fmt.Printf("Summing %d number(s):\n", len(numbers))
	for i, num := range numbers {
		sum += num
		fmt.Printf("  [%d] %d (running total: %d)\n", i+1, num, sum)
	}
	fmt.Printf("\n✓ Final sum: %d\n", sum)
	return nil
}

func copyManyAction(ctx *snap.Context) error {
	dest := ctx.MustArgString("dest", "")
	sources := ctx.MustArgStringSlice("sources", []string{})

	fmt.Printf("Copying %d file(s) to %s:\n", len(sources), dest)
	for i, source := range sources {
		fmt.Printf("  [%d] %s → %s\n", i+1, source, dest)
	}
	fmt.Println("\n✓ All files copied successfully!")
	return nil
}

func dockerRunAction(ctx *snap.Context) error {
	args := ctx.RestArgs()

	fmt.Printf("Docker run simulation with %d argument(s):\n", len(args))
	fmt.Printf("Command: docker run %s\n\n", formatArgs(args))

	fmt.Println("Parsed arguments:")
	for i, arg := range args {
		fmt.Printf("  [%d] %s\n", i, arg)
	}

	fmt.Println("\n✓ Container would be started with these arguments")
	return nil
}

func formatArgs(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		// Quote args with spaces
		if containsSpace(arg) {
			result += fmt.Sprintf("\"%s\"", arg)
		} else {
			result += arg
		}
	}
	return result
}

func containsSpace(s string) bool {
	for _, c := range s {
		if c == ' ' {
			return true
		}
	}
	return false
}
