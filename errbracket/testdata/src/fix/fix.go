package fix

import "fmt"

func open(name string) error {
	return fmt.Errorf("open %s failed", name) // want `must be written as`
}

func quoted(name string) error {
	return fmt.Errorf("got %q here", name) // want `must be written as`
}

func bracketedQ(name string) error {
	return fmt.Errorf("got [%q]", name) // want `must be written as`
}

func two(a, b string) error {
	return fmt.Errorf("%s and %q", a, b) // want `must be written as` `must be written as`
}

func padded(name string) error {
	return fmt.Errorf("padded %-10s tail", name) // want `must be written as`
}
