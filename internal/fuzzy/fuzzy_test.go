//nolint:testpackage // using package name 'fuzzy' to access unexported fields for testing
package fuzzy

import (
	"sort"
	"testing"
)

func TestMatcher_FindBest(t *testing.T) {
	matcher := NewMatcher(2)

	tests := []struct {
		name       string
		input      string
		candidates []string
		expected   string
	}{
		{
			name:       "exact match excluded",
			input:      "help",
			candidates: []string{"help", "version", "verbose"},
			expected:   "", // Exact matches are excluded from fuzzy matching
		},
		{
			name:       "simple typo",
			input:      "hep",
			candidates: []string{"help", "version", "verbose"},
			expected:   "help",
		},
		{
			name:       "single character difference",
			input:      "port",
			candidates: []string{"host", "post", "part"},
			expected:   "post", // Should be closest
		},
		{
			name:       "no good match",
			input:      "xyz",
			candidates: []string{"help", "version", "verbose"},
			expected:   "", // Distance too high
		},
		{
			name:       "prefix matching bonus",
			input:      "ver",
			candidates: []string{"very", "verify", "verso"},
			expected:   "very", // Closest match within distance limit
		},
		{
			name:       "empty input",
			input:      "x",
			candidates: []string{"help", "version"},
			expected:   "", // Too short
		},
		{
			name:       "case insensitive",
			input:      "HEP",
			candidates: []string{"help", "version"},
			expected:   "help", // Should match despite case difference
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.FindBest(tt.input, tt.candidates)
			if result != tt.expected {
				t.Errorf("FindBest(%q, %v) = %q, want %q", tt.input, tt.candidates, result, tt.expected)
			}
		})
	}
}

func TestMatcher_FindMatches(t *testing.T) {
	matcher := NewMatcher(2)

	tests := []struct {
		name       string
		input      string
		candidates []string
		minMatches int // Minimum expected matches
		maxMatches int // Maximum expected matches
	}{
		{
			name:       "multiple matches",
			input:      "hep",
			candidates: []string{"help", "heap", "deep", "version"},
			minMatches: 2, // Should find help and heap
			maxMatches: 3,
		},
		{
			name:       "no matches",
			input:      "xyz",
			candidates: []string{"help", "version", "verbose"},
			minMatches: 0,
			maxMatches: 0,
		},
		{
			name:       "ordered by quality",
			input:      "ver",
			candidates: []string{"very", "veri", "vers", "vex"},
			minMatches: 2, // Should find matches within distance limit
			maxMatches: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := matcher.FindMatches(tt.input, tt.candidates)

			if len(matches) < tt.minMatches || len(matches) > tt.maxMatches {
				t.Errorf("FindMatches(%q, %v) returned %d matches, want %d-%d",
					tt.input, tt.candidates, len(matches), tt.minMatches, tt.maxMatches)
			}

			// Verify matches are sorted by score (descending)
			for i := 1; i < len(matches); i++ {
				if matches[i-1].Score < matches[i].Score {
					t.Errorf("Matches not sorted by score: %f < %f", matches[i-1].Score, matches[i].Score)
				}
			}

			// Verify all distances are within max
			for _, match := range matches {
				if match.Distance > matcher.maxDistance {
					t.Errorf("Match distance %d exceeds max %d", match.Distance, matcher.maxDistance)
				}
			}
		})
	}
}

