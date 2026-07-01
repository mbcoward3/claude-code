// Package errbracket provides a golangci-lint analyzer that enforces a house
// style rule: every configured format verb (by default %s) inside an
// Errorf-like call must be wrapped in square brackets, e.g. [%s].
//
//	fmt.Errorf("cannot open %s", name)    // flagged
//	fmt.Errorf("cannot open [%s]", name)  // ok
//
// The analyzer offers a suggested fix, so `golangci-lint run --fix` (or
// `go vet`-style `-fix`) rewrites offenders automatically.
package errbracket

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// FunctionConfig describes an Errorf-like function to inspect.
type FunctionConfig struct {
	// Name is the fully qualified function name: the package import path,
	// a dot, and the function name. Examples:
	//   "fmt.Errorf"
	//   "github.com/pkg/errors.Wrapf"
	Name string `json:"name"`
	// FormatArg is the zero-based index of the format-string argument.
	// For fmt.Errorf it is 0; for errors.Wrapf(err, fmt, ...) it is 1.
	FormatArg int `json:"formatArg"`
}

// Settings is the user-configurable behaviour of the analyzer, decoded from
// the golangci-lint `settings` block.
type Settings struct {
	// Verbs is the set of single-letter format verbs that must be bracketed.
	// Defaults to ["s"] (i.e. %s).
	Verbs []string `json:"verbs"`
	// Functions is the list of Errorf-like functions to check. Defaults to
	// [{Name: "fmt.Errorf", FormatArg: 0}].
	Functions []FunctionConfig `json:"functions"`
}

// DefaultSettings returns the out-of-the-box configuration: bracket %s inside
// fmt.Errorf.
func DefaultSettings() Settings {
	return Settings{
		Verbs:     []string{"s"},
		Functions: []FunctionConfig{{Name: "fmt.Errorf", FormatArg: 0}},
	}
}

// withDefaults fills in any unset fields so a zero or partial Settings behaves
// sensibly.
func (s Settings) withDefaults() Settings {
	d := DefaultSettings()
	if len(s.Verbs) == 0 {
		s.Verbs = d.Verbs
	}
	if len(s.Functions) == 0 {
		s.Functions = d.Functions
	}
	return s
}

// Analyzer is the ready-to-use analyzer with default settings, suitable for
// use with singlechecker or go/analysis test harnesses.
var Analyzer = NewAnalyzer(DefaultSettings())

// NewAnalyzer builds an analyzer bound to the given settings.
func NewAnalyzer(s Settings) *analysis.Analyzer {
	c := &checker{
		verbs: verbSet(s.withDefaults().Verbs),
		funcs: funcMap(s.withDefaults().Functions),
	}
	return &analysis.Analyzer{
		Name:             "errbracket",
		Doc:              "checks that configured format verbs (default %s) inside Errorf-like calls are wrapped in brackets, e.g. [%s]",
		URL:              "https://github.com/mbcoward3/errbracket",
		Run:              c.run,
		RunDespiteErrors: false,
	}
}

type checker struct {
	verbs map[byte]bool
	funcs map[string]int // fully qualified name -> format-arg index
}

func verbSet(verbs []string) map[byte]bool {
	set := make(map[byte]bool, len(verbs))
	for _, v := range verbs {
		v = strings.TrimPrefix(v, "%")
		if len(v) == 1 {
			set[v[0]] = true
		}
	}
	return set
}

func funcMap(fns []FunctionConfig) map[string]int {
	m := make(map[string]int, len(fns))
	for _, f := range fns {
		if f.Name == "" {
			continue
		}
		m[f.Name] = f.FormatArg
	}
	return m
}

func (c *checker) run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := calleeName(pass, call)
			idx, ok := c.funcs[name]
			if !ok {
				return true
			}
			if idx < 0 || idx >= len(call.Args) {
				return true
			}
			lit, ok := call.Args[idx].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				// Only single string literals can be located precisely.
				// Concatenations, variables and constants are skipped.
				return true
			}
			c.checkFormat(pass, lit)
			return true
		})
	}
	return nil, nil
}

// calleeName resolves the fully qualified name (import path + "." + func name)
// of a called function, or "" if it cannot be determined. Using type info
// means renamed imports (import f "fmt") still resolve to "fmt.Errorf".
func calleeName(pass *analysis.Pass, call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	obj, ok := pass.TypesInfo.Uses[sel.Sel]
	if !ok {
		return ""
	}
	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil {
		return ""
	}
	return fn.Pkg().Path() + "." + fn.Name()
}

// checkFormat scans a format-string literal and reports every targeted verb
// that is not wrapped in [ ].
func (c *checker) checkFormat(pass *analysis.Pass, lit *ast.BasicLit) {
	// lit.Value keeps the surrounding quotes, and %, [, ], and verb letters
	// are all single-byte ASCII that survive Go string escaping untouched, so
	// byte offsets in lit.Value map directly onto source positions.
	raw := lit.Value
	for _, v := range scanVerbs(raw) {
		if !c.verbs[v.verb] {
			continue
		}
		if bracketed(raw, v) {
			continue
		}
		startPos := lit.Pos() + token.Pos(v.start)
		endPos := lit.Pos() + token.Pos(v.end+1)
		pass.Report(analysis.Diagnostic{
			Pos:     startPos,
			End:     endPos,
			Message: fmt.Sprintf("format verb %%%c must be wrapped in brackets: [%%%c]", v.verb, v.verb),
			SuggestedFixes: []analysis.SuggestedFix{{
				Message: fmt.Sprintf("wrap %%%c in brackets", v.verb),
				TextEdits: []analysis.TextEdit{
					{Pos: startPos, End: startPos, NewText: []byte("[")},
					{Pos: endPos, End: endPos, NewText: []byte("]")},
				},
			}},
		})
	}
}

// bracketed reports whether the verb at v is immediately preceded by '[' and
// followed by ']' in the raw literal.
func bracketed(raw string, v verbLoc) bool {
	return v.start-1 >= 0 && raw[v.start-1] == '[' &&
		v.end+1 < len(raw) && raw[v.end+1] == ']'
}

// verbLoc records the byte span of one format verb within a format string.
type verbLoc struct {
	verb       byte // the verb letter, e.g. 's'
	start, end int  // byte offsets: start is '%', end is the verb letter (inclusive)
}

// scanVerbs finds every format verb in s, skipping "%%" escapes. It follows
// the fmt grammar loosely: after '%' it consumes flags, width, precision,
// '*', and an optional [argIndex] before the terminating verb letter.
func scanVerbs(s string) []verbLoc {
	var out []verbLoc
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			continue
		}
		if i+1 < len(s) && s[i+1] == '%' {
			i++ // escaped percent, not a verb
			continue
		}
		j := i + 1
		for j < len(s) {
			ch := s[j]
			if ch == '[' { // explicit argument index, e.g. %[1]s
				for j < len(s) && s[j] != ']' {
					j++
				}
				if j < len(s) { // step past ']'
					j++
				}
				continue
			}
			if isVerbLetter(ch) {
				break
			}
			if strings.IndexByte("+-# 0.*0123456789", ch) >= 0 {
				j++
				continue
			}
			break // unexpected byte; give up on this '%'
		}
		if j < len(s) && isVerbLetter(s[j]) {
			out = append(out, verbLoc{verb: s[j], start: i, end: j})
			i = j
		}
	}
	return out
}

func isVerbLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
