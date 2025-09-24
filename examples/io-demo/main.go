package main

import (
    "fmt"
    snapio "github.com/dzonerzy/go-snap/io"
)

// Demonstrates IOManager features standalone (outside of App).
func main() {
    io := snapio.New()

    // Basic capability checks
    fmt.Printf("TTY: %v, Interactive: %v, Redirected: %v, Piped: %v\n",
        io.IsTTY(), io.IsInteractive(), io.IsRedirected(), io.IsPiped())
    fmt.Printf("Size: %dx%d (cols x rows)\n", io.Width(), io.Height())
    fmt.Printf("Color Supported: %v (level=%d)\n", io.SupportsColor(), io.ColorLevel())

    // Color helpers (will no-op if color unsupported unless forced)
    sample := "Hello, color!"
    fmt.Println(io.Bold(sample))
    fmt.Println(io.Underline(io.Colorize("red text", "31")))

    // Force color regardless of terminal detection (useful for CI previews)
    io.ForceColor()
    fmt.Println(io.Colorize("forced green", "32"))
}

