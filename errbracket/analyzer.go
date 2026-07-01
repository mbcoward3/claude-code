// Package errbracket is a golangci-lint analyzer that enforces one rule:
// every %s inside fmt.Errorf must be wrapped in square brackets.
//
//	fmt.Errorf("cannot open %s", name)    // flagged
//	fmt.Errorf("cannot open [%s]", name)  // ok
//
// It offers a suggested fix, so `golangci-lint run --fix` wraps offenders
// automatically.
package errbracket

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// Analyzer is the errbracket check.
var Analyzer = &analysis.Analyzer{
	Name: "errbracket",
	Doc:  "checks that %s inside fmt.Errorf is wrapped in brackets, e.g. [%s]",
	URL:  "https://github.com/mbcoward3/errbracket",
	Run:  run,
}

// checkedFuncs maps a fully qualified function name to the index of its
// format-string argument. To cover another Errorf-like wrapper, add a line
// here — e.g. "github.com/pkg/errors.Wrapf": 1.
var checkedFuncs = map[string]int{
	"fmt.Errorf": 0,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			argIdx, ok := checkedFuncs[calleeName(pass, call)]
			if !ok || argIdx >= len(call.Args) {
				return true
			}
			if lit, ok := call.Args[argIdx].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				check(pass, lit)
			}
			return true
		})
	}
	return nil, nil
}

// calleeName returns the "import/path.FuncName" of the called function, using
// type info so a renamed import (import f "fmt") still resolves to "fmt.Errorf".
// It returns "" when the callee can't be identified.
func calleeName(pass *analysis.Pass, call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	fn, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil {
		return ""
	}
	return fn.Pkg().Path() + "." + fn.Name()
}

// check reports every %s verb in the format literal that isn't already [%s].
func check(pass *analysis.Pass, lit *ast.BasicLit) {
	// lit.Value keeps the surrounding quotes; %, [, ], and s are single-byte
	// ASCII, so byte offsets in it map straight onto source positions.
	s := lit.Value
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			continue
		}
		start := i // the '%'
		j := i + 1
		if j < len(s) && s[j] == '%' { // "%%" is a literal percent, not a verb
			i = j
			continue
		}
		// Skip any flags/width/precision/arg-index between % and the verb
		// letter, e.g. the "-10" in %-10s or the "[1]" in %[1]s.
		for j < len(s) && !isLetter(s[j]) {
			j++
		}
		if j >= len(s) {
			break
		}
		i = j // resume scanning after this verb
		if s[j] != 's' {
			continue
		}
		if start > 0 && s[start-1] == '[' && j+1 < len(s) && s[j+1] == ']' {
			continue // already [%s]
		}

		startPos := lit.Pos() + token.Pos(start) // at '%'
		afterPos := lit.Pos() + token.Pos(j+1)   // just past 's'
		pass.Report(analysis.Diagnostic{
			Pos:     startPos,
			End:     afterPos,
			Message: "%s in fmt.Errorf must be wrapped in brackets, e.g. [%s]",
			SuggestedFixes: []analysis.SuggestedFix{{
				Message: "wrap in brackets",
				TextEdits: []analysis.TextEdit{
					{Pos: startPos, End: startPos, NewText: []byte("[")},
					{Pos: afterPos, End: afterPos, NewText: []byte("]")},
				},
			}},
		})
	}
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
