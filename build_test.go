package phpstart_test

import (
	"bytes"
	"errors"
	"fmt"
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

		buffer    *bytes.Buffer
		procMgr   *fakes.ProcMgr
		processes map[string]phpstart.Proc

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
		processes = map[string]phpstart.Proc{}
		procMgr.AddCall.Stub = func(procName string, newProc phpstart.Proc) {
			processes[procName] = newProc
		}
		build = phpstart.Build(procMgr, logEmitter)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
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

			expectedProcesses := map[string]phpstart.Proc{
				"fpm": phpstart.Proc{
					Command: "php-fpm",
					Args: []string{
						"-y",
						"fpm-conf-path",
						"-c",
						"phprc-path",
					},
				},
				"httpd": phpstart.Proc{
					Command: "httpd",
					Args: []string{
						"-f",
						"httpd-conf-path",
						"-k",
						"start",
						"-DFOREGROUND",
					},
				},
			}
			Expect(processes).To(Equal(expectedProcesses))

			Expect(procMgr.WriteFileCall.Receives.Path).To(Equal(filepath.Join(layersDir, "php-start", "procs.yml")))
			Expect(buffer.String()).To(ContainSubstring("Determining start commands to include in procs.yml:"))
			Expect(buffer.String()).To(ContainSubstring("FPM: php-fpm -y fpm-conf-path -c phprc-path"))
			Expect(buffer.String()).To(ContainSubstring("HTTPD: httpd -f httpd-conf-path -k start -DFOREGROUND"))
		})
	})

	context("the PHP_NGINX, PHP_FPM_PATH, and PHPRC env vars are set", func() {
		it.Before(func() {
			Expect(os.Setenv("PHP_NGINX_PATH", "nginx-conf-path")).To(Succeed())
			Expect(os.Setenv("PHP_FPM_PATH", "fpm-conf-path")).To(Succeed())
			Expect(os.Setenv("PHPRC", "phprc-path")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("PHP_NGINX_PATH")).To(Succeed())
			Expect(os.Unsetenv("PHP_FPM_PATH")).To(Succeed())
			Expect(os.Unsetenv("PHPRC")).To(Succeed())
		})

		it("returns a result that starts an NGINX process and an FPM process", func() {
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

			expectedProcesses := map[string]phpstart.Proc{
				"fpm": phpstart.Proc{
					Command: "php-fpm",
					Args: []string{
						"-y",
						"fpm-conf-path",
						"-c",
						"phprc-path",
					},
				},
				"nginx": phpstart.Proc{
					Command: "nginx",
					Args: []string{
						"-p",
						workingDir,
						"-c",
						"nginx-conf-path",
					},
				},
			}
			Expect(processes).To(Equal(expectedProcesses))

			Expect(procMgr.WriteFileCall.Receives.Path).To(Equal(filepath.Join(layersDir, "php-start", "procs.yml")))
			Expect(buffer.String()).To(ContainSubstring("Determining start commands to include in procs.yml:"))
			Expect(buffer.String()).To(ContainSubstring("FPM: php-fpm -y fpm-conf-path -c phprc-path"))
			Expect(buffer.String()).To(ContainSubstring(fmt.Sprintf("Nginx: nginx -p %s -c nginx-conf-path", workingDir)))
		})
	})

	context("failure cases", func() {
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

		context("when the php-start layer cannot be reset", func() {
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

		context("neither PHP_HTTPD_PATH nor PHP_NGINX_PATH are set", func() {
			it.Before(func() {
				Expect(os.Unsetenv("PHP_HTTPD_PATH")).To(Succeed())
				Expect(os.Unsetenv("PHP_NGINX_PATH")).To(Succeed())
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

				Expect(err).To(MatchError(ContainSubstring("need exactly one of: $PHP_HTTPD_PATH or $PHP_NGINX_PATH")))
			})
		})

		context("both PHP_HTTPD_PATH and PHP_NGINX_PATH are set", func() {
			it.Before(func() {
				Expect(os.Setenv("PHP_HTTPD_PATH", "some value")).To(Succeed())
				Expect(os.Setenv("PHP_NGINX_PATH", "some other value")).To(Succeed())
			})
			it.After(func() {
				Expect(os.Unsetenv("PHP_NGINX_PATH")).To(Succeed())
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

				Expect(err).To(MatchError(ContainSubstring("need exactly one of: $PHP_HTTPD_PATH or $PHP_NGINX_PATH")))
			})
		})

		context("the PHP_FPM_PATH env var is NOT set", func() {
			it.Before(func() {
				Expect(os.Unsetenv("PHP_FPM_PATH")).To(Succeed())
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

				Expect(err).To(MatchError(ContainSubstring("failed to lookup $PHP_FPM_PATH")))
			})
		})

		context("the PHP_FPM_PATH env var is set but empty", func() {
			it.Before(func() {
				Expect(os.Setenv("PHP_FPM_PATH", "")).To(Succeed())
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

				Expect(err).To(MatchError(ContainSubstring("failed to lookup $PHP_FPM_PATH")))
			})
		})

		context("the PHPRC env var is not set", func() {
			it.Before(func() {
				Expect(os.Unsetenv("PHPRC")).To(Succeed())
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

		context("the PHPRC env var is set but empty", func() {
			it.Before(func() {
				Expect(os.Setenv("PHPRC", "")).To(Succeed())
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
