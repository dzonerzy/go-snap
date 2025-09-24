package snapio

import (
    "fmt"
)

// ColorSpec represents a color in one of three spaces: basic (16), indexed (256), or truecolor (RGB)
type ColorSpec struct {
    kind  int // 1=basic, 2=indexed, 3=truecolor
    index int // for basic (0-15) and indexed (0-255)
    r, g, b uint8
}

// Basic color helpers (0-7 normal, 8-15 bright)
var (
    Black        = basic(0)
    Red          = basic(1)
    Green        = basic(2)
    Yellow       = basic(3)
    Blue         = basic(4)
    Magenta      = basic(5)
    Cyan         = basic(6)
    White        = basic(7)

    BrightBlack  = basic(8)
    BrightRed    = basic(9)
    BrightGreen  = basic(10)
    BrightYellow = basic(11)
    BrightBlue   = basic(12)
    BrightMagenta= basic(13)
    BrightCyan   = basic(14)
    BrightWhite  = basic(15)
)

func basic(i int) ColorSpec { return ColorSpec{kind:1, index:i} }
// Indexed returns a 256-color palette spec (0–255).
func Indexed(i int) ColorSpec { return ColorSpec{kind:2, index:i} }
// Truecolor returns a 24‑bit RGB color spec.
func Truecolor(r, g, b uint8) ColorSpec { return ColorSpec{kind:3, r:r, g:g, b:b} }

// Style is a fluent style builder for foreground/background colors and
// attributes (bold, faint, italic, underline, inverse).
type Style struct {
    fg, bg *ColorSpec
    bold, faint, italic, underline, inverse bool
}

// NewStyle creates a new empty style builder.
func NewStyle() *Style { return &Style{} }
func (s *Style) Fg(c ColorSpec) *Style { s.fg=&c; return s }
func (s *Style) Bg(c ColorSpec) *Style { s.bg=&c; return s }
func (s *Style) Bold() *Style      { s.bold=true; return s }
func (s *Style) Faint() *Style     { s.faint=true; return s }
func (s *Style) Italic() *Style    { s.italic=true; return s }
func (s *Style) Underline() *Style { s.underline=true; return s }
func (s *Style) Inverse() *Style   { s.inverse=true; return s }

// Sprint returns a styled string if color is supported; otherwise it returns
// the text unchanged.
func (s *Style) Sprint(io *IOManager, text string) string {
    if !io.SupportsColor() { return text }
    seq := s.ansiPrefix(io)
    if seq == "" { return text }
    return "\x1b[" + seq + "m" + text + "\x1b[0m"
}

// Sprintf formats the content with fmt.Sprintf and then applies the style.
func (s *Style) Sprintf(io *IOManager, format string, a ...any) string {
    return s.Sprint(io, fmt.Sprintf(format, a...))
}

func (s *Style) ansiPrefix(io *IOManager) string {
    codes := make([]string, 0, 6)
    // attributes first
    if s.bold { codes = append(codes, "1") }
    if s.faint { codes = append(codes, "2") }
    if s.italic { codes = append(codes, "3") }
    if s.underline { codes = append(codes, "4") }
    if s.inverse { codes = append(codes, "7") }
    // colors depending on level
    lvl := io.ColorLevel()
    if s.fg != nil {
        codes = append(codes, colorCode(*s.fg, false, lvl))
    }
    if s.bg != nil {
        codes = append(codes, colorCode(*s.bg, true, lvl))
    }
    // join
    out := ""
    for _, c := range codes {
        if c == "" { continue }
        if out != "" { out += ";" }
        out += c
    }
    return out
}

func colorCode(c ColorSpec, bg bool, level int) string {
    base := 30
    if bg { base = 40 }
    switch c.kind {
    case 1: // basic 16
        idx := c.index
        if idx < 0 { idx = 0 }
        if idx > 15 { idx = 15 }
        if idx < 8 {
            return itoa(base + idx)
        }
        // bright
        return itoa(base + 60 + (idx-8))
    case 2: // indexed 256
        if level >= 2 {
            if bg { return fmt.Sprintf("48;5;%d", c.index) }
            return fmt.Sprintf("38;5;%d", c.index)
        }
        // fallback to default fg/bg when only 16 colors available
        return ""
    case 3: // truecolor
        if level >= 3 {
            if bg { return fmt.Sprintf("48;2;%d;%d;%d", c.r, c.g, c.b) }
            return fmt.Sprintf("38;2;%d;%d;%d", c.r, c.g, c.b)
        }
        return ""
    default:
        return ""
    }
}

func itoa(n int) string {
    if n == 0 { return "0" }
    buf := [6]byte{}
    i := len(buf)
    for n > 0 && i > 0 {
        i--
        d := n % 10
        buf[i] = byte('0' + d)
        n /= 10
    }
    return string(buf[i:])
}

// Theme provides semantic colors
type Theme struct {
    Primary, Success, Warning, Error, Info, Muted ColorSpec
}

func DefaultTheme() Theme {
    return Theme{
        Primary: BrightBlue,
        Success: BrightGreen,
        Warning: BrightYellow,
        Error:   BrightRed,
        Info:    BrightCyan,
        Muted:   BrightBlack,
    }
}
