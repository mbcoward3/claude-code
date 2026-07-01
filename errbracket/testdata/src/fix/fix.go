package fix

import "fmt"

func open(name string) error {
	return fmt.Errorf("open %s failed", name) // want `format verb %s must be wrapped in brackets`
}

func two(a, b string) error {
	return fmt.Errorf("%s and %s", a, b) // want `format verb %s must be wrapped in brackets` `format verb %s must be wrapped in brackets`
}

func padded(name string) error {
	return fmt.Errorf("padded %-10s tail", name) // want `format verb %s must be wrapped in brackets`
}
