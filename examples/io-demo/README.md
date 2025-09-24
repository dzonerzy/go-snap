# IO Demo

Demonstrates IOManager usage: color, TTY detection, width/height, piped/redirected.

## Run

```
go run ./examples/io-demo
# Force color even if piped
FORCE_COLOR=1 go run ./examples/io-demo | cat
```
