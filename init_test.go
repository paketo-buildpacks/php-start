package phpstart_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitPhpStart(t *testing.T) {
	suite := spec.New("php-start", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Build", testBuild, spec.Sequential())
	suite("Detect", testDetect)
	suite("TestProcmgrLib", testProcmgrLib)
	suite.Run(t)
}
