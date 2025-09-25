package snap

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExecResult provides information about wrapped command execution
type ExecResult struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
	Error    error
}

// wrapperMode selects how child output is handled
type wrapperMode int

const (
	modePassthrough wrapperMode = iota
	modeCapture
)

// WrapperSpec captures the configured behavior for a wrapper
type WrapperSpec struct {
	Binary          string
	DiscoverOnPATH  bool
	WorkingDir      string
	Env             map[string]string
	InheritEnv      bool
	PreArgs         []string
	PostArgs        []string
	ForwardArgs     bool
	ForwardUnknown  bool // reserved for future raw-forward support
	Transform       func(*Context, []string) ([]string, error)
	TransformToolFn func(tool string, args []string) (string, []string, error)
	Mode            wrapperMode
	TeeOut          io.Writer
	TeeErr          io.Writer
	CaptureAlso     bool // when true in passthrough, also capture into ExecResult
	Dynamic         bool // reserved for toolexec dynamic shim
	// DSL helpers
	LeadingFlags []string
	AfterLeading []string
	MapBool      map[string][]string // wrapper bool flag name -> child tokens
}

// WrapperBuilder provides a fluent API to configure a wrapper.
// P is the parent type (*App or *CommandBuilder) to support .Back().
type WrapperBuilder[P any] struct {
	parent P
	spec   *WrapperSpec
	cmd    *Command // present when wrapping a command (to support HideFromHelp)
	app    *App     // present when wrapping at app level
}

// Wrap configures an application-level wrapper. When no command is provided on
// the CLI, the wrapper will execute instead of showing help.
func (a *App) Wrap(binary string) *WrapperBuilder[*App] {
	spec := &WrapperSpec{
		Binary:         binary,
		DiscoverOnPATH: true,
		InheritEnv:     true,
		ForwardArgs:    true,
		Mode:           modePassthrough,
		Env:            make(map[string]string),
	}
	a.defaultWrapper = spec
	return &WrapperBuilder[*App]{parent: a, spec: spec, app: a}
}

// Wrap configures a command-level wrapper that executes when this command runs.
func (c *CommandBuilder) Wrap(binary string) *WrapperBuilder[*CommandBuilder] {
	spec := &WrapperSpec{
		Binary:         binary,
		DiscoverOnPATH: true,
		InheritEnv:     true,
		ForwardArgs:    true,
		Mode:           modePassthrough,
		Env:            make(map[string]string),
	}
	c.command.wrapper = spec
	return &WrapperBuilder[*CommandBuilder]{parent: c, spec: spec, cmd: c.command}
}

// WrapDynamic configures a command as a dynamic wrapper (toolexec shim). By
// default it is hidden from help since users do not invoke it directly.
func (c *CommandBuilder) WrapDynamic() *WrapperBuilder[*CommandBuilder] {
	b := c.Wrap("")
	b.spec.Dynamic = true
	// Hide by default for dynamic shims
	if c.command != nil {
		c.command.Hidden = true
	}
	return b
}

// Binary sets / overrides the binary to execute.
func (b *WrapperBuilder[P]) Binary(path string) *WrapperBuilder[P] { b.spec.Binary = path; return b }

// DiscoverOnPATH enables/disables PATH lookup for bare binary names (default true).
func (b *WrapperBuilder[P]) DiscoverOnPATH(enable bool) *WrapperBuilder[P] {
	b.spec.DiscoverOnPATH = enable
	return b
}

// WorkingDir sets the working directory for the child process.
func (b *WrapperBuilder[P]) WorkingDir(dir string) *WrapperBuilder[P] {
	b.spec.WorkingDir = dir
	return b
}

// Env sets/overrides a single environment variable for the child process.
func (b *WrapperBuilder[P]) Env(key, value string) *WrapperBuilder[P] {
	if b.spec.Env == nil {
		b.spec.Env = make(map[string]string)
	}
	b.spec.Env[key] = value
	return b
}

// EnvMap sets multiple environment variables.
func (b *WrapperBuilder[P]) EnvMap(vars map[string]string) *WrapperBuilder[P] {
	if b.spec.Env == nil {
		b.spec.Env = make(map[string]string)
	}
	for k, v := range vars {
		b.spec.Env[k] = v
	}
	return b
}

