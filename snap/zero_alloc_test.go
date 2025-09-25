//nolint:testpackage // using package name 'snap' to access unexported fields for testing
package snap

import (
	"testing"
)

// TestZeroAllocParserSimple ensures the hot path parse is zero-allocation (SPECS.md guarantee)
func TestZeroAllocParserSimple(t *testing.T) {
	app := &App{
		flags: map[string]*Flag{
			"port":    {Name: "port", Type: FlagTypeInt},
			"verbose": {Name: "verbose", Type: FlagTypeBool},
		},
		commands: make(map[string]*Command),
	}

	parser := NewParser(app)
	args := []string{"--port", "8080", "--verbose"}

	allocs := testing.AllocsPerRun(1000, func() {
		res, err := parser.Parse(args)
		if err != nil || res == nil {
			t.Fatalf("unexpected parse error: %v", err)
		}
	})

	if allocs != 0 {
		t.Fatalf("expected 0 allocs/op for simple parse, got %.2f", allocs)
	}
}

// TestZeroAllocFlagGroupHappyPath verifies flag-group validation doesn’t allocate on success
func TestZeroAllocFlagGroupHappyPath(t *testing.T) {
	app := New("testapp", "").
		FlagGroup("output").
		MutuallyExclusive().
		BoolFlag("json", "").Back().
		BoolFlag("yaml", "").Back().
		EndGroup()

	parser := NewParser(app)
	args := []string{"--json"}

	allocs := testing.AllocsPerRun(1000, func() {
		res, err := parser.Parse(args)
		if err != nil || res == nil {
			t.Fatalf("unexpected parse error: %v", err)
		}
	})

	if allocs != 0 {
		t.Fatalf("expected 0 allocs/op for flag-group happy path, got %.2f", allocs)
	}
}
