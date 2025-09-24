# Flag Groups Example

Demonstrates group constraints:

- MutuallyExclusive: choose exactly one of `--json`, `--yaml`, `--table`
- AllOrNone: SSL `--cert` and `--key` must be provided together
- ExactlyOne: output format enforced

## Run

```
go run ./examples/flag-groups --help
go run ./examples/flag-groups --json
go run ./examples/flag-groups --cert cert.pem --key key.pem
```
