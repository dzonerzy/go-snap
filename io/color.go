package snapio

import (
	"fmt"
)

// ColorSpec represents a color in one of three spaces: basic (16), indexed (256), or truecolor (RGB)
type ColorSpec struct {
	kind    int // 1=basic, 2=indexed, 3=truecolor
	index   int // for basic (0-15) and indexed (0-255)
	r, g, b uint8
}

// Basic color helpers (0-7 normal, 8-15 bright)
var (
	Black   = basic(0)
	Red     = basic(1)
	Green   = basic(2)
	Yellow  = basic(3)
	Blue    = basic(4)
	Magenta = basic(5)
	Cyan    = basic(6)
	White   = basic(7)

	BrightBlack   = basic(8) // Gray
	BrightRed     = basic(9)
	BrightGreen   = basic(10)
	BrightYellow  = basic(11)
	BrightBlue    = basic(12)
	BrightMagenta = basic(13)
	BrightCyan    = basic(14)
	BrightWhite   = basic(15)
)

// 256-color palette - Extended colors
var (
	// Grays (232-255 are grayscale)
	Gray1 = Indexed(232) // Darkest gray
	Gray2 = Indexed(236)
	Gray3 = Indexed(240)
	Gray4 = Indexed(244)
	Gray5 = Indexed(248)
	Gray6 = Indexed(252) // Lightest gray

	// Reds
	DarkRed      = Indexed(88)
	Orange       = Indexed(208)
	BrightOrange = Indexed(214)

	// Greens
	DarkGreen = Indexed(22)
	LimeGreen = Indexed(118)

	// Blues
	DarkBlue = Indexed(18)
	SkyBlue  = Indexed(117)

	// Purples/Magentas
	Purple       = Indexed(93)
	LightPurple  = Indexed(141)
	BrightPurple = Indexed(165)
	Violet       = Indexed(99)

	// Cyans
	DarkCyan = Indexed(30)
	Aqua     = Indexed(51)

	// Yellows/Golds
	Gold       = Indexed(220)
	DarkYellow = Indexed(136)
)

// Truecolor palette - RGB colors with normal/bright variants
var (
	// Blacks and Grays
	TrueBlack       = Truecolor(0, 0, 0)
	TrueDarkGray    = Truecolor(85, 85, 85)
	TrueGray        = Truecolor(128, 128, 128)
	TrueLightGray   = Truecolor(192, 192, 192)
	TrueBrightWhite = Truecolor(255, 255, 255)

	// Reds
	TrueRed       = Truecolor(205, 49, 49) // Normal red
	TrueBrightRed = Truecolor(255, 85, 85) // Bright red
	TrueDarkRed   = Truecolor(139, 0, 0)   // Dark red
	TrueOrange    = Truecolor(255, 135, 0) // Orange

	// Greens
	TrueGreen       = Truecolor(19, 161, 14)  // Normal green
	TrueBrightGreen = Truecolor(80, 250, 123) // Bright green
	TrueDarkGreen   = Truecolor(0, 100, 0)    // Dark green
	TrueLimeGreen   = Truecolor(50, 205, 50)  // Lime green

	// Yellows
	TrueYellow       = Truecolor(229, 229, 16)  // Normal yellow
	TrueBrightYellow = Truecolor(255, 184, 108) // Bright yellow/orange
	TrueDarkYellow   = Truecolor(184, 134, 11)  // Dark yellow/gold
	TrueGold         = Truecolor(255, 215, 0)   // Gold

	// Blues
	TrueBlue       = Truecolor(59, 120, 255)  // Normal blue
	TrueBrightBlue = Truecolor(92, 148, 252)  // Bright blue
	TrueDarkBlue   = Truecolor(0, 0, 139)     // Dark blue
	TrueSkyBlue    = Truecolor(135, 206, 235) // Sky blue

	// Magentas/Purples
	TrueMagenta       = Truecolor(180, 0, 158)   // Normal magenta
	TrueBrightMagenta = Truecolor(255, 0, 255)   // Bright magenta
	TruePurple        = Truecolor(128, 0, 128)   // Purple
	TrueLightPurple   = Truecolor(189, 147, 249) // Light purple/lavender
	TrueBrightPurple  = Truecolor(221, 160, 221) // Bright purple/plum
	TrueViolet        = Truecolor(138, 43, 226)  // Violet

	// Cyans
	TrueCyan       = Truecolor(41, 184, 219)  // Normal cyan
	TrueBrightCyan = Truecolor(139, 233, 253) // Bright cyan
	TrueDarkCyan   = Truecolor(0, 139, 139)   // Dark cyan
	TrueAqua       = Truecolor(0, 255, 255)   // Aqua
)

func basic(i int) ColorSpec { return ColorSpec{kind: 1, index: i} }

// Indexed returns a 256-color palette spec (0–255).
func Indexed(i int) ColorSpec { return ColorSpec{kind: 2, index: i} }

// Truecolor returns a 24‑bit RGB color spec.
func Truecolor(r, g, b uint8) ColorSpec { return ColorSpec{kind: 3, r: r, g: g, b: b} }

// Style is a fluent style builder for foreground/background colors and
// attributes (bold, faint, italic, underline, inverse).
type Style struct {
	fg, bg                                  *ColorSpec
	bold, faint, italic, underline, inverse bool
}

