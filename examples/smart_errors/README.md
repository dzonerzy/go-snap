# Smart Errors

Demonstrates smart error handling with flag and command suggestions.

Shows how the error handler provides helpful suggestions when users make typos in flags or commands, using fuzzy matching with configurable edit distance.

## Run

```
go run ./examples/smart_errors --jsn
go run ./examples/smart_errors --prot 3000
go run ./examples/smart_errors --help
```

Try misspelling flags like `--jsn` (suggests `--json`), `--prot` (suggests `--port`), or `--tabel` (suggests `--table`) to see smart error suggestions in action.
