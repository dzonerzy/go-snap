package snap

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "reflect"
    "strconv"
    "strings"
    "time"

    "github.com/dzonerzy/go-snap/middleware"
)

// D is a convenient type alias for default values map, similar to bson.D or gin.H
// Usage: snap.D{"host": "localhost", "port": 8080}
type D map[string]any


// FieldSchema defines the schema for a configuration field
type FieldSchema struct {
	Type        reflect.Type
	Required    bool
	Default     any
	Validator   func(any) error
	Description string
	JSONTag     string
	EnvTag      string
	FlagTag     string
	RequiredTag string
	DefaultTag  string
	DescTag     string
	EnumTag     string
	GroupTag    string
	IgnoreTag   string   // ignore:"true" - skip flag generation
	EnumValues  []string
	GroupName   string
	Ignored     bool     // Parsed from IgnoreTag
}

// parseFlagTagOptions parses flag tag to extract name and options
// Supports syntax like: flag:"port,required,ignore"
func parseFlagTagOptions(flagTag string) (name string, options map[string]bool) {
	options = make(map[string]bool)

	if flagTag == "" {
		return "", options
	}

	parts := strings.Split(flagTag, ",")
	name = strings.TrimSpace(parts[0])

	// Parse options from remaining parts
	for i := 1; i < len(parts); i++ {
		option := strings.TrimSpace(parts[i])
		if option != "" {
			options[option] = true
		}
	}

	return name, options
}

// GroupSchema represents schema for a flag group
type GroupSchema struct {
	Fields      []string
	Constraint  GroupConstraintType
	Description string
}

// ConfigSchema represents the overall configuration schema
type ConfigSchema struct {
	Fields map[string]*FieldSchema
	Groups map[string]*GroupSchema  // Enhanced group information
}

// ConfigBuilder provides fluent API for configuration management
type ConfigBuilder struct {
	app            *App
	schema         *ConfigSchema
	target         any
	precedenceManager *PrecedenceManager
	pendingSources []func()
	flagsEnabled   bool  // Track if FromFlags() was called - enables CLI generation
}

// Config creates a standalone configuration builder with app name and description
func Config(name, description string) *ConfigBuilder {
	// Create a minimal app instance for config functionality
	app := &App{
		name:         name,
		description:  description,
		authors:      make([]Author, 0),
		flags:        make(map[string]*Flag),
		shortFlags:   make(map[rune]*Flag),
		commands:     make(map[string]*Command),
		flagGroups:   make([]*FlagGroup, 0),
		helpFlag:     true,
		versionFlag:  false,
		errorHandler: NewErrorHandler(),
		middleware:   make([]middleware.Middleware, 0),
	}

	return newConfigBuilder(app)
}

// newConfigBuilder creates a new configuration builder (internal)
func newConfigBuilder(app *App) *ConfigBuilder {
	return &ConfigBuilder{
		app:               app,
		precedenceManager: NewPrecedenceManager(),
		pendingSources:    make([]func(), 0),
	}
}

// Bind binds the configuration to a struct and processes pending sources
func (cb *ConfigBuilder) Bind(target any) *ConfigBuilder {
	cb.target = target
	cb.schema = cb.generateSchema(target)

	// Process any pending source configurations
	for _, pendingSource := range cb.pendingSources {
		pendingSource()
	}
	cb.pendingSources = nil

	return cb
}

// FromDefaults adds default values as a configuration source
// Usage: .FromDefaults(snap.D{"host": "localhost", "port": 8080})
func (cb *ConfigBuilder) FromDefaults(defaults D) *ConfigBuilder {
	if cb.schema != nil {
		cb.precedenceManager.AddSource(SourceTypeDefaults, map[string]any(defaults))
	} else {
		cb.pendingSources = append(cb.pendingSources, func() {
			cb.precedenceManager.AddSource(SourceTypeDefaults, map[string]any(defaults))
		})
	}
	return cb
}

