package snapio

import (
	stdio "io"
	"os"
	"runtime"
)

// platformIO is implemented per OS in io_unix.go and io_windows.go
type platformIO interface {
	isTerminal(*os.File) bool
	termSize(*os.File) (width, height int, ok bool)
	enableVirtualTerminal() bool
	vtEnabled() bool
	colorCapabilityLevel() int // Returns detected color level: 0=none, 1=16, 2=256, 3=truecolor
}

// newPlatformIO is provided by platform files

// IOManager centralizes IO and terminal capabilities
type IOManager struct {
	in  stdio.Reader
	out stdio.Writer
	err stdio.Writer

	forceColor         bool
	noColor            bool
	forceColorLevel    int
	hasForceColorLevel bool

	p platformIO
}

// New returns a manager bound to process stdio
func New() *IOManager {
	m := &IOManager{in: os.Stdin, out: os.Stdout, err: os.Stderr, p: newPlatformIO()}
	return m
}

// WithIn sets the input reader used by the manager and returns the manager for chaining.
func (m *IOManager) WithIn(r stdio.Reader) *IOManager { m.in = r; return m }

// WithOut sets the standard output writer and returns the manager for chaining.
func (m *IOManager) WithOut(w stdio.Writer) *IOManager { m.out = w; return m }

// WithErr sets the standard error writer and returns the manager for chaining.
func (m *IOManager) WithErr(w stdio.Writer) *IOManager { m.err = w; return m }

// ForceColor forces color output on, regardless of environment.
func (m *IOManager) ForceColor() *IOManager { m.forceColor = true; m.noColor = false; return m }

// NoColor disables color output, regardless of environment.
func (m *IOManager) NoColor() *IOManager { m.noColor = true; m.forceColor = false; return m }

// ColorAuto uses environment heuristics to determine color support.
func (m *IOManager) ColorAuto() *IOManager { m.noColor = false; m.forceColor = false; return m }

// ForceColorLevel forces a specific color level (0=none, 1=16, 2=256, 3=truecolor).
// This is useful when automatic detection fails to recognize terminal capabilities.
func (m *IOManager) ForceColorLevel(level int) *IOManager {
	m.forceColorLevel = level
	m.hasForceColorLevel = true
	return m
}

// In returns the configured input reader.
func (m *IOManager) In() stdio.Reader { return m.in }

// Out returns the configured standard output writer.
func (m *IOManager) Out() stdio.Writer { return m.out }

// Err returns the configured standard error writer.
func (m *IOManager) Err() stdio.Writer { return m.err }

// IsTTY reports whether stdout is connected to a terminal.
func (m *IOManager) IsTTY() bool         { return m.p.isTerminal(os.Stdout) }
func (m *IOManager) IsInteractive() bool { return m.p.isTerminal(os.Stdin) && os.Getenv("CI") == "" }
func (m *IOManager) Width() int {
	if w, _, ok := m.p.termSize(os.Stdout); ok && w > 0 {
		return w
	}
	if w2, _ := fallbackTermSizeFromEnv(); w2 > 0 {
		return w2
	}
	return 80
}
func (m *IOManager) Height() int {
	if _, h, ok := m.p.termSize(os.Stdout); ok && h > 0 {
		return h
	}
	if _, h2 := fallbackTermSizeFromEnv(); h2 > 0 {
		return h2
	}
	return 24
}
func (m *IOManager) IsPiped() bool      { return !m.p.isTerminal(os.Stdin) }
func (m *IOManager) IsRedirected() bool { return !m.p.isTerminal(os.Stdout) }

// SupportsColor determines ANSI color capability (0=none,1=16,2=256,3=16m via truecolor)
func (m *IOManager) SupportsColor() bool {
	if m.noColor || os.Getenv("NO_COLOR") != "" {
		return false
	}
	if m.forceColor || os.Getenv("FORCE_COLOR") != "" {
		return true
	}
	if goos() == "windows" {
		return m.p.vtEnabled()
	}
	// Unix: TTY and TERM not dumb
	if !m.IsTTY() {
		return false
	}
	term := os.Getenv("TERM")
	return term != "" && term != "dumb"
}

// ColorLevel returns 0 for none, 1 for basic, 2 for 256 colors, and 3 for truecolor.
func (m *IOManager) ColorLevel() int {
	// Check for forced color level first
	if m.hasForceColorLevel {
		return m.forceColorLevel
	}
	if !m.SupportsColor() {
		return 0
	}
	// Check for explicit truecolor/24bit environment variable
	colorterm := os.Getenv("COLORTERM")
	if colorterm == "truecolor" || colorterm == "24bit" {
		return 3
	}
	// Check TERM for truecolor indicators
	term := os.Getenv("TERM")
	if contains(term, "truecolor") || contains(term, "24bit") {
		return 3
	}
	// Check for modern terminal programs that support truecolor
	termProgram := os.Getenv("TERM_PROGRAM")
	if termProgram == "vscode" || termProgram == "zed" {
		return 3
	}

	// Windows-specific detection
	if goos() == "windows" {
		// Check for known truecolor-capable Windows terminals
		// Windows Terminal sets WT_SESSION or WT_PROFILE_ID
		if os.Getenv("WT_SESSION") != "" || os.Getenv("WT_PROFILE_ID") != "" {
			return 3
		}
		// ConEmu sets ConEmuANSI=ON for truecolor support
		if os.Getenv("ConEmuANSI") == "ON" {
			return 3
		}
		// If VT processing is enabled, assume truecolor support
		if m.p.vtEnabled() {
			return 3
		}
		// Fallback for Windows without detection
		if m.IsTTY() {
			return 2 // At least 256 colors on modern Windows
		}
	}
	// 256-color terminals
	if contains(term, "256color") {
		return 2
	}

	// Platform-specific terminal capability detection (queries actual terminal)
	// This is more reliable than environment variables alone
	if level := m.p.colorCapabilityLevel(); level > 0 {
		return level
	}

	// Basic 16-color fallback
	return 1
}

// EnableVirtualTerminal tries to enable ANSI processing on Windows consoles
func (m *IOManager) EnableVirtualTerminal() bool { return m.p.enableVirtualTerminal() }

// ANSI helpers

// Colorize wraps s with the given ANSI SGR code (e.g., "31" for red) and a
// trailing reset ("0m"). If color is not supported, it returns s unchanged.
func (m *IOManager) Colorize(s, code string) string {
	if !m.SupportsColor() {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

// Bold returns s in bold when color is supported; otherwise s unchanged.
func (m *IOManager) Bold(s string) string { return m.Colorize(s, "1") }

// Faint returns s in faint intensity when supported; otherwise s unchanged.
func (m *IOManager) Faint(s string) string { return m.Colorize(s, "2") }

// Italic returns s in italic when supported; otherwise s unchanged.
func (m *IOManager) Italic(s string) string { return m.Colorize(s, "3") }

// Underline returns s underlined when supported; otherwise s unchanged.
func (m *IOManager) Underline(s string) string { return m.Colorize(s, "4") }

// helpers
func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
func goos() string {
	if v := os.Getenv("SNAP_GOOS"); v != "" {
		return v
	}
	return runtime.GOOS
}
