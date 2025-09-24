## Simple build automation for go-snap

GO ?= go
PKG := ./...

.PHONY: help build test bench fmt vet tidy examples

help:
	@echo "Targets:"
	@echo "  build     - build all packages"
	@echo "  test      - run unit tests"
	@echo "  bench     - run benchmarks (smoke)"
	@echo "  fmt       - go fmt all packages"
	@echo "  vet       - go vet all packages"
	@echo "  tidy      - go mod tidy"
	@echo "  examples  - build example programs"

build:
	$(GO) build $(PKG)

test:
	$(GO) test -count=1 $(PKG)

# Keep benchmarks lightweight in CI; adjust locally as needed
bench:
	$(GO) test -run=^$$ -bench=. -benchmem $(PKG) || true

fmt:
	$(GO) fmt $(PKG)

vet:
	$(GO) vet $(PKG)

tidy:
	$(GO) mod tidy

examples:
	$(GO) build ./examples/...

