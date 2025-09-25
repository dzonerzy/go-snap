//nolint:testpackage // using package name 'benchmark' to access unexported fields for testing
package benchmark

import (
	"testing"

	fuzzy "github.com/dzonerzy/go-snap/internal/fuzzy"
)

// Category: fuzzy (exported paths only)

func BenchmarkMatcher_FindBest(b *testing.B) {
	matcher := fuzzy.NewMatcher(2)
	candidates := []string{
		"help", "version", "verbose", "config", "output", "input",
		"force", "debug", "port", "host", "timeout", "retry",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.FindBest("hep", candidates)
	}
}

func BenchmarkMatcher_FindMatches(b *testing.B) {
	matcher := fuzzy.NewMatcher(2)
	candidates := []string{
		"help", "version", "verbose", "config", "output", "input",
		"force", "debug", "port", "host", "timeout", "retry",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.FindMatches("ver", candidates)
	}
}

func BenchmarkConvenienceFunctions(b *testing.B) {
	flags := []string{
		"help", "version", "verbose", "config", "output", "input",
		"force", "debug", "port", "host", "timeout", "retry",
	}
	b.Run("FindBestFlag", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fuzzy.FindBestFlag("hep", flags, 2)
		}
	})
	b.Run("FindSuggestions", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fuzzy.FindSuggestions("ver", flags, 2, 3)
		}
	})
}
