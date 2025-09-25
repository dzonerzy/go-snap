// Package intern provides efficient string interning for go-snap
// Used by parser for flag names, command names, and other repeated strings
package intern

import (
	"sync"
	"unsafe"
)

// StringInterner provides thread-safe string interning
type StringInterner struct {
	strings map[string]string
	mutex   sync.RWMutex
}

// NewStringInterner creates a new string interner with optional pre-allocated capacity
func NewStringInterner(capacity int) *StringInterner {
	if capacity <= 0 {
		capacity = 64 // Default capacity
	}
	return &StringInterner{
		strings: make(map[string]string, capacity),
	}
}

// Intern interns a string, returning the canonical version
// Thread-safe and optimized for high-frequency access
func (si *StringInterner) Intern(s string) string {
	// Fast path: read lock for common case
	si.mutex.RLock()
	if interned, exists := si.strings[s]; exists {
		si.mutex.RUnlock()
		return interned
	}
	si.mutex.RUnlock()

	// Slow path: write lock for insertion
	si.mutex.Lock()
	defer si.mutex.Unlock()

	// Double-check after acquiring write lock
	if interned, exists := si.strings[s]; exists {
		return interned
	}

	// Store and return the string
	si.strings[s] = s
	return s
}

// InternBytes interns a byte slice as string without extra allocation
func (si *StringInterner) InternBytes(b []byte) string {
	// Convert bytes to string without allocation for lookup
	str := bytesToString(b)
	return si.Intern(str)
}

// InternByte interns a single byte as string using pre-allocated lookup
func (si *StringInterner) InternByte(b byte) string {
	// Use pre-allocated single character strings for common cases
	if b >= 'a' && b <= 'z' {
		return singleCharStrings[b-'a']
	}
	if b >= 'A' && b <= 'Z' {
		return singleCharStrings[26+b-'A']
	}
	if b >= '0' && b <= '9' {
		return singleCharStrings[52+b-'0']
	}
	// For other characters, intern normally (rare case)
	return si.Intern(string(rune(b)))
}

// PreIntern adds common strings to avoid allocation during parsing
func (si *StringInterner) PreIntern(strings []string) {
	si.mutex.Lock()
	defer si.mutex.Unlock()

	for _, s := range strings {
		si.strings[s] = s
	}
}

// Stats returns the number of interned strings for monitoring.
func (si *StringInterner) Stats() int {
	si.mutex.RLock()
	defer si.mutex.RUnlock()
	return len(si.strings)
}

// Clear removes all interned strings (useful for testing)
func (si *StringInterner) Clear() {
	si.mutex.Lock()
	defer si.mutex.Unlock()

	// Clear map without reallocating
	for k := range si.strings {
		delete(si.strings, k)
	}
}

// Pre-allocated single character strings for zero-allocation short flags
// a-z (0-25), A-Z (26-51), 0-9 (52-61)
var singleCharStrings = [62]string{
	"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m",
	"n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
	"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
	"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
	"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
}

// CommonFlagNames contains frequently used flag names for pre-interning
var CommonFlagNames = []string{
	"help", "h", "version", "v", "verbose", "quiet", "q",
	"config", "c", "output", "o", "input", "i", "force", "f",
	"debug", "d", "port", "p", "host", "timeout", "retry",
}

// bytesToString converts byte slice to string without allocation
// Uses unsafe pointer conversion for zero-copy operation
func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// GlobalInterner is the process-wide string interner used for go-snap CLI parsing.
// It is pre-initialized with common flag names for optimal performance.
var GlobalInterner *StringInterner

//nolint:gochecknoinits // Global interner requires init for pre-interning
func init() {
	GlobalInterner = NewStringInterner(128)
	GlobalInterner.PreIntern(CommonFlagNames)
}

// Convenience functions for common use cases

// Intern interns a string using the global interner
func Intern(s string) string {
	return GlobalInterner.Intern(s)
}

// InternBytes interns a byte slice using the global interner
//
//nolint:revive // keep name for public API symmetry with Intern/InternByte
func InternBytes(b []byte) string {
	return GlobalInterner.InternBytes(b)
}

// InternByte interns a single byte using the global interner
//
//nolint:revive // keep name for public API symmetry with Intern/InternBytes
func InternByte(b byte) string {
	return GlobalInterner.InternByte(b)
}
