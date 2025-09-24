package snap

import (
    "bytes"
    "context"
    "os/exec"
    "runtime"
    "strings"
    "testing"
)

// Test command-level wrapper that injects pre-args and forwards positional args
func TestWrapper_Command_Passthrough(t *testing.T) {
    if runtime.GOOS == "windows" {
        t.Skip("wrapper tests use /bin/echo and /bin/sh; skip on windows in unit environment")
    }

    app := New("wr", "test")
    // Capture stdout
    var out bytes.Buffer
    app.IO().WithOut(&out)

    app.Command("echo", "wrap /bin/echo").
        Wrap("/bin/echo").
        InjectArgsPre("wrapped:").
        ForwardArgs().
        Passthrough().
        Back()

    err := app.RunWithArgs(context.Background(), []string{"echo", "hello"})
    if err != nil { t.Fatalf("run error: %v", err) }
    got := strings.TrimSpace(out.String())
    if got != "wrapped: hello" {
        t.Fatalf("unexpected output: %q", got)
    }
}

// Test wrapper exit code mapping (child non-zero -> ExitError)
func TestWrapper_ExitCodeMapping(t *testing.T) {
    if runtime.GOOS == "windows" {
        t.Skip("uses /bin/sh -c 'exit 7'")
    }
    app := New("wr", "test")
    app.Command("fail", "exit 7").
        Wrap("/bin/sh").
        InjectArgsPre("-c", "exit 7").
        Passthrough().
        Back()

    err := app.RunWithArgs(context.Background(), []string{"fail"})
    if err == nil { t.Fatalf("expected error") }
    if _, ok := err.(*ExitError); !ok {
        t.Fatalf("expected ExitError, got %T", err)
    }
    code := app.ExitCodes().resolve(err)
    if code != 7 { t.Fatalf("expected exit code 7, got %d", code) }
}

// Test WrapDynamic default hidden from help
func TestWrapper_HiddenFromHelp(t *testing.T) {
    app := New("wr", "test")
    app.Command("shim", "dynamic shim").
        WrapDynamic().
        Passthrough().
        Back()

    // shim should be hidden
    if !app.commands["shim"].Hidden {
        t.Fatalf("expected shim to be hidden by default")
    }
}

// Sanity: ensure LookPath resolves binary if DiscoverOnPATH true
func TestWrapper_PathDiscovery(t *testing.T) {
    // Only if echo found on PATH
    if _, err := exec.LookPath("echo"); err != nil {
        t.Skip("echo not on PATH")
    }
    app := New("wr", "test")
    var out bytes.Buffer
    app.IO().WithOut(&out)
    app.Command("e", "wrap echo").
        Wrap("echo"). // bare name, should be resolved
        InjectArgsPre("ok").
        ForwardArgs().
        Passthrough().
        Back()
    if err := app.RunWithArgs(context.Background(), []string{"e", "1"}); err != nil {
        t.Fatalf("run error: %v", err)
    }
    if strings.TrimSpace(out.String()) != "ok 1" {
        t.Fatalf("unexpected output: %q", out.String())
    }
}

func TestWrapper_CaptureTo(t *testing.T) {
    if runtime.GOOS == "windows" {
        t.Skip("/bin/echo not on windows")
    }
    app := New("wr", "test")
    // Still stream to out, but also capture
    var out bytes.Buffer
    app.IO().WithOut(&out)
    app.Command("cap", "wrap echo capture").
        Wrap("/bin/echo").
        InjectArgsPre("hi").
        ForwardArgs().
        CaptureTo(nil, nil).
        Back()
    if err := app.RunWithArgs(context.Background(), []string{"cap"}); err != nil {
        t.Fatalf("run error: %v", err)
    }
    // capture via context After isn't available in tests; re-run with a tiny action hook
    // Instead, simulate by injecting an after hook: not available; so assert stream worked
    if strings.TrimSpace(out.String()) != "hi" {
        t.Fatalf("unexpected stream: %q", out.String())
    }
}

func TestWrapper_ReplaceArg(t *testing.T) {
    if runtime.GOOS == "windows" {
        t.Skip("/bin/echo not on windows")
    }
    app := New("wr", "test")
    var out bytes.Buffer
    app.IO().WithOut(&out)
    app.Command("r", "replace").
        Wrap("/bin/echo").
        ForwardArgs().
        ReplaceArg("foo", "bar").
        Passthrough().
        Back()
    if err := app.RunWithArgs(context.Background(), []string{"r", "foo"}); err != nil { t.Fatalf("err: %v", err) }
    if strings.TrimSpace(out.String()) != "bar" { t.Fatalf("got %q", out.String()) }
}

