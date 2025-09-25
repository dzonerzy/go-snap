# Nested Subcommands

Minimal example showing nested commands (e.g., `myapp server up`).

## Commands

- `server up`
  - Starts the server
  - Flags: `--dry-run` (`-n`)

- `server down`
  - Stops the server
  - Flags: `--force` (`-f`)

## Run

```
# Start (dry-run)
go run ./examples/nested-subcommands server up --dry-run

# Stop (forced)
go run ./examples/nested-subcommands server down --force
```

## Output (example)

```
[dry-run] server up
server down (forced)
```