func TestMatcher_LevenshteinDistance(t *testing.T) {
	matcher := NewMatcher(10) // High max to test actual distances

	tests := []struct {
		a        string
		b        string
		expected int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"abc", "abc", 0},
		{"abc", "ab", 1},
		{"abc", "abcd", 1},
		{"abc", "axc", 1},
		{"help", "hep", 1},
		{"version", "ver", 4},
		{"kitten", "sitting", 3},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			result := matcher.levenshteinDistance(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestMatcher_EarlyTermination(t *testing.T) {
	matcher := NewMatcher(2)

	// Test that very different strings are terminated early
	result := matcher.levenshteinDistance("short", "verylongstring")
	if result <= 2 {
		t.Errorf("Expected early termination for very different strings, got distance %d", result)
	}

	// Test that it returns max+1 for early termination
	if result <= matcher.maxDistance {
		t.Errorf("Expected distance > maxDistance (%d) for early termination, got %d", matcher.maxDistance, result)
	}
}

func TestMatcher_ScoreCalculation(t *testing.T) {
	matcher := NewMatcher(3)

	tests := []struct {
		input     string
		candidate string
		minScore  float64 // Minimum expected score
		maxScore  float64 // Maximum expected score
	}{
		{
			input:     "help",
			candidate: "help",
			minScore:  0.0, // Exact matches excluded, but test the score calculation
			maxScore:  1.0,
		},
		{
			input:     "hep",
			candidate: "help",
			minScore:  0.7, // Should be high due to good match
			maxScore:  1.0,
		},
		{
			input:     "ver",
			candidate: "very",
			minScore:  0.7, // Should get high score for close match
			maxScore:  1.0,
		},
		{
			input:     "xyz",
			candidate: "abc",
			minScore:  0.0, // Very different
			maxScore:  0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input+"_"+tt.candidate, func(t *testing.T) {
			distance := matcher.levenshteinDistance(tt.input, tt.candidate)
			score := matcher.calculateScore(tt.input, tt.candidate, distance)

			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("calculateScore(%q, %q, %d) = %f, want %f-%f",
					tt.input, tt.candidate, distance, score, tt.minScore, tt.maxScore)
			}

			// Score should be between 0 and 1
			if score < 0.0 || score > 1.0 {
				t.Errorf("Score %f outside valid range [0.0, 1.0]", score)
			}
		})
	}
}

func TestConvenienceFunctions(t *testing.T) {
	flags := []string{"help", "version", "verbose", "config"}
	commands := []string{"serve", "deploy", "migrate", "backup"}

	// Test FindBestFlag
	result := FindBestFlag("hep", flags, 2)
	if result != "help" {
		t.Errorf("FindBestFlag(hep) = %q, want help", result)
	}

	// Test FindBestCommand
	result = FindBestCommand("serv", commands, 2)
	if result != "serve" {
		t.Errorf("FindBestCommand(serv) = %q, want serve", result)
	}

	// Test FindSuggestions with better candidates
	suggestions := FindSuggestions("hep", flags, 2, 3)
	if len(suggestions) == 0 {
		t.Errorf("FindSuggestions(hep) returned no suggestions")
	}
	if len(suggestions) > 3 {
		t.Errorf("FindSuggestions(ver) returned %d suggestions, max was 3", len(suggestions))
	}
}

func TestMatch_Sorting(t *testing.T) {
	matches := []Match{
		{Value: "low", Distance: 3, Score: 0.2},
		{Value: "high", Distance: 1, Score: 0.8},
		{Value: "medium", Distance: 2, Score: 0.5},
		{Value: "tied_high", Distance: 2, Score: 0.8}, // Same score, different distance
	}

	// Sort using the same logic as FindMatches
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].Distance < matches[j].Distance
		}
		return matches[i].Score > matches[j].Score
	})

	// Verify sorting: high score first, then by distance for ties
	expected := []string{"high", "tied_high", "medium", "low"}
	for i, match := range matches {
		if match.Value != expected[i] {
			t.Errorf("Position %d: got %q, want %q", i, match.Value, expected[i])
		}
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test min function
	if min(5, 3) != 3 {
		t.Errorf("min(5, 3) = %d, want 3", min(5, 3))
	}

	// Test max function
	if max(5, 3) != 5 {
		t.Errorf("max(5, 3) = %d, want 5", max(5, 3))
	}

	// Test abs function
	if abs(-5) != 5 {
		t.Errorf("abs(-5) = %d, want 5", abs(-5))
	}
	if abs(5) != 5 {
		t.Errorf("abs(5) = %d, want 5", abs(5))
	}

	// Test minThree function
	if minThree(5, 3, 7) != 3 {
		t.Errorf("minThree(5, 3, 7) = %d, want 3", minThree(5, 3, 7))
	}
}

func TestCommonPrefixLength(t *testing.T) {
	matcher := NewMatcher(2)

	tests := []struct {
		a        string
		b        string
		expected int
	}{
		{"", "", 0},
		{"abc", "", 0},
		{"", "abc", 0},
		{"abc", "abc", 3},
		{"abc", "ab", 2},
		{"abc", "axc", 1},
		{"help", "hello", 3},
		{"version", "verbose", 3},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			result := matcher.commonPrefixLength(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("commonPrefixLength(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestCountCommonChars(t *testing.T) {
	matcher := NewMatcher(2)

	tests := []struct {
		a        string
		b        string
		expected int
	}{
		{"", "", 0},
		{"abc", "", 0},
		{"", "abc", 0},
		{"abc", "abc", 3},
		{"abc", "bca", 3},
		{"abc", "def", 0},
		{"help", "hello", 3}, // h, e, l (l only counted once per occurrence in first string)
		{"aab", "abb", 2},    // a, b (a counted once, b counted once)
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			result := matcher.countCommonChars(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("countCommonChars(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// Benchmarks moved to benchmark/bench_fuzzy_test.go
