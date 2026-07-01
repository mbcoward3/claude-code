package a

import f "fmt"

// renamedImport verifies the check resolves fmt.Errorf through a renamed
// import via type information rather than the literal package name.
func renamedImport() error {
	return f.Errorf("cannot open %s", "x") // want `must be written as`
}
