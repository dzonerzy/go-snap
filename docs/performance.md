# Performance Notes

## Competitive Benchmarks

go-snap is **4-10x faster** than popular Go CLI libraries with significantly less memory usage:

### Benchmark Results Summary

| Scenario | go-snap | Cobra | urfave/cli |
|----------|---------|-------|------------|
| **Simple CLI** | 1.7 μs | 7.1 μs (4.2x slower) | 8.5 μs (5.0x slower) |
| **Subcommands** | 1.8 μs | 8.0 μs (4.5x slower) | 9.7 μs (5.5x slower) |
| **Many Flags (10)** | 2.0 μs | 9.0 μs (4.5x slower) | 21.0 μs (10.4x slower) |
| **Nested Commands** | 1.6 μs | 6.8 μs (4.3x slower) | 9.4 μs (5.9x slower) |

### Memory Usage

| Library | Memory per Operation | Allocations |
|---------|---------------------|-------------|
| **go-snap** | **5-6 KB** | **33-35** |
| Cobra | 17-20 KB (3x more) | 119-149 (3.6x more) |
| urfave/cli | 8-16 KB | 254-595 (8-17x more) |

**Key Takeaways:**
- go-snap is consistently **4-10x faster** across all scenarios
- Uses **3x less memory** than Cobra
- Makes **3.6-17x fewer allocations** (especially dramatic vs urfave/cli)
- Performance advantage increases with CLI complexity

*Benchmarked on AMD Ryzen 9 9950X3D 16-Core Processor. Full results: [benchmark/COMPETITIVE_BENCHMARKS.md](../benchmark/COMPETITIVE_BENCHMARKS.md)*

## Implementation Details

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