// FromFile adds file-based configuration source
func (cb *ConfigBuilder) FromFile(filename string) *ConfigBuilder {
	if cb.schema != nil {
		data, err := cb.loadFromFile(filename)
		if err == nil {
			cb.precedenceManager.AddSource(SourceTypeFile, data)
		}
	} else {
		cb.pendingSources = append(cb.pendingSources, func() {
			data, err := cb.loadFromFile(filename)
			if err == nil {
				cb.precedenceManager.AddSource(SourceTypeFile, data)
			}
		})
	}
	return cb
}

// FromEnv adds environment variable configuration source
func (cb *ConfigBuilder) FromEnv() *ConfigBuilder {
	if cb.schema != nil {
		data := cb.loadFromEnv()
		if len(data) > 0 {
			cb.precedenceManager.AddSource(SourceTypeEnv, data)
		}
	} else {
		cb.pendingSources = append(cb.pendingSources, func() {
			data := cb.loadFromEnv()
			if len(data) > 0 {
				cb.precedenceManager.AddSource(SourceTypeEnv, data)
			}
		})
	}
	return cb
}

// FromFlags enables CLI flag generation and adds flag-based configuration source
// This is the trigger that transforms a pure config loader into a full CLI application
func (cb *ConfigBuilder) FromFlags() *ConfigBuilder {
	cb.flagsEnabled = true  // Enable CLI functionality

	if cb.schema != nil {
		// Schema exists, generate flags immediately
		cb.generateFlags()
		cb.precedenceManager.AddSource(SourceTypeFlags, make(map[string]any))
	} else {
		// Schema not ready, defer flag generation until Bind()
		cb.pendingSources = append(cb.pendingSources, func() {
			cb.generateFlags()
			cb.precedenceManager.AddSource(SourceTypeFlags, make(map[string]any))
		})
	}
	return cb
}

// Build resolves the configuration and returns either an App (if FromFlags was used) or populates struct immediately
func (cb *ConfigBuilder) Build() (*App, error) {
	if cb.target == nil || cb.schema == nil {
		return nil, fmt.Errorf("must call Bind() before Build()")
	}

	if cb.flagsEnabled {
		// CLI mode: generate flags and return App for later Run()
		cb.generateFlags()

		// Store the config builder in the app for later use during Run()
		cb.app.configBuilder = cb

		return cb.app, nil
	} else {
		// Config-only mode: populate struct immediately and return nil App
		err := cb.buildConfigOnly()
		return nil, err
	}
}

// buildConfigOnly handles immediate config population (no CLI parsing)
func (cb *ConfigBuilder) buildConfigOnly() error {
	// Execute any pending source additions
	for _, addSource := range cb.pendingSources {
		addSource()
	}

	// Resolve configuration with precedence using the precedence manager
	resolved, err := cb.precedenceManager.ResolveWithSchema(cb.schema)
	if err != nil {
		return err
	}

	// Apply resolved configuration to target struct
	return cb.applyToStruct(resolved)
}

// generateSchema creates schema from struct reflection
func (cb *ConfigBuilder) generateSchema(target any) *ConfigSchema {
	schema := &ConfigSchema{
		Fields: make(map[string]*FieldSchema),
		Groups: make(map[string]*GroupSchema),
	}

	targetType := reflect.TypeOf(target)
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	cb.parseStructFields(targetType, "", schema)
	return schema
}

// parseStructFields recursively parses struct fields to build schema
func (cb *ConfigBuilder) parseStructFields(structType reflect.Type, prefix string, schema *ConfigSchema) {
	cb.parseStructFieldsWithGroup(structType, prefix, "", schema)
}

