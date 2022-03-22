package main

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitProcmgr(t *testing.T) {
	suite := spec.New("cmd/procmgry-binary", spec.Report(report.Terminal{}))
	suite("Procmgr Binary", testProcmgr)
	suite.Run(t)
}
