package main

import (
	"fmt"
	"time"

	"github.com/dzonerzy/go-snap/snap"
)

// Example demonstrating Before/After hooks for commands and BeforeExec/AfterExec for wrappers
//
// Usage:
//   go run ./examples/lifecycle-hooks deploy --env prod
//   go run ./examples/lifecycle-hooks docker-build --tag v1.0.0

func main() {
	app := snap.New("lifecycle-demo", "Demonstrates Before/After lifecycle hooks").
		Version("1.0.0")

	// App-level hooks - run for ALL commands
	app.Before(func(ctx *snap.Context) error {
		fmt.Println("🚀 [App Before] Starting application...")
		return nil
	})

	app.After(func(ctx *snap.Context) error {
		fmt.Println("✅ [App After] Application completed successfully")
		return nil
	})

	// Command with Before/After hooks for setup and cleanup
	app.Command("deploy", "Deploy application to environment").
		StringFlag("env", "Target environment").Default("staging").Back().
		Before(func(ctx *snap.Context) error {
			env, _ := ctx.String("env")
			fmt.Printf("📋 [Deploy Before] Validating credentials for %s...\n", env)
			fmt.Printf("📋 [Deploy Before] Checking deployment requirements...\n")
			time.Sleep(100 * time.Millisecond) // Simulate validation
			fmt.Println("✓  Validation complete")
			return nil
		}).
		Action(func(ctx *snap.Context) error {
			env, _ := ctx.String("env")
			fmt.Printf("🔨 [Deploy Action] Deploying to %s environment...\n", env)
			time.Sleep(200 * time.Millisecond) // Simulate deployment
			fmt.Println("✓  Deployment successful")
			return nil
		}).
		After(func(ctx *snap.Context) error {
			env, _ := ctx.String("env")
			fmt.Printf("🔔 [Deploy After] Sending notification to team...\n")
			fmt.Printf("🔔 [Deploy After] Deployment to %s is live!\n", env)
			return nil
		})

	// Wrapper with BeforeExec/AfterExec hooks for enhanced docker build
	app.Command("docker-build", "Enhanced Docker build with logging").
		StringFlag("tag", "Docker image tag").Required().Back().
		StringFlag("platform", "Target platform").Default("linux/amd64").Back().
		Wrap("docker").
		BeforeExec(func(ctx *snap.Context, args []string) ([]string, error) {
			tag, _ := ctx.String("tag")
			platform, _ := ctx.String("platform")

			fmt.Println("🐳 [BeforeExec] Preparing Docker build...")
			fmt.Printf("   Tag: %s\n", tag)
			fmt.Printf("   Platform: %s\n", platform)

			// Inject build arguments
			finalArgs := []string{
				"build",
				"--platform", platform,
				"--tag", tag,
				"--build-arg", fmt.Sprintf("BUILD_TIME=%s", time.Now().Format(time.RFC3339)),
				".",
			}

			fmt.Printf("   Final args: %v\n", finalArgs)
			return finalArgs, nil
		}).
		Passthrough().
		AfterExec(func(ctx *snap.Context, result *snap.ExecResult) error {
			fmt.Println("📊 [AfterExec] Docker build completed!")
			fmt.Printf("   Exit Code: %d\n", result.ExitCode)

			if result.Error != nil {
				fmt.Printf("   ❌ Build failed: %v\n", result.Error)
			} else {
				tag, _ := ctx.String("tag")
				fmt.Printf("   ✅ Image %s built successfully!\n", tag)
				fmt.Println("   💾 Logging build metrics...")
				// Here you could log to monitoring system, send notifications, etc.
			}
			return nil
		}).
		Back()

	// Wrapper example with error handling in AfterExec
	app.Command("enhanced-test", "Run tests with timing and notifications").
		Wrap("go").
		BeforeExec(func(ctx *snap.Context, args []string) ([]string, error) {
			fmt.Println("🧪 [BeforeExec] Starting test suite...")
			ctx.Set("start_time", time.Now())
			return append([]string{"test", "-v", "./..."}, args...), nil
		}).
		Passthrough().
		AfterExec(func(ctx *snap.Context, result *snap.ExecResult) error {
			startTime := ctx.Get("start_time").(time.Time)
			duration := time.Since(startTime)

			fmt.Printf("\n📊 [AfterExec] Test Results:\n")
			fmt.Printf("   Duration: %v\n", duration)
			fmt.Printf("   Exit Code: %d\n", result.ExitCode)

			if result.ExitCode == 0 {
				fmt.Println("   ✅ All tests passed!")
			} else {
				fmt.Println("   ❌ Some tests failed")
			}

			return nil
		}).
		Back()

	app.RunAndExit()
}