// parseStructFieldsWithGroup recursively parses struct fields with group inheritance
func (cb *ConfigBuilder) parseStructFieldsWithGroup(structType reflect.Type, prefix string, inheritedGroup string, schema *ConfigSchema) {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldType := field.Type

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		fieldName := cb.getFieldName(field, prefix)

		// Handle nested structs
		if fieldType.Kind() == reflect.Struct && fieldType != reflect.TypeOf(time.Time{}) && fieldType != reflect.TypeOf(time.Duration(0)) {
			nestedGroupName := field.Tag.Get("group")
			if nestedGroupName == "" {
				nestedGroupName = strings.ToLower(field.Name)
			}

			// Create or update group schema for nested struct
			cb.ensureGroupSchema(nestedGroupName, field, schema)

			// Process nested struct with its own group (not inherited group)
			cb.parseStructFieldsWithGroup(fieldType, fieldName+".", nestedGroupName, schema)
			continue
		}

		// Parse flag tag to extract name and options
		flagTag := field.Tag.Get("flag")
		flagName, flagOptions := parseFlagTagOptions(flagTag)

		// Create field schema
		fieldSchema := &FieldSchema{
			Type:        fieldType,
			JSONTag:     field.Tag.Get("json"),
			EnvTag:      field.Tag.Get("env"),
			FlagTag:     flagName, // Store just the name part
			RequiredTag: field.Tag.Get("required"),
			DefaultTag:  field.Tag.Get("default"),
			DescTag:     field.Tag.Get("description"),
			EnumTag:     field.Tag.Get("enum"),
			GroupTag:    field.Tag.Get("group"),
			IgnoreTag:   field.Tag.Get("ignore"),
		}

		// Parse ignore from flag options first, then fall back to separate ignore tag
		if flagOptions["ignore"] {
			fieldSchema.Ignored = true
		} else if ignoreValue, ignoreExists := field.Tag.Lookup("ignore"); ignoreExists {
			fieldSchema.Ignored = ignoreValue == "true" || ignoreValue == ""
		}

		// Skip ignored fields completely
		if fieldSchema.Ignored {
			continue
		}

		// Parse required from flag options first, then fall back to separate required tag
		if flagOptions["required"] {
			fieldSchema.Required = true
		} else if requiredValue, requiredExists := field.Tag.Lookup("required"); requiredExists {
			fieldSchema.Required = requiredValue == "true" || requiredValue == ""
		}

		// Parse default tag
		if fieldSchema.DefaultTag != "" {
			fieldSchema.Default = cb.parseDefaultValue(fieldSchema.DefaultTag, fieldType)
		}

		// Parse description tag
		if fieldSchema.DescTag != "" {
			fieldSchema.Description = fieldSchema.DescTag
		}

		// Parse enum tag
		if fieldSchema.EnumTag != "" {
			fieldSchema.EnumValues = strings.Split(fieldSchema.EnumTag, ",")
			for i, val := range fieldSchema.EnumValues {
				fieldSchema.EnumValues[i] = strings.TrimSpace(val)
			}
		}

		// Set group name - use explicit group tag or inherited group
		if fieldSchema.GroupTag != "" {
			fieldSchema.GroupName = fieldSchema.GroupTag
		} else if inheritedGroup != "" {
			fieldSchema.GroupName = inheritedGroup
		}

		schema.Fields[fieldName] = fieldSchema

		// Add to group if specified
		if fieldSchema.GroupName != "" {
			if schema.Groups[fieldSchema.GroupName] == nil {
				schema.Groups[fieldSchema.GroupName] = &GroupSchema{
					Fields:      make([]string, 0),
					Constraint:  GroupNoConstraint, // Default
					Description: "",
				}
			}
			schema.Groups[fieldSchema.GroupName].Fields = append(schema.Groups[fieldSchema.GroupName].Fields, fieldName)
		}
	}
}

// ensureGroupSchema creates or updates group schema from struct field tags
func (cb *ConfigBuilder) ensureGroupSchema(groupName string, field reflect.StructField, schema *ConfigSchema) {
	if schema.Groups[groupName] == nil {
		schema.Groups[groupName] = &GroupSchema{
			Fields:      make([]string, 0),
			Constraint:  GroupNoConstraint, // Default
			Description: "",
		}
	}

	groupSchema := schema.Groups[groupName]

	// Parse group_constraint tag
	if constraintTag := field.Tag.Get("group_constraint"); constraintTag != "" {
		switch strings.ToLower(constraintTag) {
		case "mutually", "mutually_exclusive":
			groupSchema.Constraint = GroupMutuallyExclusive
		case "all_or_none":
			groupSchema.Constraint = GroupAllOrNone
		case "at_least_one", "one_or_more":
			groupSchema.Constraint = GroupAtLeastOne
		case "exactly_one":
			groupSchema.Constraint = GroupExactlyOne
		default:
			groupSchema.Constraint = GroupNoConstraint
		}
	}

	// Parse group_description tag
	if descTag := field.Tag.Get("group_description"); descTag != "" {
		groupSchema.Description = descTag
	} else if groupSchema.Description == "" {
		// Fallback to auto-generated description only if not already set
		groupSchema.Description = fmt.Sprintf("%s configuration", strings.ToUpper(string(groupName[0]))+groupName[1:])
	}
}

