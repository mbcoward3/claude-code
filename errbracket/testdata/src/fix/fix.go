package fix

import "fmt"

func open(name string) error {
	return fmt.Errorf("open %s failed", name) // want `must be wrapped in brackets`
}

func two(a, b string) error {
	return fmt.Errorf("%s and %s", a, b) // want `must be wrapped in brackets` `must be wrapped in brackets`
}

func padded(name string) error {
	return fmt.Errorf("padded %-10s tail", name) // want `must be wrapped in brackets`
}
