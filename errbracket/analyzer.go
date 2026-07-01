// Package errbracket is a golangci-lint analyzer that standardizes string
// formatting inside fmt.Errorf: every string verb must be written as [%s].
//
//	fmt.Errorf("cannot open %s", name)    // -> [%s]
//	fmt.Errorf("cannot open %q", name)    // -> [%s]
//	fmt.Errorf("cannot open [%q]", name)  // -> [%s]
//	fmt.Errorf("cannot open [%s]", name)  // ok
//
// It offers a suggested fix, so `golangci-lint run --fix` rewrites offenders
// automatically (adding brackets and normalizing the verb to s).
package errbracket

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// Analyzer is the errbracket check.
var Analyzer = &analysis.Analyzer{
	Name: "errbracket",
	Doc:  "checks that string verbs inside fmt.Errorf are written as [%s]",
	URL:  "https://github.com/mbcoward3/errbracket",
	Run:  run,
}

// checkedFuncs maps a fully qualified function name to the index of its
// format-string argument. To cover another Errorf-like wrapper, add a line
// here — e.g. "github.com/pkg/errors.Wrapf": 1.
var checkedFuncs = map[string]int{
	"fmt.Errorf": 0,
}

// stringVerbs are the format verbs treated as string formatting and normalized
// to [%s]. %v is intentionally excluded: it prints any type, so rewriting it to
// %s would be an unsafe fix. Add a verb here only if it always formats a string
// in your codebase.
var stringVerbs = map[byte]bool{
	's': true,
	'q': true,
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

// check reports every string verb in the format literal that isn't already
// written as [%s], and offers a fix to normalize it.
func check(pass *analysis.Pass, lit *ast.BasicLit) {
	// lit.Value keeps the surrounding quotes; %, [, ], and verb letters are
	// single-byte ASCII, so byte offsets in it map straight onto source
	// positions.
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
		verb := s[j]
		if !stringVerbs[verb] {
			continue
		}
		bracketed := start > 0 && s[start-1] == '[' && j+1 < len(s) && s[j+1] == ']'
		if bracketed && verb == 's' {
			continue // already [%s]
		}

		startPos := lit.Pos() + token.Pos(start) // at '%'
		verbPos := lit.Pos() + token.Pos(j)      // at the verb letter
		afterPos := lit.Pos() + token.Pos(j+1)   // just past the verb letter

		// Build the minimal set of edits to reach [%s]: add brackets if
		// missing, and rewrite the verb letter to s if it isn't already.
		var edits []analysis.TextEdit
		if !bracketed {
			edits = append(edits,
				analysis.TextEdit{Pos: startPos, End: startPos, NewText: []byte("[")},
				analysis.TextEdit{Pos: afterPos, End: afterPos, NewText: []byte("]")},
			)
		}
		if verb != 's' {
			edits = append(edits, analysis.TextEdit{Pos: verbPos, End: afterPos, NewText: []byte("s")})
		}

		pass.Report(analysis.Diagnostic{
			Pos:     startPos,
			End:     afterPos,
			Message: fmt.Sprintf("string verb %%%c in fmt.Errorf must be written as [%%s]", verb),
			SuggestedFixes: []analysis.SuggestedFix{{
				Message:   "standardize as [%s]",
				TextEdits: edits,
			}},
		})
	}
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
