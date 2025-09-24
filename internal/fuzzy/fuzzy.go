// Package fuzzy provides efficient fuzzy matching for CLI suggestions
// Used by snap/errors.go for smart error suggestions with typo detection
package fuzzy

import (
	"sort"
	"strings"
)

// Matcher provides fuzzy matching functionality for CLI suggestions
type Matcher struct {
	maxDistance int
	minLength   int
}

// NewMatcher creates a new fuzzy matcher with the given max edit distance
func NewMatcher(maxDistance int) *Matcher {
	return &Matcher{
		maxDistance: maxDistance,
		minLength:   2, // Don't suggest for very short inputs
	}
}

// Match represents a fuzzy match result
type Match struct {
	Value    string
	Distance int
	Score    float64 // 0.0 to 1.0, higher is better
}

// FindBest finds the best matching string from candidates
// Returns empty string if no good match found
func (m *Matcher) FindBest(input string, candidates []string) string {
	if len(input) < m.minLength {
		return ""
	}

	matches := m.FindMatches(input, candidates)
	if len(matches) == 0 {
		return ""
	}

	return matches[0].Value
}

// FindMatches finds all matching strings from candidates, sorted by quality
func (m *Matcher) FindMatches(input string, candidates []string) []Match {
	if len(input) < m.minLength {
		return nil
	}

	var matches []Match
	input = strings.ToLower(input)

	for _, candidate := range candidates {
		candidateLower := strings.ToLower(candidate)

		// Skip exact matches (not fuzzy)
		if input == candidateLower {
			continue
		}

		distance := m.levenshteinDistance(input, candidateLower)
		if distance <= m.maxDistance {
			score := m.calculateScore(input, candidateLower, distance)
			matches = append(matches, Match{
				Value:    candidate,
				Distance: distance,
				Score:    score,
			})
		}
	}

	// Sort by score (descending) then by distance (ascending)
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].Distance < matches[j].Distance
		}
		return matches[i].Score > matches[j].Score
	})

	return matches
}

// calculateScore computes a match quality score (0.0 to 1.0)
// Factors: edit distance, length difference, prefix matching, common subsequence
func (m *Matcher) calculateScore(input, candidate string, distance int) float64 {
	if distance > m.maxDistance {
		return 0.0
	}

	maxLen := max(len(input), len(candidate))
	if maxLen == 0 {
		return 1.0
	}

	// Base score from edit distance
	editScore := 1.0 - (float64(distance) / float64(maxLen))

	// Bonus for prefix matching
	prefixBonus := 0.0
	prefixLen := m.commonPrefixLength(input, candidate)
	if prefixLen > 0 {
		prefixBonus = float64(prefixLen) / float64(min(len(input), len(candidate))) * 0.3
	}

	// Bonus for length similarity
	lengthDiff := abs(len(input) - len(candidate))
	lengthBonus := (1.0 - float64(lengthDiff)/float64(maxLen)) * 0.2

	// Bonus for common character ratio
	commonChars := m.countCommonChars(input, candidate)
	charBonus := float64(commonChars) / float64(maxLen) * 0.1

	score := editScore + prefixBonus + lengthBonus + charBonus
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// levenshteinDistance calculates edit distance between two strings
// Optimized version with early termination when distance exceeds max
func (m *Matcher) levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Early termination if length difference exceeds max distance
	if abs(len(a)-len(b)) > m.maxDistance {
		return m.maxDistance + 1
	}

	// Use smaller string as first argument for memory efficiency
	if len(a) > len(b) {
		a, b = b, a
	}

	// Create only two rows instead of full matrix for memory efficiency
	previousRow := make([]int, len(a)+1)
	currentRow := make([]int, len(a)+1)

	// Initialize first row
	for i := range previousRow {
		previousRow[i] = i
	}

	for i := range len(b) {
		i++ // Adjust for 1-based indexing
		currentRow[0] = i
		minInRow := i

		for j := range len(a) {
			j++ // Adjust for 1-based indexing
			cost := 0
			if a[j-1] != b[i-1] {
				cost = 1
			}

			currentRow[j] = minThree(
				currentRow[j-1]+1,      // insertion
				previousRow[j]+1,       // deletion
				previousRow[j-1]+cost,  // substitution
			)

			if currentRow[j] < minInRow {
				minInRow = currentRow[j]
			}
		}

		// Early termination: if minimum in current row exceeds max distance,
		// final distance will definitely exceed max distance
		if minInRow > m.maxDistance {
			return m.maxDistance + 1
		}

		// Swap rows
		previousRow, currentRow = currentRow, previousRow
	}

	return previousRow[len(a)]
}

// commonPrefixLength returns the length of the common prefix
func (m *Matcher) commonPrefixLength(a, b string) int {
	maxLen := min(len(a), len(b))
	for i := range maxLen {
		if a[i] != b[i] {
			return i
		}
	}
	return maxLen
}

// countCommonChars counts characters that appear in both strings
func (m *Matcher) countCommonChars(a, b string) int {
	charCount := make(map[rune]int)

	// Count characters in first string
	for _, r := range a {
		charCount[r]++
	}

	// Count common characters
	common := 0
	for _, r := range b {
		if charCount[r] > 0 {
			common++
			charCount[r]--
		}
	}

	return common
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func minThree(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// Convenience functions for CLI usage

// FindBestFlag finds the best matching flag name
func FindBestFlag(input string, flags []string, maxDistance int) string {
	matcher := NewMatcher(maxDistance)
	return matcher.FindBest(input, flags)
}

// FindBestCommand finds the best matching command name
func FindBestCommand(input string, commands []string, maxDistance int) string {
	matcher := NewMatcher(maxDistance)
	return matcher.FindBest(input, commands)
}

// FindSuggestions finds multiple suggestions for CLI error messages
func FindSuggestions(input string, candidates []string, maxDistance, maxSuggestions int) []string {
	matcher := NewMatcher(maxDistance)
	matches := matcher.FindMatches(input, candidates)

	suggestions := make([]string, 0, min(len(matches), maxSuggestions))
	for i, match := range matches {
		if i >= maxSuggestions {
			break
		}
		suggestions = append(suggestions, match.Value)
	}

	return suggestions
}