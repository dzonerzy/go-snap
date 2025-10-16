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

// ArgBuilder provides a fluent interface for building positional arguments.
// Similar to FlagBuilder, it uses two type parameters:
// - T: the value type (string, int, bool, etc.)
// - P: the parent type (*App or *CommandBuilder)
type ArgBuilder[T any, P any] struct {
	arg    *Arg
	parent P
}

// Required marks the argument as required and returns the builder for chaining
func (b *ArgBuilder[T, P]) Required() *ArgBuilder[T, P] {
	b.arg.Required = true
	return b
}

// Default sets the default value for an optional argument and returns parent for chaining
func (b *ArgBuilder[T, P]) Default(value T) P {
	b.arg.Required = false
	switch v := any(value).(type) {
	case string:
		b.arg.DefaultString = v
	case int:
		b.arg.DefaultInt = v
	case bool:
		b.arg.DefaultBool = v
	case time.Duration:
		b.arg.DefaultDuration = v
	case float64:
		b.arg.DefaultFloat = v
	case []string:
		b.arg.DefaultStringSlice = v
	case []int:
		b.arg.DefaultIntSlice = v
	}
	return b.parent
}

// Variadic marks the argument as variadic (accepts multiple values)
// Only valid for StringSliceArg and must be the last positional argument
// Returns parent to complete the chain
func (b *ArgBuilder[T, P]) Variadic() P {
	b.arg.Variadic = true
	return b.parent
}

// Validate adds a validation function for the argument and returns builder for chaining
func (b *ArgBuilder[T, P]) Validate(fn func(T) error) *ArgBuilder[T, P] {
	b.arg.Validator = fn
	return b
}

// Back returns to the parent builder (App or CommandBuilder) to continue chaining
func (b *ArgBuilder[T, P]) Back() P {
	return b.parent
}

// Helper functions for creating argument builders

func newStringArg[P any](name, description string, position int, parent P) *ArgBuilder[string, P] {
	arg := &Arg{
		Name:        name,
		Description: description,
		Type:        ArgTypeString,
		Position:    position,
		Required:    false, // Default to optional
	}
	return &ArgBuilder[string, P]{arg: arg, parent: parent}
}

func newIntArg[P any](name, description string, position int, parent P) *ArgBuilder[int, P] {
	arg := &Arg{
		Name:        name,
		Description: description,
		Type:        ArgTypeInt,
		Position:    position,
		Required:    false,
	}
	return &ArgBuilder[int, P]{arg: arg, parent: parent}
}

func newBoolArg[P any](name, description string, position int, parent P) *ArgBuilder[bool, P] {
	arg := &Arg{
		Name:        name,
		Description: description,
		Type:        ArgTypeBool,
		Position:    position,
		Required:    false,
	}
	return &ArgBuilder[bool, P]{arg: arg, parent: parent}
}

func newFloatArg[P any](name, description string, position int, parent P) *ArgBuilder[float64, P] {
	arg := &Arg{
		Name:        name,
		Description: description,
		Type:        ArgTypeFloat,
		Position:    position,
		Required:    false,
	}
	return &ArgBuilder[float64, P]{arg: arg, parent: parent}
}

func newDurationArg[P any](name, description string, position int, parent P) *ArgBuilder[time.Duration, P] {
	arg := &Arg{
		Name:        name,
		Description: description,
		Type:        ArgTypeDuration,
		Position:    position,
		Required:    false,
	}
	return &ArgBuilder[time.Duration, P]{arg: arg, parent: parent}
}

func newStringSliceArg[P any](name, description string, position int, parent P) *ArgBuilder[[]string, P] {
	arg := &Arg{
		Name:        name,
		Description: description,
		Type:        ArgTypeStringSlice,
		Position:    position,
		Required:    false,
		Variadic:    false, // Must explicitly call .Variadic()
	}
	return &ArgBuilder[[]string, P]{arg: arg, parent: parent}
}

func newIntSliceArg[P any](name, description string, position int, parent P) *ArgBuilder[[]int, P] {
	arg := &Arg{
		Name:        name,
		Description: description,
		Type:        ArgTypeIntSlice,
		Position:    position,
		Required:    false,
		Variadic:    false, // Must explicitly call .Variadic()
	}
	return &ArgBuilder[[]int, P]{arg: arg, parent: parent}
}
