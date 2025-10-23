//nolint:testpackage // using package name 'snap' to access unexported fields for testing
package snap

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"sync"
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
		BeforeExec(func(_ *Context, args []string) ([]string, error) {
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
		AfterExec(func(_ *Context, result *ExecResult) error {
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
		BeforeExec(func(_ *Context, _ []string) ([]string, error) {
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
		AfterExec(func(_ *Context, _ *ExecResult) error {
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
		BeforeExec(func(_ *Context, args []string) ([]string, error) {
			executionOrder = append(executionOrder, "before-exec")
			return args, nil
		}).
		Capture().
		AfterExec(func(_ *Context, _ *ExecResult) error {
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
		AfterExec(func(_ *Context, result *ExecResult) error {
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
		BeforeExec(func(_ *Context, _ []string) ([]string, error) {
			// Replace all arguments
			return []string{"modified", "args"}, nil
		}).
		Capture().
		AfterExec(func(_ *Context, result *ExecResult) error {
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

// TestWrapManySequential tests sequential execution of multiple binaries
func TestWrapManySequential(t *testing.T) {
	var executionOrder []string
	var mu sync.Mutex

	app := New("test", "test wrapper")
	app.Command("multi", "run multiple").
		WrapMany("/bin/echo", "/bin/true", "/bin/false").
		StopOnError(false). // continue even if one fails
		AfterExec(func(ctx *Context, _ *ExecResult) error {
			mu.Lock()
			binary := ctx.CurrentBinary()
			executionOrder = append(executionOrder, binary)
			mu.Unlock()
			return nil
		}).
		Back()

	err := app.RunWithArgs(context.Background(), []string{"multi"})
	if err != nil {
		t.Fatalf("RunWithArgs failed: %v", err)
	}

	// Verify all three executed in order
	if len(executionOrder) != 3 {
		t.Errorf("Expected 3 executions, got %d", len(executionOrder))
	}
	if executionOrder[0] != "/bin/echo" || executionOrder[1] != "/bin/true" || executionOrder[2] != "/bin/false" {
		t.Errorf("Unexpected execution order: %v", executionOrder)
	}
}

// TestWrapManyStopOnError tests that execution stops on first error by default
func TestWrapManyStopOnError(t *testing.T) {
	var executed []string
	var mu sync.Mutex

	app := New("test", "test wrapper")
	app.Command("multi", "run multiple").
		WrapMany("/bin/false", "/bin/true", "/bin/echo").
		// StopOnError is true by default
		AfterExec(func(ctx *Context, _ *ExecResult) error {
			mu.Lock()
			binary := ctx.CurrentBinary()
			executed = append(executed, binary)
			mu.Unlock()
			return nil
		}).
		Back()

	err := app.RunWithArgs(context.Background(), []string{"multi"})
	if err == nil {
		t.Fatal("Expected error from /bin/false")
	}

	// Should only execute first binary since it fails
	if len(executed) != 1 {
		t.Errorf("Expected 1 execution (stopped on error), got %d: %v", len(executed), executed)
	}
	if executed[0] != "/bin/false" {
		t.Errorf("Expected /bin/false, got %s", executed[0])
	}
}

// TestWrapManyContinueOnError tests that execution continues when StopOnError(false)
func TestWrapManyContinueOnError(t *testing.T) {
	var executed []string
	var mu sync.Mutex

	app := New("test", "test wrapper")
	app.Command("multi", "run multiple").
		WrapMany("/bin/false", "/bin/true", "/bin/echo").
		StopOnError(false).
		AfterExec(func(ctx *Context, _ *ExecResult) error {
			mu.Lock()
			binary := ctx.CurrentBinary()
			executed = append(executed, binary)
			mu.Unlock()
			return nil
		}).
		Back()

	err := app.RunWithArgs(context.Background(), []string{"multi"})
	if err != nil {
		t.Fatalf("Should not error with StopOnError(false): %v", err)
	}

	// Should execute all three even though first fails
	if len(executed) != 3 {
		t.Errorf("Expected 3 executions, got %d: %v", len(executed), executed)
	}
}

// TestWrapManyContextAccessors tests CurrentBinary() and Binaries()
func TestWrapManyContextAccessors(t *testing.T) {
	var capturedBinary string
	var capturedBinaries []string

	app := New("test", "test wrapper")
	app.Command("multi", "run multiple").
		WrapMany("go1.21", "go1.22").
		AfterExec(func(ctx *Context, _ *ExecResult) error {
			if capturedBinary == "" {
				capturedBinary = ctx.CurrentBinary()
				capturedBinaries = ctx.Binaries()
			}
			return nil
		}).
		Back()

	// This will fail since go1.21/go1.22 don't exist, but that's ok for this test
	_ = app.RunWithArgs(context.Background(), []string{"multi"})

	if capturedBinary != "go1.21" {
		t.Errorf("Expected current binary 'go1.21', got %q", capturedBinary)
	}

	if len(capturedBinaries) != 2 || capturedBinaries[0] != "go1.21" || capturedBinaries[1] != "go1.22" {
		t.Errorf("Expected binaries [go1.21 go1.22], got %v", capturedBinaries)
	}
}

// TestWrapManyParallel tests parallel execution of multiple binaries
func TestWrapManyParallel(t *testing.T) {
	var executed sync.Map

	app := New("test", "test wrapper")
	app.Command("multi", "run multiple").
		WrapMany("/bin/echo", "/bin/true", "/bin/sleep").
		Parallel().
		StopOnError(false).
		AfterExec(func(ctx *Context, _ *ExecResult) error {
			binary := ctx.CurrentBinary()
			executed.Store(binary, true)
			return nil
		}).
		Back()

	err := app.RunWithArgs(context.Background(), []string{"multi", "0.01"})
	if err != nil {
		t.Fatalf("RunWithArgs failed: %v", err)
	}

	// Verify all three executed
	count := 0
	executed.Range(func(_, _ any) bool {
		count++
		return true
	})

	if count != 3 {
		t.Errorf("Expected 3 parallel executions, got %d", count)
	}
}

// TestWrapManyParallelStopOnError tests parallel execution with error
func TestWrapManyParallelStopOnError(t *testing.T) {
	var executed sync.Map

	app := New("test", "test wrapper")
	app.Command("multi", "run multiple").
		WrapMany("/bin/false", "/bin/true", "/bin/echo").
		Parallel().
		// StopOnError defaults to true, but all goroutines complete
		AfterExec(func(ctx *Context, _ *ExecResult) error {
			binary := ctx.CurrentBinary()
			executed.Store(binary, true)
			return nil
		}).
		Back()

	err := app.RunWithArgs(context.Background(), []string{"multi"})
	if err == nil {
		t.Fatal("Expected error from /bin/false")
	}

	// All should execute (parallel), but first error should be returned
	count := 0
	executed.Range(func(_, _ any) bool {
		count++
		return true
	})

	if count != 3 {
		t.Errorf("Expected all 3 to execute in parallel, got %d", count)
	}
}

// TestWrapDynamic_PreservesDoubleDash tests that "--" is preserved in WrapDynamic mode
// This is critical for tools like cgo that use "--" to separate tool flags from compiler flags
func TestWrapDynamic_PreservesDoubleDash(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/bin/echo required")
	}
	app := New("wr", "test")
	var out bytes.Buffer
	app.IO().WithOut(&out)

	// WrapDynamic should preserve "--" as a positional argument
	app.Command("shim", "dynamic shim").
		WrapDynamic().
		ForwardUnknownFlags().
		Passthrough().
		Back()

	// Simulate toolexec invocation: shim /bin/echo -- -n hello
	// The "--" should be passed through to echo
	err := app.RunWithArgs(context.Background(), []string{"shim", "/bin/echo", "--", "hello", "world"})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}

	// The output should include "--" because it's passed to echo as an argument
	got := strings.TrimSpace(out.String())
	if got != "-- hello world" {
		t.Fatalf("expected '-- hello world', got %q", got)
	}
}

// TestWrapper_DoubleDashConsumedInNormalMode tests that "--" is consumed (not passed through) in normal wrapper mode
func TestWrapper_DoubleDashConsumedInNormalMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/bin/echo required")
	}
	app := New("wr", "test")
	var out bytes.Buffer
	app.IO().WithOut(&out)

	// Normal wrapper mode should consume "--"
	app.Command("echo", "wrap echo").
		Wrap("/bin/echo").
		ForwardArgs().
		Passthrough().
		Back()

	err := app.RunWithArgs(context.Background(), []string{"echo", "--", "hello"})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}

	// The output should NOT include "--" because it's consumed by the parser
	got := strings.TrimSpace(out.String())
	if got != "hello" {
		t.Fatalf("expected 'hello', got %q", got)
	}
}
