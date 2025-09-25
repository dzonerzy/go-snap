package middleware

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// ValidatorFunc represents a custom validation function for business logic validation.
// Use this to implement validation that goes beyond flag structure (which should be
// handled by FlagGroups in the CLI definition).
//
// Examples of appropriate middleware validation:
// - Conditional requirements based on business logic
// - File system checks (file/directory existence)
// - API connectivity validation
// - Cross-cutting validation that spans multiple components
type ValidatorFunc func(ctx Context) error

// Validator creates a middleware that performs business logic validation.
//
// This middleware is a framework for implementing custom business logic validation.
// It does NOT handle flag relationship validation (required, mutually exclusive, etc.)
// - those should be handled by FlagGroups in the CLI definition.
//
// Use this middleware for:
// - Runtime file system checks
// - Conditional business logic validation
// - API access validation
// - Any validation that requires external state checks
func Validator(options ...MiddlewareOption) Middleware {
	config := DefaultConfig()
	for _, option := range options {
		option(config)
	}

	return func(next ActionFunc) ActionFunc {
		return func(ctx Context) error {
			// Run custom validators (business logic validation)
			for name, validator := range config.CustomValidators {
				if err := validator(ctx); err != nil {
					// If it's already a ValidationError, return it directly
					validationErr := &ValidationError{}
					if errors.As(err, &validationErr) {
						return validationErr
					}
					// Otherwise, wrap it
					return &ValidationError{
						Field:   name,
						Message: "validation failed",
						Cause:   err,
					}
				}
			}

			// Execute the action
			return next(ctx)
		}
	}
}

// ValidatorWithCustom creates a validator middleware with custom validation functions
// ValidatorWithCustom composes a middleware that runs the provided named
// validators before the action. The map key is used in error reporting.
func ValidatorWithCustom(validators map[string]ValidatorFunc) Middleware {
	return func(next ActionFunc) ActionFunc {
		return func(ctx Context) error {
			// Run custom validators
			for name, validator := range validators {
				if err := validator(ctx); err != nil {
					// If it's already a ValidationError, return it directly
					validationErr := &ValidationError{}
					if errors.As(err, &validationErr) {
						return validationErr
					}
					// Otherwise, wrap it
					return &ValidationError{
						Field:   name,
						Message: "validation failed",
						Cause:   err,
					}
				}
			}

			return next(ctx)
		}
	}
}

// NamedValidator wraps a ValidatorFunc with a display name for friendly APIs.
// NamedValidator associates a human-readable name with a ValidatorFunc for
// clearer error reporting and easier composition.
type NamedValidator struct {
	Name string
	Fn   ValidatorFunc
}

// Custom wraps an arbitrary ValidatorFunc with a name for reporting.
func Custom(name string, fn ValidatorFunc) NamedValidator {
	return NamedValidator{Name: name, Fn: fn}
}

// File returns a NamedValidator that ensures the given file flags exist. Both
// command-local and global flags are checked.
func File(flagNames ...string) NamedValidator {
	return NamedValidator{Name: "file_exists", Fn: FileExists(flagNames...)}
}

// Dir returns a NamedValidator that ensures the given directory flags exist.
// Both command-local and global flags are checked.
func Dir(flagNames ...string) NamedValidator {
	return NamedValidator{Name: "directory_exists", Fn: DirectoryExists(flagNames...)}
}

// Validate composes a set of NamedValidators into a single Middleware.
//
// Example:
//
//	app.Use(middleware.Validate(
//	    middleware.Custom("port_range", checkPort),
//	    middleware.File("config"),
//	))
func Validate(validators ...NamedValidator) Middleware {
	m := make(map[string]ValidatorFunc, len(validators))
	for _, v := range validators {
		if v.Name == "" || v.Fn == nil {
			continue
		}
		m[v.Name] = v.Fn
	}
	return ValidatorWithCustom(m)
}

// ConditionalRequired creates a validator that makes flags required based on conditions
// This is useful for business logic validation where requirements depend on other values
func ConditionalRequired(condition ValidatorFunc, requiredFlags ...string) ValidatorFunc {
	return func(ctx Context) error {
		// Check if condition is met
		if err := condition(ctx); err == nil {
			// Condition is met, check required flags
			var missing []string
			for _, flagName := range requiredFlags {
				if hasFlag := checkFlagPresence(ctx, flagName); !hasFlag {
					missing = append(missing, flagName)
				}
			}

			if len(missing) > 0 {
				return &ValidationError{
					Field:   strings.Join(missing, ", "),
					Message: fmt.Sprintf("flags required when condition is met: %s", strings.Join(missing, ", ")),
				}
			}
		}
		// Condition not met, no validation needed
		return nil
	}
}

