# Wrapper: echo with injected prefix

Simple command-level wrapper that prefixes all messages using `/bin/echo` (UNIX).

## Run

```
go run ./examples/wrapper-echo -- hello world
```

Outputs:

```
[prefix] hello world
```

