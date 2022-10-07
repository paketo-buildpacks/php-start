package phpstart_test

import (
	"bytes"
	"errors"
	"fmt"
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
		reloader  *fakes.Reloader
		processes map[string]phpstart.Proc

		buildContext packit.BuildContext
		build        packit.BuildFunc
	)

	it.Before(func() {
		layersDir = t.TempDir()
		cnbDir = t.TempDir()
		workingDir = t.TempDir()

		Expect(os.Mkdir(filepath.Join(cnbDir, "bin"), 0700)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(cnbDir, "bin", "procmgr-binary"), []byte{}, 0644)).To(Succeed())

		buffer = bytes.NewBuffer(nil)
		logEmitter := scribe.NewEmitter(buffer).WithLevel("DEBUG")

		procMgr = &fakes.ProcMgr{}
		reloader = &fakes.Reloader{}
		processes = map[string]phpstart.Proc{}
		procMgr.AddCall.Stub = func(procName string, newProc phpstart.Proc) {
			processes[procName] = newProc
		}

		buildContext = packit.BuildContext{
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
		}
		build = phpstart.Build(procMgr, logEmitter, reloader)
	})

	context("[HTTPD] the PHP_HTTPD, PHP_FPM_PATH, and PHPRC env vars are set", func() {
		it.Before(func() {
			t.Setenv("PHP_HTTPD_PATH", "httpd-conf-path")
			t.Setenv("PHP_FPM_PATH", "fpm-conf-path")
			t.Setenv("PHPRC", "phprc-path")
		})

		it("returns a result that starts an HTTPD process and an FPM process", func() {
			result, err := build(buildContext)
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
				"fpm": {
					Command: "php-fpm",
					Args: []string{
						"-y",
						"fpm-conf-path",
						"-c",
						"phprc-path",
					},
				},
				"httpd": {
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

		context("when live reload is enabled", func() {
			it.Before(func() {
				reloader.ShouldEnableLiveReloadCall.Returns.Bool = true
			})

			context("the watch directories exist", func() {
				it.Before(func() {
					Expect(os.MkdirAll(filepath.Join(workingDir, ".php.fpm.d"), os.ModePerm)).To(Succeed())
					Expect(os.MkdirAll(filepath.Join(workingDir, ".httpd.conf.d"), os.ModePerm)).To(Succeed())
				})

				it("should add watchexec processes to the process file", func() {
					result, err := build(buildContext)
					Expect(err).NotTo(HaveOccurred())

					expectedProcesses := map[string]phpstart.Proc{
						"fpm": {
							Command: "watchexec",
							Args: []string{
								"--watch", "/workspace/.php.fpm.d",
								"--on-busy-update", "signal",
								"--signal", "SIGUSR2",
								"--shell", "none",
								"--", "php-fpm",
								"-y", "fpm-conf-path",
								"-c", "phprc-path",
							},
						},
						"httpd": {
							Command: "watchexec",
							Args: []string{
								"--watch", "/workspace/.httpd.conf.d",
								"--on-busy-update", "signal",
								"--signal", "SIGHUP",
								"--shell", "none",
								"--", "httpd",
								"-f", "httpd-conf-path",
								"-k", "start",
								"-DFOREGROUND",
							},
						},
					}
					Expect(processes).To(Equal(expectedProcesses))

					Expect(result.Launch.Processes).To(ConsistOf(packit.Process{
						Type:    "web",
						Command: "procmgr-binary",
						Args:    []string{filepath.Join(layersDir, "php-start", "procs.yml")},
						Default: true,
						Direct:  true,
					}))

					Expect(reloader.TransformReloadableProcessesCall.CallCount).To(Equal(0))
				})
			})

			context("the watch directories do not exist", func() {
				it("should add non-reloadable processes to the process file", func() {
					_, err := build(buildContext)
					Expect(err).NotTo(HaveOccurred())

					expectedProcesses := map[string]phpstart.Proc{
						"fpm": {
							Command: "php-fpm",
							Args: []string{
								"-y",
								"fpm-conf-path",
								"-c",
								"phprc-path",
							},
						},
						"httpd": {
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
					Expect(buffer.String()).To(ContainSubstring("HTTPD configuration will not be reloadable since .httpd.conf.d folder not found"))
					Expect(buffer.String()).To(ContainSubstring("FPM will not be reloadable since .php.fpm.d folder not found"))
				})
			})

			context("failure cases", func() {
				context("when reloader returns an error", func() {
					it.Before(func() {
						reloader.ShouldEnableLiveReloadCall.Returns.Error = errors.New("reload error")
					})

					it("will return the error", func() {
						_, err := build(buildContext)
						Expect(err).To(MatchError("reload error"))
					})
				})
			})
		})
	})

	context("[NGINX] the PHP_NGINX, PHP_FPM_PATH, and PHPRC env vars are set", func() {
		it.Before(func() {
			t.Setenv("PHP_NGINX_PATH", "nginx-conf-path")
			t.Setenv("PHP_FPM_PATH", "fpm-conf-path")
			t.Setenv("PHPRC", "phprc-path")
		})

		it("returns a result that starts an NGINX process and an FPM process", func() {
			result, err := build(buildContext)
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
				"fpm": {
					Command: "php-fpm",
					Args: []string{
						"-y",
						"fpm-conf-path",
						"-c",
						"phprc-path",
					},
				},
				"nginx": {
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

		context("when live reload is enabled", func() {
			it.Before(func() {
				reloader.ShouldEnableLiveReloadCall.Returns.Bool = true
			})

			context("the watch directories exist", func() {
				it.Before(func() {
					Expect(os.MkdirAll(filepath.Join(workingDir, ".php.fpm.d"), os.ModePerm)).To(Succeed())
					Expect(os.MkdirAll(filepath.Join(workingDir, ".nginx.conf.d"), os.ModePerm)).To(Succeed())
				})

				it("should add watchexec processes to the process file", func() {
					result, err := build(buildContext)
					Expect(err).NotTo(HaveOccurred())

					expectedProcesses := map[string]phpstart.Proc{
						"fpm": {
							Command: "watchexec",
							Args: []string{
								"--watch", "/workspace/.php.fpm.d",
								"--on-busy-update", "signal",
								"--signal", "SIGUSR2",
								"--shell", "none",
								"--", "php-fpm",
								"-y", "fpm-conf-path",
								"-c", "phprc-path",
							},
						},
						"nginx": {
							Command: "watchexec",
							Args: []string{
								"--watch", "/workspace/.nginx.conf.d",
								"--on-busy-update", "signal",
								"--signal", "SIGHUP",
								"--shell", "none",
								"--", "nginx",
								"-p", workingDir,
								"-c", "nginx-conf-path"},
						},
					}
					Expect(processes).To(Equal(expectedProcesses))

					Expect(result.Launch.Processes).To(ConsistOf(packit.Process{
						Type:    "web",
						Command: "procmgr-binary",
						Args:    []string{filepath.Join(layersDir, "php-start", "procs.yml")},
						Default: true,
						Direct:  true,
					}))

					Expect(reloader.TransformReloadableProcessesCall.CallCount).To(Equal(0))
				})
			})

			context("the watch directories do not exist", func() {
				it("should add non-reloadable processes to the process file", func() {
					_, err := build(buildContext)
					Expect(err).NotTo(HaveOccurred())

					expectedProcesses := map[string]phpstart.Proc{
						"fpm": {
							Command: "php-fpm",
							Args: []string{
								"-y",
								"fpm-conf-path",
								"-c",
								"phprc-path",
							},
						},
						"nginx": {
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
					Expect(buffer.String()).To(ContainSubstring("NGINX configuration will not be reloadable since .nginx.conf.d folder not found"))
					Expect(buffer.String()).To(ContainSubstring("FPM will not be reloadable since .php.fpm.d folder not found"))
				})
			})

			context("failure cases", func() {
				context("when reloader returns an error", func() {
					it.Before(func() {
						reloader.ShouldEnableLiveReloadCall.Returns.Error = errors.New("reload error")
					})

					it("will return the error", func() {
						_, err := build(buildContext)
						Expect(err).To(MatchError("reload error"))
					})
				})
			})
		})
	})

	context("failure cases", func() {
		it.Before(func() {
			t.Setenv("PHP_HTTPD_PATH", "httpd-conf-path")
			t.Setenv("PHP_FPM_PATH", "fpm-conf-path")
			t.Setenv("PHPRC", "phprc-path")
		})

		context("the php-start layer cannot be gotten", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(layersDir, "php-start.toml"), nil, 0000)).To(Succeed())
			})

			it("it returns an error", func() {
				_, err := build(buildContext)
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
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("could not remove file")))
			})
		})

		context("neither PHP_HTTPD_PATH nor PHP_NGINX_PATH are set", func() {
			it.Before(func() {
				Expect(os.Unsetenv("PHP_HTTPD_PATH")).To(Succeed())
				Expect(os.Unsetenv("PHP_NGINX_PATH")).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("need exactly one of: $PHP_HTTPD_PATH or $PHP_NGINX_PATH")))
			})
		})

		context("both PHP_HTTPD_PATH and PHP_NGINX_PATH are set", func() {
			it.Before(func() {
				t.Setenv("PHP_HTTPD_PATH", "some value")
				t.Setenv("PHP_NGINX_PATH", "some other value")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("need exactly one of: $PHP_HTTPD_PATH or $PHP_NGINX_PATH")))
			})
		})

		context("the PHP_FPM_PATH env var is NOT set", func() {
			it.Before(func() {
				Expect(os.Unsetenv("PHP_FPM_PATH")).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("failed to lookup $PHP_FPM_PATH")))
			})
		})

		context("the PHP_FPM_PATH env var is set but empty", func() {
			it.Before(func() {
				t.Setenv("PHP_FPM_PATH", "")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("failed to lookup $PHP_FPM_PATH")))
			})
		})

		context("the PHPRC env var is not set", func() {
			it.Before(func() {
				Expect(os.Unsetenv("PHPRC")).To(Succeed())
			})

			it("returns an error, since the PHPRC is needed for FPM start command", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("failed to lookup $PHPRC path for FPM")))
			})
		})

		context("the PHPRC env var is set but empty", func() {
			it.Before(func() {
				t.Setenv("PHPRC", "")
			})

			it("returns an error, since the PHPRC is needed for FPM start command", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("failed to lookup $PHPRC path for FPM")))
			})
		})

		context("when the procs.yml cannot be written", func() {
			it.Before(func() {
				procMgr.WriteFileCall.Returns.Error = errors.New("failed to write procs.yml")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
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
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("failed to copy procmgr-binary into layer:")))
			})
		})
	})
}
