//nolint:testpackage // using package name 'snap' to access unexported fields for testing
package snap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// TestComprehensiveFlagTypes tests all implemented flag types with zero allocations
func TestComprehensiveFlagTypes(t *testing.T) {
	app := &App{
		flags: map[string]*Flag{
			// Core types
			"name":    {Name: "name", Type: FlagTypeString},
			"port":    {Name: "port", Type: FlagTypeInt},
			"verbose": {Name: "verbose", Type: FlagTypeBool},
			"timeout": {Name: "timeout", Type: FlagTypeDuration},
			"ratio":   {Name: "ratio", Type: FlagTypeFloat},

			// Collection types
			"tags":  {Name: "tags", Type: FlagTypeStringSlice},
			"ports": {Name: "ports", Type: FlagTypeIntSlice},

			// Global flag
			"debug": {Name: "debug", Type: FlagTypeBool, Global: true},
		},
		commands: make(map[string]*Command),
	}

	parser := NewParser(app)

	// Test comprehensive argument parsing with all types
	args := []string{
		"--debug",           // Global bool flag
		"--name", "go-snap", // String flag
		"--port", "0xFF", // Int flag (hex)
		"--verbose",          // Bool flag
		"--timeout", "1h30m", // Duration flag
		"--ratio", "3.14", // Float flag
		"--tags", "cli,parser,go", // String slice
		"--ports", "80,443,8080", // Int slice
	}

	result, err := parser.Parse(args)
	if err != nil {
		t.Fatalf("Failed to parse comprehensive args: %v", err)
	}

	// Verify all flag types using new method-based API
	if name, ok := result.GetString("name"); !ok || name != "go-snap" {
		t.Errorf("Expected name='go-snap', got %v", name)
	}

	if port, ok := result.GetInt("port"); !ok || port != 255 {
		t.Errorf("Expected port=255 (0xFF), got %v", port)
	}

	if verbose, ok := result.GetBool("verbose"); !ok || !verbose {
		t.Errorf("Expected verbose=true, got %v", verbose)
	}

	if timeout, ok := result.GetDuration("timeout"); !ok || timeout != 1*time.Hour+30*time.Minute {
		t.Errorf("Expected timeout=1h30m, got %v", timeout)
	}

	if ratio, ok := result.GetFloat("ratio"); !ok || ratio != 3.14 {
		t.Errorf("Expected ratio=3.14, got %v", ratio)
	}

	if tags, ok := result.GetStringSlice("tags"); !ok || len(tags) != 3 || tags[0] != "cli" || tags[1] != "parser" ||
		tags[2] != "go" {
		t.Errorf("Expected tags=[cli,parser,go], got %v", tags)
	}

	if ports, ok := result.GetIntSlice("ports"); !ok || len(ports) != 3 || ports[0] != 80 || ports[1] != 443 ||
		ports[2] != 8080 {
		t.Errorf("Expected ports=[80,443,8080], got %v", ports)
	}

	if debug, ok := result.GetGlobalBool("debug"); !ok || !debug {
		t.Errorf("Expected global debug=true, got %v", debug)
	}
}

// BenchmarkComprehensiveFlagTypes benchmarks all implemented flag types with zero allocations
// Parser benchmarks moved to benchmark/bench_parser_test.go

// TestEnumFlag tests enum flag functionality
func TestEnumFlag(t *testing.T) {
	app := &App{
		flags: map[string]*Flag{
			"level": {
				Name:        "level",
				Type:        FlagTypeEnum,
				EnumValues:  []string{"debug", "info", "warn", "error"},
				DefaultEnum: "info",
			},
		},
		commands: make(map[string]*Command),
	}

	parser := NewParser(app)

	// Test valid enum value
	result, err := parser.Parse([]string{"--level", "debug"})
	if err != nil {
		t.Fatalf("Failed to parse valid enum: %v", err)
	}

	if level, ok := result.GetEnum("level"); !ok || level != "debug" {
		t.Errorf("Expected level='debug', got %v", level)
	}

	// Test invalid enum value
	_, err = parser.Parse([]string{"--level", "invalid"})
	if err == nil {
		t.Fatal("Expected error for invalid enum value")
	}

	parseErr := &ParseError{}
	if errors.As(err, &parseErr) {
		if parseErr.Type != ErrorTypeInvalidValue {
			t.Errorf("Expected ErrorTypeInvalidValue, got %v", parseErr.Type)
		}
	} else {
		t.Errorf("Expected ParseError, got %T", err)
	}
}

