//nolint:testpackage // using package name 'benchmark' to access unexported fields for testing
package benchmark

import (
	"testing"
	"time"

	mw "github.com/dzonerzy/go-snap/middleware"
)

// Minimal bench context implementing middleware.Context
type benchCtx struct {
	done chan struct{}
}

func newBenchCtx() *benchCtx { return &benchCtx{done: make(chan struct{})} }

func (b *benchCtx) Done() <-chan struct{}                         { return b.done }
func (b *benchCtx) Cancel()                                       { close(b.done) }
func (b *benchCtx) Args() []string                                { return nil }
func (b *benchCtx) Set(_ string, _ any)                           {}
func (b *benchCtx) Get(_ string) any                              { return nil }
func (b *benchCtx) String(_ string) (string, bool)                { return "", false }
func (b *benchCtx) Int(_ string) (int, bool)                      { return 0, false }
func (b *benchCtx) Bool(_ string) (bool, bool)                    { return false, false }
func (b *benchCtx) Duration(_ string) (time.Duration, bool)       { return 0, false }
func (b *benchCtx) Float(_ string) (float64, bool)                { return 0, false }
func (b *benchCtx) Enum(_ string) (string, bool)                  { return "", false }
func (b *benchCtx) StringSlice(_ string) ([]string, bool)         { return nil, false }
func (b *benchCtx) IntSlice(_ string) ([]int, bool)               { return nil, false }
func (b *benchCtx) GlobalString(_ string) (string, bool)          { return "", false }
func (b *benchCtx) GlobalInt(_ string) (int, bool)                { return 0, false }
func (b *benchCtx) GlobalBool(_ string) (bool, bool)              { return false, false }
func (b *benchCtx) GlobalDuration(_ string) (time.Duration, bool) { return 0, false }
func (b *benchCtx) GlobalFloat(_ string) (float64, bool)          { return 0, false }
func (b *benchCtx) GlobalEnum(_ string) (string, bool)            { return "", false }
func (b *benchCtx) GlobalStringSlice(_ string) ([]string, bool)   { return nil, false }
func (b *benchCtx) GlobalIntSlice(_ string) ([]int, bool)         { return nil, false }

// Command name is used by middleware for messages; provide a stub
type benchCmd struct{}

func (benchCmd) Name() string           { return "bench" }
func (benchCmd) Description() string    { return "" }
func (b *benchCtx) Command() mw.Command { return benchCmd{} }

var noop = func(_ mw.Context) error { return nil }

func BenchmarkMW_SilentLogger(b *testing.B) {
	m := mw.SilentLogger()
	action := m(noop)
	ctx := newBenchCtx()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = action(ctx)
	}
}

func BenchmarkMW_Recovery_NoStack(b *testing.B) {
	m := mw.Recovery(mw.WithStackTrace(false))
	action := m(noop)
	ctx := newBenchCtx()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = action(ctx)
	}
}

func BenchmarkMW_NoTimeout(b *testing.B) {
	m := mw.NoTimeout()
	action := m(noop)
	ctx := newBenchCtx()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = action(ctx)
	}
}

func BenchmarkMW_Timeout_10ms(b *testing.B) {
	m := mw.Timeout(10 * time.Millisecond)
	action := m(noop)
	ctx := newBenchCtx()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// The action returns immediately; timeout path won't trigger
		_ = action(ctx)
	}
}

func BenchmarkMW_Chain_NoTimeout(b *testing.B) {
	chain := mw.Chain(mw.SilentLogger(), mw.Recovery(mw.WithStackTrace(false)), mw.NoopValidator())
	action := chain.Apply(noop)
	ctx := newBenchCtx()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = action(ctx)
	}
}

func BenchmarkMW_Chain_Timeout(b *testing.B) {
	chain := mw.Chain(mw.SilentLogger(), mw.Recovery(mw.WithStackTrace(false)), mw.Timeout(10*time.Millisecond))
	action := chain.Apply(noop)
	ctx := newBenchCtx()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = action(ctx)
	}
}
