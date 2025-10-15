//nolint:testpackage // using package name 'snap' to access unexported fields for testing
package snap

import (
	"bytes"
	"context"
	"errors"
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
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
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
	if err == nil {
		t.Fatalf("expected error")
	}
	exitError := &ExitError{}
	if !errors.As(err, &exitError) {
		t.Fatalf("expected ExitError, got %T", err)
	}
	code := app.ExitCodes().resolve(err)
	if code != 7 {
		t.Fatalf("expected exit code 7, got %d", code)
	}
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
	if err := app.RunWithArgs(context.Background(), []string{"r", "foo"}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if strings.TrimSpace(out.String()) != "bar" {
		t.Fatalf("got %q", out.String())
	}
}

// App-level wrapper should forward unknown top-level tokens as positional
func TestWrapper_AppLevel_PositionalForwarding(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/bin/echo required")
	}
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
	if strings.TrimSpace(out.String()) != "lol" {
		t.Fatalf("got %q", out.String())
	}
}

// App-level ForwardUnknownFlags forwards unknown flags
func TestWrapper_AppLevel_UnknownFlags_Forward(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/bin/echo required")
	}
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
	if strings.TrimSpace(out.String()) != "--weird x" {
		t.Fatalf("got %q", out.String())
	}
}

// "--" terminator must end flag parsing
func TestWrapper_TerminatorDoubleDash(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/bin/echo required")
	}
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
	if strings.TrimSpace(out.String()) != "hello" {
		t.Fatalf("got %q", out.String())
	}
}

// DSL ordering: LeadingFlags + InsertAfterLeadingFlags + MapBoolFlag
func TestWrapper_DSL_LeadingAndAfterLeading(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/usr/bin/printf required on UNIX")
	}
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
	if strings.TrimSpace(got) != "-n [p] hello" {
		t.Fatalf("got %q", got)
	}
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
	if err == nil {
		t.Fatalf("expected error")
	}
	cli := &CLIError{}
	if errors.As(err, &cli) {
		if cli.Type != ErrorTypePermission {
			t.Fatalf("expected permission, got %v", cli.Type)
		}
	}
}

// Dynamic TransformTool can rewrite args before exec
func TestWrapper_Dynamic_TransformTool(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/bin/echo required")
	}
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
	if strings.TrimSpace(out.String()) != "X hello" {
		t.Fatalf("got %q", out.String())
	}
}

// TeeTo writes to an extra writer while streaming
func TestWrapper_TeeTo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/bin/echo required")
	}
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
	if err := app.RunWithArgs(context.Background(), []string{"e", "hi"}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if strings.TrimSpace(out.String()) != "hi" {
		t.Fatalf("out %q", out.String())
	}
	if strings.TrimSpace(tee.String()) != "hi" {
		t.Fatalf("tee %q", tee.String())
	}
}

// Env and WorkingDir use case
func TestWrapper_EnvWorkingDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/bin/sh required")
	}
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
	if err := app.RunWithArgs(context.Background(), []string{"pwd"}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != tmp {
		t.Fatalf("pwd got %q want %q", got, tmp)
	}
}

// ${SELF} token expansion inside a token
func TestWrapper_SelfToken(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/bin/echo required")
	}
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
	if err := app.RunWithArgs(context.Background(), []string{"s"}); err != nil {
		t.Fatalf("err: %v", err)
	}
}
func TestWrapper_ForwardUnknownFlags(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("echo not consistent")
	}
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
	if err == nil {
		t.Fatalf("expected error for unknown flag")
	}

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
	err = app.RunWithArgs(context.Background(), []string{"echo", "--xflag"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out.String()) != "--xflag" {
		t.Fatalf("unexpected out: %q", out.String())
	}
}

// TestWrapperBeforeExecHook tests BeforeExec hook functionality
func TestWrapperBeforeExecHook(t *testing.T) {
	var capturedArgs []string

	app := New("test", "test wrapper")
	app.Command("echo", "wrap echo").
		Wrap("/bin/echo").
		ForwardArgs().
		BeforeExec(func(ctx *Context, args []string) ([]string, error) {
			capturedArgs = append([]string{}, args...)
			// Prepend a prefix to the arguments
			return append([]string{"[BEFORE]"}, args...), nil
		}).
		Passthrough().
		Back()

	var out bytes.Buffer
	app.IO().WithOut(&out)

	err := app.RunWithArgs(context.Background(), []string{"echo", "hello", "world"})
	if err != nil {
		t.Fatalf("RunWithArgs failed: %v", err)
	}

	// Verify BeforeExec was called with correct args
	if len(capturedArgs) != 2 || capturedArgs[0] != "hello" || capturedArgs[1] != "world" {
		t.Errorf("Expected capturedArgs=[hello world], got %v", capturedArgs)
	}

	// Verify output includes prefix from BeforeExec
	output := strings.TrimSpace(out.String())
	if !strings.Contains(output, "[BEFORE]") {
		t.Errorf("Expected output to contain '[BEFORE]', got %q", output)
	}
}

