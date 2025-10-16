package snap

import (
	"time"
)

// ArgType represents the type of a positional argument
type ArgType string

const (
	// ArgTypeString indicates a string-valued argument.
	ArgTypeString ArgType = "string"
	// ArgTypeBool indicates a boolean argument.
	ArgTypeBool ArgType = "bool"
	// ArgTypeInt indicates an integer argument.
	ArgTypeInt ArgType = "int"
	// ArgTypeDuration indicates a time.Duration argument.
	ArgTypeDuration ArgType = "duration"
	// ArgTypeFloat indicates a float64 argument.
	ArgTypeFloat ArgType = "float64"
	// ArgTypeStringSlice indicates a []string argument (variadic).
	ArgTypeStringSlice ArgType = "[]string"
	// ArgTypeIntSlice indicates a []int argument (variadic).
	ArgTypeIntSlice ArgType = "[]int"
)

// Arg represents a positional command-line argument with all its properties
type Arg struct {
	Name               string
	Description        string
	Type               ArgType
	Position           int // 0-indexed position
	DefaultString      string
	DefaultInt         int
	DefaultBool        bool
	DefaultDuration    time.Duration
	DefaultFloat       float64
	DefaultStringSlice []string
	DefaultIntSlice    []int
	Required           bool
	Variadic           bool // Only valid for last arg, only for StringSlice/IntSlice types

	// Type-safe validation function (will be cast to func(T) error at runtime)
	Validator interface{}
}

// IsRequired returns true if the argument is required
func (a *Arg) IsRequired() bool {
	return a.Required
}

// IsVariadic returns true if the argument accepts multiple values (variadic)
func (a *Arg) IsVariadic() bool {
	return a.Variadic
}

// ArgBuilder provides a fluent interface for building positional arguments
type ArgBuilder[T any] struct {
	arg       *Arg
	parentApp *App            // Set if parent is *App
	parentCmd *CommandBuilder // Set if parent is *CommandBuilder
}

// Required marks the argument as required
func (b *ArgBuilder[T]) Required() *ArgBuilder[T] {
	b.arg.Required = true
	return b
}

// Default sets the default value for an optional argument
func (b *ArgBuilder[T]) Default(value T) *ArgBuilder[T] {
	b.arg.Required = false
	switch any(value).(type) {
	case string:
		b.arg.DefaultString = any(value).(string)
	case int:
		b.arg.DefaultInt = any(value).(int)
	case bool:
		b.arg.DefaultBool = any(value).(bool)
	case time.Duration:
		b.arg.DefaultDuration = any(value).(time.Duration)
	case float64:
		b.arg.DefaultFloat = any(value).(float64)
	case []string:
		b.arg.DefaultStringSlice = any(value).([]string)
	case []int:
		b.arg.DefaultIntSlice = any(value).([]int)
	}
	return b
}

// Variadic marks the argument as variadic (accepts multiple values)
// Only valid for StringSliceArg and must be the last positional argument
func (b *ArgBuilder[T]) Variadic() *ArgBuilder[T] {
	b.arg.Variadic = true
	return b
}

// Validate adds a validation function for the argument
func (b *ArgBuilder[T]) Validate(fn func(T) error) *ArgBuilder[T] {
	b.arg.Validator = fn
	return b
}

// App returns to the parent App builder (panics if parent is not *App)
func (b *ArgBuilder[T]) App() *App {
	if b.parentApp != nil {
		return b.parentApp
	}
	panic("ArgBuilder parent is not *App")
}

// Command returns to the parent CommandBuilder (panics if parent is not *CommandBuilder)
func (b *ArgBuilder[T]) Command() *CommandBuilder {
	if b.parentCmd != nil {
		return b.parentCmd
	}
	panic("ArgBuilder parent is not *CommandBuilder")
}

// Helper functions for creating argument builders

func newStringArg(name, description string, position int, parent interface{}) *ArgBuilder[string] {
	arg := &Arg{
		Name:        name,
		Description: description,
		Type:        ArgTypeString,
		Position:    position,
		Required:    false, // Default to optional
	}
	builder := &ArgBuilder[string]{arg: arg}
	if app, ok := parent.(*App); ok {
		builder.parentApp = app
	} else if cmd, ok := parent.(*CommandBuilder); ok {
		builder.parentCmd = cmd
	}
	return builder
}

func newIntArg(name, description string, position int, parent interface{}) *ArgBuilder[int] {
	arg := &Arg{
		Name:        name,
		Description: description,
		Type:        ArgTypeInt,
		Position:    position,
		Required:    false,
	}
	builder := &ArgBuilder[int]{arg: arg}
	if app, ok := parent.(*App); ok {
		builder.parentApp = app
	} else if cmd, ok := parent.(*CommandBuilder); ok {
		builder.parentCmd = cmd
	}
	return builder
}

func newBoolArg(name, description string, position int, parent interface{}) *ArgBuilder[bool] {
	arg := &Arg{
		Name:        name,
		Description: description,
		Type:        ArgTypeBool,
		Position:    position,
		Required:    false,
	}
	builder := &ArgBuilder[bool]{arg: arg}
	if app, ok := parent.(*App); ok {
		builder.parentApp = app
	} else if cmd, ok := parent.(*CommandBuilder); ok {
		builder.parentCmd = cmd
	}
	return builder
}

func newFloatArg(name, description string, position int, parent interface{}) *ArgBuilder[float64] {
	arg := &Arg{
		Name:        name,
		Description: description,
		Type:        ArgTypeFloat,
		Position:    position,
		Required:    false,
	}
	builder := &ArgBuilder[float64]{arg: arg}
	if app, ok := parent.(*App); ok {
		builder.parentApp = app
	} else if cmd, ok := parent.(*CommandBuilder); ok {
		builder.parentCmd = cmd
	}
	return builder
}

func newDurationArg(name, description string, position int, parent interface{}) *ArgBuilder[time.Duration] {
	arg := &Arg{
		Name:        name,
		Description: description,
		Type:        ArgTypeDuration,
		Position:    position,
		Required:    false,
	}
	builder := &ArgBuilder[time.Duration]{arg: arg}
	if app, ok := parent.(*App); ok {
		builder.parentApp = app
	} else if cmd, ok := parent.(*CommandBuilder); ok {
		builder.parentCmd = cmd
	}
	return builder
}

func newStringSliceArg(name, description string, position int, parent interface{}) *ArgBuilder[[]string] {
	arg := &Arg{
		Name:        name,
		Description: description,
		Type:        ArgTypeStringSlice,
		Position:    position,
		Required:    false,
		Variadic:    false, // Must explicitly call .Variadic()
	}
	builder := &ArgBuilder[[]string]{arg: arg}
	if app, ok := parent.(*App); ok {
		builder.parentApp = app
	} else if cmd, ok := parent.(*CommandBuilder); ok {
		builder.parentCmd = cmd
	}
	return builder
}

func newIntSliceArg(name, description string, position int, parent interface{}) *ArgBuilder[[]int] {
	arg := &Arg{
		Name:        name,
		Description: description,
		Type:        ArgTypeIntSlice,
		Position:    position,
		Required:    false,
		Variadic:    false, // Must explicitly call .Variadic()
	}
	builder := &ArgBuilder[[]int]{arg: arg}
	if app, ok := parent.(*App); ok {
		builder.parentApp = app
	} else if cmd, ok := parent.(*CommandBuilder); ok {
		builder.parentCmd = cmd
	}
	return builder
}
