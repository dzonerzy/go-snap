# go-snap Documentation

Welcome to the official documentation for the go-snap CLI library.

go-snap is a lean, high-performance CLI toolkit for Go focused on:
- Zero-allocation parsing (fast, GC-friendly)
- Type-safe fluent builders for apps, commands, flags and groups
- Smart errors with suggestions and contextual help
- Configuration from struct tags with source precedence and auto flag generation
- First-class middleware (logger, recovery, timeout, validator)
- A powerful wrapper DSL for enhancing existing CLIs (incl. dynamic toolexec shims)
- Clean IO/terminal detection with ANSI color helpers (Windows VT supported)

This documentation reflects the current implementation in this repository.

Note: Stable early beta focused on the core features. Please use [GitHub Discussions](https://github.com/dzonerzy/go-snap/discussions) to propose new features or share ideas.

Quick links
- [Quick Start](./quickstart.md)
- [App & Commands](./app-and-commands.md)
- [Flags & Flag Groups](./flags-and-groups.md)
- [Parsing & Context](./parsing-and-context.md)
- [Configuration (struct tags + precedence)](./configuration.md)
- [Middleware](./middleware.md)
- [IO & Color](./io-and-color.md)
- [Wrapper DSL](./wrapper.md)
- [Errors & Exit Codes](./errors-and-exit-codes.md)
- [Best Practices](./best-practices.md)
- [Migration from Cobra / urfave/cli](./migration.md)
- [Performance Notes](./performance.md)
- [Examples Tour](./examples.md)
- [FAQ](./faq.md)



Link graph
```
README -> quickstart
       -> app-and-commands -> flags-and-groups -> parsing-and-context
       -> configuration ----^                \-> errors-and-exit-codes
       -> middleware ------------------------/
       -> io-and-color
       -> wrapper
       -> best-practices
       -> migration
       -> performance
       -> examples
       -> faq
```

---

Package imports used in examples
- Core: `github.com/dzonerzy/go-snap/snap`
- Middleware: `github.com/dzonerzy/go-snap/middleware`
- IO/Color: `github.com/dzonerzy/go-snap/io` (package `snapio`)

---

## Contributing to Docs

Keep it brief, accurate, and cross-linked.

- Scope: Document only what exists in the codebase. Avoid future/roadmap items in main flows.
- Links: Use relative links like `[Configuration](./configuration.md)`; prefer section anchors for deep links.
- Style: Favor small, runnable code snippets that import:
  - `github.com/dzonerzy/go-snap/snap`
  - `github.com/dzonerzy/go-snap/middleware`
  - `github.com/dzonerzy/go-snap/io` as `snapio`
- Headings: Start at `#` for page title, then `##`/`###` for subsections.
- Examples: Prefer borrowing from `examples/` to ensure parity with code.
- Gotchas: Call out pitfalls and precedence rules explicitly.

Quick checklist before opening a PR:
- [ ] Added/updated a page under `docs/` with consistent headings.
- [ ] All intra-doc links are relative and valid.
- [ ] Snippets compile against current package paths.
- [ ] No references to removed/speculative files.