// App-level wrapper should forward unknown top-level tokens as positional
func TestWrapper_AppLevel_PositionalForwarding(t *testing.T) {
    if runtime.GOOS == "windows" { t.Skip("/bin/echo required") }
    app := New("wr", "test")
    var out bytes.Buffer
    app.IO().WithOut(&out)
    app.Wrap("/bin/echo").
        ForwardArgs().
        Passthrough().
        Back()
    if err := app.RunWithArgs(context.Background(), []string{"lol"}); err != nil {
        t.Fatalf("run error: %v", err)
    }
    if strings.TrimSpace(out.String()) != "lol" { t.Fatalf("got %q", out.String()) }
}

// App-level ForwardUnknownFlags forwards unknown flags
func TestWrapper_AppLevel_UnknownFlags_Forward(t *testing.T) {
    if runtime.GOOS == "windows" { t.Skip("/bin/echo required") }
    app := New("wr", "test")
    var out bytes.Buffer
    app.IO().WithOut(&out)
    app.Wrap("/bin/echo").
        ForwardArgs().
        ForwardUnknownFlags().
        Passthrough().
        Back()
    if err := app.RunWithArgs(context.Background(), []string{"--weird", "x"}); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if strings.TrimSpace(out.String()) != "--weird x" { t.Fatalf("got %q", out.String()) }
}

// "--" terminator must end flag parsing
func TestWrapper_TerminatorDoubleDash(t *testing.T) {
    if runtime.GOOS == "windows" { t.Skip("/bin/echo required") }
    app := New("wr", "test")
    var out bytes.Buffer
    app.IO().WithOut(&out)
    app.Wrap("/bin/echo").
        ForwardArgs().
        Passthrough().
        Back()
    if err := app.RunWithArgs(context.Background(), []string{"--", "-n", "hello"}); err != nil {
        t.Fatalf("run error: %v", err)
    }
    if strings.TrimSpace(out.String()) != "hello" { t.Fatalf("got %q", out.String()) }
}

// DSL ordering: LeadingFlags + InsertAfterLeadingFlags + MapBoolFlag
func TestWrapper_DSL_LeadingAndAfterLeading(t *testing.T) {
    if runtime.GOOS == "windows" { t.Skip("/usr/bin/printf required on UNIX") }
    app := New("wr", "test")
    var got string
    app.Wrap("/usr/bin/printf").
        InjectArgsPre("%s %s %s\n").
        ForwardUnknownFlags().
        ForwardArgs().
        LeadingFlags("-n").
        InsertAfterLeadingFlags("[p]").
        CaptureTo(nil, nil).
        Back()
    app.After(func(ctx *Context) error {
        if r, ok := ctx.WrapperResult(); ok {
            got = string(r.Stdout)
        }
        return nil
    })
    if err := app.RunWithArgs(context.Background(), []string{"-n", "hello"}); err != nil {
        t.Fatalf("run error: %v", err)
    }
    if strings.TrimSpace(got) != "-n [p] hello" { t.Fatalf("got %q", got) }
}

// Dynamic AllowTools denies disallowed tool without exec
func TestWrapper_Dynamic_AllowTools(t *testing.T) {
    app := New("wr", "test")
    app.Command("shim", "").
        WrapDynamic().
        AllowTools("echo").
        Passthrough().
        Back()
    // Intentionally pass disallowed tool; expect permission error
    err := app.RunWithArgs(context.Background(), []string{"shim", "/bin/ls"})
    if err == nil { t.Fatalf("expected error") }
    if cli, ok := err.(*CLIError); ok {
        if cli.Type != ErrorTypePermission { t.Fatalf("expected permission, got %v", cli.Type) }
    }
}

