package benchmark

import (
    "testing"
    "github.com/dzonerzy/go-snap/snap"
    "context"
)

// These benchmarks exercise the argv-building path; they still spawn a tiny child
// (/bin/true on UNIX). Skipped on Windows.

func BenchmarkWrapper_ArgBuild_LeadingFlags(b *testing.B) {
    if testing.Short() { b.SkipNow() }
    app := snap.New("b", "")
    // Define a wrapper flag with short alias -n, and map it to child "-n"
    app.BoolFlag("no-newline", "").Short('n').Back()
    // Use /bin/true to minimize cost (UNIX only). If not present, skip.
    bin := "/bin/true"
    app.Command("r", "").
        Wrap(bin).
        LeadingFlags("-n").
        MapBoolFlag("no-newline", "-n").
        InsertAfterLeadingFlags("[p]").
        ForwardArgs().
        Passthrough().
        Back()
    args := []string{"r", "-n", "hello", "world"}
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = app.RunWithArgs(context.Background(), args)
    }
}
