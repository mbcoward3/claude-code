package a

import "fmt"

func good() error {
	return fmt.Errorf("cannot open [%s]", "f")
}

func bad() error {
	return fmt.Errorf("cannot open %s", "f") // want `format verb %s must be wrapped in brackets`
}

func mixedVerbs() error {
	// %d is not targeted; only %s is flagged.
	return fmt.Errorf("code %d for %s", 1, "x") // want `format verb %s must be wrapped in brackets`
}

func multiple() error {
	return fmt.Errorf("%s then %s", "a", "b") // want `format verb %s must be wrapped in brackets` `format verb %s must be wrapped in brackets`
}

func escapedPercent() error {
	return fmt.Errorf("literal %%s stays put")
}

func indexedArg() error {
	// Explicit argument index; the [1] is fmt syntax, not our bracketing.
	return fmt.Errorf("value %[1]s", "x") // want `format verb %s must be wrapped in brackets`
}

func widthFlag() error {
	return fmt.Errorf("padded %-10s here", "x") // want `format verb %s must be wrapped in brackets`
}

func notErrorf() string {
	// fmt.Sprintf is not in the checked-function set.
	return fmt.Sprintf("cannot open %s", "f")
}
