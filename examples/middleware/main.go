package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dzonerzy/go-snap/middleware"
	"github.com/dzonerzy/go-snap/snap"
)

func main() {
	app := snap.New("server", "A demo server application with middleware").
		Version("1.0.0").
		Author("go-snap", "support@go-snap.dev").
		// Global middleware applied to all commands
        Use(
            middleware.Logger(middleware.WithLogLevel(middleware.LogLevelInfo)),
            middleware.Recovery(middleware.WithStackTrace(true)),
            middleware.Timeout(30*time.Second),
            // Friendlier validation syntax
            middleware.Validate(
                middleware.Custom("port_range", validatePortRange),
            ),
        ).
		// Global flags with environment variable support
		BoolFlag("verbose", "Enable verbose logging").
		Short('v').
		FromEnv("VERBOSE", "DEBUG").Back().
        StringFlag("config", "Configuration file").
        Default("config/config.json").Global().
        FromEnv("CONFIG_FILE", "APP_CONFIG").Back()

	// Serve command with command-specific middleware
	app.Command("serve", "Start the HTTP server").
		// Command-specific middleware (runs after global middleware)
            Use(
                middleware.Validate(
                    middleware.Custom("port_range", validatePortRange),
                ),
            ).
		IntFlag("port", "Server port").
		Default(8080).
		FromEnv("PORT", "SERVER_PORT").Back().
		StringFlag("host", "Server host").
		Default("localhost").
		FromEnv("HOST", "SERVER_HOST").Back().
		Action(serveAction)

	// Status command (only uses global middleware)
	app.Command("status", "Check server status").
		Action(statusAction)

	// Config command with different middleware configuration
    app.Command("config", "Manage configuration").
        Use(
            middleware.JSONLogger(), // JSON format for config operations
        ).
        Command("validate", "Validate configuration file").
        // Ensure the provided --config points to an existing file
        Use(middleware.Validate(middleware.File("config"))).
        Action(validateConfigAction)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// serveAction starts the HTTP server
func serveAction(ctx *snap.Context) error {
	port, _ := ctx.Int("port")
	host, _ := ctx.String("host")
	verbose, _ := ctx.GlobalBool("verbose")

	if verbose {
		fmt.Printf("Starting server in verbose mode...\n")
	}

	fmt.Printf("Server starting on %s:%d\n", host, port)

	// Simulate server work
	time.Sleep(2 * time.Second)
	fmt.Println("Server started successfully!")

	return nil
}

// statusAction checks server status
func statusAction(ctx *snap.Context) error {
	fmt.Println("Checking server status...")

	// Simulate status check
	time.Sleep(500 * time.Millisecond)
	fmt.Println("Server status: OK")

	return nil
}

// validateConfigAction validates the configuration file
func validateConfigAction(ctx *snap.Context) error {
	config, _ := ctx.GlobalString("config")

	fmt.Printf("Validating configuration file: %s\n", config)

	// Simulate config validation
	time.Sleep(1 * time.Second)
	fmt.Println("Configuration is valid!")

	return nil
}

// validatePortRange is a custom validator function
func validatePortRange(ctx middleware.Context) error {
	if port, exists := ctx.Int("port"); exists {
		if port < 1 || port > 65535 {
			return &middleware.ValidationError{
				Field:   "port",
				Value:   port,
				Message: "port must be between 1 and 65535",
			}
		}
		if port < 1024 {
			fmt.Fprintf(os.Stderr, "Warning: using privileged port %d (requires root)\n", port)
		}
	}
	return nil
}
