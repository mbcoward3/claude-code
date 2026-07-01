package errbracket

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

// Register errbracket with golangci-lint's module plugin system. "errbracket"
// is the name you reference in .golangci.yml.
func init() {
	register.Plugin("errbracket", New)
}

// New is the plugin constructor. errbracket takes no settings, so the argument
// is ignored.
func New(any) (register.LinterPlugin, error) {
	return plugin{}, nil
}

type plugin struct{}

func (plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{Analyzer}, nil
}

// GetLoadMode requests type information, which calleeName uses to resolve
// fmt.Errorf even through renamed imports.
func (plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}
