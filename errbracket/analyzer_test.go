package errbracket_test

import (
	"testing"

	"github.com/mbcoward3/errbracket"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestDefault(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), errbracket.Analyzer, "a")
}

func TestSuggestedFixes(t *testing.T) {
	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), errbracket.Analyzer, "fix")
}
