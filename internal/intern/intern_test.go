package intern

import (
	"sync"
	"testing"
)

func TestStringInterner_Intern(t *testing.T) {
	interner := NewStringInterner(0)

	// Test basic interning
	s1 := interner.Intern("test")
	s2 := interner.Intern("test")

	if s1 != s2 {
		t.Errorf("Expected same string instances, got different")
	}

	// Test different strings
	s3 := interner.Intern("other")
	if s1 == s3 {
		t.Errorf("Expected different string instances for different values")
	}
}

func TestStringInterner_InternBytes(t *testing.T) {
	interner := NewStringInterner(0)

	bytes1 := []byte("test")
	bytes2 := []byte("test")

	s1 := interner.InternBytes(bytes1)
	s2 := interner.InternBytes(bytes2)

	if s1 != s2 {
		t.Errorf("Expected same string instances from byte slices, got different")
	}

	if s1 != "test" {
		t.Errorf("Expected 'test', got %q", s1)
	}
}

func TestStringInterner_InternByte(t *testing.T) {
	interner := NewStringInterner(0)

	tests := []struct {
		input    byte
		expected string
	}{
		{'a', "a"},
		{'Z', "Z"},
		{'5', "5"},
		{'@', "@"}, // Non-alphanumeric
	}

	for _, test := range tests {
		result := interner.InternByte(test.input)
		if result != test.expected {
			t.Errorf("InternByte(%c) = %q, want %q", test.input, result, test.expected)
		}

		// Test that repeated calls return same instance for alphanumeric
		if (test.input >= 'a' && test.input <= 'z') ||
			(test.input >= 'A' && test.input <= 'Z') ||
			(test.input >= '0' && test.input <= '9') {
			result2 := interner.InternByte(test.input)
			if result != result2 {
				t.Errorf("InternByte(%c) returned different instances", test.input)
			}
		}
	}
}

func TestStringInterner_PreIntern(t *testing.T) {
	interner := NewStringInterner(0)

	preStrings := []string{"flag1", "flag2", "flag3"}
	interner.PreIntern(preStrings)

	for _, s := range preStrings {
		interned := interner.Intern(s)
		if interned != s {
			t.Errorf("Expected pre-interned string %q to be returned as-is", s)
		}
	}
}

func TestStringInterner_Stats(t *testing.T) {
	interner := NewStringInterner(0)

	// Start with 0 count
	if count := interner.Stats(); count != 0 {
		t.Errorf("Expected 0 strings, got %d", count)
	}

	// Add some strings
	interner.Intern("test1")
	interner.Intern("test2")
	interner.Intern("test1") // Duplicate - shouldn't increase count

	if count := interner.Stats(); count != 2 {
		t.Errorf("Expected 2 strings, got %d", count)
	}
}

func TestStringInterner_Clear(t *testing.T) {
	interner := NewStringInterner(0)

	interner.Intern("test1")
	interner.Intern("test2")

	if count := interner.Stats(); count != 2 {
		t.Errorf("Expected 2 strings before clear, got %d", count)
	}

	interner.Clear()

	if count := interner.Stats(); count != 0 {
		t.Errorf("Expected 0 strings after clear, got %d", count)
	}
}

func TestStringInterner_Concurrent(t *testing.T) {
	interner := NewStringInterner(0)

	const numGoroutines = 100
	const numOperations = 1000

	var wg sync.WaitGroup
	results := make([][]string, numGoroutines)

	// Run concurrent interning operations
	for i := range numGoroutines {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			results[goroutineID] = make([]string, numOperations)

			for j := range numOperations {
				// Intern the same string from all goroutines
				results[goroutineID][j] = interner.Intern("concurrent-test")
			}
		}(i)
	}

	wg.Wait()

	// Verify all results are the same instance
	expected := results[0][0]
	for i := range numGoroutines {
		for j := range numOperations {
			if results[i][j] != expected {
				t.Errorf("Concurrent interning failed: got different instances")
				return
			}
		}
	}

	// Should only have one instance in the interner
	if count := interner.Stats(); count != 1 {
		t.Errorf("Expected 1 string after concurrent operations, got %d", count)
	}
}

func TestGlobalInterner(t *testing.T) {
	// Test that global convenience functions work
	s1 := Intern("global-test")
	s2 := Intern("global-test")

	if s1 != s2 {
		t.Errorf("Global Intern() returned different instances")
	}

	// Test global byte interning
	bytes := []byte("byte-test")
	s3 := InternBytes(bytes)
	s4 := InternBytes(bytes)

	if s3 != s4 {
		t.Errorf("Global InternBytes() returned different instances")
	}

	// Test global byte interning
	s5 := InternByte('x')
	s6 := InternByte('x')

	if s5 != s6 {
		t.Errorf("Global InternByte() returned different instances")
	}
}

func TestCommonFlagNames(t *testing.T) {
	// Test that common flag names are pre-interned in global interner
	for _, flagName := range CommonFlagNames {
		interned := Intern(flagName)
		if interned != flagName {
			t.Errorf("Common flag %q not properly pre-interned", flagName)
		}
	}
}

// Benchmarks moved to benchmark/bench_intern_test.go
