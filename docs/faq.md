# FAQ

Why do I get help instead of my command?
- If no command is provided and no app-level wrapper is configured, the app shows help.

How do global flags work?
- Flags defined with `.Global()` are available across commands and readable via `ctx.Global*` helpers.

What is the precedence of config sources?
- Flags > Environment > File (JSON) > Defaults.

Why only JSON for config files?
- The current code supports JSON only; other formats are future ideas.

Windows color output looks plain.
- The app attempts to enable VT processing automatically when writing to a TTY. Set `SNAP_DISABLE_VT=1` to opt out.

How do I forward unknown flags to a wrapped tool?
- Use `ForwardUnknownFlags()` on the wrapper builder.

Where are interactive prompts and shell completion?
- Not implemented in this codebase. They are planned features and not available here.