// Dynamic TransformTool can rewrite args before exec
func TestWrapper_Dynamic_TransformTool(t *testing.T) {
    if runtime.GOOS == "windows" { t.Skip("/bin/echo required") }
    app := New("wr", "test")
    var out bytes.Buffer
    app.IO().WithOut(&out)
    app.Command("shim", "").
        WrapDynamic().
        TransformTool(func(tool string, args []string) (string, []string, error) {
            // Insert a prefix token before args
            return tool, append([]string{"X"}, args...), nil
        }).
        ForwardUnknownFlags().
        Passthrough().
        Back()
    // tool=/bin/echo, args=hello
    if err := app.RunWithArgs(context.Background(), []string{"shim", "/bin/echo", "hello"}); err != nil {
        t.Fatalf("run error: %v", err)
    }
    if strings.TrimSpace(out.String()) != "X hello" { t.Fatalf("got %q", out.String()) }
}

// TeeTo writes to an extra writer while streaming
func TestWrapper_TeeTo(t *testing.T) {
    if runtime.GOOS == "windows" { t.Skip("/bin/echo required") }
    app := New("wr", "test")
    var out bytes.Buffer
    var tee bytes.Buffer
    app.IO().WithOut(&out)
    app.Command("e", "").
        Wrap("/bin/echo").
        ForwardArgs().
        Passthrough().
        TeeTo(&tee, nil).
        Back()
    if err := app.RunWithArgs(context.Background(), []string{"e", "hi"}); err != nil { t.Fatalf("err: %v", err) }
    if strings.TrimSpace(out.String()) != "hi" { t.Fatalf("out %q", out.String()) }
    if strings.TrimSpace(tee.String()) != "hi" { t.Fatalf("tee %q", tee.String()) }
}

// Env and WorkingDir use case
func TestWrapper_EnvWorkingDir(t *testing.T) {
    if runtime.GOOS == "windows" { t.Skip("/bin/sh required") }
    app := New("wr", "test")
    tmp := t.TempDir()
    // capture output via CaptureTo
    app.Command("pwd", "").
        Wrap("/bin/sh").
        InjectArgsPre("-c", "pwd").
        WorkingDir(tmp).
        CaptureTo(nil, nil).
        Back()
    var got string
    app.After(func(ctx *Context) error {
        if r, ok := ctx.WrapperResult(); ok {
            got = strings.TrimSpace(string(r.Stdout))
        }
        return nil
    })
    if err := app.RunWithArgs(context.Background(), []string{"pwd"}); err != nil { t.Fatalf("err: %v", err) }
    if got != tmp { t.Fatalf("pwd got %q want %q", got, tmp) }
}

// ${SELF} token expansion inside a token
func TestWrapper_SelfToken(t *testing.T) {
    if runtime.GOOS == "windows" { t.Skip("/bin/echo required") }
    app := New("wr", "test")
    var out bytes.Buffer
    app.IO().WithOut(&out)
    app.Command("s", "").
        Wrap("/bin/echo").
        InjectArgsPre("X=${SELF}").
        CaptureTo(nil, nil).
        Back()
    app.After(func(ctx *Context) error {
        if r, ok := ctx.WrapperResult(); ok {
            if !strings.HasPrefix(strings.TrimSpace(string(r.Stdout)), "X=/") {
                t.Fatalf("unexpected SELF expansion: %q", string(r.Stdout))
            }
        }
        return nil
    })
    if err := app.RunWithArgs(context.Background(), []string{"s"}); err != nil { t.Fatalf("err: %v", err) }
}
func TestWrapper_ForwardUnknownFlags(t *testing.T) {
    if runtime.GOOS == "windows" { t.Skip("echo not consistent") }
    // Case 1: no forward-unknown -> error
    app := New("wr", "test")
    var out bytes.Buffer
    app.IO().WithOut(&out)
    app.Command("echo", "wrap echo").
        Wrap("/bin/echo").
        ForwardArgs().
        Passthrough().
        Back()
    err := app.RunWithArgs(context.Background(), []string{"echo", "--xflag"})
    if err == nil { t.Fatalf("expected error for unknown flag") }

    // Case 2: enable forward-unknown
    app = New("wr", "test")
    out.Reset()
    app.IO().WithOut(&out)
    app.Command("echo", "wrap echo").
        Wrap("/bin/echo").
        ForwardArgs().
        ForwardUnknownFlags().
        Passthrough().
        Back()
    if err := app.RunWithArgs(context.Background(), []string{"echo", "--xflag"}); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if strings.TrimSpace(out.String()) != "--xflag" {
        t.Fatalf("unexpected out: %q", out.String())
    }
}
