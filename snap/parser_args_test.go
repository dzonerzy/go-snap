package snap

import (
	"testing"
	"time"
)

// TestPositionalArgsBasic tests basic positional argument parsing
func TestPositionalArgsBasic(t *testing.T) {
	app := New("test", "Test application")
	app.StringArg("name", "Name argument").Required()
	app.IntArg("age", "Age argument").Default(25)

	parser := NewParser(app)
	result, err := parser.Parse([]string{"John", "30"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Test string arg
	name, exists := result.GetArgString("name")
	if !exists {
		t.Error("Expected 'name' argument to exist")
	}
	if name != "John" {
		t.Errorf("Expected name='John', got '%s'", name)
	}

	// Test int arg
	age, exists := result.GetArgInt("age")
	if !exists {
		t.Error("Expected 'age' argument to exist")
	}
	if age != 30 {
		t.Errorf("Expected age=30, got %d", age)
	}

	// Test raw arg access
	if result.Args[0] != "John" {
		t.Errorf("Expected Args[0]='John', got '%s'", result.Args[0])
	}
	if result.Args[1] != "30" {
		t.Errorf("Expected Args[1]='30', got '%s'", result.Args[1])
	}
}

// TestPositionalArgsDefaults tests default values for optional args
func TestPositionalArgsDefaults(t *testing.T) {
	app := New("test", "Test application")
	app.StringArg("name", "Name argument").Required()
	app.IntArg("age", "Age argument").Default(25)
	app.BoolArg("active", "Active flag").Default(true)

	parser := NewParser(app)
	result, err := parser.Parse([]string{"John"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Required arg should exist
	name, exists := result.GetArgString("name")
	if !exists || name != "John" {
		t.Errorf("Expected name='John', got '%s' (exists=%v)", name, exists)
	}

	// Optional arg with default
	age, exists := result.GetArgInt("age")
	if !exists || age != 25 {
		t.Errorf("Expected age=25 (default), got %d (exists=%v)", age, exists)
	}

	// Optional bool with default
	active, exists := result.GetArgBool("active")
	if !exists || !active {
		t.Errorf("Expected active=true (default), got %v (exists=%v)", active, exists)
	}
}

// TestPositionalArgsMissingRequired tests error when required arg is missing
func TestPositionalArgsMissingRequired(t *testing.T) {
	app := New("test", "Test application")
	app.StringArg("name", "Name argument").Required()
	app.IntArg("age", "Age argument").Required()

	parser := NewParser(app)
	_, err := parser.Parse([]string{"John"})
	if err == nil {
		t.Fatal("Expected error for missing required argument")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("Expected *ParseError, got %T", err)
	}
	if parseErr.Type != ErrorTypeInvalidArgument {
		t.Errorf("Expected ErrorTypeInvalidArgument, got %s", parseErr.Type)
	}
}

// TestPositionalArgsAllTypes tests all argument types
func TestPositionalArgsAllTypes(t *testing.T) {
	app := New("test", "Test application")
	app.StringArg("str", "String arg").Required()
	app.IntArg("num", "Int arg").Required()
	app.BoolArg("flag", "Bool arg").Required()
	app.FloatArg("price", "Float arg").Required()
	app.DurationArg("timeout", "Duration arg").Required()

	parser := NewParser(app)
	result, err := parser.Parse([]string{"hello", "42", "true", "3.14", "5s"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if str, _ := result.GetArgString("str"); str != "hello" {
		t.Errorf("Expected str='hello', got '%s'", str)
	}
	if num, _ := result.GetArgInt("num"); num != 42 {
		t.Errorf("Expected num=42, got %d", num)
	}
	if flag, _ := result.GetArgBool("flag"); !flag {
		t.Errorf("Expected flag=true, got %v", flag)
	}
	if price, _ := result.GetArgFloat("price"); price != 3.14 {
		t.Errorf("Expected price=3.14, got %f", price)
	}
	if timeout, _ := result.GetArgDuration("timeout"); timeout != 5*time.Second {
		t.Errorf("Expected timeout=5s, got %v", timeout)
	}
}

// TestPositionalArgsVariadicString tests variadic string slice arguments
func TestPositionalArgsVariadicString(t *testing.T) {
	app := New("test", "Test application")
	app.StringArg("cmd", "Command").Required()
	app.StringSliceArg("files", "Files to process").Variadic()

	parser := NewParser(app)
	result, err := parser.Parse([]string{"rm", "file1.txt", "file2.txt", "file3.txt"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	cmd, _ := result.GetArgString("cmd")
	if cmd != "rm" {
		t.Errorf("Expected cmd='rm', got '%s'", cmd)
	}

	files, exists := result.GetArgStringSlice("files")
	if !exists {
		t.Fatal("Expected 'files' variadic arg to exist")
	}
	if len(files) != 3 {
		t.Fatalf("Expected 3 files, got %d", len(files))
	}
	expected := []string{"file1.txt", "file2.txt", "file3.txt"}
	for i, file := range files {
		if file != expected[i] {
			t.Errorf("Expected files[%d]='%s', got '%s'", i, expected[i], file)
		}
	}
}

// TestPositionalArgsVariadicInt tests variadic int slice arguments
func TestPositionalArgsVariadicInt(t *testing.T) {
	app := New("test", "Test application")
	app.StringArg("cmd", "Command").Required()
	app.IntSliceArg("numbers", "Numbers to sum").Variadic()

	parser := NewParser(app)
	result, err := parser.Parse([]string{"sum", "1", "2", "3", "4", "5"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	numbers, exists := result.GetArgIntSlice("numbers")
	if !exists {
		t.Fatal("Expected 'numbers' variadic arg to exist")
	}
	if len(numbers) != 5 {
		t.Fatalf("Expected 5 numbers, got %d", len(numbers))
	}

	sum := 0
	for _, num := range numbers {
		sum += num
	}
	if sum != 15 {
		t.Errorf("Expected sum=15, got %d", sum)
	}
}

// TestPositionalArgsRestArgs tests RestArgs functionality
func TestPositionalArgsRestArgs(t *testing.T) {
	app := New("test", "Test application")
	app.RestArgs()

	parser := NewParser(app)
	result, err := parser.Parse([]string{"docker", "run", "-it", "ubuntu", "bash"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(result.RestArgs) != 5 {
		t.Fatalf("Expected 5 rest args, got %d", len(result.RestArgs))
	}

	expected := []string{"docker", "run", "-it", "ubuntu", "bash"}
	for i, arg := range result.RestArgs {
		if arg != expected[i] {
			t.Errorf("Expected RestArgs[%d]='%s', got '%s'", i, expected[i], arg)
		}
	}
}

// TestPositionalArgsWithFlags tests positional args combined with flags
func TestPositionalArgsWithFlags(t *testing.T) {
	app := New("test", "Test application")
	app.BoolFlag("verbose", "Verbose output").Short('v')
	app.StringArg("input", "Input file").Required()
	app.StringArg("output", "Output file").Required()

	parser := NewParser(app)
	result, err := parser.Parse([]string{"-v", "input.txt", "output.txt"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check flag
	verbose, _ := result.GetBool("verbose")
	if !verbose {
		t.Error("Expected verbose=true")
	}

	// Check positional args
	input, _ := result.GetArgString("input")
	if input != "input.txt" {
		t.Errorf("Expected input='input.txt', got '%s'", input)
	}

	output, _ := result.GetArgString("output")
	if output != "output.txt" {
		t.Errorf("Expected output='output.txt', got '%s'", output)
	}
}

// TestPositionalArgsInvalidType tests type conversion errors
func TestPositionalArgsInvalidType(t *testing.T) {
	app := New("test", "Test application")
	app.IntArg("number", "A number").Required()

	parser := NewParser(app)
	_, err := parser.Parse([]string{"not-a-number"})
	if err == nil {
		t.Fatal("Expected error for invalid int value")
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("Expected *ParseError, got %T", err)
	}
	if parseErr.Type != ErrorTypeInvalidArgument {
		t.Errorf("Expected ErrorTypeInvalidArgument, got %s", parseErr.Type)
	}
}

// BenchmarkPositionalArgsZeroAlloc verifies zero allocations for positional arg parsing
func BenchmarkPositionalArgsZeroAlloc(b *testing.B) {
	app := New("test", "Test application")
	app.StringArg("name", "Name").Required()
	app.IntArg("age", "Age").Required()
	app.BoolArg("active", "Active").Default(true)

	parser := NewParser(app)
	args := []string{"John", "30", "true"}

	b.ResetTimer()
	b.ReportAllocs()

	allocs := testing.AllocsPerRun(100, func() {
		parser.reset()
		result, err := parser.Parse(args)
		if err != nil {
			b.Fatal(err)
		}
		// Access values to ensure they're actually stored
		_, _ = result.GetArgString("name")
		_, _ = result.GetArgInt("age")
		_, _ = result.GetArgBool("active")
	})

	if allocs > 0 {
		b.Errorf("Expected 0 allocations, got %.2f", allocs)
	}
}

// BenchmarkVariadicArgsAlloc tests allocations for variadic args
func BenchmarkVariadicArgsAlloc(b *testing.B) {
	app := New("test", "Test application")
	app.StringArg("cmd", "Command").Required()
	app.StringSliceArg("files", "Files").Variadic()

	parser := NewParser(app)
	args := []string{"rm", "file1.txt", "file2.txt", "file3.txt"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		parser.reset()
		result, err := parser.Parse(args)
		if err != nil {
			b.Fatal(err)
		}
		_, _ = result.GetArgString("cmd")
		_, _ = result.GetArgStringSlice("files")
	}
}

// BenchmarkPositionalArgsVsCobra compares performance with Cobra (conceptual)
func BenchmarkPositionalArgsVsCobra(b *testing.B) {
	b.Run("go-snap", func(b *testing.B) {
		app := New("test", "Test application")
		app.StringArg("input", "Input file").Required()
		app.StringArg("output", "Output file").Required()

		parser := NewParser(app)
		args := []string{"input.txt", "output.txt"}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			parser.reset()
			_, err := parser.Parse(args)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Note: Cobra benchmark would go here for comparison
	// b.Run("cobra", func(b *testing.B) { ... })
}
