package main

import (
	"time"

	snapio "github.com/dzonerzy/go-snap/io"
	"github.com/dzonerzy/go-snap/snap"
)

// Example demonstrating the structured logging system with various formats and customization
//
// Usage:
//   go run ./examples/logging-demo deploy
//   go run ./examples/logging-demo deploy --format symbols
//   go run ./examples/logging-demo deploy --format tagged
//   go run ./examples/logging-demo deploy --format custom
//   go run ./examples/logging-demo deploy --timestamp

func main() {
	app := snap.New("logging-demo", "Demonstrates structured logging with various formats").
		Version("1.0.0")

	// Global flag for log format
	app.StringFlag("format", "Log format (circles, symbols, tagged, plain, custom)").
		Default("circles").
		Global().
		Back()

	app.BoolFlag("timestamp", "Include timestamps in log output").
		Global().
		Back()

	// Deploy command - demonstrates all log levels
	app.Command("deploy", "Deploy application with detailed logging").
		StringFlag("env", "Target environment").Default("production").Back().
		IntFlag("replicas", "Number of replicas").Default(3).Back().
		Action(func(ctx *snap.Context) error {
			// Configure logger based on flags
			configureLogger(ctx)

			env, _ := ctx.String("env")
			replicas, _ := ctx.Int("replicas")

			ctx.LogInfo("Starting deployment to %s environment", env)
			ctx.LogDebug("Deployment configuration: replicas=%d", replicas)

			// Simulate deployment steps
			time.Sleep(200 * time.Millisecond)
			ctx.LogInfo("Building Docker image...")

			time.Sleep(200 * time.Millisecond)
			ctx.LogSuccess("Docker image built successfully")

			ctx.LogInfo("Pushing image to registry...")
			time.Sleep(200 * time.Millisecond)
			ctx.LogSuccess("Image pushed to registry")

			ctx.LogWarning("Existing pods will be terminated")
			ctx.LogInfo("Rolling out %d new pods...", replicas)
			time.Sleep(300 * time.Millisecond)

			ctx.LogSuccess("Deployment to %s complete!", env)
			ctx.LogInfo("Application is live at https://%s.example.com", env)

			return nil
		})

	// Database command - demonstrates error logging
	app.Command("db-migrate", "Run database migrations").
		BoolFlag("dry-run", "Show what would be done without executing").Back().
		Action(func(ctx *snap.Context) error {
			configureLogger(ctx)

			dryRun, _ := ctx.Bool("dry-run")

			if dryRun {
				ctx.LogWarning("Running in dry-run mode - no changes will be made")
			}

			ctx.LogInfo("Connecting to database...")
			time.Sleep(100 * time.Millisecond)
			ctx.LogSuccess("Connected to database")

			ctx.LogInfo("Checking for pending migrations...")
			time.Sleep(100 * time.Millisecond)

			ctx.LogDebug("Found 3 pending migrations")
			ctx.LogInfo("Applying migration 001_create_users_table")
			time.Sleep(150 * time.Millisecond)
			ctx.LogSuccess("Migration 001 applied")

			ctx.LogInfo("Applying migration 002_add_indexes")
			time.Sleep(150 * time.Millisecond)
			ctx.LogSuccess("Migration 002 applied")

			ctx.LogInfo("Applying migration 003_add_timestamps")
			time.Sleep(150 * time.Millisecond)

			// Simulate an error
			if !dryRun {
				ctx.LogError("Failed to apply migration 003: column 'created_at' already exists")
				ctx.LogWarning("Rolling back migration 002...")
				time.Sleep(100 * time.Millisecond)
				ctx.LogInfo("Rollback complete")
				return snap.NewError(snap.ErrorTypeValidation, "migration failed")
			}

			ctx.LogSuccess("All migrations completed successfully")
			return nil
		})

	// Format showcase - shows all formats side by side
	app.Command("showcase", "Show all log formats").
		Action(func(ctx *snap.Context) error {
			formats := []struct {
				name   string
				format snapio.LogFormat
			}{
				{"Circles (Default)", snapio.LogFormatCircles},
				{"Symbols", snapio.LogFormatSymbols},
				{"Tagged", snapio.LogFormatTagged},
				{"Plain", snapio.LogFormatPlain},
			}

			for _, f := range formats {
				ctx.App.Logger().WithFormat(f.format)
				ctx.LogInfo("=== %s ===", f.name)
				ctx.LogDebug("This is a debug message")
				ctx.LogInfo("This is an info message")
				ctx.LogSuccess("This is a success message")
				ctx.LogWarning("This is a warning message")
				ctx.LogError("This is an error message")
				ctx.LogInfo("") // Empty line for separation
			}

			// Custom format example
			ctx.App.Logger().WithFormat(snapio.LogFormatCustom).
				WithTemplate("[{{.Level}}] {{.Time}} - {{.Message}}")
			ctx.LogInfo("=== Custom Template ===")
			ctx.LogInfo("Custom formatted message with timestamp")
			ctx.LogSuccess("Another custom message")

			return nil
		})

	// Timestamp demo
	app.Command("with-timestamps", "Demonstrate timestamp logging").
		Action(func(ctx *snap.Context) error {
			ctx.App.Logger().WithTimestamp(true)

			ctx.LogInfo("Starting task 1...")
			time.Sleep(500 * time.Millisecond)
			ctx.LogSuccess("Task 1 completed")

			ctx.LogInfo("Starting task 2...")
			time.Sleep(700 * time.Millisecond)
			ctx.LogSuccess("Task 2 completed")

			ctx.LogInfo("Starting task 3...")
			time.Sleep(300 * time.Millisecond)
			ctx.LogError("Task 3 failed")

			// Custom time format
			ctx.App.Logger().WithTimeFormat("15:04:05.000")
			ctx.LogInfo("Using millisecond precision timestamps")

			return nil
		})

	app.RunAndExit()
}

// configureLogger applies user-specified format preferences
func configureLogger(ctx *snap.Context) {
	format, _ := ctx.GlobalString("format")
	timestamp, _ := ctx.GlobalBool("timestamp")

	switch format {
	case "circles":
		ctx.App.Logger().WithFormat(snapio.LogFormatCircles)
	case "symbols":
		ctx.App.Logger().WithFormat(snapio.LogFormatSymbols)
	case "tagged":
		ctx.App.Logger().WithFormat(snapio.LogFormatTagged)
	case "plain":
		ctx.App.Logger().WithFormat(snapio.LogFormatPlain)
	case "custom":
		ctx.App.Logger().WithFormat(snapio.LogFormatCustom).
			WithTemplate("[{{.Level}}] {{.Message}}")
	}

	if timestamp {
		ctx.App.Logger().WithTimestamp(true)
	}
}
