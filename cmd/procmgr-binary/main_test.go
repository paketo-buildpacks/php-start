package main

import (
	"testing"

	. "github.com/onsi/gomega"
	phpstart "github.com/paketo-buildpacks/php-start"
	"github.com/sclevine/spec"
)

// TODO: remove print/echo statements
func testProcmgr(t *testing.T, context spec.G, it spec.S) {
	var Expect = NewWithT(t).Expect

	context("given a process", func() {
		it("should run it", func() {
			err := runProcs(phpstart.Procs{
				Processes: map[string]phpstart.Proc{
					"proc1": {
						Command: "echo",
						Args:    []string{"'Hello World!"},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	context("given a process that doesn't exist", func() {
		it("should fail", func() {
			err := runProcs(phpstart.Procs{
				Processes: map[string]phpstart.Proc{
					"proc1": {
						Command: "idontexist",
						Args:    []string{},
					},
				},
			})
			Expect(err).To(HaveOccurred())
		})
	})

	context("given two processes", func() {
		it("should run both", func() {
			err := runProcs(phpstart.Procs{
				Processes: map[string]phpstart.Proc{
					"proc1": {
						Command: "echo",
						Args:    []string{"'Hello World!"},
					},
					"proc2": {
						Command: "echo",
						Args:    []string{"'Good-bye World!"},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	context("given a process that exits non-zero", func() {
		it("fails", func() {
			err := runProcs(phpstart.Procs{
				Processes: map[string]phpstart.Proc{
					"proc1": {
						Command: "false",
						Args:    []string{""},
					},
				},
			})
			Expect(err).To(HaveOccurred())
		})
	})

	context("given two processes where one is shorter", func() {
		it("should succeed in running both", func() {
			err := runProcs(phpstart.Procs{
				Processes: map[string]phpstart.Proc{
					"sleep0.25": {
						Command: "sleep",
						Args:    []string{"0.25"},
					},
					"sleep1": {
						Command: "sleep",
						Args:    []string{"1"},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})
}
