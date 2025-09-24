# Exit Codes Example

Shows how to map errors to process exit codes and how to request exit from inside a command action.

## Run

```
go run ./examples/exit-codes success
go run ./examples/exit-codes not-found
go run ./examples/exit-codes custom-exit
echo $?   # inspect exit code
```