// getFieldName determines the field name for configuration
func (cb *ConfigBuilder) getFieldName(field reflect.StructField, prefix string) string {
	// Priority: flag tag > json tag > field name
	if flagTag := field.Tag.Get("flag"); flagTag != "" {
		// Parse flag tag to extract just the name part (ignore options)
		flagName, _ := parseFlagTagOptions(flagTag)
		if flagName != "" {
			return prefix + flagName
		}
	}
	if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
		parts := strings.Split(jsonTag, ",")
		return prefix + parts[0]
	}
	return prefix + strings.ToLower(field.Name)
}

// parseDefaultValue parses default value string to appropriate type
func (cb *ConfigBuilder) parseDefaultValue(defaultStr string, fieldType reflect.Type) any {
	pm := NewPrecedenceManager()
	switch fieldType.Kind() {
	case reflect.String:
		return defaultStr
	case reflect.Bool:
		val, _ := pm.parseBoolString(defaultStr)
		return val
	case reflect.Int:
		val, _ := strconv.Atoi(defaultStr)
		return val
	case reflect.Int64:
		if fieldType == reflect.TypeOf(time.Duration(0)) {
			val, _ := pm.parseDurationString(defaultStr)
			return val
		}
		val, _ := strconv.ParseInt(defaultStr, 10, 64)
		return val
	case reflect.Float64:
		val, _ := strconv.ParseFloat(defaultStr, 64)
		return val
	case reflect.Slice:
		// Handle slice types
		if fieldType.Elem().Kind() == reflect.String {
			return cb.parseStringSliceString(defaultStr)
		} else if fieldType.Elem().Kind() == reflect.Int {
			val, _ := cb.parseIntSliceString(defaultStr)
			return val
		}
		return defaultStr
	default:
		return defaultStr
	}
}