// FileExists creates a validator that ensures file flags point to existing files
func FileExists(flagNames ...string) ValidatorFunc {
	return func(ctx Context) error {
		for _, flagName := range flagNames {
			// Check local then global scope
			var (
				path   string
				exists bool
			)
			if path, exists = ctx.String(flagName); !exists || path == "" {
				if g, ok := ctx.GlobalString(flagName); ok && g != "" {
					path, exists = g, true
				}
			}
			if exists && path != "" {
				if err := validateFileExists(path); err != nil {
					return &ValidationError{
						Field:   flagName,
						Value:   path,
						Message: fmt.Sprintf("file validation failed for flag '%s'", flagName),
						Cause:   err,
					}
				}
			}
		}
		return nil
	}
}

// DirectoryExists creates a validator that ensures directory flags point to existing directories
func DirectoryExists(flagNames ...string) ValidatorFunc {
	return func(ctx Context) error {
		for _, flagName := range flagNames {
			// Check local then global scope
			var (
				path   string
				exists bool
			)
			if path, exists = ctx.String(flagName); !exists || path == "" {
				if g, ok := ctx.GlobalString(flagName); ok && g != "" {
					path, exists = g, true
				}
			}
			if exists && path != "" {
				if err := validateDirectoryExists(path); err != nil {
					return &ValidationError{
						Field:   flagName,
						Value:   path,
						Message: fmt.Sprintf("directory validation failed for flag '%s'", flagName),
						Cause:   err,
					}
				}
			}
		}
		return nil
	}
}

// Helper functions

// checkFlagPresence checks if a flag is present (has any non-zero value)
// This is useful for business logic validation where we need to check if a flag was actually set
func checkFlagPresence(ctx Context, flagName string) bool {
	// Check string flags
	if value, exists := ctx.String(flagName); exists && value != "" {
		return true
	}

	// Check int flags
	if value, exists := ctx.Int(flagName); exists && value != 0 {
		return true
	}

	// Check bool flags
	if value, exists := ctx.Bool(flagName); exists && value {
		return true
	}

	// Check duration flags
	if value, exists := ctx.Duration(flagName); exists && value != 0 {
		return true
	}

	// Check float flags
	if value, exists := ctx.Float(flagName); exists && value != 0 {
		return true
	}

	// Check enum flags
	if value, exists := ctx.Enum(flagName); exists && value != "" {
		return true
	}

	// Check slice flags
	if value, exists := ctx.StringSlice(flagName); exists && len(value) > 0 {
		return true
	}

	if value, exists := ctx.IntSlice(flagName); exists && len(value) > 0 {
		return true
	}

	return false
}

// validateFileExists checks if a file exists
func validateFileExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	return nil
}

// validateDirectoryExists checks if a directory exists
func validateDirectoryExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

// Convenience constructors

// NoopValidator creates a validator that doesn't perform any validation.
// Useful for testing or when you want to disable validation in certain environments.
func NoopValidator() Middleware {
	return func(next ActionFunc) ActionFunc {
		return next // No validation
	}
}

// FileSystemValidator creates a validator that checks file and directory existence.
// This is a common use case for middleware validation since it requires runtime checks.
func FileSystemValidator(fileFlags, dirFlags []string) Middleware {
	validators := make(map[string]ValidatorFunc)

	if len(fileFlags) > 0 {
		validators["file_exists"] = FileExists(fileFlags...)
	}

	if len(dirFlags) > 0 {
		validators["directory_exists"] = DirectoryExists(dirFlags...)
	}

	return ValidatorWithCustom(validators)
}

// WithCustomValidators adds custom validators to the middleware config
func WithCustomValidators(validators map[string]ValidatorFunc) MiddlewareOption {
	return func(config *MiddlewareConfig) {
		if config.CustomValidators == nil {
			config.CustomValidators = make(map[string]ValidatorFunc)
		}
		for name, validator := range validators {
			config.CustomValidators[name] = validator
		}
	}
}