// TestDualAPI tests both GetXXX and MustGetXXX patterns
func TestDualAPI(t *testing.T) {
	app := &App{
		flags: map[string]*Flag{
			"port": {Name: "port", Type: FlagTypeInt},    // No default value
			"host": {Name: "host", Type: FlagTypeString}, // No default value
		},
		commands: make(map[string]*Command),
	}

	parser := NewParser(app)
	result, err := parser.Parse([]string{"--port", "9000"}) // Only port provided
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Test GetXXX pattern (safe access) - explicit existence checking
	if port, exists := result.GetInt("port"); exists {
		if port != 9000 {
			t.Errorf("Expected port=9000, got %d", port)
		}
	} else {
		t.Error("Expected port to exist (was provided)")
	}

	if _, exists := result.GetString("host"); exists {
		t.Error("Expected host to not exist (not provided)")
	}

	// Test MustGetXXX pattern (convenience with defaults) - runtime fallbacks
	port := result.MustGetInt("port", 8080)           // Should return 9000 (provided)
	host := result.MustGetString("host", "localhost") // Should return "localhost" (fallback)

	if port != 9000 {
		t.Errorf("MustGetInt: expected 9000, got %d", port)
	}

	if host != "localhost" {
		t.Errorf("MustGetString: expected 'localhost' (fallback), got '%s'", host)
	}
}

// TestHasFlag tests the new HasFlag and HasGlobalFlag methods
func TestHasFlag(t *testing.T) {
	app := &App{
		flags: map[string]*Flag{
			"global": {Name: "global", Type: FlagTypeBool, Global: true, DefaultBool: true},
			"port":   {Name: "port", Type: FlagTypeInt},
			"host":   {Name: "host", Type: FlagTypeString, DefaultString: "localhost"},
		},
		commands: make(map[string]*Command),
	}

	parser := NewParser(app)
	result, err := parser.Parse([]string{"--port", "9000"}) // Only port provided
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Test HasFlag for provided flag
	if !result.HasFlag("port") {
		t.Error("HasFlag should return true for provided flag")
	}

	// Test HasFlag for flag with default value
	if !result.HasFlag("host") {
		t.Error("HasFlag should return true for flag with default value")
	}

	// Test HasFlag for non-existent flag
	if result.HasFlag("nonexistent") {
		t.Error("HasFlag should return false for non-existent flag")
	}

	// Test HasGlobalFlag for global flag with default
	if !result.HasGlobalFlag("global") {
		t.Error("HasGlobalFlag should return true for global flag with default")
	}

	// Test HasGlobalFlag for non-existent global flag
	if result.HasGlobalFlag("nonexistent") {
		t.Error("HasGlobalFlag should return false for non-existent global flag")
	}
}

// TestFluentAPI tests the fluent API builder pattern
func TestFluentAPI(t *testing.T) {
	// This test verifies that our fluent API compiles and chains correctly
	app := New("testapp", "Test application").
		Version("1.0.0").
		Author("Test Author", "test@example.com").
		StringFlag("name", "Your name").
		Default("Go User").Back().
		IntFlag("age", "Your age").
		Default(30).Back().
		BoolFlag("verbose", "Enable verbose output").
		Short('v').Back().
		DurationFlag("timeout", "Request timeout").Back().
		IntFlag("port", "Server port").
		Default(8080).Back()

	app.Command("serve", "Start server").
		StringFlag("host", "Server host").
		Default("localhost")

	// Just verify the app was created properly
	if app.name != "testapp" {
		t.Errorf("Expected app name 'testapp', got '%s'", app.name)
	}

	if app.version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", app.version)
	}
}

