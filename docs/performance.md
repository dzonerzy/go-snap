# Performance Notes

Zero-allocation parsing
- The parser uses pooled buffers and typed maps to avoid interface boxing and allocations on the hot path.
- String interning (`internal/intern`) removes duplicate allocations for flag/command names.
- Object pools (`internal/pool`) manage parse results and slices.

Typed storage
- Parsed values are stored in typed maps (string/int/bool/float/duration/enum) instead of `any`.
- Slice flags use pooled slices referenced by offsets.

Suggestions
- Fuzzy matching runs only on error paths and uses early-exit strategies to minimize cost.

Durations and floats
- CLI float parsing handles common cases; env/file parsing uses `strconv.ParseFloat`.
- Durations support extended formats (e.g., `MM:SS`, `HH:MM:SS`, `1d`, `1w`, `1M`, `1Y`, and Go style).

Help output
- Sorted output for deterministic help in flags/commands/groups.

Wrapper execution
- Passthrough streams directly; Capture uses buffers with optional tee to preserve performance.

Keep it fast
- Avoid unnecessary string concatenations in actions; write to `ctx.Stdout()`.
- Prefer validators over ad-hoc parsing in actions.

See also
- [Parsing & Context](./parsing-and-context.md)
- [Wrapper DSL](./wrapper.md)