// generateFlags automatically generates CLI flags from schema
func (cb *ConfigBuilder) generateFlags() {
	if cb.app == nil {
		return
	}

    // Create flag groups first
    groupBuilders := make(map[string]*FlagGroupBuilder[*App])
	for groupName, groupSchema := range cb.schema.Groups {
		groupBuilder := cb.app.FlagGroup(groupName).Description(groupSchema.Description)

		// Apply group constraint based on schema
		switch groupSchema.Constraint {
		case GroupMutuallyExclusive:
			groupBuilder = groupBuilder.MutuallyExclusive()
		case GroupAllOrNone:
			groupBuilder = groupBuilder.AllOrNone()
		case GroupAtLeastOne:
			groupBuilder = groupBuilder.AtLeastOne()
		case GroupExactlyOne:
			groupBuilder = groupBuilder.ExactlyOne()
		case GroupNoConstraint:
			// No constraint needed - flags work independently
		}

		groupBuilders[groupName] = groupBuilder
	}

	// Generate flags for each field
	for fieldName, fieldSchema := range cb.schema.Fields {
		flagName := fieldName
		if fieldSchema.FlagTag != "" {
			flagName = fieldSchema.FlagTag
		}

		description := fieldSchema.Description
		if description == "" {
			description = fmt.Sprintf("Configuration for %s", fieldName)
		}

		// Determine if this flag belongs to a group
		var flagBuilder interface{}

		if fieldSchema.GroupName != "" && groupBuilders[fieldSchema.GroupName] != nil {
			// Add flag to group
			groupBuilder := groupBuilders[fieldSchema.GroupName]

			// Generate appropriate flag type based on field type
			switch fieldSchema.Type.Kind() {
            case reflect.String:
                if len(fieldSchema.EnumValues) > 0 {
                    // For enum fields, add validation note and create enum flag
                    description += fmt.Sprintf(" (valid values: %s)", strings.Join(fieldSchema.EnumValues, ", "))
                    flagBuilder = groupBuilder.EnumFlag(flagName, description, fieldSchema.EnumValues...)
                } else {
                    flagBuilder = groupBuilder.StringFlag(flagName, description)
                }
            case reflect.Bool:
                flagBuilder = groupBuilder.BoolFlag(flagName, description)
			case reflect.Int:
				flagBuilder = groupBuilder.IntFlag(flagName, description)
			case reflect.Int64:
				if fieldSchema.Type == reflect.TypeOf(time.Duration(0)) {
					flagBuilder = groupBuilder.DurationFlag(flagName, description)
				} else {
					flagBuilder = groupBuilder.IntFlag(flagName, description)
				}
			case reflect.Float64:
				flagBuilder = groupBuilder.FloatFlag(flagName, description)
            case reflect.Slice:
                if fieldSchema.Type.Elem().Kind() == reflect.String {
                    flagBuilder = groupBuilder.StringSliceFlag(flagName, description)
                } else if fieldSchema.Type.Elem().Kind() == reflect.Int {
                    flagBuilder = groupBuilder.IntSliceFlag(flagName, description)
                }
            }
        } else {
            // Add flag directly to app
            switch fieldSchema.Type.Kind() {
            case reflect.String:
                if len(fieldSchema.EnumValues) > 0 {
                    // For enum fields, add validation note and create enum flag
                    description += fmt.Sprintf(" (valid values: %s)", strings.Join(fieldSchema.EnumValues, ", "))
                    flagBuilder = cb.app.EnumFlag(flagName, description, fieldSchema.EnumValues...)
                } else {
                    flagBuilder = cb.app.StringFlag(flagName, description)
                }
            case reflect.Bool:
                flagBuilder = cb.app.BoolFlag(flagName, description)
            case reflect.Int:
                flagBuilder = cb.app.IntFlag(flagName, description)
            case reflect.Int64:
                if fieldSchema.Type == reflect.TypeOf(time.Duration(0)) {
                    flagBuilder = cb.app.DurationFlag(flagName, description)
                } else {
                    flagBuilder = cb.app.IntFlag(flagName, description)
                }
            case reflect.Float64:
                flagBuilder = cb.app.FloatFlag(flagName, description)
            case reflect.Slice:
                if fieldSchema.Type.Elem().Kind() == reflect.String {
                    flagBuilder = cb.app.StringSliceFlag(flagName, description)
                } else if fieldSchema.Type.Elem().Kind() == reflect.Int {
                    flagBuilder = cb.app.IntSliceFlag(flagName, description)
                }
            }
        }

        // Apply common flag settings based on field type
        cb.applyFlagSettings(flagBuilder, fieldSchema)
    }

	// Close all flag groups
	for _, groupBuilder := range groupBuilders {
		groupBuilder.EndGroup()
	}
}

