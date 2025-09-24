//go:build !windows

package snapio

import (
	"os"
	"syscall"
	"unsafe"
)

type unixPlatform struct{}

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
