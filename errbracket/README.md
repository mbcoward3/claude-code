# errbracket

A [golangci-lint **v2** module plugin](https://golangci-lint.run/docs/plugins/module-plugins/)
that enforces one rule:

> Every string-formatting verb inside `fmt.Errorf` must be written as `[%s]`.

```go
fmt.Errorf("cannot open %s", name)    // ✗ -> [%s]
fmt.Errorf("cannot open %q", name)    // ✗ -> [%s]
fmt.Errorf("cannot open [%q]", name)  // ✗ -> [%s]
fmt.Errorf("cannot open [%s]", name)  // ✓ ok
```

It ships a suggested fix, so `golangci-lint run --fix` rewrites offenders for
you — adding the brackets **and** normalizing the verb (`%q` → `%s`). `%d`,
`%v`, and non-`fmt.Errorf` calls are left alone. Renamed imports
(`import f "fmt"; f.Errorf(...)`) are still caught, because the check resolves
calls via type information rather than by name.

`%v` is deliberately not rewritten: it formats any type, so turning it into
`%s` would be an unsafe fix (`%s` on a non-`Stringer` value prints `%!s(...)`).

The whole linter is two short files — [`analyzer.go`](./analyzer.go) (the check)
and [`plugin.go`](./plugin.go) (golangci-lint registration).

---

## Integrating into your existing `.golangci.yml`

> **The one thing to know:** golangci-lint v2 can't load a plugin into the
> stock binary. Module plugins are *compiled in*, so you build a small custom
> golangci-lint binary once and run **that** instead of `golangci-lint`. Your
> `.golangci.yml` is otherwise unchanged apart from the two additions below.

### 1. Add the linter to your config

In your existing `.golangci.yml` (which must be `version: "2"`), enable the
linter and register it as a custom module. Merge these two keys into what you
already have:

```yaml
version: "2"

linters:
  enable:
    # ...your existing linters...
    - errbracket

  settings:
    custom:
      errbracket:
        type: module
        description: Standardize string verbs inside fmt.Errorf as [%s].
```

There is no plugin-specific `settings:` block to fill in — the rule has no knobs.

### 2. Tell `golangci-lint custom` how to build the binary

Add a `.custom-gcl.yml` next to your `.golangci.yml` (one already lives in this
repo you can copy). Set `version` to the golangci-lint version your team uses:

```yaml
version: v2.5.0        # match your golangci-lint version
name: custom-gcl
destination: ./bin
plugins:
  - module: github.com/mbcoward3/errbracket
    version: v0.1.0    # a tagged release once you publish the module
    # For local development against a checkout instead of a release, drop
    # `version` and point at the path:
    #   path: ../path/to/errbracket
```

### 3. Build the custom binary

```bash
golangci-lint custom     # reads .custom-gcl.yml -> ./bin/custom-gcl
```

### 4. Run `./bin/custom-gcl` wherever you run `golangci-lint`

It's a drop-in replacement — same flags, same config:

```bash
./bin/custom-gcl run ./...
./bin/custom-gcl run --fix ./...     # auto-wrap offenders
```

Update the invocation in the places that call the linter:

- **Makefile / scripts:** replace `golangci-lint run` with `./bin/custom-gcl run`.
- **pre-commit:** point the hook's `entry` at `./bin/custom-gcl`.
- **CI:** build it in a step (`golangci-lint custom`) and call `./bin/custom-gcl`,
  or use the [`golangci-lint-custom` action](https://github.com/marketplace/actions/golangci-lint-custom).
  Cache `./bin` keyed on the plugin version to skip rebuilds.

---

## Extending it (optional)

Both knobs are single-line edits in [`analyzer.go`](./analyzer.go).

To cover another `Errorf`-like wrapper, add to `checkedFuncs` — key is
`importpath.FuncName`, value is the zero-based index of the format argument:

```go
var checkedFuncs = map[string]int{
	"fmt.Errorf":                  0,
	"github.com/pkg/errors.Wrapf": 1, // Wrapf(err, format, ...)
}
```

To treat another verb as a string verb (normalized to `[%s]`), add to
`stringVerbs`. Only do this for verbs that always format a string in your code
— e.g. adding `'v'` would rewrite `%v` to `%s`, which breaks non-`Stringer`
arguments:

```go
var stringVerbs = map[byte]bool{
	's': true,
	'q': true,
}
```

## Develop

```bash
go test ./...   # analysistest: diagnostics (testdata/src/a) + autofix golden (testdata/src/fix)
```

If you host this somewhere other than `github.com/mbcoward3/errbracket`, update
the module path in `go.mod`, `plugin.go`'s `URL`, and `.custom-gcl.yml`.