// applyFlagSettings applies common settings to flag builders
func (cb *ConfigBuilder) applyFlagSettings(flagBuilder interface{}, fieldSchema *FieldSchema) {
	// Apply settings based on the flag builder type using type assertions
    switch fb := flagBuilder.(type) {
    case *FlagBuilder[string, *App]:
        if fieldSchema.Default != nil {
            fb.Default(fieldSchema.Default.(string))
        }
        if fieldSchema.Required {
            fb.Required()
        }
        fb.Global()
    case *FlagBuilder[string, *FlagGroupBuilder[*App]]:
        if fieldSchema.Default != nil {
            fb.Default(fieldSchema.Default.(string))
        }
        if fieldSchema.Required {
            fb.Required()
        }
        fb.Global()
    case *FlagBuilder[bool, *App]:
        if fieldSchema.Default != nil {
            fb.Default(fieldSchema.Default.(bool))
        }
        fb.Global()
    case *FlagBuilder[bool, *FlagGroupBuilder[*App]]:
        if fieldSchema.Default != nil {
            fb.Default(fieldSchema.Default.(bool))
        }
        fb.Global()
    case *FlagBuilder[int, *App]:
        if fieldSchema.Default != nil {
            fb.Default(fieldSchema.Default.(int))
        }
        if fieldSchema.Required {
            fb.Required()
        }
        fb.Global()
    case *FlagBuilder[int, *FlagGroupBuilder[*App]]:
        if fieldSchema.Default != nil {
            fb.Default(fieldSchema.Default.(int))
        }
        if fieldSchema.Required {
            fb.Required()
        }
        fb.Global()
    case *FlagBuilder[time.Duration, *App]:
        if fieldSchema.Default != nil {
            fb.Default(fieldSchema.Default.(time.Duration))
        }
        if fieldSchema.Required {
            fb.Required()
        }
        fb.Global()
    case *FlagBuilder[time.Duration, *FlagGroupBuilder[*App]]:
        if fieldSchema.Default != nil {
            fb.Default(fieldSchema.Default.(time.Duration))
        }
        if fieldSchema.Required {
            fb.Required()
        }
        fb.Global()
    case *FlagBuilder[float64, *App]:
        if fieldSchema.Default != nil {
            fb.Default(fieldSchema.Default.(float64))
        }
        if fieldSchema.Required {
            fb.Required()
        }
        fb.Global()
    case *FlagBuilder[float64, *FlagGroupBuilder[*App]]:
        if fieldSchema.Default != nil {
            fb.Default(fieldSchema.Default.(float64))
        }
        if fieldSchema.Required {
            fb.Required()
        }
        fb.Global()
    case *FlagBuilder[[]string, *App]:
        if fieldSchema.Default != nil {
            fb.Default(fieldSchema.Default.([]string))
        }
        if fieldSchema.Required {
            fb.Required()
        }
        fb.Global()
    case *FlagBuilder[[]string, *FlagGroupBuilder[*App]]:
        if fieldSchema.Default != nil {
            fb.Default(fieldSchema.Default.([]string))
        }
        if fieldSchema.Required {
            fb.Required()
        }
        fb.Global()
    case *FlagBuilder[[]int, *App]:
        if fieldSchema.Default != nil {
            fb.Default(fieldSchema.Default.([]int))
        }
        if fieldSchema.Required {
            fb.Required()
        }
        fb.Global()
    case *FlagBuilder[[]int, *FlagGroupBuilder[*App]]:
        if fieldSchema.Default != nil {
            fb.Default(fieldSchema.Default.([]int))
        }
        if fieldSchema.Required {
            fb.Required()
        }
        fb.Global()
    }
}

// loadFromFile loads configuration from JSON file
func (cb *ConfigBuilder) loadFromFile(filename string) (map[string]any, error) {
    // Support JSON only; ignore other formats by returning an error so caller skips adding the source
    ext := strings.ToLower(filepath.Ext(filename))
    if ext != ".json" {
        return nil, fmt.Errorf("unsupported config format: %s (only .json supported)", ext)
    }

    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }

    var config map[string]any
    if err := json.Unmarshal(data, &config); err != nil {
        return nil, err
    }

    return config, nil
}

// loadFromEnv loads configuration from environment variables based on struct tags
func (cb *ConfigBuilder) loadFromEnv() map[string]any {
	data := make(map[string]any)

	for fieldName, fieldSchema := range cb.schema.Fields {
		if fieldSchema.EnvTag != "" {
			if value := os.Getenv(fieldSchema.EnvTag); value != "" {
				data[fieldName] = value
			}
		}
	}

	return data
}

