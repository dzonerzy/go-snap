//go:build windows

package snapio

import (
	stdio "io"
	"os"
	"testing"
)

func TestWindows_EnvFallbackSize(t *testing.T) {
	t.Setenv("COLUMNS", "90")
	t.Setenv("LINES", "33")
	m := New()
	if m.Width() != 90 || m.Height() != 33 {
		t.Fatalf("want 90x33, got %dx%d", m.Width(), m.Height())
	}
}

func TestWindows_VT_Enable_NoPanic(t *testing.T) {
	m := New()
	_ = m.EnableVirtualTerminal() // smoke, may return false
	_ = m.SupportsColor()
}

func TestWindows_FileIsNotTerminal(t *testing.T) {
	f, err := os.CreateTemp("", "iofile")
	if err != nil {
		t.Fatalf("tmp: %v", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if New().p.isTerminal(f) {
		t.Fatalf("regular file must not be terminal")
	}
}

func TestWindows_FluentRedirects(t *testing.T) {
	m := New().WithOut(stdio.Discard)
	if m.Out() == nil {
		t.Fatalf("missing writer")
	}
}

func TestWindows_ColorOverrides(t *testing.T) {
	m := New()
	t.Setenv("NO_COLOR", "1")
	if m.SupportsColor() {
		t.Fatalf("NO_COLOR should disable")
	}
	os.Unsetenv("NO_COLOR")
	if !m.ForceColor().SupportsColor() {
		t.Fatalf("ForceColor should enable")
	}
}
