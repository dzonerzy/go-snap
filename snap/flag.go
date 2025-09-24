package snap

import (
	"fmt"
	"os"
	"regexp"
	"time"
)

// FlagParent interface allows both App and CommandBuilder to be used as flag parents
type FlagParent interface {
	addShortFlag(short rune, flag *Flag)
}

// FlagType represents the type of a flag
type FlagType string

const (
	// Core types
	FlagTypeString   FlagType = "string"
	FlagTypeBool     FlagType = "bool"
	FlagTypeInt      FlagType = "int"
	FlagTypeDuration FlagType = "duration"
	FlagTypeFloat    FlagType = "float64"
	FlagTypeEnum     FlagType = "enum"

	// Collection types
	FlagTypeStringSlice FlagType = "[]string"
	FlagTypeIntSlice    FlagType = "[]int"
)

// Flag represents a command-line flag with all its properties
type Flag struct {
    Name            string
    Description     string
    Type            FlagType
    DefaultString   string
    DefaultInt      int
    DefaultBool     bool
    DefaultDuration time.Duration
    DefaultFloat    float64
    DefaultEnum     string
    DefaultStringSlice []string
    DefaultIntSlice    []int
    Global          bool
    Required        bool
    Hidden          bool
    Short           rune
    EnvVars         []string // Environment variables to check (in precedence order)
	Usage           string

	// Enum-specific fields
	EnumValues []string // Valid enum values

	// Type-safe validation function (will be cast to func(T) error at runtime)
	Validator interface{}
}

// RequiresValue returns true if the flag type requires a value
func (f *Flag) RequiresValue() bool {
	return f.Type != FlagTypeBool
}

// IsGlobal returns true if the flag is global
func (f *Flag) IsGlobal() bool {
	return f.Global
}

// Validation helper functions

// ValidateFile creates a validation function for file paths
func ValidateFile(mustExist bool) func(string) error {
	return func(path string) error {
		if path == "" {
			return fmt.Errorf("file path cannot be empty")
		}
		if mustExist {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("file does not exist: %s", path)
			} else if err != nil {
				return fmt.Errorf("cannot access file %s: %v", path, err)
			}
		}
		return nil
	}
}

// ValidateDir creates a validation function for directory paths
func ValidateDir(mustExist bool) func(string) error {
	return func(path string) error {
		if path == "" {
			return fmt.Errorf("directory path cannot be empty")
		}
		if mustExist {
			info, err := os.Stat(path)
			if os.IsNotExist(err) {
				return fmt.Errorf("directory does not exist: %s", path)
			} else if err != nil {
				return fmt.Errorf("cannot access directory %s: %v", path, err)
			} else if !info.IsDir() {
				return fmt.Errorf("path is not a directory: %s", path)
			}
		}
		return nil
	}
}

// ValidateRegex creates a validation function that validates strings against a regex pattern
func ValidateRegex(pattern string) func(string) error {
	// Compile the regex once during function creation
	regex, err := regexp.Compile(pattern)
	if err != nil {
		// Return a function that always returns this compilation error
		return func(string) error {
			return fmt.Errorf("invalid regex pattern '%s': %v", pattern, err)
		}
	}

	return func(value string) error {
		if !regex.MatchString(value) {
			return fmt.Errorf("value '%s' does not match pattern '%s'", value, pattern)
		}
		return nil
	}
}

// ValidateOneOf creates a validation function that ensures the value is one of the allowed values
func ValidateOneOf[T comparable](values ...T) func(T) error {
	return func(value T) error {
		for _, v := range values {
			if value == v {
				return nil
			}
		}
		return fmt.Errorf("value %v is not one of the allowed values: %v", value, values)
	}
}

// FlagBuilder provides fluent API for configuring flags with type safety
// T is the flag value type, P is the parent type (either *App or *CommandBuilder)
type FlagBuilder[T any, P FlagParent] struct {
	flag   *Flag
	parent P
}

// Flag modifiers - Type-safe configuration methods

// Default sets the default value for the flag
func (f *FlagBuilder[T, P]) Default(value T) *FlagBuilder[T, P] {
    switch f.flag.Type {
    case FlagTypeString:
        if v, ok := any(value).(string); ok {
            f.flag.DefaultString = v
        }
    case FlagTypeInt:
        if v, ok := any(value).(int); ok {
            f.flag.DefaultInt = v
        }
    case FlagTypeBool:
        if v, ok := any(value).(bool); ok {
            f.flag.DefaultBool = v
        }
    case FlagTypeDuration:
        if v, ok := any(value).(time.Duration); ok {
            f.flag.DefaultDuration = v
        }
    case FlagTypeFloat:
        if v, ok := any(value).(float64); ok {
            f.flag.DefaultFloat = v
        }
    case FlagTypeEnum:
        if v, ok := any(value).(string); ok {
            f.flag.DefaultEnum = v
        }
    case FlagTypeStringSlice:
        if v, ok := any(value).([]string); ok {
            f.flag.DefaultStringSlice = v
        }
    case FlagTypeIntSlice:
        if v, ok := any(value).([]int); ok {
            f.flag.DefaultIntSlice = v
        }
    }
    return f
}