// NewStyle creates a new empty style builder.
func NewStyle() *Style                 { return &Style{} }
func (s *Style) Fg(c ColorSpec) *Style { s.fg = &c; return s }
func (s *Style) Bg(c ColorSpec) *Style { s.bg = &c; return s }
func (s *Style) Bold() *Style          { s.bold = true; return s }
func (s *Style) Faint() *Style         { s.faint = true; return s }
func (s *Style) Italic() *Style        { s.italic = true; return s }
func (s *Style) Underline() *Style     { s.underline = true; return s }
func (s *Style) Inverse() *Style       { s.inverse = true; return s }

// Sprint returns a styled string if color is supported; otherwise it returns
// the text unchanged.
func (s *Style) Sprint(io *IOManager, text string) string {
	if !io.SupportsColor() {
		return text
	}
	seq := s.ansiPrefix(io)
	if seq == "" {
		return text
	}
	return "\x1b[" + seq + "m" + text + "\x1b[0m"
}

// Sprintf formats the content with fmt.Sprintf and then applies the style.
func (s *Style) Sprintf(io *IOManager, format string, a ...any) string {
	return s.Sprint(io, fmt.Sprintf(format, a...))
}

func (s *Style) ansiPrefix(io *IOManager) string {
	codes := make([]string, 0, 6)
	// attributes first
	if s.bold {
		codes = append(codes, "1")
	}
	if s.faint {
		codes = append(codes, "2")
	}
	if s.italic {
		codes = append(codes, "3")
	}
	if s.underline {
		codes = append(codes, "4")
	}
	if s.inverse {
		codes = append(codes, "7")
	}
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
		if c == "" {
			continue
		}
		if out != "" {
			out += ";"
		}
		out += c
	}
	return out
}

func colorCode(c ColorSpec, bg bool, level int) string {
	base := 30
	if bg {
		base = 40
	}
	switch c.kind {
	case 1: // basic 16
		idx := c.index
		if idx < 0 {
			idx = 0
		}
		if idx > 15 {
			idx = 15
		}
		if idx < 8 {
			return itoa(base + idx)
		}
		// bright
		return itoa(base + 60 + (idx - 8))
	case 2: // indexed 256
		if level >= 2 {
			if bg {
				return fmt.Sprintf("48;5;%d", c.index)
			}
			return fmt.Sprintf("38;5;%d", c.index)
		}
		// fallback to default fg/bg when only 16 colors available
		return ""
	case 3: // truecolor
		if level >= 3 {
			if bg {
				return fmt.Sprintf("48;2;%d;%d;%d", c.r, c.g, c.b)
			}
			return fmt.Sprintf("38;2;%d;%d;%d", c.r, c.g, c.b)
		}
		return ""
	default:
		return ""
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
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
	Primary, Success, Warning, Error, Info, Debug, Muted ColorSpec
}

// DefaultTheme16 returns a theme using basic 16 colors (ANSI colors 0-15).
// Best for terminals with limited color support or ColorLevel 1.
func DefaultTheme16() Theme {
	return Theme{
		Primary: BrightBlue,
		Success: BrightGreen,
		Warning: BrightYellow,
		Error:   BrightRed,
		Info:    BrightCyan,
		Debug:   BrightMagenta, // Best we can do with 16 colors
		Muted:   BrightBlack,   // Gray
	}
}

// DefaultTheme256 returns a theme using 256-color palette.
// Best for terminals with ColorLevel 2 (256 colors).
func DefaultTheme256() Theme {
	return Theme{
		Primary: BrightBlue,
		Success: BrightGreen,
		Warning: BrightYellow,
		Error:   BrightRed,
		Info:    BrightCyan,
		Debug:   LightPurple, // Indexed(141) - proper light purple
		Muted:   BrightBlack,
	}
}

// DefaultThemeTruecolor returns a theme using 24-bit RGB colors.
// Best for terminals with ColorLevel 3 (truecolor/16M colors).
func DefaultThemeTruecolor() Theme {
	return Theme{
		Primary: TrueBrightBlue,   // RGB(92, 148, 252)
		Success: TrueBrightGreen,  // RGB(80, 250, 123)
		Warning: TrueBrightYellow, // RGB(255, 184, 108)
		Error:   TrueBrightRed,    // RGB(255, 85, 85)
		Info:    TrueBrightCyan,   // RGB(139, 233, 253)
		Debug:   TrueLightPurple,  // RGB(189, 147, 249)
		Muted:   TrueGray,         // RGB(128, 128, 128)
	}
}

// DefaultTheme returns the appropriate default theme based on IOManager's color level.
// Automatically selects DefaultTheme16, DefaultTheme256, or DefaultThemeTruecolor.
func DefaultTheme(io *IOManager) Theme {
	switch io.ColorLevel() {
	case 3:
		return DefaultThemeTruecolor()
	case 2:
		return DefaultTheme256()
	case 1:
		return DefaultTheme16()
	default:
		// No color support, but return basic theme anyway (colors won't be rendered)
		return DefaultTheme16()
	}
}

// TruecolorTheme is an alias for DefaultThemeTruecolor for backward compatibility.
// Deprecated: Use DefaultThemeTruecolor() or DefaultTheme(io) instead.
func TruecolorTheme() Theme {
	return DefaultThemeTruecolor()
}