// collectFlagValues collects values from parsed flags
func (cb *ConfigBuilder) collectFlagValues() {
    flagData := make(map[string]any)

    // Collect flag values from app context
    for fieldName, fieldSchema := range cb.schema.Fields {
        flagName := fieldName
        if fieldSchema.FlagTag != "" {
            flagName = fieldSchema.FlagTag
        }

        // Try to get flag value based on type
        switch fieldSchema.Type.Kind() {
        case reflect.String:
            // If schema declares enum values, pull from enum storage first
            if len(fieldSchema.EnumValues) > 0 {
                if value, exists := cb.app.getEnumFlagValue(flagName); exists {
                    // Skip if equals default to avoid promoting defaults to Flags precedence
                    if def, ok := fieldSchema.Default.(string); !ok || value != def {
                        flagData[fieldName] = value
                    }
                }
            } else {
                if value, exists := cb.app.getStringFlagValue(flagName); exists {
                    if def, ok := fieldSchema.Default.(string); !ok || value != def {
                        flagData[fieldName] = value
                    }
                }
            }
        case reflect.Bool:
            if value, exists := cb.app.getBoolFlagValue(flagName); exists {
                if def, ok := fieldSchema.Default.(bool); !ok || value != def {
                    flagData[fieldName] = value
                }
            }
        case reflect.Int:
            if value, exists := cb.app.getIntFlagValue(flagName); exists {
                if def, ok := fieldSchema.Default.(int); !ok || value != def {
                    flagData[fieldName] = value
                }
            }
        case reflect.Int64:
            if fieldSchema.Type == reflect.TypeOf(time.Duration(0)) {
                if value, exists := cb.app.getDurationFlagValue(flagName); exists {
                    if def, ok := fieldSchema.Default.(time.Duration); !ok || value != def {
                        flagData[fieldName] = value
                    }
                }
            }
        case reflect.Float64:
            if value, exists := cb.app.getFloatFlagValue(flagName); exists {
                if def, ok := fieldSchema.Default.(float64); !ok || value != def {
                    flagData[fieldName] = value
                }
            }
        case reflect.Slice:
            // Handle []string and []int
            if fieldSchema.Type.Elem().Kind() == reflect.String {
                if value, exists := cb.app.getStringSliceFlagValue(flagName); exists {
                    // compare lengths and items if default exists
                    if def, ok := fieldSchema.Default.([]string); ok {
                        if len(def) == len(value) {
                            equal := true
                            for i := range def { if def[i] != value[i] { equal = false; break } }
                            if !equal { flagData[fieldName] = value }
                        } else { flagData[fieldName] = value }
                    } else {
                        flagData[fieldName] = value
                    }
                }
            } else if fieldSchema.Type.Elem().Kind() == reflect.Int {
                if value, exists := cb.app.getIntSliceFlagValue(flagName); exists {
                    if def, ok := fieldSchema.Default.([]int); ok {
                        if len(def) == len(value) {
                            equal := true
                            for i := range def { if def[i] != value[i] { equal = false; break } }
                            if !equal { flagData[fieldName] = value }
                        } else { flagData[fieldName] = value }
                    } else {
                        flagData[fieldName] = value
                    }
                }
            }
        }
    }

    // Add collected flag data to precedence manager
    if len(flagData) > 0 {
        cb.precedenceManager.AddSource(SourceTypeFlags, flagData)
    }
}

// Helper methods to get flag values from current parse result
func (a *App) getStringFlagValue(name string) (string, bool) {
    if a.currentResult == nil {
        return "", false
    }
    if v, ok := a.currentResult.GetString(name); ok {
        return v, true
    }
    return a.currentResult.GetGlobalString(name)
}

func (a *App) getBoolFlagValue(name string) (bool, bool) {
    if a.currentResult == nil {
        return false, false
    }
    if v, ok := a.currentResult.GetBool(name); ok {
        return v, true
    }
    return a.currentResult.GetGlobalBool(name)
}

func (a *App) getIntFlagValue(name string) (int, bool) {
    if a.currentResult == nil {
        return 0, false
    }
    if v, ok := a.currentResult.GetInt(name); ok {
        return v, true
    }
    return a.currentResult.GetGlobalInt(name)
}

func (a *App) getDurationFlagValue(name string) (time.Duration, bool) {
    if a.currentResult == nil {
        return 0, false
    }
    if v, ok := a.currentResult.GetDuration(name); ok {
        return v, true
    }
    return a.currentResult.GetGlobalDuration(name)
}

func (a *App) getFloatFlagValue(name string) (float64, bool) {
    if a.currentResult == nil {
        return 0, false
    }
    if v, ok := a.currentResult.GetFloat(name); ok {
        return v, true
    }
    return a.currentResult.GetGlobalFloat(name)
}

