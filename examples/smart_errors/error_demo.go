package main

import (
	"github.com/dzonerzy/go-snap/snap"
)

func main() {
	app := snap.New("demo", "Demo app showing smart error handling").
		// Create a flag group with mutually exclusive options
		FlagGroup("output").
		MutuallyExclusive().
		Description("Output format selection").
		BoolFlag("json", "Output as JSON").Short('j').Back().
		BoolFlag("yaml", "Output as YAML").Short('y').Back().
		BoolFlag("table", "Output as table").Short('t').Back().
		EndGroup().
		// Regular flags
		StringFlag("config", "Configuration file").Short('c').Back().
		IntFlag("port", "Server port").Short('p').Default(8080).Back()

	// Configure error handler for smart suggestions
	app.ErrorHandler().
		SuggestFlags(true).
		SuggestCommands(true).
		MaxDistance(2)

	// Demonstrate different types of errors
	err := app.Run()
	if err != nil {
		// Print the error message with suggestions if applicable
		println("Error:", err.Error())
	}
}
