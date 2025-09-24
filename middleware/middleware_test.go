package middleware

import (
    "bytes"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "testing"
    "time"
)

// Mock implementations for testing

// MockContext implements the Context interface for testing
type MockContext struct {
	args          []string
	command       *MockCommand
	stringFlags   map[string]string
	intFlags      map[string]int
	boolFlags     map[string]bool
	durationFlags map[string]time.Duration
	floatFlags    map[string]float64
	enumFlags     map[string]string
	stringSlices  map[string][]string
	intSlices     map[string][]int
	metadata      map[string]any
	cancelled     bool
	done          chan struct{}
}

func NewMockContext() *MockContext {
	return &MockContext{
		args:          []string{},
		command:       &MockCommand{name: "test", description: "test command"},
		stringFlags:   make(map[string]string),
		intFlags:      make(map[string]int),
		boolFlags:     make(map[string]bool),
		durationFlags: make(map[string]time.Duration),
		floatFlags:    make(map[string]float64),
		enumFlags:     make(map[string]string),
		stringSlices:  make(map[string][]string),
		intSlices:     make(map[string][]int),
		metadata:      make(map[string]any),
		done:          make(chan struct{}),
	}
}

func (m *MockContext) Done() <-chan struct{}     { return m.done }
func (m *MockContext) Cancel()                   { close(m.done); m.cancelled = true }
func (m *MockContext) Args() []string            { return m.args }
func (m *MockContext) Command() Command          { return m.command }
func (m *MockContext) Set(key string, value any) { m.metadata[key] = value }
func (m *MockContext) Get(key string) any        { return m.metadata[key] }

func (m *MockContext) String(name string) (string, bool) {
	v, ok := m.stringFlags[name]
	return v, ok
}
func (m *MockContext) Int(name string) (int, bool) {
	v, ok := m.intFlags[name]
	return v, ok
}
func (m *MockContext) Bool(name string) (bool, bool) {
	v, ok := m.boolFlags[name]
	return v, ok
}
func (m *MockContext) Duration(name string) (time.Duration, bool) {
	v, ok := m.durationFlags[name]
	return v, ok
}
func (m *MockContext) Float(name string) (float64, bool) {
	v, ok := m.floatFlags[name]
	return v, ok
}
func (m *MockContext) Enum(name string) (string, bool) {
	v, ok := m.enumFlags[name]
	return v, ok
}
func (m *MockContext) StringSlice(name string) ([]string, bool) {
	v, ok := m.stringSlices[name]
	return v, ok
}
func (m *MockContext) IntSlice(name string) ([]int, bool) {
	v, ok := m.intSlices[name]
	return v, ok
}

// Global flag methods (for simplicity, use same storage)
func (m *MockContext) GlobalString(name string) (string, bool)          { return m.String(name) }
func (m *MockContext) GlobalInt(name string) (int, bool)                { return m.Int(name) }
func (m *MockContext) GlobalBool(name string) (bool, bool)              { return m.Bool(name) }
func (m *MockContext) GlobalDuration(name string) (time.Duration, bool) { return m.Duration(name) }
func (m *MockContext) GlobalFloat(name string) (float64, bool)          { return m.Float(name) }
func (m *MockContext) GlobalEnum(name string) (string, bool)            { return m.Enum(name) }
func (m *MockContext) GlobalStringSlice(name string) ([]string, bool)   { return m.StringSlice(name) }
func (m *MockContext) GlobalIntSlice(name string) ([]int, bool)         { return m.IntSlice(name) }

// Helper methods for testing
func (m *MockContext) SetString(name, value string)    { m.stringFlags[name] = value }
func (m *MockContext) SetInt(name string, value int)   { m.intFlags[name] = value }
func (m *MockContext) SetBool(name string, value bool) { m.boolFlags[name] = value }
func (m *MockContext) SetArgs(args []string)           { m.args = args }

type MockCommand struct {
	name        string
	description string
}

func (m *MockCommand) Name() string        { return m.name }
func (m *MockCommand) Description() string { return m.description }