// InheritEnv controls whether to inherit the parent environment (default true).
func (b *WrapperBuilder[P]) InheritEnv(enable bool) *WrapperBuilder[P] {
	b.spec.InheritEnv = enable
	return b
}

// InjectArgsPre inserts arguments before forwarded args.
func (b *WrapperBuilder[P]) InjectArgsPre(args ...string) *WrapperBuilder[P] {
	b.spec.PreArgs = append(b.spec.PreArgs, args...)
	return b
}

// InjectArgsPost appends arguments after forwarded args.
func (b *WrapperBuilder[P]) InjectArgsPost(args ...string) *WrapperBuilder[P] {
	b.spec.PostArgs = append(b.spec.PostArgs, args...)
	return b
}

// ForwardArgs forwards positional args to the child (default true).
func (b *WrapperBuilder[P]) ForwardArgs() *WrapperBuilder[P] { b.spec.ForwardArgs = true; return b }

// ForwardUnknownFlags forwards unknown flags (those not defined in this CLI)
// as positional arguments to the wrapped binary instead of failing parsing.
// Applies in the current command context (or app-level when using app.Wrap()).
func (b *WrapperBuilder[P]) ForwardUnknownFlags() *WrapperBuilder[P] {
	b.spec.ForwardUnknown = true
	return b
}

// TransformArgs provides full control over the final argv.
func (b *WrapperBuilder[P]) TransformArgs(fn func(*Context, []string) ([]string, error)) *WrapperBuilder[P] {
	b.spec.Transform = fn
	return b
}

// ReplaceArg finds the first occurrence of 'find' in the final argv and replaces
// it with 'repl'. Useful for quick in-place edits without writing a full
// TransformArgs function.
func (b *WrapperBuilder[P]) ReplaceArg(find string, repl ...string) *WrapperBuilder[P] {
	prev := b.spec.Transform
	b.spec.Transform = func(c *Context, in []string) ([]string, error) {
		out := make([]string, 0, len(in)+len(repl))
		replaced := false
		for _, a := range in {
			if !replaced && a == find {
				out = append(out, repl...)
				replaced = true
			} else {
				out = append(out, a)
			}
		}
		if prev != nil {
			return prev(c, out)
		}
		return out, nil
	}
	return b
}

// TransformTool sets a function to modify the tool path and its arguments in
// dynamic mode (WrapDynamic). Only evaluated when Dynamic==true.
func (b *WrapperBuilder[P]) TransformTool(
	fn func(tool string, args []string) (string, []string, error),
) *WrapperBuilder[P] {
	b.spec.TransformToolFn = fn
	return b
}

// Passthrough streams child stdout/stderr to the app IO writers (default).
func (b *WrapperBuilder[P]) Passthrough() *WrapperBuilder[P] { b.spec.Mode = modePassthrough; return b }

// Capture captures child stdout/stderr into ExecResult and does not write to IO.
func (b *WrapperBuilder[P]) Capture() *WrapperBuilder[P] { b.spec.Mode = modeCapture; return b }

// TeeTo tees child output to the given writers (nil to ignore) in passthrough mode.
func (b *WrapperBuilder[P]) TeeTo(out, err io.Writer) *WrapperBuilder[P] {
	b.spec.TeeOut = out
	b.spec.TeeErr = err
	return b
}

// CaptureTo streams to app IO (like Passthrough) and also captures stdout/stderr
// into ExecResult (optionally teeing to extra writers).
func (b *WrapperBuilder[P]) CaptureTo(out, err io.Writer) *WrapperBuilder[P] {
	b.spec.Mode = modePassthrough
	b.spec.CaptureAlso = true
	b.spec.TeeOut = out
	b.spec.TeeErr = err
	return b
}

// AllowTools restricts dynamic wrapping (WrapDynamic) to the given tool base names.
// When set, the dynamic tool must match one of the allowed names (by filepath.Base).
func (b *WrapperBuilder[P]) AllowTools(names ...string) *WrapperBuilder[P] {
	// piggyback on TransformTool to enforce at runtime
	prev := b.spec.TransformToolFn
	b.spec.TransformToolFn = func(tool string, args []string) (string, []string, error) {
		base := filepath.Base(tool)
		ok := false
		for _, n := range names {
			if n == base {
				ok = true
				break
			}
		}
		if !ok {
			return tool, args, NewError(ErrorTypePermission, "tool not allowed: "+base)
		}
		if prev != nil {
			return prev(tool, args)
		}
		return tool, args, nil
	}
	return b
}

