//nolint:testpackage // using package name 'benchmark' to access unexported fields for testing
package benchmark

import (
	"testing"

	"github.com/dzonerzy/go-snap/snap"
)

// Category: parser

func buildSimpleApp() *snap.App {
	return snap.New("bench", "bench").
		IntFlag("port", "").Default(8080).Back().
		BoolFlag("verbose", "").Back()
}

func BenchmarkParserSimple(b *testing.B) {
	app := buildSimpleApp()
	parser := snap.NewParser(app)
	args := []string{"--port", "8080", "--verbose"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := parser.Parse(args)
		if err != nil || result == nil {
			b.Fatal(err)
		}
		if v, ok := result.GetBool("verbose"); !ok || !v {
			b.Fatalf("verbose not parsed")
		}
	}
}

func BenchmarkParserComplex(b *testing.B) {
	app := snap.New("bench", "bench").
		BoolFlag("global", "").Global().Back().
		IntFlag("port", "").Default(8080).Back().
		StringFlag("host", "").Default("localhost").Back()
	app.Command("serve", "")
	parser := snap.NewParser(app)
	args := []string{"--global", "serve", "--port", "8080", "--host", "localhost"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := parser.Parse(args)
		if err != nil || result == nil {
			b.Fatal(err)
		}
		if result.Command == nil || result.Command.Name() != "serve" {
			b.Fatalf("command mismatch")
		}
	}
}

func BenchmarkParserLongFlags(b *testing.B) {
	app := snap.New("bench", "bench").
		IntFlag("port", "").Default(8080).Back().
		BoolFlag("verbose", "").Back().
		StringFlag("config", "").Back()
	parser := snap.NewParser(app)
	args := []string{"--port=8080", "--verbose=true", "--config=/path/to/config.json"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := parser.Parse(args)
		if err != nil || result == nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParserShortFlags(b *testing.B) {
	app := snap.New("bench", "bench").
		BoolFlag("v", "").Back().
		BoolFlag("h", "").Back().
		IntFlag("p", "").Default(8080).Back()
	parser := snap.NewParser(app)
	args := []string{"-vhp", "8080"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := parser.Parse(args)
		if err != nil || result == nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParserErrorSuggestion(b *testing.B) {
	app := snap.New("bench", "bench").
		IntFlag("port", "").Default(8080).Back().
		BoolFlag("verbose", "").Back()
	parser := snap.NewParser(app)
	args := []string{"--prot", "8080"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := parser.Parse(args); err == nil {
			b.Fatal("expected error")
		}
	}
}

func BenchmarkComprehensiveFlagTypes(b *testing.B) {
	app := snap.New("bench", "bench").
		StringFlag("name", "").Back().
		IntFlag("port", "").Back().
		BoolFlag("verbose", "").Back().
		DurationFlag("timeout", "").Back().
		FloatFlag("ratio", "").Back().
		StringSliceFlag("tags", "").Back().
		IntSliceFlag("ports", "").Back().
		BoolFlag("debug", "").Global().Back()
	parser := snap.NewParser(app)
	args := []string{
		"--debug",
		"--name",
		"go-snap",
		"--port",
		"0xFF",
		"--verbose",
		"--timeout",
		"1h30m",
		"--ratio",
		"3.14",
		"--tags",
		"cli,parser,go",
		"--ports",
		"80,443,8080",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := parser.Parse(args)
		if err != nil || result == nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFlagGroupParsing(b *testing.B) {
	app := snap.New("bench", "bench").
		FlagGroup("output").
		MutuallyExclusive().
		BoolFlag("json", "").Back().
		BoolFlag("yaml", "").Back().
		EndGroup().
		StringFlag("config", "").Back()
	parser := snap.NewParser(app)
	args := []string{"--json", "--config", "test.conf"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := parser.Parse(args)
		if err != nil || result == nil {
			b.Fatal(err)
		}
	}
}
