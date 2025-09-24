package snap

import (
    "errors"
    "reflect"

    "github.com/dzonerzy/go-snap/middleware"
)

// ExitError is a sentinel used to request a specific exit code from inside actions.
type ExitError struct {
    Code int
    Err  error
}

func (e *ExitError) Error() string {
    if e.Err != nil { return e.Err.Error() }
    return "exit"
}

// ExitCodeDefaults holds common default codes.
type ExitCodeDefaults struct {
    Success         int // default: 0
    GeneralError    int // default: 1
    MisusageError   int // default: 2
    ValidationError int // default: 3
    NotFoundError   int // default: 127
    PermissionError int // default: 126
}

func defaultExitDefaults() ExitCodeDefaults {
    return ExitCodeDefaults{Success:0, GeneralError:1, MisusageError:2, ValidationError:3, NotFoundError:127, PermissionError:126}
}

// ExitCodeManager maps errors and categories to process exit codes.
type ExitCodeManager struct {
    codesByName map[string]int
    codesByType map[reflect.Type]int
    codesByCLI  map[ErrorType]int
    defaults    ExitCodeDefaults
}

func newExitCodeManager() *ExitCodeManager {
    m := &ExitCodeManager{
        codesByName: make(map[string]int),
        codesByType: make(map[reflect.Type]int),
        codesByCLI:  make(map[ErrorType]int),
        defaults:    defaultExitDefaults(),
    }
    // Prewire common CLI mappings
    m.codesByCLI[ErrorTypeValidation] = m.defaults.ValidationError
    m.codesByCLI[ErrorTypePermission] = m.defaults.PermissionError
    m.codesByCLI[ErrorTypeMissingRequired] = m.defaults.MisusageError
    m.codesByCLI[ErrorTypeUnknownFlag] = m.defaults.MisusageError
    m.codesByCLI[ErrorTypeUnknownCommand] = m.defaults.MisusageError
    m.codesByCLI[ErrorTypeFlagGroupViolation] = m.defaults.MisusageError

    // Prewire middleware types
    m.codesByType[reflect.TypeOf(&middleware.TimeoutError{})] = m.defaults.GeneralError
    m.codesByType[reflect.TypeOf(&middleware.ValidationError{})] = m.defaults.ValidationError
    m.codesByType[reflect.TypeOf(&middleware.RecoveryError{})] = m.defaults.GeneralError
    return m
}

// Exit code configuration

// Define registers a named exit-code mapping. The name is user-defined and
// intended for documentation or convenience; it does not affect resolution
// precedence. Use DefineError/DefineCLI to affect mapping at runtime.
func (e *ExitCodeManager) Define(name string, code int) *ExitCodeManager { e.codesByName[name] = code; return e }

// DefineError maps a concrete error value (by its dynamic type) to an exit
// code. During resolution, a matching error type takes precedence over the
// default codes but is secondary to an explicit ExitError requested by the
// action.
func (e *ExitCodeManager) DefineError(err error, code int) *ExitCodeManager {
    if err == nil { return e }
    e.codesByType[reflect.TypeOf(err)] = code
    return e
}

// DefineCLI overrides the exit code used for a specific CLI error category
// produced by the parser (e.g., unknown flag/command, validation). CLI mappings
// are applied when the error is a *CLIError.
func (e *ExitCodeManager) DefineCLI(typ ErrorType, code int) *ExitCodeManager { e.codesByCLI[typ] = code; return e }

// Default replaces the manager's default codes (Success, Misusage, etc.).
// Defaults apply when no specific mapping matches.
func (e *ExitCodeManager) Default(d ExitCodeDefaults) *ExitCodeManager { e.defaults = d; return e }

// resolve converts an error to an exit code according to registered mappings.
// Precedence:
//   1) ExitError (requested code)
//   2) CLIError category mapping (DefineCLI)
//   3) Concrete error type mapping (DefineError)
//   4) Default codes
func (e *ExitCodeManager) resolve(err error) int {
    if err == nil { return e.defaults.Success }

    // ExitError wins
    var exitErr *ExitError
    if errors.As(err, &exitErr) {
        return exitErr.Code
    }

    // CLIError mapping
    var cli *CLIError
    if errors.As(err, &cli) {
        if code, ok := e.codesByCLI[cli.Type]; ok {
            return code
        }
        return e.defaults.GeneralError
    }

    // middleware errors by concrete type
    for t, code := range e.codesByType {
        if errors.As(err, reflect.New(t).Interface()) {
            return code
        }
    }

    // Fallback
    return e.defaults.GeneralError
}
