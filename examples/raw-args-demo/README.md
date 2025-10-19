# Raw Args Demo

Demonstrates RawArgs() usage for audit logging, debugging, and proxying commands.

Shows the difference between raw arguments (as typed by user) and parsed arguments (after flag processing), useful for audit trails, debugging, and forwarding commands to external tools.

## Run

```
go run ./examples/raw-args-demo audit --verbose serve --port 8080 file1.txt
go run ./examples/raw-args-demo debug -abc test
go run ./examples/raw-args-demo compare --verbose process --workers 4 data.txt
go run ./examples/raw-args-demo proxy --any args here
```
