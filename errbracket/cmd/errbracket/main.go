// Command errbracket runs the errbracket analyzer as a standalone vet-style
// tool, independent of golangci-lint. Useful for quick local runs and CI that
// doesn't use golangci-lint:
//
//	go run ./cmd/errbracket ./...
//	go run ./cmd/errbracket -fix ./...
package main

import (
	"github.com/mbcoward3/errbracket"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(errbracket.Analyzer)
}
