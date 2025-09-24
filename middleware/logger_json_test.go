package middleware

import (
    "bytes"
    "strings"
    "testing"
)

func TestJSONLoggerEscapesStrings(t *testing.T) {
    var buf bytes.Buffer
    mw := LoggerWithWriter(&buf, func(c *MiddlewareConfig) {
        c.LogFormat = LogFormatJSON
        c.IncludeArgs = true
        c.LogLevel = LogLevelInfo
    })

    ctx := NewMockContext()
    ctx.SetArgs([]string{`a "quoted"`, "line1\nline2"})

    err := mw(successAction)(ctx)
    if err != nil { t.Fatalf("unexpected err: %v", err) }

    out := buf.String()
    if !strings.Contains(out, `"args":[`) {
        t.Fatalf("missing args array: %s", out)
    }
    if !strings.Contains(out, `\"quoted\"`) {
        t.Fatalf("expected escaped quotes in args, got: %s", out)
    }
    if !strings.Contains(out, `line1\nline2`) {
        t.Fatalf("expected escaped newline in args, got: %s", out)
    }
}