// HideFromHelp hides the wrapped command from help. Only valid for command-level wrappers.
func (b *WrapperBuilder[P]) HideFromHelp() *WrapperBuilder[P] {
	if b.cmd != nil {
		b.cmd.Hidden = true
	}
	return b
}

// Visible marks the wrapped command as visible in help.
func (b *WrapperBuilder[P]) Visible() *WrapperBuilder[P] {
	if b.cmd != nil {
		b.cmd.Hidden = false
	}
	return b
}

// Back returns to the parent fluent builder context
func (b *WrapperBuilder[P]) Back() P { return b.parent }

// LeadingFlags declares which child flags should be considered "leading" and
// kept before positional arguments. Useful for echo-like tools (e.g., -n, -e).
func (b *WrapperBuilder[P]) LeadingFlags(flags ...string) *WrapperBuilder[P] {
	b.spec.LeadingFlags = append(b.spec.LeadingFlags, flags...)
	return b
}

// InsertAfterLeadingFlags inserts tokens after the leading flags but before the
// remaining arguments.
func (b *WrapperBuilder[P]) InsertAfterLeadingFlags(tokens ...string) *WrapperBuilder[P] {
	b.spec.AfterLeading = append(b.spec.AfterLeading, tokens...)
	return b
}

// MapBoolFlag maps a wrapper boolean flag to child tokens (inserted among
// leading flags when the flag is set).
func (b *WrapperBuilder[P]) MapBoolFlag(wrapperFlag string, childTokens ...string) *WrapperBuilder[P] {
	if b.spec.MapBool == nil {
		b.spec.MapBool = make(map[string][]string)
	}
	b.spec.MapBool[wrapperFlag] = append([]string{}, childTokens...)
	return b
}

