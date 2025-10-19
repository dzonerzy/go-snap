//go:build !windows

package snapio

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

type unixPlatform struct {
	colorCapOnce sync.Once
	colorCap     int // Cached result: -1=unknown, 0=none, 8=basic, 256=256color, 16777216=truecolor
}

func newPlatformIO() platformIO { return &unixPlatform{} }

type winsize struct{ Row, Col, Xpixel, Ypixel uint16 }

func (u *unixPlatform) isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	var ws winsize
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws)))
	if errno == 0 {
		return true
	}
	// fallback: character device check
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func (u *unixPlatform) termSize(f *os.File) (int, int, bool) {
	if f == nil {
		return 0, 0, false
	}
	var ws winsize
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws)))
	if errno != 0 {
		return 0, 0, false
	}
	if ws.Col == 0 || ws.Row == 0 {
		return 0, 0, false
	}
	return int(ws.Col), int(ws.Row), true
}

func (u *unixPlatform) enableVirtualTerminal() bool { return true }
func (u *unixPlatform) vtEnabled() bool             { return true }

// detectColorCapability queries the terminal for its actual color capability
func (u *unixPlatform) detectColorCapability() int {
	u.colorCapOnce.Do(func() {
		u.colorCap = -1 // Unknown by default

		// First check tput RGB - most reliable for truecolor detection
		cmd := exec.Command("tput", "RGB")
		if err := cmd.Run(); err == nil {
			// If RGB capability exists, we have truecolor
			u.colorCap = 16777216 // 2^24
			return
		}

		// Try tput colors command to get the number of colors supported
		cmd = exec.Command("tput", "colors")
		cmd.Env = os.Environ()
		output, err := cmd.Output()
		if err == nil {
			colors := strings.TrimSpace(string(output))
			if n, parseErr := strconv.Atoi(colors); parseErr == nil {
				u.colorCap = n
				return
			}
		}

		// Last resort: parse TERM for hints
		term := os.Getenv("TERM")
		if strings.Contains(term, "256") {
			u.colorCap = 256
		} else if term != "" && term != "dumb" {
			u.colorCap = 8 // Basic colors
		}
	})
	return u.colorCap
}

// colorCapabilityLevel returns the color level based on detected capability
func (u *unixPlatform) colorCapabilityLevel() int {
	capability := u.detectColorCapability()
	if capability >= 16777216 {
		return 3 // Truecolor
	}
	if capability >= 256 {
		return 2 // 256 colors
	}
	if capability >= 8 {
		return 1 // Basic 16 colors
	}
	return 0 // No color
}
