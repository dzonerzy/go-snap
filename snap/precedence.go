package snap

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// SourceType represents the type of configuration source
type SourceType int

const (
	SourceTypeDefaults SourceType = iota
	SourceTypeFile
	SourceTypeEnv
	SourceTypeFlags
)

// ConfigSource represents a configuration source with data and priority
type ConfigSource struct {
	Type     SourceType
	Data     map[string]any
	Priority int
}

// PrecedenceManager handles configuration precedence and resolution
type PrecedenceManager struct {
	sources []ConfigSource
}

// NewPrecedenceManager creates a new precedence manager
func NewPrecedenceManager() *PrecedenceManager {
	return &PrecedenceManager{
		sources: make([]ConfigSource, 0),
	}
}

// AddSource adds a configuration source with its priority
func (pm *PrecedenceManager) AddSource(sourceType SourceType, data map[string]any) {
	source := ConfigSource{
		Type:     sourceType,
		Data:     data,
		Priority: int(sourceType),
	}
	pm.sources = append(pm.sources, source)
}

// Resolve resolves configuration with proper precedence
// Returns the final configuration map with highest priority values
func (pm *PrecedenceManager) Resolve() map[string]any {
    result := make(map[string]any)

	// Process sources in priority order (lowest to highest)
	// This ensures higher priority sources override lower priority ones
	for priority := int(SourceTypeDefaults); priority <= int(SourceTypeFlags); priority++ {
		for _, source := range pm.sources {
			if source.Priority == priority {
				pm.mergeWithPrecedence(result, source.Data)
			}
		}
	}

    // Flatten nested maps to dotted keys so schema lookups match struct fields
    flat := make(map[string]any)
    flattenMap("", result, flat)
    return flat
}

// flattenMap converts nested maps to dotted keys (e.g., {"a":{"b":1}} => {"a.b":1})
func flattenMap(prefix string, src map[string]any, dst map[string]any) {
    for k, v := range src {
        key := k
        if prefix != "" {
            key = prefix + "." + k
        }
        if sub, ok := v.(map[string]any); ok {
            flattenMap(key, sub, dst)
            continue
        }
        dst[key] = v
    }
}

// mergeWithPrecedence merges source data into result with precedence rules
func (pm *PrecedenceManager) mergeWithPrecedence(result, source map[string]any) {
	for key, value := range source {
		// Handle nested objects
		if existingValue, exists := result[key]; exists {
			if existingMap, ok := existingValue.(map[string]any); ok {
				if sourceMap, ok := value.(map[string]any); ok {
					// Recursively merge nested maps
					pm.mergeWithPrecedence(existingMap, sourceMap)
					continue
				}
			}
		}

		// Override or set new value
		result[key] = value
	}
}

// ResolveWithSchema resolves configuration using schema for validation and type conversion
func (pm *PrecedenceManager) ResolveWithSchema(schema *ConfigSchema) (map[string]any, error) {
	// First get the merged configuration
	config := pm.Resolve()

	// Validate required fields
	if err := pm.validateRequired(config, schema); err != nil {
		return nil, err
	}

	// Apply type conversions and defaults
	if err := pm.applySchemaDefaults(config, schema); err != nil {
		return nil, err
	}

	return config, nil
}

// validateRequired checks that all required fields are present
func (pm *PrecedenceManager) validateRequired(config map[string]any, schema *ConfigSchema) error {
	for fieldName, fieldSchema := range schema.Fields {
		if fieldSchema.Required {
			if _, exists := config[fieldName]; !exists {
				return fmt.Errorf("required field '%s' is missing", fieldName)
			}
		}
	}
	return nil
}

