# Contributing to go-snap

Thanks for your interest in contributing! This project aims to be a fast, focused, and well-documented CLI library. Contributions that improve API quality, performance, docs, or examples are welcome.

Stable early beta focused on the core features. Please use
[GitHub Discussions](https://github.com/dzonerzy/go-snap/discussions)
to propose new features or share ideas.

## Requirements
- Go 1.22+
- A GitHub account to open issues and pull requests

## Quick Start (dev loop)
```bash
# clone your fork, then
make tidy   # optional: ensure modules are tidy
make build  # compile all packages
make test   # run tests
make bench  # quick smoke benchmark (non-blocking)
```

Before pushing a PR:
- run `make fmt vet test`
- ensure examples still compile: `make examples`
- update docs (see below) and CHANGELOG if the change is user-visible

## Code Style & Design
- Use idiomatic Go and keep APIs type-safe.
- Prefer minimal, composable APIs over feature bloat.
- Avoid breaking public APIs. If a breaking change is necessary, open an issue to discuss first; we follow semantic versioning.
- Keep zero-allocation and performance goals in mind; add a `*_test.go` or micro-bench if relevant.

## Tests & Benchmarks
- Add unit tests for new behavior.
- If your change affects performance or allocations, add a benchmark and include `go test -bench=. -benchmem` output in the PR description.

## Documentation
- All user docs live in `docs/`.
- Follow the guidance in “Contributing to Docs” inside `docs/README.md`.
- Keep intra-doc links relative (e.g., ``[Configuration](./docs/configuration.md)`` from the repo root or ``[Configuration](./configuration.md)`` from inside `docs/`).
- Add or update a small example in `examples/` when introducing new features.

## Commit Messages & PRs
- Descriptive commit messages are sufficient; conventional commits are welcome (e.g., `feat:`, `fix:`, `docs:`).
- Keep PRs scoped and focused. Include:
  - What changed and why
  - API surface touched (if any)
  - Tests and docs updates
  - Bench/alloc notes when applicable

## CI & Releases
- CI runs on pushes and PRs (`.github/workflows/ci.yml`).
- Releases are created automatically when a tag `vX.Y.Z` is pushed (`.github/workflows/release.yml`). Maintainers handle tagging.

## Reporting Issues & Asking Questions
- Use GitHub Issues for bugs and other actionable tasks.
- Use [GitHub Discussions](https://github.com/dzonerzy/go-snap/discussions) for feature proposals, roadmap ideas, and design questions.
- If unsure whether something belongs in core vs. an external helper, start a Discussion to explore trade-offs.

Thanks again for helping make go-snap better!
