package a

import "fmt"

func good() error {
	return fmt.Errorf("cannot open [%s]", "f")
}

func bareS() error {
	return fmt.Errorf("cannot open %s", "f") // want `must be written as`
}

func bareQ() error {
	return fmt.Errorf("cannot open %q", "f") // want `must be written as`
}

func bracketedQ() error {
	// Bracketed but wrong verb: must normalize to [%s].
	return fmt.Errorf("cannot open [%q]", "f") // want `must be written as`
}

func mixedVerbs() error {
	// %d is not a string verb; only the %s is flagged.
	return fmt.Errorf("code %d for %s", 1, "x") // want `must be written as`
}

func multiple() error {
	return fmt.Errorf("%s then %q", "a", "b") // want `must be written as` `must be written as`
}

func escapedPercent() error {
	return fmt.Errorf("literal %%s stays put")
}

func indexedArg() error {
	// Explicit argument index; the [1] is fmt syntax, not our bracketing.
	return fmt.Errorf("value %[1]s", "x") // want `must be written as`
}

func widthFlag() error {
	return fmt.Errorf("padded %-10s here", "x") // want `must be written as`
}

func notErrorf() string {
	// fmt.Sprintf is not in the checked-function set.
	return fmt.Sprintf("cannot open %q", "f")
}
