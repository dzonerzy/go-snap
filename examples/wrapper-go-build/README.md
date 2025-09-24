# Wrapper: go build with toolexec logging

This example wraps `go build` and injects a `--toolexec` shim that logs every tool invocation (compile, link, asm, vet, ...).

## How it works

- `build` command wraps `go` and injects a single `--toolexec` value set to "<self> log".
- The go tool then calls the hidden `log` command for each tool with the pattern:
  
  `<self> log /path/to/tool <tool-args>`

- The `log` command is a dynamic wrapper (WrapDynamic) that:
  - forwards all flags/args to the tool (ForwardUnknownFlags)
  - prints a pretty log line to stderr
  - then executes the tool (Passthrough)

## Run

```
go run ./examples/wrapper-go-build build -o main ./examples/wrapper-go-build/main.go
```

You should see lines like:

```
[toolexec] compile "-importcfg" "..." "-p" "main" ...
[toolexec] link "-importcfg" "..." "-o" "..."
```

## Notes

- `${SELF}` token expands to the path of the running wrapper binary.
- We pass the entire toolexec value as a single argv element ("${SELF} log"), so `go` interprets it correctly.
- `ForwardUnknownFlags()` is required on the dynamic shim to avoid parse errors for flags intended for the tool.