// Required marks the flag as required
func (f *FlagBuilder[T, P]) Required() *FlagBuilder[T, P] {
	f.flag.Required = true
	return f
}

// Short sets a short flag alias (single character)
func (f *FlagBuilder[T, P]) Short(short rune) *FlagBuilder[T, P] {
	f.flag.Short = short

	// Use parent interface for O(1) lookup - no need to check types!
	f.parent.addShortFlag(short, f.flag)

	return f
}

// Global marks the flag as global (available to all commands)
func (f *FlagBuilder[T, P]) Global() *FlagBuilder[T, P] {
	f.flag.Global = true
	return f
}

// Hidden hides the flag from help output
func (f *FlagBuilder[T, P]) Hidden() *FlagBuilder[T, P] {
	f.flag.Hidden = true
	return f
}

// FromEnv binds the flag to environment variables (checked in precedence order)
func (f *FlagBuilder[T, P]) FromEnv(envVars ...string) *FlagBuilder[T, P] {
	f.flag.EnvVars = envVars
	return f
}

// Usage sets a detailed usage description
func (f *FlagBuilder[T, P]) Usage(usage string) *FlagBuilder[T, P] {
	f.flag.Usage = usage
	return f
}

// Validate adds a validation function for the flag value
func (f *FlagBuilder[T, P]) Validate(fn func(T) error) *FlagBuilder[T, P] {
	// Store the type-safe validation function
	f.flag.Validator = fn
	return f
}

// Builder termination - returns to parent builder

// Build finalizes the flag configuration and returns to the parent builder
func (f *FlagBuilder[T, P]) Build() interface{} {
	// This method allows chaining back to the parent builder
	// The return type would be either *AppBuilder or *CommandBuilder
	// For now, we'll handle this through method chaining patterns
	return f
}

// Convenience methods - syntactic sugar over validation functions

// Range sets inclusive min/max validation for numeric flags (int and float64).
// The value must satisfy min <= value <= max.
func Range[T int | float64, P FlagParent](f *FlagBuilder[T, P], min, max T) *FlagBuilder[T, P] {
	return f.Validate(func(value T) error {
		if value < min || value > max {
			return fmt.Errorf("value %v is not within range [%v, %v]", value, min, max)
		}
		return nil
	})
}

// OneOf sets validation to ensure the value is one of the allowed values (for string flags)
func OneOf[P FlagParent](f *FlagBuilder[string, P], values ...string) *FlagBuilder[string, P] {
	return f.Validate(ValidateOneOf(values...))
}

// File sets file path validation for string flags
func File[P FlagParent](f *FlagBuilder[string, P], mustExist bool) *FlagBuilder[string, P] {
	return f.Validate(ValidateFile(mustExist))
}

// Dir sets directory path validation for string flags
func Dir[P FlagParent](f *FlagBuilder[string, P], mustExist bool) *FlagBuilder[string, P] {
	return f.Validate(ValidateDir(mustExist))
}

// Regex sets regex pattern validation for string flags
func Regex[P FlagParent](f *FlagBuilder[string, P], pattern string) *FlagBuilder[string, P] {
	return f.Validate(ValidateRegex(pattern))
}

// Back returns to the parent builder context for continued chaining.
// Returns *App for app-level flags, *CommandBuilder for command-level flags.
func (f *FlagBuilder[T, P]) Back() P {
	return f.parent
}

// Flag Group System

// GroupConstraintType represents the type of constraint for flag groups
type GroupConstraintType int

const (
	GroupNoConstraint      GroupConstraintType = iota // Flags work independently (DEFAULT)
	GroupMutuallyExclusive                           // Only one flag can be set
	GroupAllOrNone                                   // Either all flags or no flags
	GroupAtLeastOne                                  // At least one flag must be set
	GroupExactlyOne                                  // Exactly one flag must be set
	GroupRequiredGroup                               // Alias for GroupAtLeastOne (deprecated)
)

// FlagGroup represents a group of related flags with constraints
type FlagGroup struct {
	Name        string
	Description string
	Flags       []*Flag
	Constraint  GroupConstraintType
}

// FlagGroupParent interface for type-safe group building
type FlagGroupParent interface {
	FlagParent
	addFlagGroup(group *FlagGroup)
}

// FlagGroupBuilder provides fluent API for flag group configuration
// P is the parent type (*App or *CommandBuilder)
type FlagGroupBuilder[P FlagGroupParent] struct {
	group  *FlagGroup
	parent P
}

// Group constraint methods

// MutuallyExclusive sets the group to allow only one flag to be set
func (g *FlagGroupBuilder[P]) MutuallyExclusive() *FlagGroupBuilder[P] {
	g.group.Constraint = GroupMutuallyExclusive
	return g
}

// RequiredGroup sets the group to require at least one flag to be set
func (g *FlagGroupBuilder[P]) RequiredGroup() *FlagGroupBuilder[P] {
	g.group.Constraint = GroupRequiredGroup
	return g
}

