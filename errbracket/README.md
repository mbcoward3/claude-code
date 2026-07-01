# errbracket

A [golangci-lint **v2** module plugin](https://golangci-lint.run/docs/plugins/module-plugins/)
(and standalone `go/analysis` tool) that enforces one house rule:

> Every `%s` inside an `Errorf`-like call must be wrapped in square brackets.

```go
fmt.Errorf("cannot open %s", name)    // ✗ flagged
fmt.Errorf("cannot open [%s]", name)  // ✓ ok
```

It ships a **suggested fix**, so `--fix` rewrites offenders automatically. The
targeted verbs and functions are configurable — `%s`/`fmt.Errorf` are just the
defaults.

## How it works

- Resolves the called function through **type information**, so renamed imports
  (`import f "fmt"; f.Errorf(...)`) are still caught, and unrelated same-named
  methods are not.
- Only inspects the format argument when it is a **single string literal**.
  Verbs come from variables, constants, or concatenated strings are skipped
  (their source position can't be pinned precisely).
- Understands `fmt` verb syntax: flags/width/precision (`%-10s`), `*` width, and
  explicit argument indexes (`%[1]s`) are handled; `%%` is ignored.
- Only the configured verbs are flagged — `%d`, `%v`, etc. are left untouched
  unless you add them.

## Use it with golangci-lint

golangci-lint's module plugins are compiled into a custom binary. From this
directory:

```bash
# 1. Build a golangci-lint binary with errbracket baked in (reads .custom-gcl.yml)
golangci-lint custom      # -> ./bin/custom-gcl

# 2. Point your project's .golangci.yml at it (see .golangci.example.yml)

# 3. Lint (and optionally auto-fix)
./bin/custom-gcl run ./...
./bin/custom-gcl run --fix ./...
```

Minimal `.golangci.yml`:

```yaml
version: "2"
linters:
  enable:
    - errbracket
  settings:
    custom:
      errbracket:
        type: module
        description: Enforce bracketed format verbs inside Errorf-like calls.
        # settings omitted -> defaults: verbs [s], functions [fmt.Errorf@0]
```

See [`.golangci.example.yml`](./.golangci.example.yml) for the full settings
surface (custom verbs, extra functions like `errors.Wrapf`).

## Use it standalone (no golangci-lint)

A `go vet`-style binary is included — handy for editors and lightweight CI:

```bash
go run github.com/mbcoward3/errbracket/cmd/errbracket ./...
go run github.com/mbcoward3/errbracket/cmd/errbracket -fix ./...
```

## Configuration

| Setting            | Type   | Default                              | Meaning                                                        |
| ------------------ | ------ | ------------------------------------ | -------------------------------------------------------------- |
| `verbs`            | `[]string` | `["s"]`                          | Single-letter format verbs that must be bracketed.             |
| `functions`        | `[]object` | `[{name: "fmt.Errorf", formatArg: 0}]` | Errorf-like functions to inspect.                    |
| `functions[].name` | `string`   | —                                | Import path + `.` + function name (e.g. `github.com/pkg/errors.Wrapf`). |
| `functions[].formatArg` | `int` | —                                | Zero-based index of the format-string argument.               |

## Develop

```bash
make test          # unit tests: diagnostics + suggested-fix golden files
make demo PKG=./...
make fix  PKG=./...
```

Adjust the module path (`github.com/mbcoward3/errbracket`) in `go.mod`,
`cmd/errbracket/main.go`, and the config files if you host it elsewhere.
