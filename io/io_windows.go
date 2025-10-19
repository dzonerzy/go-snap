//go:build windows

package snapio

import (
	"os"
	"syscall"
	"unsafe"
)

type windowsPlatform struct{}

func newPlatformIO() platformIO { return &windowsPlatform{} }

// Win32 structures
type coord struct{ X, Y int16 }
type smallRect struct{ Left, Top, Right, Bottom int16 }
type consoleScreenBufferInfo struct {
	DwSize              coord
	DwCursorPosition    coord
	WAttributes         uint16
	SrWindow            smallRect
	DwMaximumWindowSize coord
}

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode             = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	procGetStdHandle               = kernel32.NewProc("GetStdHandle")
)

const (
	stdOutputHandle                 = ^uintptr(10) + 1 // (uintptr)(-11)
	stdInputHandle                  = ^uintptr(8) + 1  // (uintptr)(-10)
	enableVirtualTerminalProcessing = 0x0004
)

func stdHandle(file *os.File) uintptr {
	if file == os.Stdout {
		return stdOutputHandle
	}
	if file == os.Stdin {
		return stdInputHandle
	}
	return uintptr(file.Fd())
}

func (w *windowsPlatform) isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	var mode uint32
	h := stdHandle(f)
	r, _, _ := procGetConsoleMode.Call(h, uintptr(unsafe.Pointer(&mode)))
	return r != 0
}

func (w *windowsPlatform) termSize(f *os.File) (int, int, bool) {
	if f == nil {
		return 0, 0, false
	}
	var info consoleScreenBufferInfo
	h := stdHandle(f)
	r, _, _ := procGetConsoleScreenBufferInfo.Call(h, uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		return 0, 0, false
	}
	width := int(info.SrWindow.Right - info.SrWindow.Left + 1)
	height := int(info.SrWindow.Bottom - info.SrWindow.Top + 1)
	if width <= 0 || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func (w *windowsPlatform) enableVirtualTerminal() bool {
	var mode uint32
	h, _, _ := procGetStdHandle.Call(stdOutputHandle)
	if h == 0 || h == uintptr(^uintptr(0)) {
		return false
	}
	r1, _, _ := procGetConsoleMode.Call(h, uintptr(unsafe.Pointer(&mode)))
	if r1 == 0 {
		return false
	}
	if mode&enableVirtualTerminalProcessing != 0 {
		return true
	}
	mode |= enableVirtualTerminalProcessing
	r2, _, _ := procSetConsoleMode.Call(h, uintptr(mode))
	return r2 != 0
}

func (w *windowsPlatform) vtEnabled() bool {
	var mode uint32
	h, _, _ := procGetStdHandle.Call(stdOutputHandle)
	if h == 0 || h == uintptr(^uintptr(0)) {
		return false
	}
	r, _, _ := procGetConsoleMode.Call(h, uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		return false
	}
	return mode&enableVirtualTerminalProcessing != 0
}

// colorCapabilityLevel returns the color level for Windows terminals
// Modern Windows terminals with VT support can handle truecolor
func (w *windowsPlatform) colorCapabilityLevel() int {
	// Check for known truecolor-capable Windows terminals via environment
	if os.Getenv("WT_SESSION") != "" || os.Getenv("WT_PROFILE_ID") != "" {
		return 3 // Windows Terminal
	}
	if os.Getenv("ConEmuANSI") == "ON" {
		return 3 // ConEmu with ANSI support
	}

	// If VT processing is enabled, assume truecolor capability
	if w.vtEnabled() {
		return 3
	}

	// Fallback: if it's a console, assume at least 256 colors on modern Windows
	if w.isTerminal(os.Stdout) {
		return 2
	}

	return 0 // No color support detected
}