// applySchemaDefaults applies default values and type conversions based on schema
func (pm *PrecedenceManager) applySchemaDefaults(config map[string]any, schema *ConfigSchema) error {
	for fieldName, fieldSchema := range schema.Fields {
		if _, exists := config[fieldName]; !exists && fieldSchema.Default != nil {
			// Apply default value
			config[fieldName] = fieldSchema.Default
		}

		// Type conversion
		if value, exists := config[fieldName]; exists {
			convertedValue, err := pm.convertValueToType(value, fieldSchema.Type)
			if err != nil {
				return fmt.Errorf("failed to convert field '%s': %w", fieldName, err)
			}
			config[fieldName] = convertedValue
		}

		// Validate enum values
		if len(fieldSchema.EnumValues) > 0 {
			if value, exists := config[fieldName]; exists {
				valueStr := fmt.Sprintf("%v", value)
				found := false
				for _, enumValue := range fieldSchema.EnumValues {
					if valueStr == enumValue {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("field '%s' must be one of: %s (got '%s')",
						fieldName, strings.Join(fieldSchema.EnumValues, ", "), valueStr)
				}
			}
		}
	}
	return nil
}

// convertValueToType converts a value to the specified type
func (pm *PrecedenceManager) convertValueToType(value any, targetType reflect.Type) (any, error) {
	valueReflect := reflect.ValueOf(value)

	// If already the correct type, return as-is
	if valueReflect.Type() == targetType {
		return value, nil
	}

	// Handle string conversions (common from JSON/env vars)
	if valueReflect.Kind() == reflect.String {
		return pm.convertStringToType(value.(string), targetType)
	}

	// Handle numeric conversions
	if valueReflect.Type().ConvertibleTo(targetType) {
		return valueReflect.Convert(targetType).Interface(), nil
	}

	return nil, fmt.Errorf("cannot convert %T to %s", value, targetType)
}

// convertStringToType converts string values to specific types
func (pm *PrecedenceManager) convertStringToType(str string, targetType reflect.Type) (any, error) {
	switch targetType.Kind() {
	case reflect.String:
		return str, nil

	case reflect.Bool:
		return pm.parseBoolString(str)

	case reflect.Int:
		return strconv.Atoi(str)

	case reflect.Int8:
		if val, err := strconv.ParseInt(str, 10, 8); err != nil {
			return nil, err
		} else {
			return int8(val), nil
		}

	case reflect.Int16:
		if val, err := strconv.ParseInt(str, 10, 16); err != nil {
			return nil, err
		} else {
			return int16(val), nil
		}

	case reflect.Int32:
		if val, err := strconv.ParseInt(str, 10, 32); err != nil {
			return nil, err
		} else {
			return int32(val), nil
		}

	case reflect.Int64:
		if targetType == reflect.TypeOf(time.Duration(0)) {
			return pm.parseDurationString(str)
		}
		return strconv.ParseInt(str, 10, 64)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val, err := strconv.ParseUint(str, 10, int(targetType.Size()*8)); err != nil {
			return nil, err
		} else {
			return reflect.ValueOf(val).Convert(targetType).Interface(), nil
		}

	case reflect.Float32:
		if val, err := strconv.ParseFloat(str, 32); err != nil {
			return nil, err
		} else {
			return float32(val), nil
		}

	case reflect.Float64:
		return strconv.ParseFloat(str, 64)

	default:
		return nil, fmt.Errorf("unsupported type conversion to %s", targetType)
	}
}

// parseBoolString parses boolean values from strings with go-snap compatibility
func (pm *PrecedenceManager) parseBoolString(str string) (bool, error) {
	str = strings.ToLower(strings.TrimSpace(str))
	switch str {
	case "true", "t", "yes", "y", "1", "on":
		return true, nil
	case "false", "f", "no", "n", "0", "off", "":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", str)
	}
}

// ConfigurationPrecedence documents the precedence order
const ConfigurationPrecedence = `
Configuration Precedence (highest to lowest):
1. Command line flags        (Priority 3)
2. Environment variables     (Priority 2)
3. Configuration files       (Priority 1)
4. Default values           (Priority 0)

When the same configuration key is found in multiple sources,
the source with higher precedence wins.
`

// DebugPrecedence returns a debug string showing how configuration was resolved
func (pm *PrecedenceManager) DebugPrecedence() string {
	var debug strings.Builder
	debug.WriteString("Configuration Sources (in resolution order):\n")

	for priority := int(SourceTypeDefaults); priority <= int(SourceTypeFlags); priority++ {
		for _, source := range pm.sources {
			if source.Priority == priority {
				debug.WriteString(fmt.Sprintf("  Priority %d (%s): %d keys\n",
					priority, pm.sourceTypeName(source.Type), len(source.Data)))
			}
		}
	}

	return debug.String()
}

// sourceTypeName returns human-readable source type name
func (pm *PrecedenceManager) sourceTypeName(sourceType SourceType) string {
	switch sourceType {
	case SourceTypeDefaults:
		return "Defaults"
	case SourceTypeFile:
		return "Files"
	case SourceTypeEnv:
		return "Environment"
	case SourceTypeFlags:
		return "Flags"
	default:
		return "Unknown"
	}
}

// parseDurationString implements the same logic as parseDurationBytes but for strings
// Supports: "MM:SS", "HH:MM:SS", "1d", "1w", "1M", "1Y", "1h30m15s"
func (pm *PrecedenceManager) parseDurationString(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	// 1. Check for colon format: "MM:SS" or "HH:MM:SS"
	if strings.Contains(s, ":") {
		return pm.parseColonDurationString(s)
	}

	// 2. Check for extended units: "1d", "1w", "1M", "1Y"
	if duration, ok := pm.parseExtendedDurationString(s); ok {
		return duration, nil
	}

	// 3. Fall back to standard Go duration format: "1h30m15s"
	return time.ParseDuration(s)
}

// parseColonDurationString parses "MM:SS" or "HH:MM:SS" format
func (pm *PrecedenceManager) parseColonDurationString(s string) (time.Duration, error) {
	parts := strings.Split(s, ":")

	if len(parts) == 2 {
		// Format: "MM:SS" - minutes:seconds
		minutes, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("invalid minutes: %s", parts[0])
		}

		seconds, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("invalid seconds: %s", parts[1])
		}

		return time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second, nil

	} else if len(parts) == 3 {
		// Format: "HH:MM:SS" - hours:minutes:seconds
		hours, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("invalid hours: %s", parts[0])
		}

		minutes, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("invalid minutes: %s", parts[1])
		}

		seconds, err := strconv.Atoi(parts[2])
		if err != nil {
			return 0, fmt.Errorf("invalid seconds: %s", parts[2])
		}

		return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second, nil
	}

	return 0, fmt.Errorf("invalid colon duration format: %s", s)
}

// parseExtendedDurationString parses "1d", "1w", "1M", "1Y" format
func (pm *PrecedenceManager) parseExtendedDurationString(s string) (time.Duration, bool) {
	if len(s) < 2 {
		return 0, false
	}

	// Find the last character (unit)
	lastChar := s[len(s)-1]
	unit := strings.ToLower(string(lastChar))

	var multiplier time.Duration
	switch unit {
	case "d":
		multiplier = 24 * time.Hour // 1 day = 24 hours
	case "w":
		multiplier = 7 * 24 * time.Hour // 1 week = 7 days
	case "m":
		// Check if it's 'M' (month) vs 'm' (minute) - only 'M' is extended
		if lastChar == 'M' {
			multiplier = 30 * 24 * time.Hour // 1 month = 30 days (assumption)
		} else {
			return 0, false // Regular minute - handled by standard parsing
		}
	case "y":
		multiplier = 365 * 24 * time.Hour // 1 year = 365 days (assumption)
	default:
		return 0, false
	}

	// Parse the number part
	numberStr := s[:len(s)-1]
	number, err := strconv.Atoi(numberStr)
	if err != nil {
		return 0, false
	}

	return time.Duration(number) * multiplier, true
}
