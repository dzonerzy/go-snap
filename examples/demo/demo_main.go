package main

import (
	"fmt"

	"github.com/dzonerzy/go-snap/snap"
)

func main() {
	fmt.Println("=== Running DemoHelpSystem ===")

	app := snap.New("myapp", "My awesome CLI application").
		HelpText("This is a comprehensive CLI application that demonstrates\nthe go-snap library capabilities including commands, flags,\nand help system.\n\nFor more information, visit: https://example.com").
		Version("1.0.0").
		Author("John Doe", "john@example.com").
		FlagGroup("Global Options").MutuallyExclusive().
		StringFlag("config", "Configuration file").Global().Back().
		IntFlag("age", "Your age").Default(30).Global().Back().
		EndGroup()

	app.Command("serve", "Start the web server").
		HelpText("Start the web server on the specified port.\n\nThis command will start a web server and listen for incoming connections.\nYou can specify the port using the --port flag.").
		StringFlag("host", "Server hostname").Default("localhost").Back().
		IntFlag("port", "Server port").Default(8080).Back()

	app.Command("static", "Serve static files").
		HelpText("Serve static files from a directory.\n\nThis subcommand serves static files from the specified directory.")

	err := app.Run()
	if err != nil {
		fmt.Println("Error:", err)
	}
}
