# Best Practices

Designing flags
- Prefer typed defaults (`Default`) over post-parse fallback logic.
- Use flag groups to encode mutual-exclusion and all-or-none relationships.
- Bind env vars with `FromEnv` to honor container/CI environments.

Validation boundaries
- Structural validity (required, mutual exclusion) → use flag groups.
- Business logic (ranges, file existence, conditional requirements) → use middleware `Validate(...)`.

Error handling
- Keep user-facing errors short and actionable.
- Leverage smart suggestions; tune with `ErrorHandler().MaxDistance(n)`.
- Map domain errors to exit codes via `ExitCodes()`.

Middleware
- Apply global logger/recovery; use timeouts where commands may block.
- Prefer `RecoveryToError()` in production to avoid noisy stacks; print stacks in dev.

Configuration
- Keep a single source of truth in the struct definition; let `FromFlags()` create CLI flags automatically.
- Use `enum` tags to constrain values (the parser validates enums on input).

Wrapper DSL
- Use `ForwardUnknownFlags()` when wrapping complex CLIs to minimize maintenance.
- With echo-style tools, declare `LeadingFlags` and `InsertAfterLeadingFlags` to preserve familiar flag order.
- For toolexec shims, keep `log`/shim commands hidden from help.

IO/TTY
- Respect `IsInteractive()` for UI/pretty output and fall back to plain output when piped.
- Defer to `IO().SupportsColor()` and `ColorLevel()` before using styles.

Windows
- VT enablement is best-effort and automatic when writing to a TTY. Respect `SNAP_DISABLE_VT` to disable.