// slice and enum helpers for collectFlagValues
func (a *App) getStringSliceFlagValue(name string) ([]string, bool) {
    if a.currentResult == nil {
        return nil, false
    }
    if v, ok := a.currentResult.GetStringSlice(name); ok {
        return v, true
    }
    // also check global
    if v, ok := a.currentResult.GetGlobalStringSlice(name); ok {
        return v, true
    }
    return nil, false
}

func (a *App) getIntSliceFlagValue(name string) ([]int, bool) {
    if a.currentResult == nil {
        return nil, false
    }
    if v, ok := a.currentResult.GetIntSlice(name); ok {
        return v, true
    }
    if v, ok := a.currentResult.GetGlobalIntSlice(name); ok {
        return v, true
    }
    return nil, false
}

func (a *App) getEnumFlagValue(name string) (string, bool) {
    if a.currentResult == nil {
        return "", false
    }
    if v, ok := a.currentResult.GetEnum(name); ok {
        return v, true
    }
    if v, ok := a.currentResult.GetGlobalEnum(name); ok {
        return v, true
    }
    // Fallback to string map if stored there
    return a.currentResult.GetString(name)
}



// applyToStruct applies the resolved configuration to the target struct
func (cb *ConfigBuilder) applyToStruct(config map[string]any) error {
	targetValue := reflect.ValueOf(cb.target)
	if targetValue.Kind() != reflect.Ptr || targetValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}

	targetStruct := targetValue.Elem()
	return cb.setStructFields(targetStruct, targetStruct.Type(), "", config)
}

// setStructFields recursively sets struct fields from configuration
func (cb *ConfigBuilder) setStructFields(structValue reflect.Value, structType reflect.Type, prefix string, config map[string]any) error {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := structValue.Field(i)

		if !field.IsExported() || !fieldValue.CanSet() {
			continue
		}

		fieldName := cb.getFieldName(field, prefix)

		// Handle nested structs
		if fieldValue.Kind() == reflect.Struct && field.Type != reflect.TypeOf(time.Time{}) && field.Type != reflect.TypeOf(time.Duration(0)) {
			if err := cb.setStructFields(fieldValue, field.Type, fieldName+".", config); err != nil {
				return err
			}
			continue
		}

		// Set field value if present in config
		if value, exists := config[fieldName]; exists {
			if err := cb.setFieldValue(fieldValue, value); err != nil {
				return fmt.Errorf("failed to set field %s: %w", fieldName, err)
			}
		}
	}

	return nil
}

// setFieldValue sets a single field value with type conversion
func (cb *ConfigBuilder) setFieldValue(fieldValue reflect.Value, value any) error {
	valueReflect := reflect.ValueOf(value)

	// Direct assignment if types match
	if valueReflect.Type() == fieldValue.Type() {
		fieldValue.Set(valueReflect)
		return nil
	}

	// Type conversion if possible
	if valueReflect.Type().ConvertibleTo(fieldValue.Type()) {
		fieldValue.Set(valueReflect.Convert(fieldValue.Type()))
		return nil
	}

	// String conversion
	if valueReflect.Kind() == reflect.String {
		pm := NewPrecedenceManager()
		convertedValue, err := pm.convertStringToType(value.(string), fieldValue.Type())
		if err != nil {
			return err
		}
		fieldValue.Set(reflect.ValueOf(convertedValue))
		return nil
	}

	return fmt.Errorf("cannot convert %T to %s", value, fieldValue.Type())
}


// parseStringSliceString parses comma-separated strings: "item1,item2,item3"
func (cb *ConfigBuilder) parseStringSliceString(s string) []string {
	if s == "" {
		return []string{}
	}

	parts := strings.Split(s, ",")
	result := make([]string, len(parts))
	for i, part := range parts {
		result[i] = strings.TrimSpace(part)
	}
	return result
}

// parseIntSliceString parses comma-separated integers: "1,2,3"
func (cb *ConfigBuilder) parseIntSliceString(s string) ([]int, error) {
	if s == "" {
		return []int{}, nil
	}

	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		value, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid integer in slice: %s", part)
		}
		result = append(result, value)
	}

	return result, nil
}