// run executes the wrapper with the given context and original args slice.
//
//nolint:gocognit,gocyclo,cyclop,funlen // Wrapper execution covers resolution, arg building, env, and IO wiring.
func (w *WrapperSpec) run(ctx *Context, _ []string) error {
	// Resolve binary
	bin := w.Binary
	if bin == "" && w.Dynamic {
		// Dynamic shim requires first positional arg as tool - sanity check
		if len(ctx.Args()) == 0 {
			return NewError(ErrorTypeInvalidValue, "missing tool for dynamic wrapper")
		}
		bin = ctx.Args()[0]
	}
	if bin == "" {
		return NewError(ErrorTypeInvalidValue, "missing wrapper binary")
	}
	if w.DiscoverOnPATH && !filepath.IsAbs(bin) {
		if p, err := exec.LookPath(bin); err == nil {
			bin = p
		}
	}

	// Build argv
	argv := make([]string, 0, len(w.PreArgs)+len(w.PostArgs)+len(ctx.Args())+8)
	pre := substituteTokens(w.PreArgs)
	forwarded := make([]string, 0, len(ctx.Args()))
	if w.ForwardArgs {
		// For dynamic: forward tool args (skip tool path)
		if w.Dynamic {
			if len(ctx.Args()) > 1 {
				forwarded = append(forwarded, ctx.Args()[1:]...)
			}
		} else {
			forwarded = append(forwarded, ctx.Args()...)
		}
	}
	// DSL reordering for leading flags and after-leading tokens
	if len(w.LeadingFlags) > 0 || len(w.AfterLeading) > 0 || len(w.MapBool) > 0 {
		leading, rest := splitLeading(forwarded, w.LeadingFlags)
		// mapped wrapper bool flags
		if len(w.MapBool) > 0 {
			for name, child := range w.MapBool {
				if v, ok := ctx.Bool(name); ok && v {
					leading = append(child, leading...)
				}
			}
		}
		forwarded = make([]string, 0, len(leading)+len(w.AfterLeading)+len(rest))
		forwarded = append(forwarded, leading...)
		forwarded = append(forwarded, substituteTokens(w.AfterLeading)...)
		forwarded = append(forwarded, rest...)
	}
	argv = append(argv, pre...)
	argv = append(argv, forwarded...)
	argv = append(argv, substituteTokens(w.PostArgs)...)
	// Dynamic tool transform (allows replacing tool path or its args)
	if w.Dynamic && w.TransformToolFn != nil {
		toolArgs := argv
		var err error
		bin, toolArgs, err = w.TransformToolFn(bin, toolArgs)
		if err != nil {
			return err
		}
		argv = toolArgs
	}
	if w.Transform != nil {
		var err error
		argv, err = w.Transform(ctx, argv)
		if err != nil {
			return err
		}
	}

	// Prepare command
	cmd := exec.CommandContext(ctx.Context(), bin, argv...)
	if w.WorkingDir != "" {
		cmd.Dir = w.WorkingDir
	}

	// Environment
	if w.InheritEnv {
		cmd.Env = append(cmd.Env, os.Environ()...)
	}
	if len(w.Env) > 0 {
		for k, v := range w.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	// IO wiring
	switch w.Mode {
	case modePassthrough:
		outW := ctx.Stdout()
		errW := ctx.Stderr()
		var outBuf, errBuf bytes.Buffer
		//nolint:nestif // IO wiring needs explicit nested branches to avoid subtle bugs.
		if w.CaptureAlso {
			// capture while streaming
			mwOut := []io.Writer{outW}
			if w.TeeOut != nil {
				mwOut = append(mwOut, w.TeeOut)
			}
			mwOut = append(mwOut, &outBuf)
			outW = io.MultiWriter(mwOut...)

			mwErr := []io.Writer{errW}
			if w.TeeErr != nil {
				mwErr = append(mwErr, w.TeeErr)
			}
			mwErr = append(mwErr, &errBuf)
			errW = io.MultiWriter(mwErr...)
		} else {
			if w.TeeOut != nil {
				outW = io.MultiWriter(outW, w.TeeOut)
			}
			if w.TeeErr != nil {
				errW = io.MultiWriter(errW, w.TeeErr)
			}
		}
		cmd.Stdout = outW
		cmd.Stderr = errW
		cmd.Stdin = ctx.Stdin()
		runErr := cmd.Run()
		if w.CaptureAlso {
			res := &ExecResult{Stdout: outBuf.Bytes(), Stderr: errBuf.Bytes(), Error: runErr}
			if ee := toExitError(runErr); ee != nil {
				res.ExitCode = ee.Code
				ctx.Set("__wrapper_result__", res)
				return ee
			}
			ctx.Set("__wrapper_result__", res)
		}
		if runErr != nil {
			return toExitError(runErr)
		}
		return nil
	case modeCapture:
		var outBuf, errBuf bytes.Buffer
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf
		cmd.Stdin = ctx.Stdin()
		err := cmd.Run()
		res := &ExecResult{Stdout: outBuf.Bytes(), Stderr: errBuf.Bytes(), Error: err}
		if ee := toExitError(err); ee != nil {
			// Attach exit code
			res.ExitCode = ee.Code
			// Expose via context metadata for PostHook usage if needed
			ctx.Set("__wrapper_result__", res)
			return ee
		}
		ctx.Set("__wrapper_result__", res)
		return nil
	default:
		return NewError(ErrorTypeInternal, "invalid wrapper mode")
	}
}

func toExitError(err error) *ExitError {
	if err == nil {
		return nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return &ExitError{Code: ee.ExitCode(), Err: err}
	}
	// Non-ExitError: treat as general failure with code 1
	return &ExitError{Code: 1, Err: err}
}

func substituteTokens(args []string) []string {
	if len(args) == 0 {
		return args
	}
	out := make([]string, 0, len(args))
	self, _ := os.Executable()
	for _, a := range args {
		if strings.Contains(a, "${SELF}") {
			a = strings.ReplaceAll(a, "${SELF}", self)
		}
		out = append(out, a)
	}
	return out
}

func splitLeading(args []string, leadingSet []string) ([]string, []string) {
	if len(leadingSet) == 0 {
		return nil, args
	}
	isLead := func(s string) bool {
		for _, f := range leadingSet {
			if s == f {
				return true
			}
		}
		return false
	}
	var leading []string
	var rest []string
	i := 0
	for ; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			i++
			break
		}
		if !isLead(a) {
			break
		}
		leading = append(leading, a)
	}
	rest = append(rest, args[i:]...)
	return leading, rest
}