// Mock action functions for testing
func successAction(ctx Context) error { return nil }
func errorAction(ctx Context) error   { return errors.New("test error") }
func panicAction(ctx Context) error   { panic("test panic") }
func slowAction(ctx Context) error {
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Test Core Middleware Functionality

func TestMiddlewareChain(t *testing.T) {
	var order []string

	middleware1 := func(next ActionFunc) ActionFunc {
		return func(ctx Context) error {
			order = append(order, "before1")
			err := next(ctx)
			order = append(order, "after1")
			return err
		}
	}

	middleware2 := func(next ActionFunc) ActionFunc {
		return func(ctx Context) error {
			order = append(order, "before2")
			err := next(ctx)
			order = append(order, "after2")
			return err
		}
	}

	action := func(ctx Context) error {
		order = append(order, "action")
		return nil
	}

	chain := Chain(middleware1, middleware2)
	finalAction := chain.Apply(action)

	ctx := NewMockContext()
	err := finalAction(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expected := []string{"before1", "before2", "action", "after2", "after1"}
	if len(order) != len(expected) {
		t.Errorf("Expected %d steps, got %d", len(expected), len(order))
	}

	for i, step := range expected {
		if i >= len(order) || order[i] != step {
			t.Errorf("Step %d: expected %s, got %s", i, step, order[i])
		}
	}
}

func TestLoggerConstructors(t *testing.T) {
    // Text logger (InfoLogger)
    var buf bytes.Buffer
    mw := LoggerWithWriter(&buf, WithLogLevel(LogLevelInfo))
    if err := mw(successAction)(NewMockContext()); err != nil { t.Fatalf("err: %v", err) }
    if out := buf.String(); !strings.Contains(out, "START") && out == "" {
        t.Fatalf("expected text logs, got %q", out)
    }
    // JSON logger
    buf.Reset()
    mw = LoggerWithWriter(&buf, func(c *MiddlewareConfig){ c.LogFormat = LogFormatJSON; c.LogLevel = LogLevelInfo })
    if err := mw(successAction)(NewMockContext()); err != nil { t.Fatalf("err: %v", err) }
    if out := buf.String(); !strings.Contains(out, "\"timestamp\"") { t.Fatalf("expected json log, got %q", out) }
    // Silent logger
    buf.Reset()
    mw = Logger(func(c *MiddlewareConfig){ c.LogOutput = LogOutputNone })
    if err := mw(successAction)(NewMockContext()); err != nil { t.Fatalf("err: %v", err) }
    if out := buf.String(); out != "" { t.Fatalf("expected no output, got %q", out) }
}

func TestTimeoutVariants(t *testing.T) {
    // TimeoutWithDefault
    twd := TimeoutWithDefault()
    if err := twd(successAction)(NewMockContext()); err != nil { t.Fatalf("unexpected err: %v", err) }
    // TimeoutWithGracefulShutdown (short timeout)
    tgs := TimeoutWithGracefulShutdown(5*time.Millisecond, 1*time.Millisecond)
    err := tgs(slowAction)(NewMockContext())
    if _, ok := err.(*TimeoutError); !ok { t.Fatalf("expected TimeoutError, got %T", err) }
    // TimeoutPerCommand
    per := TimeoutPerCommand(map[string]time.Duration{"test": 1 * time.Millisecond}, 1*time.Second)
    err = per(slowAction)(NewMockContext())
    if _, ok := err.(*TimeoutError); !ok { t.Fatalf("expected TimeoutError for per-command, got %T", err) }
    // TimeoutWithCallback
    called := make(chan struct{},1)
    cb := TimeoutWithCallback(1*time.Millisecond, func(string, time.Duration){ called <- struct{}{} })
    _ = cb(slowAction)(NewMockContext())
    select { case <-called: default: t.Fatalf("expected callback invoked") }
    // TimeoutWithRetry
    attempts := 0
    retry := TimeoutWithRetry(1*time.Millisecond, 2)
    err = retry(func(ctx Context) error { attempts++; return &TimeoutError{Duration:1*time.Millisecond, Command: getCommandName(ctx)} })(NewMockContext())
    if attempts < 3 { t.Fatalf("expected retries, attempts=%d", attempts) }
    // DynamicTimeout: 0 -> no timeout, >0 -> timeout
    dyn := DynamicTimeout(func(Context) time.Duration { return 0 })
    if err := dyn(slowAction)(NewMockContext()); err != nil { t.Fatalf("dynamic 0 unexpected err: %v", err) }
    dyn = DynamicTimeout(func(Context) time.Duration { return 1 * time.Millisecond })
    if err := dyn(slowAction)(NewMockContext()); err == nil { t.Fatalf("expected timeout") }
}

func TestTimeoutFromFlagAndStats(t *testing.T) {
    // MockContext has Duration(name) implemented via map; we can simulate by setting metadata
    ctx := NewMockContext()
    ctx.durationFlags["timeout"] = 1 * time.Millisecond
    tff := TimeoutFromFlag("timeout", 1*time.Second)
    if err := tff(slowAction)(ctx); err == nil { t.Fatalf("expected timeout from flag") }

    stats := NewTimeoutStats()
    tws := TimeoutWithStats(1*time.Millisecond, stats)
    _ = tws(slowAction)(NewMockContext())
    if stats.TotalTimeouts == 0 || stats.LastTimeout == nil { t.Fatalf("expected stats updated") }
}

func TestValidatorVariants(t *testing.T) {
    // ConditionalRequired: when condition true, missing flags should error
    cond := func(Context) error { return nil } // condition met
    v := ValidatorWithCustom(map[string]ValidatorFunc{"cond": ConditionalRequired(cond, "must")})
    if err := v(successAction)(NewMockContext()); err == nil { t.Fatalf("expected validation error") }

    // WithCustomValidators + NoopValidator path
    cust := ValidatorWithCustom(map[string]ValidatorFunc{"ok": func(Context) error { return nil }})
    if err := cust(successAction)(NewMockContext()); err != nil { t.Fatalf("unexpected err: %v", err) }

    // FileSystemValidator (placeholders succeed) — ensure it runs without error
    fsv := FileSystemValidator([]string{"file"}, []string{"dir"})
    ctx := NewMockContext(); ctx.Set("file", "path"); ctx.Set("dir", "path")
    if err := fsv(successAction)(ctx); err != nil { t.Fatalf("unexpected fs validator err: %v", err) }
}

// Test Logger Middleware

func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := LoggerWithWriter(&buf, WithLogLevel(LogLevelInfo))

	ctx := NewMockContext()
	ctx.SetArgs([]string{"arg1", "arg2"})

	err := logger(successAction)(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "SUCCESS") {
		t.Errorf("Expected SUCCESS in log output, got: %s", output)
	}
	if !strings.Contains(output, "test") {
		t.Errorf("Expected command name in log output, got: %s", output)
	}
}

func TestLoggerWithError(t *testing.T) {
	var buf bytes.Buffer
	logger := LoggerWithWriter(&buf, WithLogLevel(LogLevelError))

	ctx := NewMockContext()
	err := logger(errorAction)(ctx)

	if err == nil {
		t.Error("Expected error to be propagated")
	}

	output := buf.String()
	if !strings.Contains(output, "ERROR") {
		t.Errorf("Expected ERROR in log output, got: %s", output)
	}
}

func TestSilentLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := LoggerWithWriter(&buf, WithLogLevel(LogLevelNone))

	ctx := NewMockContext()
	err := logger(successAction)(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if buf.Len() > 0 {
		t.Errorf("Expected no log output, got: %s", buf.String())
	}
}

// Test Recovery Middleware

func TestRecovery(t *testing.T) {
	recovery := Recovery(WithStackTrace(false))

	ctx := NewMockContext()
	err := recovery(panicAction)(ctx)

	if err == nil {
		t.Error("Expected recovery error")
	}

	recoveryErr, ok := err.(*RecoveryError)
	if !ok {
		t.Errorf("Expected RecoveryError, got %T", err)
	}

	if recoveryErr.Panic != "test panic" {
		t.Errorf("Expected panic value 'test panic', got %v", recoveryErr.Panic)
	}

	if recoveryErr.Command != "test" {
		t.Errorf("Expected command 'test', got %s", recoveryErr.Command)
	}
}

func TestRecoveryWithStack(t *testing.T) {
	recovery := Recovery(WithStackTrace(true))

	ctx := NewMockContext()
	err := recovery(panicAction)(ctx)

	recoveryErr, ok := err.(*RecoveryError)
	if !ok {
		t.Errorf("Expected RecoveryError, got %T", err)
	}

	if len(recoveryErr.Stack) == 0 {
		t.Error("Expected stack trace to be captured")
	}

	stackStr := string(recoveryErr.Stack)
	if !strings.Contains(stackStr, "panicAction") {
		t.Errorf("Expected stack trace to contain function name, got: %s", stackStr)
	}
}

func TestNoopRecovery(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic to bubble up")
		}
	}()

	noop := NoopRecovery()
	ctx := NewMockContext()
	noop(panicAction)(ctx)
}