// TestFlagGroups tests the flag group functionality
func TestFlagGroups(t *testing.T) {
	// Test mutually exclusive group
	app := New("testapp", "Test app").
		FlagGroup("output").
		MutuallyExclusive().
		Description("Output format selection").
		BoolFlag("json", "Output as JSON").Back().
		BoolFlag("yaml", "Output as YAML").Back().
		BoolFlag("table", "Output as table").Back().
		EndGroup().
		StringFlag("config", "Config file").Back()

	// Verify app was built correctly
	if app.name != "testapp" {
		t.Errorf("Expected app name 'testapp', got '%s'", app.name)
	}

	// Verify flag groups were added
	if len(app.flagGroups) != 1 {
		t.Errorf("Expected 1 flag group, got %d", len(app.flagGroups))
	}

	group := app.flagGroups[0]
	if group.Name != "output" {
		t.Errorf("Expected group name 'output', got '%s'", group.Name)
	}

	if group.Constraint != GroupMutuallyExclusive {
		t.Errorf("Expected GroupMutuallyExclusive constraint, got %v", group.Constraint)
	}

	if len(group.Flags) != 3 {
		t.Errorf("Expected 3 flags in group, got %d", len(group.Flags))
	}

	// Verify flags were added to app.flags for parsing
	if _, exists := app.flags["json"]; !exists {
		t.Error("Flag 'json' should exist in app.flags")
	}
	if _, exists := app.flags["yaml"]; !exists {
		t.Error("Flag 'yaml' should exist in app.flags")
	}
	if _, exists := app.flags["table"]; !exists {
		t.Error("Flag 'table' should exist in app.flags")
	}
	if _, exists := app.flags["config"]; !exists {
		t.Error("Flag 'config' should exist in app.flags")
	}
}

// TestFlagGroupsRequiredGroup tests the required group constraint
func TestFlagGroupsRequiredGroup(t *testing.T) {
	app := New("testapp", "Test app").
		FlagGroup("auth").
		RequiredGroup().
		StringFlag("username", "Username").Back().
		StringFlag("token", "Auth token").Back().
		EndGroup()

	// Verify group constraint
	if len(app.flagGroups) != 1 {
		t.Errorf("Expected 1 flag group, got %d", len(app.flagGroups))
	}

	group := app.flagGroups[0]
	if group.Constraint != GroupRequiredGroup {
		t.Errorf("Expected GroupRequiredGroup constraint, got %v", group.Constraint)
	}
}

// TestFlagGroupsAllOrNone tests the all-or-none constraint
func TestFlagGroupsAllOrNone(t *testing.T) {
	app := New("testapp", "Test app").
		FlagGroup("ssl").
		AllOrNone().
		StringFlag("cert", "SSL certificate").Back().
		StringFlag("key", "SSL key").Back().
		EndGroup()

	// Verify group constraint
	group := app.flagGroups[0]
	if group.Constraint != GroupAllOrNone {
		t.Errorf("Expected GroupAllOrNone constraint, got %v", group.Constraint)
	}
}

// TestFlagGroupsFluentAPI tests the fluent API with complex nesting
func TestFlagGroupsFluentAPI(t *testing.T) {
	app := New("testapp", "Test app").
		// Global flags
		BoolFlag("verbose", "Verbose output").Global().Back().
		// First group
		FlagGroup("output").
		MutuallyExclusive().
		BoolFlag("json", "JSON output").Back().
		BoolFlag("yaml", "YAML output").Back().
		EndGroup().
		// Second group
		FlagGroup("auth").
		AllOrNone().
		StringFlag("user", "Username").Back().
		StringFlag("pass", "Password").Back().
		EndGroup().
		// Regular flag after groups
		StringFlag("config", "Config file").Back()

	// Verify structure
	if len(app.flagGroups) != 2 {
		t.Errorf("Expected 2 flag groups, got %d", len(app.flagGroups))
	}

	// Verify all flags exist
	expectedFlags := []string{"verbose", "json", "yaml", "user", "pass", "config"}
	for _, flagName := range expectedFlags {
		if _, exists := app.flags[flagName]; !exists {
			t.Errorf("Flag '%s' should exist in app.flags", flagName)
		}
	}
}

