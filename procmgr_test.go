package phpstart_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	phpstart "github.com/paketo-buildpacks/php-start"
	"github.com/sclevine/spec"
)

func testProcmgrLib(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect       = NewWithT(t).Expect
		tmpDir       string
		proc1, proc2 phpstart.Proc
		procs        phpstart.Procs
	)

	it.Before(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "procmgr")
		Expect(err).ToNot(HaveOccurred())

		proc1 = phpstart.NewProc("echo", []string{"Hello World!"})
		proc2 = phpstart.NewProc("echo", []string{"Good-bye World!"})
	})

	it.After(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	context("Add", func() {
		it("adds each process to the process list", func() {
			procs = phpstart.NewProcs()
			procs.Add("echo1", proc1)
			procs.Add("echo2", proc2)
			Expect(len(procs.Processes)).To(Equal(2))
			Expect(procs.Processes["echo1"].Command).To(Equal("echo"))
			Expect(procs.Processes["echo2"].Command).To(Equal("echo"))
			Expect(procs.Processes["echo1"].Args).To(Equal([]string{"Hello World!"}))
			Expect(procs.Processes["echo2"].Args).To(Equal([]string{"Good-bye World!"}))
		})
	})

	context("WriteFile", func() {
		context("given a process list and path", func() {
			it.Before(func() {
				procs = phpstart.NewProcs()
				procs.Add("echo1", proc1)
				procs.Add("echo2", proc2)
			})

			it("writes the processes to given path", func() {
				procsFilePath := filepath.Join(tmpDir, "procs.yml")
				Expect(procs.WriteFile(procsFilePath)).To(Succeed())
				content, err := os.ReadFile(procsFilePath)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(ContainSubstring("command: echo"))
				Expect(string(content)).To(ContainSubstring(" - Hello World!"))
				Expect(string(content)).To(ContainSubstring(" - Good-bye World!"))
			})
		})

		context("failure case", func() {
			context("cannot write procs.yml to the given path", func() {
				var badProcsFilePath string
				it.Before(func() {
					badProcsFilePath = filepath.Join(tmpDir, "procs.yml")
					Expect(os.WriteFile(badProcsFilePath, []byte{}, 0000)).To(Succeed())
				})
				it.After(func() {
					Expect(os.RemoveAll(badProcsFilePath)).To(Succeed())
				})
				it("returns an error", func() {
					err := procs.WriteFile(badProcsFilePath)
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})

	context("ReadProcs", func() {
		context("given a process list and path", func() {
			var procsFilePath string
			it.Before(func() {
				procsStr := `{"processes": {"echo1": {"command": "echo", "args": ["'Hello World!'"]}}}`
				procsFilePath = filepath.Join(tmpDir, "procs.yml")
				Expect(os.WriteFile(procsFilePath, []byte(procsStr), os.ModePerm)).To(Succeed())
			})
			it.After(func() {
				Expect(os.RemoveAll(procsFilePath)).To(Succeed())
			})

			it("reads the file and unmarshals it into a Procs type", func() {
				procs, err := phpstart.ReadProcs(procsFilePath)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(procs.Processes)).To(Equal(1))
				Expect(procs.Processes["echo1"]).To(Equal(phpstart.Proc{"echo", []string{"'Hello World!'"}}))
			})
		})

		context("given an empty proc file", func() {
			it("returns an empty Procs struct", func() {
				procs, err := phpstart.ReadProcs("nonexistent-path")
				Expect(err).ToNot(HaveOccurred())
				Expect(procs.Processes).To(Equal(map[string]phpstart.Proc{}))
			})
		})

		context("failure case", func() {
			var badProcsFilePath string
			context("given procs file cannot be opened", func() {
				it.Before(func() {
					procsStr := `{"processes": {"echo1": {"command": "echo", "args": ["'Hello World!'"]}}}`
					badProcsFilePath = filepath.Join(tmpDir, "procs.yml")
					Expect(os.WriteFile(badProcsFilePath, []byte(procsStr), 0000)).To(Succeed())
				})
				it.After(func() {
					Expect(os.RemoveAll(badProcsFilePath)).To(Succeed())
				})
				it("returns an error", func() {
					_, err := phpstart.ReadProcs(badProcsFilePath)
					Expect(err).To(MatchError(ContainSubstring("failed to open proc.yml:")))
				})
			})

			context("given procs file is malformed", func() {
				it.Before(func() {
					procsStr := `non-yaml content`
					badProcsFilePath = filepath.Join(tmpDir, "procs.yml")
					Expect(os.WriteFile(badProcsFilePath, []byte(procsStr), os.ModePerm)).To(Succeed())
				})
				it.After(func() {
					Expect(os.RemoveAll(badProcsFilePath)).To(Succeed())
				})
				it("returns an error", func() {
					_, err := phpstart.ReadProcs(badProcsFilePath)
					Expect(err).To(MatchError(ContainSubstring("invalid proc.yml contents")))
				})
			})
		})
	})
}