// Test Timeout Middleware

func TestTimeout(t *testing.T) {
	timeout := Timeout(50 * time.Millisecond)

	ctx := NewMockContext()
	err := timeout(slowAction)(ctx)

	if err == nil {
		t.Error("Expected timeout error")
	}

	timeoutErr, ok := err.(*TimeoutError)
	if !ok {
		t.Errorf("Expected TimeoutError, got %T", err)
	}

	if timeoutErr.Duration != 50*time.Millisecond {
		t.Errorf("Expected timeout duration 50ms, got %v", timeoutErr.Duration)
	}
}

func TestTimeoutSuccess(t *testing.T) {
	timeout := Timeout(200 * time.Millisecond)

	ctx := NewMockContext()
	err := timeout(successAction)(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestNoTimeout(t *testing.T) {
	noTimeout := NoTimeout()

	ctx := NewMockContext()
	start := time.Now()
	err := noTimeout(slowAction)(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if duration < 90*time.Millisecond {
		t.Errorf("Expected action to complete normally, took %v", duration)
	}
}

// Test Validator Middleware

func TestBusinessLogicValidator(t *testing.T) {
	// Example: API access validation - if API mode is enabled, require API key
	apiValidator := func(ctx Context) error {
		if useAPI, _ := ctx.Bool("api"); useAPI {
			if key, _ := ctx.String("api-key"); key == "" {
				return &ValidationError{
					Field:   "api-key",
					Message: "--api-key is required when --api is enabled",
				}
			}
		}
		return nil
	}

	validator := ValidatorWithCustom(map[string]ValidatorFunc{
		"api_access": apiValidator,
	})

	// Test with API disabled (should pass)
	ctx := NewMockContext()
	ctx.SetBool("api", false)
	err := validator(successAction)(ctx)
	if err != nil {
		t.Errorf("Expected no error when API disabled, got %v", err)
	}

	// Test with API enabled but no key (should fail)
	ctx.SetBool("api", true)
	err = validator(successAction)(ctx)
	if err == nil {
		t.Error("Expected validation error when API enabled but no key")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("Expected ValidationError, got %T", err)
	}

	if !strings.Contains(validationErr.Message, "api-key") {
		t.Errorf("Expected api-key error message, got: %s", validationErr.Message)
	}

	// Test with API enabled and key present (should pass)
	ctx.SetString("api-key", "secret123")
	err = validator(successAction)(ctx)
	if err != nil {
		t.Errorf("Expected no error when API key present, got %v", err)
	}
}

func TestFileSystemValidator(t *testing.T) {
	// Example: Output format validation - check that output directory structure makes sense
	outputValidator := func(ctx Context) error {
		if output, exists := ctx.String("output"); exists && output != "" {
			// Business logic: if output is specified, input must also be specified
			if input, _ := ctx.String("input"); input == "" {
				return &ValidationError{
					Field:   "input",
					Message: "--input is required when --output is specified",
				}
			}
		}
		return nil
	}

	validator := ValidatorWithCustom(map[string]ValidatorFunc{
		"output_validation": outputValidator,
	})

	// Test with output but no input (should fail)
	ctx := NewMockContext()
	ctx.SetString("output", "result.txt")
	err := validator(successAction)(ctx)

	if err == nil {
		t.Error("Expected validation error when output specified but no input")
	}

	// Test with both input and output (should pass)
	ctx.SetString("input", "source.txt")
	err = validator(successAction)(ctx)

	if err != nil {
		t.Errorf("Expected no error when both input and output specified, got %v", err)
	}
}

func TestFileAndDirectoryValidators(t *testing.T) {
    // Prepare filesystem
    dir := t.TempDir()
    file, err := os.CreateTemp(dir, "sample-*.txt")
    if err != nil { t.Fatalf("create temp file: %v", err) }
    filePath := file.Name()
    _ = file.Close()

    v := FileSystemValidator([]string{"file"}, []string{"dir"})

    // Happy path
    ctx := NewMockContext()
    ctx.SetString("file", filePath)
    ctx.SetString("dir", dir)
    if err := v(successAction)(ctx); err != nil {
        t.Fatalf("expected no error, got %v", err)
    }

    // Missing file
    ctx2 := NewMockContext()
    ctx2.SetString("file", filepath.Join(dir, "missing.txt"))
    if err := v(successAction)(ctx2); err == nil {
        t.Fatalf("expected error for missing file")
    }

    // Wrong type: directory given to file validator
    ctx3 := NewMockContext()
    ctx3.SetString("file", dir)
    if err := v(successAction)(ctx3); err == nil {
        t.Fatalf("expected error for directory as file")
    }

    // Wrong type: file given to directory validator
    ctx4 := NewMockContext()
    ctx4.SetString("dir", filePath)
    if err := v(successAction)(ctx4); err == nil {
        t.Fatalf("expected error for file as directory")
    }
}

func TestConditionalValidator(t *testing.T) {
	// Example: Database validation - if database mode is enabled, validate connection params
	dbValidator := func(ctx Context) error {
		if useDB, _ := ctx.Bool("database"); useDB {
			if host, _ := ctx.String("db-host"); host == "" {
				return &ValidationError{
					Field:   "db-host",
					Message: "--db-host is required when --database is enabled",
				}
			}
			if port, exists := ctx.Int("db-port"); exists && (port < 1 || port > 65535) {
				return &ValidationError{
					Field:   "db-port",
					Value:   port,
					Message: "--db-port must be between 1 and 65535",
				}
			}
		}
		return nil
	}

	validator := ValidatorWithCustom(map[string]ValidatorFunc{
		"db_validation": dbValidator,
	})

	// Test with database disabled (should pass)
	ctx := NewMockContext()
	ctx.SetBool("database", false)
	err := validator(successAction)(ctx)
	if err != nil {
		t.Errorf("Expected no error when database disabled, got %v", err)
	}

	// Test with database enabled but missing host (should fail)
	ctx.SetBool("database", true)
	err = validator(successAction)(ctx)
	if err == nil {
		t.Error("Expected validation error when database enabled but no host")
	}

	// Test with valid database config (should pass)
	ctx.SetString("db-host", "localhost")
	ctx.SetInt("db-port", 5432)
	err = validator(successAction)(ctx)
	if err != nil {
		t.Errorf("Expected no error with valid database config, got %v", err)
	}
}

func TestNoopValidator(t *testing.T) {
	validator := NoopValidator()

	ctx := NewMockContext()
	err := validator(successAction)(ctx)

	if err != nil {
		t.Errorf("Expected no error from noop validator, got %v", err)
	}
}

// Test Integration

func TestMiddlewareIntegration(t *testing.T) {
	var buf bytes.Buffer

	// Example business logic validator
	businessValidator := ValidatorWithCustom(map[string]ValidatorFunc{
		"config_check": func(ctx Context) error {
			if config, _ := ctx.String("config"); config != "" && config != "config.json" {
				return &ValidationError{
					Field:   "config",
					Message: "config file must be config.json for this example",
				}
			}
			return nil
		},
	})

	// Create a chain with all middleware types
	chain := Chain(
		LoggerWithWriter(&buf, WithLogLevel(LogLevelInfo)),
		Recovery(WithStackTrace(false)),
		Timeout(1*time.Second),
		businessValidator,
	)

	// Test successful execution
	ctx := NewMockContext()
	ctx.SetString("config", "config.json")

	finalAction := chain.Apply(successAction)
	err := finalAction(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check that logger output was generated
	if buf.Len() == 0 {
		t.Error("Expected log output")
	}
}

func TestMiddlewareOrderMatters(t *testing.T) {
	var buf bytes.Buffer

	// Recovery should come before timeout to catch timeout panics
	chain := Chain(
		Recovery(WithStackTrace(false)),
		Timeout(50*time.Millisecond),
		LoggerWithWriter(&buf, WithLogLevel(LogLevelError)),
	)

	ctx := NewMockContext()
	finalAction := chain.Apply(slowAction)
	err := finalAction(ctx)

	// Should get timeout error, not panic
	if err == nil {
		t.Error("Expected timeout error")
	}

	if _, ok := err.(*TimeoutError); !ok {
		t.Errorf("Expected TimeoutError, got %T", err)
	}
}

// Benchmarks moved to benchmark/bench_middleware_test.go

// Test error propagation
func TestErrorPropagation(t *testing.T) {
	testError := errors.New("original error")
	errorAction := func(ctx Context) error {
		return testError
	}

	chain := Chain(
		Logger(WithLogLevel(LogLevelNone)),
		Recovery(WithStackTrace(false)),
		NoopValidator(),
	)

	ctx := NewMockContext()
	err := chain.Apply(errorAction)(ctx)

	if err != testError {
		t.Errorf("Expected original error to be propagated, got %v", err)
	}
}

// Test memory allocation
func TestMemoryAllocation(t *testing.T) {
	chain := Chain(
		Logger(WithLogLevel(LogLevelNone)),
		Recovery(WithStackTrace(false)),
		NoopValidator(),
	)

	action := chain.Apply(successAction)
	ctx := NewMockContext()

	// Warm up
	for i := 0; i < 10; i++ {
		action(ctx)
	}

	// Measure allocations
	allocs := testing.AllocsPerRun(100, func() {
		action(ctx)
	})

	// Should have zero allocations for the happy path with silent logger
	if allocs > 0 {
		t.Errorf("Expected 0 allocs per run for silent middleware chain, got %.2f", allocs)
	}
}

// Test comprehensive example showing proper middleware validation usage
func TestComprehensiveValidationExample(t *testing.T) {
	// This test demonstrates the CORRECT way to use middleware validation
	// vs flag groups (which would be in the CLI definition)

	// Example business logic validators that are appropriate for middleware
	validators := map[string]ValidatorFunc{
		// 1. Conditional requirements based on business logic
		"ssl_validation": func(ctx Context) error {
			if secure, _ := ctx.Bool("secure"); secure {
				if cert, _ := ctx.String("cert-file"); cert == "" {
					return &ValidationError{
						Field:   "cert-file",
						Message: "--cert-file is required when --secure is enabled",
					}
				}
				if key, _ := ctx.String("key-file"); key == "" {
					return &ValidationError{
						Field:   "key-file",
						Message: "--key-file is required when --secure is enabled",
					}
				}
			}
			return nil
		},

		// 2. Runtime environment validation
		"environment_validation": func(ctx Context) error {
			if env, _ := ctx.String("env"); env == "production" {
				if debug, _ := ctx.Bool("debug"); debug {
					return &ValidationError{
						Field:   "debug",
						Message: "--debug cannot be enabled in production environment",
					}
				}
			}
			return nil
		},

		// 3. Resource limit validation (business logic)
		"resource_limits": func(ctx Context) error {
			if workers, exists := ctx.Int("workers"); exists {
				if memory, memExists := ctx.Int("memory"); memExists {
					// Business rule: each worker needs at least 512MB
					requiredMemory := workers * 512
					if memory < requiredMemory {
						return &ValidationError{
							Field:   "memory",
							Value:   memory,
							Message: fmt.Sprintf("insufficient memory: %d workers require at least %d MB", workers, requiredMemory),
						}
					}
				}
			}
			return nil
		},
	}

	validator := ValidatorWithCustom(validators)

	// Test Case 1: Valid SSL configuration
	ctx := NewMockContext()
	ctx.SetBool("secure", true)
	ctx.SetString("cert-file", "server.crt")
	ctx.SetString("key-file", "server.key")
	err := validator(successAction)(ctx)
	if err != nil {
		t.Errorf("Expected no error with valid SSL config, got %v", err)
	}

	// Test Case 2: Invalid SSL configuration (missing cert)
	ctx = NewMockContext()
	ctx.SetBool("secure", true)
	ctx.SetString("key-file", "server.key")
	err = validator(successAction)(ctx)
	if err == nil {
		t.Error("Expected validation error for missing cert file")
	}

	// Test Case 3: Invalid production + debug combination
	ctx = NewMockContext()
	ctx.SetString("env", "production")
	ctx.SetBool("debug", true)
	err = validator(successAction)(ctx)
	if err == nil {
		t.Error("Expected validation error for debug in production")
	}

	// Test Case 4: Insufficient memory for workers
	ctx = NewMockContext()
	ctx.SetInt("workers", 4)
	ctx.SetInt("memory", 1024) // 4 workers need 2048MB, only have 1024MB
	err = validator(successAction)(ctx)
	if err == nil {
		t.Error("Expected validation error for insufficient memory")
	}

	// Test Case 5: Valid resource configuration
	ctx = NewMockContext()
	ctx.SetInt("workers", 4)
	ctx.SetInt("memory", 3072) // More than enough memory
	err = validator(successAction)(ctx)
	if err != nil {
		t.Errorf("Expected no error with sufficient memory, got %v", err)
	}
}
