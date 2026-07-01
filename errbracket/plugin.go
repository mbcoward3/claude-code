package errbracket

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

// init registers errbracket with golangci-lint's module plugin system. The
// name here ("errbracket") is what you reference under
// linters.settings.custom and linters.enable in .golangci.yml.
func init() {
	register.Plugin("errbracket", New)
}

// New is the plugin constructor. golangci-lint passes the decoded `settings`
// block from .golangci.yml as `any`.
func New(settings any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[Settings](settings)
	if err != nil {
		return nil, err
	}
	return &plugin{settings: s}, nil
}

type plugin struct {
	settings Settings
}

// BuildAnalyzers returns the analyzers this plugin contributes.
func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{NewAnalyzer(p.settings)}, nil
}

// GetLoadMode requests type information, which the analyzer uses to resolve
// calls like fmt.Errorf even through renamed imports.
func (p *plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}