// TestFlagGroupValidation tests that flag group constraints are properly validated
func TestFlagGroupValidation(t *testing.T) {
	// Test mutually exclusive violation
	app := New("testapp", "Test app").
		FlagGroup("output").
		MutuallyExclusive().
		BoolFlag("json", "JSON output").Back().
		BoolFlag("yaml", "YAML output").Back().
		EndGroup()

	parser := NewParser(app)

	// This should fail - both json and yaml provided
	_, err := parser.Parse([]string{"--json", "--yaml"})
	if err == nil {
		t.Error("Expected error for mutually exclusive flags, got none")
	}

	parseErr := &ParseError{}
	if errors.As(err, &parseErr) {
		if parseErr.Type != ErrorTypeFlagGroupViolation {
			t.Errorf("Expected ErrorTypeFlagGroupViolation, got %v", parseErr.Type)
		}
	} else {
		t.Errorf("Expected ParseError, got %T", err)
	}

	// This should succeed - only one flag provided
	result, err := parser.Parse([]string{"--json"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if json, ok := result.GetBool("json"); !ok || !json {
		t.Errorf("Expected json=true, got %v", json)
	}
}

// TestSmartErrorHandling tests the smart error handling system
func TestSmartErrorHandling(t *testing.T) {
	app := New("testapp", "Test app").
		StringFlag("port", "Server port").Back().
		StringFlag("host", "Server host").Back()

	// Test flag suggestion for typo
	app.ErrorHandler().SuggestFlags(true).MaxDistance(2)

	parser := NewParser(app)
	_, err := parser.Parse([]string{"--prot", "8080"}) // Typo: should be --port

	if err == nil {
		t.Error("Expected error for unknown flag, got none")
	}

	parseErr := &ParseError{}
	if errors.As(err, &parseErr) {
		if parseErr.Type != ErrorTypeUnknownFlag {
			t.Errorf("Expected ErrorTypeUnknownFlag, got %v", parseErr.Type)
		}
	} else {
		t.Errorf("Expected ParseError, got %T", err)
	}
}

// TestFlagGroupRequiredGroup tests required group validation
func TestFlagGroupRequiredGroup(t *testing.T) {
	app := New("testapp", "Test app").
		FlagGroup("auth").
		RequiredGroup().
		StringFlag("username", "Username").Back().
		StringFlag("token", "Auth token").Back().
		EndGroup()

	parser := NewParser(app)

	// This should fail - no auth flags provided
	_, err := parser.Parse([]string{})
	if err == nil {
		t.Error("Expected error for missing required group flags, got none")
	}

	// This should succeed - username provided
	result, err := parser.Parse([]string{"--username", "testuser"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if username, ok := result.GetString("username"); !ok || username != "testuser" {
		t.Errorf("Expected username=testuser, got %v", username)
	}
}

// BenchmarkFlagGroupParsing benchmarks parsing with flag groups (should maintain zero allocations)
// Parser benchmarks moved to benchmark/bench_parser_test.go

// ExampleParser demonstrates basic parser usage
func ExampleParser() {
	app := &App{
		flags: map[string]*Flag{
			"port": {
				Name:       "port",
				Type:       FlagTypeInt,
				DefaultInt: 8080,
			},
		},
		commands: make(map[string]*Command),
	}

	parser := NewParser(app)
	result, err := parser.Parse([]string{"--port", "9000"})

	if err != nil {
		panic(err)
	}

	port, _ := result.GetInt("port")
	fmt.Printf("Port: %d\n", port)
	// Output: Port: 9000
}

// TestFromEnv tests the FromEnv functionality for various flag types
func TestFromEnv(t *testing.T) {
	// Set up environment variables for testing
	envVars := map[string]string{
		"TEST_HOST":    "example.com",
		"TEST_PORT":    "9000",
		"TEST_VERBOSE": "true",
		"TEST_TIMEOUT": "30s",
		"TEST_RATIO":   "3.14",
		"TEST_LEVEL":   "debug",
	}

	// Set environment variables
	for key, value := range envVars {
		t.Setenv(key, value)
	}

	app := New("testapp", "Test app").
		StringFlag("host", "Server host").
		Default("localhost").
		FromEnv("TEST_HOST").Back().
		IntFlag("port", "Server port").
		Default(8080).
		FromEnv("TEST_PORT").Back().
		BoolFlag("verbose", "Verbose mode").
		FromEnv("TEST_VERBOSE").Back().
		DurationFlag("timeout", "Request timeout").
		Default(time.Second).
		FromEnv("TEST_TIMEOUT").Back().
		FloatFlag("ratio", "Some ratio").
		Default(1.0).
		FromEnv("TEST_RATIO").Back().
		EnumFlag("level", "Log level", "debug", "info", "warn", "error").
		Default("info").
		FromEnv("TEST_LEVEL").Back()

	parser := NewParser(app)
	result, err := parser.Parse([]string{}) // No command line args, should use env vars

	if err != nil {
		t.Fatalf("Failed to parse with environment variables: %v", err)
	}

	// Test that environment variables were used
	if host, ok := result.GetString("host"); !ok || host != "example.com" {
		t.Errorf("Expected host='example.com' from env, got %v", host)
	}

	if port, ok := result.GetInt("port"); !ok || port != 9000 {
		t.Errorf("Expected port=9000 from env, got %v", port)
	}

	if verbose, ok := result.GetBool("verbose"); !ok || !verbose {
		t.Errorf("Expected verbose=true from env, got %v", verbose)
	}

	if timeout, ok := result.GetDuration("timeout"); !ok || timeout != 30*time.Second {
		t.Errorf("Expected timeout=30s from env, got %v", timeout)
	}

	if ratio, ok := result.GetFloat("ratio"); !ok || ratio != 3.14 {
		t.Errorf("Expected ratio=3.14 from env, got %v", ratio)
	}

	if level, ok := result.GetEnum("level"); !ok || level != "debug" {
		t.Errorf("Expected level='debug' from env, got %v", level)
	}
}

// Exit code manager: minimal coverage to ensure default mappings work
func TestExitCodes_Minimal(t *testing.T) {
	app := New("t", "")
	// CLI validation -> 3
	if code := app.ExitCodes().resolve(NewError(ErrorTypeValidation, "")); code != app.ExitCodes().defaults.ValidationError {
		t.Fatalf("expected validation=%d got %d", app.ExitCodes().defaults.ValidationError, code)
	}
	// Unknown flag -> misusage
	if code := app.ExitCodes().resolve(NewError(ErrorTypeUnknownFlag, "")); code != app.ExitCodes().defaults.MisusageError {
		t.Fatalf("expected misusage=%d got %d", app.ExitCodes().defaults.MisusageError, code)
	}
}

// captureStderr captures stderr output during fn
func captureStderr(fn func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	//nolint:reassign // intentionally redirect os.Stderr in tests to capture output
	os.Stderr = w
	fn()
	w.Close()
	//nolint:reassign // intentionally redirect os.Stderr in tests to capture output
	os.Stderr = old
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, _ := r.Read(buf)
		if n <= 0 {
			break
		}
		sb.Write(buf[:n])
		if n < len(buf) {
			break
		}
	}
	return sb.String()
}

// Error display should include group help context for flag group violations
func TestErrorDisplay_GroupViolation_ShowsGroupHelp(t *testing.T) {
	app := New("x", "")
	// Create a group with exactly one constraint
	g := app.FlagGroup("output").ExactlyOne()
	g.BoolFlag("json", "").Back()
	g.BoolFlag("yaml", "").Back()
	g.EndGroup()
	// Parse with both flags -> violation
	p := NewParser(app)
	out := captureStderr(func() {
		_, err := p.Parse([]string{"--json", "--yaml"})
		if err != nil {
			// send through app's handler to display group context
			pe := &ParseError{}
			if errors.As(err, &pe) {
				_ = app.handleParseError(pe)
			} else {
				t.Fatalf("unexpected error type: %T", err)
			}
		} else {
			t.Fatalf("expected error")
		}
	})
	if !strings.Contains(out, "Flag group 'output'") || !strings.Contains(out, "Constraint:") {
		t.Fatalf("expected group help in stderr, got: %s", out)
	}
}

// Help/version flags should be honored at top-level and subcommand contexts
func TestHelpAndVersionAcrossContexts(t *testing.T) {
	app := New("tool", "desc").Version("1.0.0")
	sub := app.Command("serve", "serves").Build()
	// top-level --help
	if err := app.RunWithArgs(context.Background(), []string{"--help"}); !errors.Is(err, ErrHelpShown) {
		t.Fatalf("expected ErrHelpShown, got %v", err)
	}
	// subcommand --help
	p := NewParser(app)
	res, _ := p.Parse([]string{"serve", "--help"})
	app.currentResult = res
	if err := app.RunWithArgs(context.Background(), []string{"serve", "--help"}); !errors.Is(err, ErrHelpShown) {
		t.Fatalf("expected ErrHelpShown for subcommand, got %v", err)
	}
	_ = sub // silence
}

// Subcommand suggestion should prefer current command's children
func TestSubcommandSuggestionPrefersChild(t *testing.T) {
	app := New("t", "")
	srv := app.Command("serve", "").Build()
	srv.Command("status", "").Build()
	srv.Command("start", "").Build()

	parser := NewParser(app)
	_, err := parser.Parse([]string{"serve", "sttus"})
	if err == nil {
		t.Fatalf("expected error")
	}
	pe := &ParseError{}
	if errors.As(err, &pe) {
		if pe.Suggestion != "status" {
			t.Fatalf("expected suggestion 'status', got %q", pe.Suggestion)
		}
	} else {
		t.Fatalf("unexpected error type %T", err)
	}
}

// Int parsing should accept MaxInt64 and overflow on +1 (on 64-bit)
func TestIntParsing64BitBoundaries(t *testing.T) {
	if ^uint(0)>>63 == 0 {
		t.Skip("32-bit platform")
	}
	app := New("t", "")
	app.IntFlag("n", "").Back()
	p := NewParser(app)
	// MaxInt64
	if _, err := p.Parse([]string{"--n", "9223372036854775807"}); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	// overflow
	if _, err := p.Parse([]string{"--n", "9223372036854775808"}); err == nil {
		t.Fatalf("expected overflow error")
	}
}

// Duration env should accept extended formats like 1d
func TestDurationEnvExtended(t *testing.T) {
	app := New("t", "")
	app.DurationFlag("timeout", "").FromEnv("TIMEOUT").Back()
	t.Setenv("TIMEOUT", "1d")
	p := NewParser(app)
	if _, err := p.Parse([]string{}); err != nil {
		t.Fatalf("parse: %v", err)
	}
}

// Slice defaults should apply when not provided
func TestSliceDefaultsApply(t *testing.T) {
	app := New("t", "")
	app.StringSliceFlag("names", "").Default([]string{"a", "b"}).Back()
	app.IntSliceFlag("ports", "").Default([]int{1, 2}).Back()
	p := NewParser(app)
	res, err := p.Parse([]string{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if v := res.MustGetStringSlice("names", nil); len(v) != 2 || v[0] != "a" || v[1] != "b" {
		t.Fatalf("names default missing: %#v", v)
	}
	if v := res.MustGetIntSlice("ports", nil); len(v) != 2 || v[0] != 1 || v[1] != 2 {
		t.Fatalf("ports default missing: %#v", v)
	}
}

// Enum flags should be generated and collected in config precedence
func TestConfig_EnumAndSlices_Collected(t *testing.T) {
	type C struct {
		Names []string `flag:"names"`
		Ports []int    `flag:"ports"`
		Mode  string   `flag:"mode"  enum:"red,blue,green"`
	}
	var cfg C
	app, err := Config("app", "").FromFlags().Bind(&cfg).Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	parser := NewParser(app)
	res, err := parser.Parse([]string{"--names", "a,b", "--ports", "10,20", "--mode", "green"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	app.currentResult = res
	if pErr := app.populateConfiguration(); pErr != nil {
		t.Fatalf("populate: %v", pErr)
	}
	if cfg.Mode != "green" || len(cfg.Names) != 2 || cfg.Names[0] != "a" || cfg.Ports[0] != 10 {
		t.Fatalf("bad config: %#v", cfg)
	}
}

// IO integration: writing via ctx.Stdout goes to configured writer
func TestIO_Integration_Write(t *testing.T) {
	var buf strings.Builder
	app := New("io", "")
	app.IO().WithOut(&buf)
	app.Command("hello", " ").Action(func(c *Context) error {
		_, _ = c.Stdout().Write([]byte("hi"))
		return nil
	}).Build()
	parser := NewParser(app)
	res, _ := parser.Parse([]string{"hello"})
	app.currentResult = res
	_ = app.RunWithArgs(context.Background(), []string{"hello"})
	if buf.String() != "hi" {
		t.Fatalf("expected 'hi', got %q", buf.String())
	}
}

// TestFromEnvMultipleVars tests environment variable precedence with multiple variables
func TestFromEnvMultipleVars(t *testing.T) {
	// Set up multiple environment variables
	t.Setenv("PRIMARY_HOST", "primary.example.com")
	t.Setenv("SECONDARY_HOST", "secondary.example.com")
	t.Setenv("FALLBACK_HOST", "fallback.example.com")

	app := New("testapp", "Test app").
		StringFlag("host", "Server host").
		Default("localhost").
		FromEnv("MISSING_HOST", "PRIMARY_HOST", "SECONDARY_HOST", "FALLBACK_HOST").
		Back()

	parser := NewParser(app)
	result, err := parser.Parse([]string{})

	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Should use the first available env var (PRIMARY_HOST)
	if host, ok := result.GetString("host"); !ok || host != "primary.example.com" {
		t.Errorf("Expected host='primary.example.com' (first available), got %v", host)
	}
}

// TestFromEnvCommandLinePrecedence tests that command line flags override environment variables
func TestFromEnvCommandLinePrecedence(t *testing.T) {
	t.Setenv("TEST_PORT", "9000")

	app := New("testapp", "Test app").
		IntFlag("port", "Server port").
		Default(8080).
		FromEnv("TEST_PORT").
		Back()

	parser := NewParser(app)
	result, err := parser.Parse([]string{"--port", "3000"}) // Command line should override

	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Command line value should override environment variable
	if port, ok := result.GetInt("port"); !ok || port != 3000 {
		t.Errorf("Expected port=3000 (command line override), got %v", port)
	}
}

// TestFromEnvFallbackToDefault tests fallback to default values when env vars are not set
func TestFromEnvFallbackToDefault(t *testing.T) {
	app := New("testapp", "Test app").
		StringFlag("host", "Server host").
		Default("localhost").
		FromEnv("MISSING_HOST").
		Back().
		IntFlag("port", "Server port").
		Default(8080).
		FromEnv("MISSING_PORT").
		Back()

	parser := NewParser(app)
	result, err := parser.Parse([]string{})

	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Should fall back to default values
	if host, ok := result.GetString("host"); !ok || host != "localhost" {
		t.Errorf("Expected host='localhost' (default fallback), got %v", host)
	}

	if port, ok := result.GetInt("port"); !ok || port != 8080 {
		t.Errorf("Expected port=8080 (default fallback), got %v", port)
	}
}

// TestFromEnvGlobalFlags tests environment variables with global flags
func TestFromEnvGlobalFlags(t *testing.T) {
	t.Setenv("GLOBAL_DEBUG", "true")
	t.Setenv("CMD_PORT", "9000")

	app := New("testapp", "Test app").
		BoolFlag("debug", "Debug mode").
		Global().
		FromEnv("GLOBAL_DEBUG").
		Back()

	app.Command("serve", "Start server").
		IntFlag("port", "Server port").
		Default(8080).
		FromEnv("CMD_PORT").
		Back()

	parser := NewParser(app)
	result, err := parser.Parse([]string{"serve"})

	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Test global flag from environment
	if debug, ok := result.GetGlobalBool("debug"); !ok || !debug {
		t.Errorf("Expected global debug=true from env, got %v", debug)
	}

	// Test command flag from environment
	if port, ok := result.GetInt("port"); !ok || port != 9000 {
		t.Errorf("Expected port=9000 from env, got %v", port)
	}
}

// TestCommandBeforeAfterHooks tests command-level Before/After hooks
func TestCommandBeforeAfterHooks(t *testing.T) {
	var executionOrder []string

	app := New("test", "Test app")
	app.Before(func(ctx *Context) error {
		executionOrder = append(executionOrder, "app-before")
		return nil
	})
	app.After(func(ctx *Context) error {
		executionOrder = append(executionOrder, "app-after")
		return nil
	})

	app.Command("serve", "Start server").
		Before(func(ctx *Context) error {
			executionOrder = append(executionOrder, "command-before")
			return nil
		}).
		Action(func(ctx *Context) error {
			executionOrder = append(executionOrder, "action")
			return nil
		}).
		After(func(ctx *Context) error {
			executionOrder = append(executionOrder, "command-after")
			return nil
		})

	err := app.RunWithArgs(context.Background(), []string{"serve"})
	if err != nil {
		t.Fatalf("RunWithArgs failed: %v", err)
	}

	expected := []string{"app-before", "command-before", "action", "command-after", "app-after"}
	if len(executionOrder) != len(expected) {
		t.Fatalf("Expected %d execution steps, got %d: %v", len(expected), len(executionOrder), executionOrder)
	}

	for i, step := range expected {
		if executionOrder[i] != step {
			t.Errorf("Step %d: expected %q, got %q", i, step, executionOrder[i])
		}
	}
}

// TestContextAppMetadata tests app metadata accessors in Context
func TestContextAppMetadata(t *testing.T) {
	app := New("myapp", "My application").
		Version("1.2.3").
		Author("Alice", "alice@example.com").
		Author("Bob", "bob@example.com")

	var capturedName string
	var capturedVersion string
	var capturedDescription string
	var capturedAuthors []Author

	app.Command("test", "Test command").
		Action(func(ctx *Context) error {
			capturedName = ctx.AppName()
			capturedVersion = ctx.AppVersion()
			capturedDescription = ctx.AppDescription()
			capturedAuthors = ctx.AppAuthors()
			return nil
		})

	err := app.RunWithArgs(context.Background(), []string{"test"})
	if err != nil {
		t.Fatalf("RunWithArgs failed: %v", err)
	}

	if capturedName != "myapp" {
		t.Errorf("Expected app name 'myapp', got %q", capturedName)
	}

	if capturedVersion != "1.2.3" {
		t.Errorf("Expected version '1.2.3', got %q", capturedVersion)
	}

	if capturedDescription != "My application" {
		t.Errorf("Expected description 'My application', got %q", capturedDescription)
	}

	if len(capturedAuthors) != 2 {
		t.Fatalf("Expected 2 authors, got %d", len(capturedAuthors))
	}

	if capturedAuthors[0].Name != "Alice" || capturedAuthors[0].Email != "alice@example.com" {
		t.Errorf("Expected first author Alice <alice@example.com>, got %s <%s>",
			capturedAuthors[0].Name, capturedAuthors[0].Email)
	}

	if capturedAuthors[1].Name != "Bob" || capturedAuthors[1].Email != "bob@example.com" {
		t.Errorf("Expected second author Bob <bob@example.com>, got %s <%s>",
			capturedAuthors[1].Name, capturedAuthors[1].Email)
	}
}

// TestCommandBeforeError tests that Before hook errors stop execution
func TestCommandBeforeError(t *testing.T) {
	var executed bool

	app := New("test", "Test app")
	app.Command("serve", "Start server").
		Before(func(ctx *Context) error {
			return errors.New("before error")
		}).
		Action(func(ctx *Context) error {
			executed = true
			return nil
		})

	err := app.RunWithArgs(context.Background(), []string{"serve"})
	if err == nil {
		t.Fatal("Expected error from Before hook")
	}

	if err.Error() != "before error" {
		t.Errorf("Expected 'before error', got %q", err.Error())
	}

	if executed {
		t.Error("Action should not have been executed after Before error")
	}
}

// TestCommandAfterError tests that After hook errors are returned
func TestCommandAfterError(t *testing.T) {
	var actionExecuted bool

	app := New("test", "Test app")
	app.Command("serve", "Start server").
		Action(func(ctx *Context) error {
			actionExecuted = true
			return nil
		}).
		After(func(ctx *Context) error {
			return errors.New("after error")
		})

	err := app.RunWithArgs(context.Background(), []string{"serve"})
	if err == nil {
		t.Fatal("Expected error from After hook")
	}

	if err.Error() != "after error" {
		t.Errorf("Expected 'after error', got %q", err.Error())
	}

	if !actionExecuted {
		t.Error("Action should have been executed before After hook")
	}
}

// TestCommandAfterWithActionError tests that After runs even if Action fails
func TestCommandAfterWithActionError(t *testing.T) {
	var afterExecuted bool

	app := New("test", "Test app")
	app.Command("serve", "Start server").
		Action(func(ctx *Context) error {
			return errors.New("action error")
		}).
		After(func(ctx *Context) error {
			afterExecuted = true
			return nil
		})

	err := app.RunWithArgs(context.Background(), []string{"serve"})
	if err == nil {
		t.Fatal("Expected error from Action")
	}

	if err.Error() != "action error" {
		t.Errorf("Expected 'action error', got %q", err.Error())
	}

	if !afterExecuted {
		t.Error("After hook should have been executed even after Action error")
	}
}

// TestNestedCommandBeforeAfter tests Before/After with nested subcommands
func TestNestedCommandBeforeAfter(t *testing.T) {
	var executionOrder []string

	app := New("test", "Test app")

	server := app.Command("server", "Server management").
		Before(func(ctx *Context) error {
			executionOrder = append(executionOrder, "server-before")
			return nil
		}).
		After(func(ctx *Context) error {
			executionOrder = append(executionOrder, "server-after")
			return nil
		})

	server.Command("start", "Start server").
		Before(func(ctx *Context) error {
			executionOrder = append(executionOrder, "start-before")
			return nil
		}).
		Action(func(ctx *Context) error {
			executionOrder = append(executionOrder, "start-action")
			return nil
		}).
		After(func(ctx *Context) error {
			executionOrder = append(executionOrder, "start-after")
			return nil
		})

	err := app.RunWithArgs(context.Background(), []string{"server", "start"})
	if err != nil {
		t.Fatalf("RunWithArgs failed: %v", err)
	}

	// Note: Only the deepest command's Before/After hooks are called
	expected := []string{"start-before", "start-action", "start-after"}
	if len(executionOrder) != len(expected) {
		t.Fatalf("Expected %d execution steps, got %d: %v", len(expected), len(executionOrder), executionOrder)
	}

	for i, step := range expected {
		if executionOrder[i] != step {
			t.Errorf("Step %d: expected %q, got %q", i, step, executionOrder[i])
		}
	}
}
