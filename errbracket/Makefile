.PHONY: test lint-build fix demo

# Run the unit tests (diagnostics + suggested-fix golden files).
test:
	go test ./...

# Build a golangci-lint binary with errbracket compiled in -> ./bin/custom-gcl
lint-build:
	golangci-lint custom

# Run the standalone analyzer over a target (default ./...) and auto-fix.
# Usage: make fix PKG=./...
PKG ?= ./...
fix:
	go run ./cmd/errbracket -fix $(PKG)

# Show diagnostics only, no fixes.
demo:
	go run ./cmd/errbracket $(PKG)
