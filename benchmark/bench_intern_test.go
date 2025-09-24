package benchmark

import (
    "testing"

    intern "github.com/dzonerzy/go-snap/internal/intern"
)

// Category: intern

func BenchmarkStringInterner_Intern(b *testing.B) {
    interner := intern.NewStringInterner(0)
    testStrings := []string{"flag1", "flag2", "help", "version", "config"}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        interner.Intern(testStrings[i%len(testStrings)])
    }
}

func BenchmarkStringInterner_InternBytes(b *testing.B) {
    interner := intern.NewStringInterner(0)
    testBytes := [][]byte{
        []byte("flag1"),
        []byte("flag2"),
        []byte("help"),
        []byte("version"),
        []byte("config"),
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        interner.InternBytes(testBytes[i%len(testBytes)])
    }
}

func BenchmarkStringInterner_InternByte(b *testing.B) {
    interner := intern.NewStringInterner(0)
    testBytes := []byte{'a', 'h', 'v', 'c', 'p', 'd'}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        interner.InternByte(testBytes[i%len(testBytes)])
    }
}

func BenchmarkGlobalIntern(b *testing.B) {
    testStrings := []string{"flag1", "flag2", "help", "version", "config"}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        intern.Intern(testStrings[i%len(testStrings)])
    }
}

