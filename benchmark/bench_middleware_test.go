//nolint:testpackage // using package name 'benchmark' to access unexported fields for testing
package benchmark

import (
	"context"
	"testing"
	"time"

	mw "github.com/dzonerzy/go-snap/middleware"
	"github.com/dzonerzy/go-snap/snap"
)

// Category: middleware

func BenchmarkMiddlewareChain(b *testing.B) {
	app := snap.New("bench", "bench").
		BoolFlag("v", "verbose").Back()

	chain := mw.Chain(mw.SilentLogger(), mw.Recovery(), mw.Timeout(10*time.Millisecond))

	cmd := app.Command("run", "").Action(func(_ *snap.Context) error {
		return nil
	})
	cmd.Use(chain...)

	args := []string{"run", "-v"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = app.RunWithArgs(context.Background(), args)
	}
}

// NOTE: micro-benchmarks for individual middleware are exercised via the chain benchmark above.
// Keeping chain-based benchmark avoids duplicating mock context plumbing here.