// AllOrNone sets the group to require either all flags or no flags
func (g *FlagGroupBuilder[P]) AllOrNone() *FlagGroupBuilder[P] {
	g.group.Constraint = GroupAllOrNone
	return g
}

// ExactlyOne sets the group to require exactly one flag to be set
func (g *FlagGroupBuilder[P]) ExactlyOne() *FlagGroupBuilder[P] {
	g.group.Constraint = GroupExactlyOne
	return g
}

// AtLeastOne sets the group to require at least one flag to be set (alias for RequiredGroup)
func (g *FlagGroupBuilder[P]) AtLeastOne() *FlagGroupBuilder[P] {
	g.group.Constraint = GroupAtLeastOne
	return g
}

// Description sets a description for the flag group
func (g *FlagGroupBuilder[P]) Description(desc string) *FlagGroupBuilder[P] {
	g.group.Description = desc
	return g
}

// Flag creation methods for groups - return FlagBuilder with FlagGroupBuilder as parent

// StringFlag creates a string flag within the group
func (g *FlagGroupBuilder[P]) StringFlag(name, description string) *FlagBuilder[string, *FlagGroupBuilder[P]] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeString,
	}
	g.group.Flags = append(g.group.Flags, flag)
	return &FlagBuilder[string, *FlagGroupBuilder[P]]{
		flag:   flag,
		parent: g,
	}
}

// IntFlag creates an integer flag within the group
func (g *FlagGroupBuilder[P]) IntFlag(name, description string) *FlagBuilder[int, *FlagGroupBuilder[P]] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeInt,
	}
	g.group.Flags = append(g.group.Flags, flag)
	return &FlagBuilder[int, *FlagGroupBuilder[P]]{
		flag:   flag,
		parent: g,
	}
}

// BoolFlag creates a boolean flag within the group
func (g *FlagGroupBuilder[P]) BoolFlag(name, description string) *FlagBuilder[bool, *FlagGroupBuilder[P]] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeBool,
	}
	g.group.Flags = append(g.group.Flags, flag)
	return &FlagBuilder[bool, *FlagGroupBuilder[P]]{
		flag:   flag,
		parent: g,
	}
}

// DurationFlag creates a duration flag within the group
func (g *FlagGroupBuilder[P]) DurationFlag(name, description string) *FlagBuilder[time.Duration, *FlagGroupBuilder[P]] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeDuration,
	}
	g.group.Flags = append(g.group.Flags, flag)
	return &FlagBuilder[time.Duration, *FlagGroupBuilder[P]]{
		flag:   flag,
		parent: g,
	}
}

// FloatFlag creates a float64 flag within the group
func (g *FlagGroupBuilder[P]) FloatFlag(name, description string) *FlagBuilder[float64, *FlagGroupBuilder[P]] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeFloat,
	}
	g.group.Flags = append(g.group.Flags, flag)
	return &FlagBuilder[float64, *FlagGroupBuilder[P]]{
		flag:   flag,
		parent: g,
	}
}

// StringSliceFlag creates a string slice flag within the group
func (g *FlagGroupBuilder[P]) StringSliceFlag(name, description string) *FlagBuilder[[]string, *FlagGroupBuilder[P]] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeStringSlice,
	}
	g.group.Flags = append(g.group.Flags, flag)
	return &FlagBuilder[[]string, *FlagGroupBuilder[P]]{
		flag:   flag,
		parent: g,
	}
}

// IntSliceFlag creates an integer slice flag within the group
func (g *FlagGroupBuilder[P]) IntSliceFlag(name, description string) *FlagBuilder[[]int, *FlagGroupBuilder[P]] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeIntSlice,
	}
	g.group.Flags = append(g.group.Flags, flag)
	return &FlagBuilder[[]int, *FlagGroupBuilder[P]]{
		flag:   flag,
		parent: g,
	}
}

// EnumFlag creates an enum flag within the group
func (g *FlagGroupBuilder[P]) EnumFlag(name, description string, values ...string) *FlagBuilder[string, *FlagGroupBuilder[P]] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeEnum,
		EnumValues:  values,
	}
	g.group.Flags = append(g.group.Flags, flag)
	return &FlagBuilder[string, *FlagGroupBuilder[P]]{
		flag:   flag,
		parent: g,
	}
}

// Navigation methods

// EndGroup terminates the group and returns to the parent builder
func (g *FlagGroupBuilder[P]) EndGroup() P {
	// Add the group to the parent before returning
	g.parent.addFlagGroup(g.group)
	return g.parent
}

// App provides direct escape to the app context (for deep nesting)
func (g *FlagGroupBuilder[P]) App() *App {
	// Add the group to the parent before escaping
	g.parent.addFlagGroup(g.group)

	// Navigate to app context
	switch p := any(g.parent).(type) {
	case *App:
		return p
	case interface{ App() *App }:
		return p.App()
	default:
		panic("unsupported parent type for App() navigation")
	}
}

// addShortFlag implementation for FlagGroupBuilder (to satisfy FlagParent interface)
func (g *FlagGroupBuilder[P]) addShortFlag(short rune, flag *Flag) {
	// Delegate to parent for short flag registration
	g.parent.addShortFlag(short, flag)
}
