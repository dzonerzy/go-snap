package benchmark

import (
    "bytes"
    "testing"

    snapio "github.com/dzonerzy/go-snap/io"
)

// Category: io

func BenchmarkIO_Colorize(b *testing.B) {
    io := snapio.New().ForceColor()
    s := "hello world"
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = io.Colorize(s, "31") // red
    }
}

func BenchmarkIO_Styling(b *testing.B) {
    io := snapio.New().ForceColor()
    s := "hello world"
    b.Run("Bold", func(b *testing.B) {
        for i := 0; i < b.N; i++ { _ = io.Bold(s) }
    })
    b.Run("Underline", func(b *testing.B) {
        for i := 0; i < b.N; i++ { _ = io.Underline(s) }
    })
}

func BenchmarkIO_Write(b *testing.B) {
    buf := &bytes.Buffer{}
    io := snapio.New().WithOut(buf)
    data := []byte("some output line\n")
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = io.Out().Write(data)
        buf.Reset()
    }
}

