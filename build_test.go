package phpstart_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	phpstart "github.com/paketo-buildpacks/php-start"
	"github.com/paketo-buildpacks/php-start/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir  string
		workingDir string
		cnbDir     string

		buffer  *bytes.Buffer
		procMgr *fakes.ProcMgr

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Mkdir(filepath.Join(cnbDir, "bin"), 0700)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(cnbDir, "bin", "procmgr-binary"), []byte{}, 0644)).To(Succeed())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		buffer = bytes.NewBuffer(nil)
		logEmitter := scribe.NewEmitter(buffer)

		procMgr = &fakes.ProcMgr{}
		build = phpstart.Build(procMgr, logEmitter)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("the PHP_HTTPD_PATH env var is set", func() {
		it.Before(func() {
			Expect(os.Setenv("PHP_HTTPD_PATH", "httpd-conf-path")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("PHP_HTTPD_PATH")).To(Succeed())
		})

		it("returns a result that sets an HTTPD start command", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "some-stack",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Launch.Processes[0]).To(Equal(packit.Process{
				Type:    "web",
				Command: "procmgr-binary",
				Args: []string{
					filepath.Join(layersDir, "php-start", "procs.yml"),
				},
				Default: true,
				Direct:  true,
			}))

			Expect(result.Layers[0].Name).To(Equal("php-start"))
			Expect(result.Layers[0].Path).To(Equal(filepath.Join(layersDir, "php-start")))
			Expect(result.Layers[0].Launch).To(BeTrue())
			Expect(result.Layers[0].Build).To(BeFalse())

			Expect(procMgr.AddCall.CallCount).To(Equal(1))
			Expect(procMgr.AddCall.Receives.Name).To(Equal("httpd"))
			Expect(procMgr.AddCall.Receives.Proc.Command).To(Equal("httpd"))
			Expect(procMgr.AddCall.Receives.Proc.Args).To(Equal([]string{
				"-f",
				"httpd-conf-path",
				"-k",
				"start",
				"-DFOREGROUND",
			}))

			Expect(procMgr.WriteFileCall.Receives.Path).To(Equal(filepath.Join(layersDir, "php-start", "procs.yml")))
			Expect(buffer.String()).To(ContainSubstring("Determining start commands to include in procs.yml:"))
			Expect(buffer.String()).To(ContainSubstring("HTTPD: httpd -f httpd-conf-path -k start -DFOREGROUND"))
			Expect(buffer.String()).ToNot(ContainSubstring("FPM: php-fpm -y fpm-conf-path -c phprc-path"))
		})
	})

	context("the PHP_HTTPD, PHP_FPM_PATH, and PHPRC env vars are set", func() {
		it.Before(func() {
			Expect(os.Setenv("PHP_HTTPD_PATH", "httpd-conf-path")).To(Succeed())
			Expect(os.Setenv("PHP_FPM_PATH", "fpm-conf-path")).To(Succeed())
			Expect(os.Setenv("PHPRC", "phprc-path")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("PHP_HTTPD_PATH")).To(Succeed())
			Expect(os.Unsetenv("PHP_FPM_PATH")).To(Succeed())
			Expect(os.Unsetenv("PHPRC")).To(Succeed())
		})

		it("returns a result that starts an HTTPD process and an FPM process", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "some-stack",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Launch.Processes[0]).To(Equal(packit.Process{
				Type:    "web",
				Command: "procmgr-binary",
				Args: []string{
					filepath.Join(layersDir, "php-start", "procs.yml"),
				},
				Default: true,
				Direct:  true,
			}))

			Expect(procMgr.AddCall.CallCount).To(Equal(2))

			// can't check what procMgr.AddCall received first
			Expect(procMgr.AddCall.Receives.Name).To(Equal("fpm"))
			Expect(procMgr.AddCall.Receives.Proc.Command).To(Equal("php-fpm"))
			Expect(procMgr.AddCall.Receives.Proc.Args).To(Equal([]string{
				"-y",
				"fpm-conf-path",
				"-c",
				"phprc-path",
			}))

			Expect(procMgr.WriteFileCall.Receives.Path).To(Equal(filepath.Join(layersDir, "php-start", "procs.yml")))
			Expect(buffer.String()).To(ContainSubstring("Determining start commands to include in procs.yml:"))
			Expect(buffer.String()).To(ContainSubstring("FPM: php-fpm -y fpm-conf-path -c phprc-path"))
		})
	})

	context("failure cases", func() {

		context("the php-start layer cannot be gotten", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(filepath.Join(layersDir, "php-start.toml"), nil, 0000)).To(Succeed())
			})
			it("it returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("failed to parse layer content metadata:")))
			})
		})

		context("when the python layer cannot be reset", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(layersDir, "php-start", "something"), os.ModePerm)).To(Succeed())
				Expect(os.Chmod(filepath.Join(layersDir, "php-start"), 0500)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(filepath.Join(layersDir, "php-start"), os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("could not remove file")))
			})
		})

		context("the PHP_FPM_PATH is set but PHPRC is not", func() {
			it.Before(func() {
				Expect(os.Setenv("PHP_FPM_PATH", "fpm-conf-path")).To(Succeed())
			})

			it.After(func() {
				Expect(os.Unsetenv("PHP_FPM_PATH")).To(Succeed())
			})

			it("returns an error, since the PHPRC is needed for FPM start command", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("failed to lookup $PHPRC path for FPM")))
			})
		})

		context("when the procs.yml cannot be written", func() {
			it.Before(func() {
				procMgr.WriteFileCall.Returns.Error = errors.New("failed to write procs.yml")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("failed to write procs.yml")))
			})
		})

		context("when the procmgr binary cannot be copied into layer", func() {
			it.Before(func() {
				Expect(os.Chmod(filepath.Join(cnbDir, "bin", "procmgr-binary"), 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(filepath.Join(cnbDir, "bin", "procmgr-binary"), os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("failed to copy procmgr-binary into layer:")))
			})
		})
	})
}
