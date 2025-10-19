# Lifecycle Hooks

Demonstrates Before/After hooks for commands and BeforeExec/AfterExec for wrapper commands.

Shows how to use lifecycle hooks for setup, validation, cleanup, logging, and metrics. Includes app-level hooks that run for all commands, command-specific hooks, and wrapper hooks for enhancing external command execution.

## Run

```
go run ./examples/lifecycle-hooks deploy --env prod
go run ./examples/lifecycle-hooks docker-build --tag v1.0.0
go run ./examples/lifecycle-hooks enhanced-test
```
