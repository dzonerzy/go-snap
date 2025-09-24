//go:build !windows

package snapio

import (
    stdio "io"
    "os"
    "strings"
    "syscall"
    "testing"
    "unsafe"
)

func TestUnix_EnvFallbackSize(t *testing.T) {
    t.Setenv("COLUMNS", "101")
    t.Setenv("LINES", "55")
    m := New()
    if m.Width() != 101 || m.Height() != 55 {
        t.Fatalf("want 101x55, got %dx%d", m.Width(), m.Height())
    }
}

func TestUnix_ColorOverridesAndLevels(t *testing.T) {
    m := New().ColorAuto()
    t.Setenv("NO_COLOR", "1")
    if m.SupportsColor() { t.Fatalf("NO_COLOR should disable") }
    os.Unsetenv("NO_COLOR")
    if !m.ForceColor().SupportsColor() { t.Fatalf("ForceColor should enable") }
    // 256 color
    t.Setenv("TERM", "xterm-256color")
    if m.ColorLevel() < 2 { t.Fatalf("expected at least 2 for 256color") }
    // truecolor
    t.Setenv("COLORTERM", "truecolor")
    if m.ColorLevel() != 3 { t.Fatalf("expected truecolor level 3") }
}

func TestUnix_ANSIStyles(t *testing.T) {
    m := New().ForceColor()
    out := NewStyle().Bold().Underline().Fg(BrightBlue).Sprint(m, "x")
    if !strings.Contains(out, "\x1b[") || !strings.HasSuffix(out, "\x1b[0m") {
        t.Fatalf("missing ANSI: %q", out)
    }
    // 256
    t.Setenv("TERM", "xterm-256color")
    out = NewStyle().Fg(Indexed(202)).Sprint(m, "x")
    if !strings.Contains(out, "38;5;202") { t.Fatalf("expected 256 code, got %q", out) }
    // truecolor
    t.Setenv("COLORTERM", "truecolor")
    out = NewStyle().Fg(Truecolor(1,2,3)).Bg(Truecolor(4,5,6)).Sprint(m, "x")
    if !strings.Contains(out, "38;2;1;2;3") || !strings.Contains(out, "48;2;4;5;6") {
        t.Fatalf("expected truecolor codes, got %q", out)
    }
}

func unixIsTerminal(f *os.File) bool {
    if f == nil { return false }
    var ws struct{ Row, Col, Xpixel, Ypixel uint16 }
    _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws)))
    if errno == 0 { return true }
    fi, err := f.Stat()
    if err != nil { return false }
    return (fi.Mode() & os.ModeCharDevice) != 0
}

func TestUnix_TTY_Piped_Redirected(t *testing.T) {
    outIsTTY := unixIsTerminal(os.Stdout)
    inIsTTY := unixIsTerminal(os.Stdin)
    m := New()
    if m.IsTTY() != outIsTTY { t.Fatalf("IsTTY mismatch: %v vs %v", m.IsTTY(), outIsTTY) }
    if m.IsPiped() != (!inIsTTY) { t.Fatalf("IsPiped mismatch: %v vs %v", m.IsPiped(), !inIsTTY) }
    if m.IsRedirected() != (!outIsTTY) { t.Fatalf("IsRedirected mismatch: %v vs %v", m.IsRedirected(), !outIsTTY) }
}

func TestUnix_FluentRedirects(t *testing.T) {
    m := New().WithOut(stdio.Discard)
    if m.Out() == nil { t.Fatalf("missing writer") }
}
