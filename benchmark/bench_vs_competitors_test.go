package benchmark_test

import (
	"context"
	"testing"

	"github.com/dzonerzy/go-snap/snap"
	"github.com/spf13/cobra"
	"github.com/urfave/cli/v2"
)

// Benchmark simple CLI with basic flags
// Tests parsing performance with int and bool flags
// All three execute a command with flags for fair comparison

func BenchmarkSimpleCLI_GoSnap(b *testing.B) {
	app := snap.New("bench", "benchmark app")
	app.Command("run", "Run benchmark").
		IntFlag("port", "Server port").Default(8080).Back().
		BoolFlag("verbose", "Verbose output").Back().
		Action(func(_ *snap.Context) error { return nil })

	args := []string{"run", "--port", "9000", "--verbose"}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = app.RunWithArgs(context.Background(), args)
	}
}

func BenchmarkSimpleCLI_Cobra(b *testing.B) {
	args := []string{"run", "--port", "9000", "--verbose"}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rootCmd := &cobra.Command{Use: "bench"}
		runCmd := &cobra.Command{
			Use: "run",
			Run: func(_ *cobra.Command, _ []string) {},
		}
		runCmd.Flags().IntP("port", "p", 8080, "Server port")
		runCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
		rootCmd.AddCommand(runCmd)
		rootCmd.SetArgs(args)
		_ = rootCmd.Execute()
	}
}

func BenchmarkSimpleCLI_Urfave(b *testing.B) {
	args := []string{"bench", "run", "--port", "9000", "--verbose"}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		app := &cli.App{
			Name: "bench",
			Commands: []*cli.Command{
				{
					Name: "run",
					Flags: []cli.Flag{
						&cli.IntFlag{Name: "port", Value: 8080, Usage: "Server port"},
						&cli.BoolFlag{Name: "verbose", Usage: "Verbose output"},
					},
					Action: func(_ *cli.Context) error { return nil },
				},
			},
		}
		_ = app.Run(args)
	}
}

// Benchmark with subcommands
// Tests command routing and flag parsing in subcommands

func BenchmarkSubcommands_GoSnap(b *testing.B) {
	app := snap.New("bench", "benchmark app").
		BoolFlag("global", "Global flag").Back()
	app.Command("serve", "Start server").
		IntFlag("port", "Server port").Default(8080).Back().
		StringFlag("host", "Server host").Default("localhost").Back().
		Action(func(_ *snap.Context) error { return nil })

	args := []string{"--global", "serve", "--port", "9000", "--host", "0.0.0.0"}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = app.RunWithArgs(context.Background(), args)
	}
}

func BenchmarkSubcommands_Cobra(b *testing.B) {
	args := []string{"--global", "serve", "--port", "9000", "--host", "0.0.0.0"}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rootCmd := &cobra.Command{Use: "bench"}
		rootCmd.PersistentFlags().Bool("global", false, "Global flag")

		serveCmd := &cobra.Command{
			Use: "serve",
			Run: func(_ *cobra.Command, _ []string) {},
		}
		serveCmd.Flags().IntP("port", "p", 8080, "Server port")
		serveCmd.Flags().String("host", "localhost", "Server host") // Removed -h shorthand to avoid conflict with help
		rootCmd.AddCommand(serveCmd)

		rootCmd.SetArgs(args)
		_ = rootCmd.Execute()
	}
}

func BenchmarkSubcommands_Urfave(b *testing.B) {
	args := []string{"bench", "--global", "serve", "--port", "9000", "--host", "0.0.0.0"}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		app := &cli.App{
			Name: "bench",
			Flags: []cli.Flag{
				&cli.BoolFlag{Name: "global", Usage: "Global flag"},
			},
			Commands: []*cli.Command{
				{
					Name: "serve",
					Flags: []cli.Flag{
						&cli.IntFlag{Name: "port", Value: 8080, Usage: "Server port"},
						&cli.StringFlag{Name: "host", Value: "localhost", Usage: "Server host"},
					},
					Action: func(_ *cli.Context) error { return nil },
				},
			},
		}
		_ = app.Run(args)
	}
}

// Benchmark many flags
// Tests performance with many flags (realistic CLI tool scenario)
// All three execute a command with multiple flags for fair comparison

