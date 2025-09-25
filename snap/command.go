package snap

import (
	"time"

	"github.com/dzonerzy/go-snap/middleware"
)

// Command represents a CLI command or subcommand
type Command struct {
	name        string
	description string
	HelpText    string
	Aliases     []string
	Hidden      bool
	flags       map[string]*Flag
	shortFlags  map[rune]*Flag // O(1) lookup for short flags
	subcommands map[string]*Command
	flagGroups  []*FlagGroup // Flag groups for validation
	Action      ActionFunc
	middleware  []middleware.Middleware // Command-level middleware
	wrapper     *WrapperSpec            // Optional wrapper configuration
}

// Name returns the command name (implements middleware.Command interface)
func (c *Command) Name() string {
	return c.name
}

// Description returns the command description (implements middleware.Command interface)
func (c *Command) Description() string {
	return c.description
}

// CommandBuilder provides fluent API for building commands
type CommandBuilder struct {
	command *Command
	app     *App
}

// Command configuration methods

// Alias adds aliases for the command
func (c *CommandBuilder) Alias(aliases ...string) *CommandBuilder {
	c.command.Aliases = append(c.command.Aliases, aliases...)
	return c
}

// Action sets the action function for the command
func (c *CommandBuilder) Action(fn ActionFunc) *CommandBuilder {
	c.command.Action = fn
	return c
}

// Hidden marks the command as hidden from help
func (c *CommandBuilder) Hidden() *CommandBuilder {
	c.command.Hidden = true
	return c
}

// HelpText sets detailed help text for the command
func (c *CommandBuilder) HelpText(help string) *CommandBuilder {
	c.command.HelpText = help
	return c
}

// Use adds middleware to the command
func (c *CommandBuilder) Use(middleware ...middleware.Middleware) *CommandBuilder {
	c.command.middleware = append(c.command.middleware, middleware...)
	return c
}

// Flag builders for command-specific flags

// StringFlag adds a string flag to the command
func (c *CommandBuilder) StringFlag(name, description string) *FlagBuilder[string, *CommandBuilder] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeString,
	}
	c.command.flags[name] = flag
	return &FlagBuilder[string, *CommandBuilder]{flag: flag, parent: c}
}

// IntFlag adds an integer flag to the command
func (c *CommandBuilder) IntFlag(name, description string) *FlagBuilder[int, *CommandBuilder] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeInt,
	}
	c.command.flags[name] = flag
	return &FlagBuilder[int, *CommandBuilder]{flag: flag, parent: c}
}

// BoolFlag adds a boolean flag to the command
func (c *CommandBuilder) BoolFlag(name, description string) *FlagBuilder[bool, *CommandBuilder] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeBool,
	}
	c.command.flags[name] = flag
	return &FlagBuilder[bool, *CommandBuilder]{flag: flag, parent: c}
}

// DurationFlag adds a duration flag to the command
func (c *CommandBuilder) DurationFlag(name, description string) *FlagBuilder[time.Duration, *CommandBuilder] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeDuration,
	}
	c.command.flags[name] = flag
	return &FlagBuilder[time.Duration, *CommandBuilder]{flag: flag, parent: c}
}

// FloatFlag adds a float64 flag to the command
func (c *CommandBuilder) FloatFlag(name, description string) *FlagBuilder[float64, *CommandBuilder] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeFloat,
	}
	c.command.flags[name] = flag
	return &FlagBuilder[float64, *CommandBuilder]{flag: flag, parent: c}
}

// EnumFlag adds an enum flag to the command
func (c *CommandBuilder) EnumFlag(name, description string, values ...string) *FlagBuilder[string, *CommandBuilder] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeEnum,
		EnumValues:  values,
	}
	c.command.flags[name] = flag
	return &FlagBuilder[string, *CommandBuilder]{flag: flag, parent: c}
}

// StringSliceFlag adds a string slice flag to the command
func (c *CommandBuilder) StringSliceFlag(name, description string) *FlagBuilder[[]string, *CommandBuilder] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeStringSlice,
	}
	c.command.flags[name] = flag
	return &FlagBuilder[[]string, *CommandBuilder]{flag: flag, parent: c}
}

// IntSliceFlag adds an int slice flag to the command
func (c *CommandBuilder) IntSliceFlag(name, description string) *FlagBuilder[[]int, *CommandBuilder] {
	flag := &Flag{
		Name:        name,
		Description: description,
		Type:        FlagTypeIntSlice,
	}
	c.command.flags[name] = flag
	return &FlagBuilder[[]int, *CommandBuilder]{flag: flag, parent: c}
}

// Subcommand builder

// Command adds a subcommand to this command
func (c *CommandBuilder) Command(name, description string) *CommandBuilder {
	cmd := &Command{
		name:        name,
		description: description,
		Aliases:     make([]string, 0),
		Hidden:      false,
		flags:       make(map[string]*Flag),
		shortFlags:  make(map[rune]*Flag),
		subcommands: make(map[string]*Command),
		flagGroups:  make([]*FlagGroup, 0),
		middleware:  make([]middleware.Middleware, 0),
	}
	c.app.addCommandHelpFlag(cmd)
	c.command.subcommands[name] = cmd
	return &CommandBuilder{
		command: cmd,
		app:     c.app,
	}
}

// FlagParent interface implementation

// addShortFlag adds a short flag mapping for O(1) lookup to the command
func (c *CommandBuilder) addShortFlag(short rune, flag *Flag) {
	c.command.shortFlags[short] = flag
}

// addFlagGroup adds a flag group to the command (implements FlagGroupParent interface)
func (c *CommandBuilder) addFlagGroup(group *FlagGroup) {
	c.command.flagGroups = append(c.command.flagGroups, group)

	// Also add all flags in the group to the command's flag map for parsing
	for _, flag := range group.Flags {
		c.command.flags[flag.Name] = flag
		if flag.Short != 0 {
			c.command.shortFlags[flag.Short] = flag
		}
	}
}

// FlagGroup creates a new flag group builder for the command
func (c *CommandBuilder) FlagGroup(name string) *FlagGroupBuilder[*CommandBuilder] {
	group := &FlagGroup{
		Name:  name,
		Flags: make([]*Flag, 0),
	}
	return &FlagGroupBuilder[*CommandBuilder]{
		group:  group,
		parent: c,
	}
}

// Builder termination

// App returns to the app for continued chaining
func (c *CommandBuilder) App() *App {
	return c.app
}

// Build finalizes the command configuration
func (c *CommandBuilder) Build() *CommandBuilder {
	return c
}