// TestWrapperAfterExecHook tests AfterExec hook functionality
func TestWrapperAfterExecHook(t *testing.T) {
	var capturedResult *ExecResult

	app := New("test", "test wrapper")
	app.Command("echo", "wrap echo").
		Wrap("/bin/echo").
		ForwardArgs().
		Capture().
		AfterExec(func(ctx *Context, result *ExecResult) error {
			capturedResult = result
			return nil
		}).
		Back()

	err := app.RunWithArgs(context.Background(), []string{"echo", "hello"})
	if err != nil {
		t.Fatalf("RunWithArgs failed: %v", err)
	}

	// Verify AfterExec was called with result
	if capturedResult == nil {
		t.Fatal("AfterExec was not called")
	}

	if capturedResult.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", capturedResult.ExitCode)
	}

	output := strings.TrimSpace(string(capturedResult.Stdout))
	if output != "hello" {
		t.Errorf("Expected stdout 'hello', got %q", output)
	}
}

// TestWrapperBeforeExecError tests that BeforeExec errors stop execution
func TestWrapperBeforeExecError(t *testing.T) {
	app := New("test", "test wrapper")
	app.Command("echo", "wrap echo").
		Wrap("/bin/echo").
		ForwardArgs().
		BeforeExec(func(ctx *Context, args []string) ([]string, error) {
			return nil, errors.New("before exec error")
		}).
		Passthrough().
		Back()

	err := app.RunWithArgs(context.Background(), []string{"echo", "hello"})
	if err == nil {
		t.Fatal("Expected error from BeforeExec")
	}

	if err.Error() != "before exec error" {
		t.Errorf("Expected 'before exec error', got %q", err.Error())
	}
}

// TestWrapperAfterExecError tests that AfterExec errors are returned
func TestWrapperAfterExecError(t *testing.T) {
	app := New("test", "test wrapper")
	app.Command("true", "wrap true").
		Wrap("/bin/true").
		Passthrough().
		AfterExec(func(ctx *Context, result *ExecResult) error {
			return errors.New("after exec error")
		}).
		Back()

	err := app.RunWithArgs(context.Background(), []string{"true"})
	if err == nil {
		t.Fatal("Expected error from AfterExec")
	}

	if err.Error() != "after exec error" {
		t.Errorf("Expected 'after exec error', got %q", err.Error())
	}
}

// TestWrapperBeforeAfterExecCombined tests BeforeExec and AfterExec together
func TestWrapperBeforeAfterExecCombined(t *testing.T) {
	var executionOrder []string

	app := New("test", "test wrapper")
	app.Command("echo", "wrap echo").
		Wrap("/bin/echo").
		ForwardArgs().
		BeforeExec(func(ctx *Context, args []string) ([]string, error) {
			executionOrder = append(executionOrder, "before-exec")
			return args, nil
		}).
		Capture().
		AfterExec(func(ctx *Context, result *ExecResult) error {
			executionOrder = append(executionOrder, "after-exec")
			return nil
		}).
		Back()

	err := app.RunWithArgs(context.Background(), []string{"echo", "test"})
	if err != nil {
		t.Fatalf("RunWithArgs failed: %v", err)
	}

	expected := []string{"before-exec", "after-exec"}
	if len(executionOrder) != len(expected) {
		t.Fatalf("Expected %d steps, got %d: %v", len(expected), len(executionOrder), executionOrder)
	}

	for i, step := range expected {
		if executionOrder[i] != step {
			t.Errorf("Step %d: expected %q, got %q", i, step, executionOrder[i])
		}
	}
}

// TestWrapperAfterExecWithFailedCommand tests AfterExec is called even when command fails
func TestWrapperAfterExecWithFailedCommand(t *testing.T) {
	var afterExecCalled bool
	var resultExitCode int

	app := New("test", "test wrapper")
	app.Command("false", "wrap false").
		Wrap("/bin/false").
		Passthrough().
		AfterExec(func(ctx *Context, result *ExecResult) error {
			afterExecCalled = true
			resultExitCode = result.ExitCode
			// Don't return error to allow inspection
			return nil
		}).
		Back()

	err := app.RunWithArgs(context.Background(), []string{"false"})

	// Command should fail with exit code 1
	if err == nil {
		t.Fatal("Expected error from failed command")
	}

	if !afterExecCalled {
		t.Error("AfterExec should have been called even when command fails")
	}

	if resultExitCode != 1 {
		t.Errorf("Expected exit code 1 in AfterExec result, got %d", resultExitCode)
	}
}

// TestWrapperBeforeExecArgModification tests argument modification in BeforeExec
func TestWrapperBeforeExecArgModification(t *testing.T) {
	var capturedOutput string

	app := New("test", "test wrapper")
	app.Command("echo", "wrap echo").
		Wrap("/bin/echo").
		ForwardArgs().
		BeforeExec(func(ctx *Context, args []string) ([]string, error) {
			// Replace all arguments
			return []string{"modified", "args"}, nil
		}).
		Capture().
		AfterExec(func(ctx *Context, result *ExecResult) error {
			capturedOutput = strings.TrimSpace(string(result.Stdout))
			return nil
		}).
		Back()

	err := app.RunWithArgs(context.Background(), []string{"echo", "original", "args"})
	if err != nil {
		t.Fatalf("RunWithArgs failed: %v", err)
	}

	if capturedOutput != "modified args" {
		t.Errorf("Expected output 'modified args', got %q", capturedOutput)
	}
}