func BenchmarkManyFlags_GoSnap(b *testing.B) {
	app := snap.New("bench", "benchmark app")
	app.Command("run", "Run benchmark").
		StringFlag("flag1", "Flag 1").Default("value1").Back().
		StringFlag("flag2", "Flag 2").Default("value2").Back().
		StringFlag("flag3", "Flag 3").Default("value3").Back().
		StringFlag("flag4", "Flag 4").Default("value4").Back().
		StringFlag("flag5", "Flag 5").Default("value5").Back().
		IntFlag("port", "Port").Default(8080).Back().
		BoolFlag("verbose", "Verbose").Back().
		BoolFlag("debug", "Debug").Back().
		BoolFlag("quiet", "Quiet").Back().
		BoolFlag("force", "Force").Back().
		Action(func(_ *snap.Context) error { return nil })

	args := []string{
		"run",
		"--flag1", "test1",
		"--flag2", "test2",
		"--flag3", "test3",
		"--port", "9000",
		"--verbose",
		"--debug",
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = app.RunWithArgs(context.Background(), args)
	}
}

func BenchmarkManyFlags_Cobra(b *testing.B) {
	args := []string{
		"run",
		"--flag1", "test1",
		"--flag2", "test2",
		"--flag3", "test3",
		"--port", "9000",
		"--verbose",
		"--debug",
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rootCmd := &cobra.Command{Use: "bench"}
		runCmd := &cobra.Command{
			Use: "run",
			Run: func(_ *cobra.Command, _ []string) {},
		}
		runCmd.Flags().String("flag1", "value1", "Flag 1")
		runCmd.Flags().String("flag2", "value2", "Flag 2")
		runCmd.Flags().String("flag3", "value3", "Flag 3")
		runCmd.Flags().String("flag4", "value4", "Flag 4")
		runCmd.Flags().String("flag5", "value5", "Flag 5")
		runCmd.Flags().IntP("port", "p", 8080, "Port")
		runCmd.Flags().BoolP("verbose", "v", false, "Verbose")
		runCmd.Flags().Bool("debug", false, "Debug")
		runCmd.Flags().Bool("quiet", false, "Quiet")
		runCmd.Flags().Bool("force", false, "Force")
		rootCmd.AddCommand(runCmd)
		rootCmd.SetArgs(args)
		_ = rootCmd.Execute()
	}
}

func BenchmarkManyFlags_Urfave(b *testing.B) {
	args := []string{
		"bench", "run",
		"--flag1", "test1",
		"--flag2", "test2",
		"--flag3", "test3",
		"--port", "9000",
		"--verbose",
		"--debug",
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		app := &cli.App{
			Name: "bench",
			Commands: []*cli.Command{
				{
					Name: "run",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "flag1", Value: "value1", Usage: "Flag 1"},
						&cli.StringFlag{Name: "flag2", Value: "value2", Usage: "Flag 2"},
						&cli.StringFlag{Name: "flag3", Value: "value3", Usage: "Flag 3"},
						&cli.StringFlag{Name: "flag4", Value: "value4", Usage: "Flag 4"},
						&cli.StringFlag{Name: "flag5", Value: "value5", Usage: "Flag 5"},
						&cli.IntFlag{Name: "port", Value: 8080, Usage: "Port"},
						&cli.BoolFlag{Name: "verbose", Usage: "Verbose"},
						&cli.BoolFlag{Name: "debug", Usage: "Debug"},
						&cli.BoolFlag{Name: "quiet", Usage: "Quiet"},
						&cli.BoolFlag{Name: "force", Usage: "Force"},
					},
					Action: func(_ *cli.Context) error { return nil },
				},
			},
		}
		_ = app.Run(args)
	}
}

// Benchmark nested subcommands
// Tests deep command hierarchies (realistic for complex tools)

func BenchmarkNestedCommands_GoSnap(b *testing.B) {
	app := snap.New("bench", "benchmark app")
	server := app.Command("server", "Server management")
	server.Command("start", "Start server").
		Action(func(_ *snap.Context) error { return nil })

	args := []string{"server", "start"}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = app.RunWithArgs(context.Background(), args)
	}
}

func BenchmarkNestedCommands_Cobra(b *testing.B) {
	args := []string{"server", "start"}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rootCmd := &cobra.Command{Use: "bench"}
		serverCmd := &cobra.Command{Use: "server"}
		startCmd := &cobra.Command{
			Use: "start",
			Run: func(_ *cobra.Command, _ []string) {},
		}
		serverCmd.AddCommand(startCmd)
		rootCmd.AddCommand(serverCmd)
		rootCmd.SetArgs(args)
		_ = rootCmd.Execute()
	}
}

func BenchmarkNestedCommands_Urfave(b *testing.B) {
	args := []string{"bench", "server", "start"}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		app := &cli.App{
			Name: "bench",
			Commands: []*cli.Command{
				{
					Name: "server",
					Subcommands: []*cli.Command{
						{
							Name:   "start",
							Action: func(_ *cli.Context) error { return nil },
						},
					},
				},
			},
		}
		_ = app.Run(args)
	}
}
