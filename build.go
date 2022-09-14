package phpstart

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface ProcMgr --output fakes/procmgr.go

// ProcMgr manages all processes that the build  phase may need to run by
// adding them to a procs.yml file for execution at launch time.
type ProcMgr interface {
	Add(name string, proc Proc)
	WriteFile(path string) error
}

// Build will return a packit.BuildFunc that will be invoked during the build
// phase of the buildpack lifecycle.
//
// It will create a layer dedicated to storing a process manager and YAML file
// of processes to run, since there are multiple process that could be run. The
// layer is available at and launch-time, and its contents are used in the
// image launch process.
func Build(procs ProcMgr, logger scribe.Emitter, reloader Reloader) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		logger.Debug.Process("Getting the layer associated with the server start command")
		layer, err := context.Layers.Get("php-start")
		if err != nil {
			return packit.BuildResult{}, err
		}
		logger.Debug.Subprocess(layer.Path)
		logger.Break()

		layer, err = layer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}
		layer.Launch = true

		httpdConfPath := os.Getenv("PHP_HTTPD_PATH")
		nginxConfPath := os.Getenv("PHP_NGINX_PATH")

		if (httpdConfPath == "" && nginxConfPath == "") ||
			(httpdConfPath != "" && nginxConfPath != "") {
			return packit.BuildResult{}, errors.New("need exactly one of: $PHP_HTTPD_PATH or $PHP_NGINX_PATH")
		}

		shouldEnableReload, err := reloader.ShouldEnableLiveReload()
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Process("Determining start commands to include in procs.yml:")
		watchPaths := make([]string, 0)
		// HTTPD Case
		if httpdConfPath != "" {
			serverProc := NewProc("httpd", []string{"-f", httpdConfPath, "-k", "start", "-DFOREGROUND"})

			if exists, err := fs.Exists(filepath.Join(context.WorkingDir, ".httpd.conf.d")); err != nil {
				return packit.BuildResult{}, err
			} else if shouldEnableReload && exists {
				serverProc = NewProc("watchexec", []string{
					"--watch", "/workspace/.httpd.conf.d",
					"--on-busy-update", "signal",
					"--signal", "SIGHUP",
					"--", "httpd",
					"-f", httpdConfPath, "-k",
					"start",
					"-DFOREGROUND",
				})
			} else if shouldEnableReload && !exists {
				logger.Subprocess("HTTPD will not be reloadable since .httpd.conf.d folder not found")
			}

			procs.Add("httpd", serverProc)
			logger.Subprocess("HTTPD: %s %v", serverProc.Command, strings.Join(serverProc.Args, " "))
			watchPaths = append(watchPaths, "/workspace/.httpd.conf.d")
		}

		// Nginx Case
		if nginxConfPath != "" {
			serverProc := NewProc("nginx", []string{"-p", context.WorkingDir, "-c", nginxConfPath})

			if exists, err := fs.Exists(filepath.Join(context.WorkingDir, ".nginx.conf.d")); err != nil {
				return packit.BuildResult{}, err
			} else if shouldEnableReload && exists {
				serverProc = NewProc("watchexec", []string{
					"--watch", "/workspace/.nginx.conf.d",
					"--on-busy-update", "signal",
					"--signal", "SIGHUP",
					"--", "nginx",
					"-p", context.WorkingDir,
					"-c", nginxConfPath,
				})
			} else if shouldEnableReload && !exists {
				logger.Subprocess("NGINX will not be reloadable since .nginx.conf.d folder not found")
			}

			procs.Add("nginx", serverProc)
			logger.Subprocess("Nginx: %s %v", serverProc.Command, strings.Join(serverProc.Args, " "))
			watchPaths = append(watchPaths, "/workspace/.nginx.conf.d")
		}

		// FPM Case
		fpmConfPath, ok := os.LookupEnv("PHP_FPM_PATH")
		if !ok || fpmConfPath == "" {
			return packit.BuildResult{}, errors.New("failed to lookup $PHP_FPM_PATH")
		}

		phprcPath, ok := os.LookupEnv("PHPRC")
		if !ok || phprcPath == "" {
			return packit.BuildResult{}, errors.New("failed to lookup $PHPRC path for FPM")
		}
		fpmProc := NewProc("php-fpm", []string{"-y", fpmConfPath, "-c", phprcPath})

		if exists, err := fs.Exists(filepath.Join(context.WorkingDir, ".php.fpm.d")); err != nil {
			return packit.BuildResult{}, err
		} else if shouldEnableReload && exists {
			fpmProc = NewProc("watchexec", []string{
				"--watch", "/workspace/.php.fpm.d",
				"--on-busy-update", "signal",
				"--signal", "SIGHUP",
				"--", "php-fpm",
				"-y", fpmConfPath,
				"-c", phprcPath,
			})
		} else if shouldEnableReload && !exists {
			logger.Subprocess("FPM will not be reloadable since .php.fpm.d folder not found")
		}

		procs.Add("fpm", fpmProc)
		logger.Subprocess("FPM: %s %v", fpmProc.Command, strings.Join(fpmProc.Args, " "))

		// Write the process file
		logger.Debug.Subprocess("Writing process file to %s", filepath.Join(layer.Path, "procs.yml"))
		logger.Break()
		err = procs.WriteFile(filepath.Join(layer.Path, "procs.yml"))
		if err != nil {
			return packit.BuildResult{}, err
		}

		// Make the process manager binary available in layer
		logger.Debug.Process("Copying procmgr-binary into %s", filepath.Join(layer.Path, "bin", "procmgr-binary"))
		logger.Debug.Break()
		err = os.Mkdir(filepath.Join(layer.Path, "bin"), os.ModePerm)
		if err != nil {
			//untested
			return packit.BuildResult{}, err
		}

		err = fs.Copy(filepath.Join(context.CNBPath, "bin", "procmgr-binary"), filepath.Join(layer.Path, "bin", "procmgr-binary"))
		if err != nil {
			return packit.BuildResult{}, fmt.Errorf("failed to copy procmgr-binary into layer: %w", err)
		}

		originalProcess := packit.Process{
			Type:    "web",
			Command: "procmgr-binary",
			Args:    []string{filepath.Join(layer.Path, "procs.yml")},
			Default: true,
			Direct:  true,
		}

		processes := make([]packit.Process, 0)

		if _, err := reloader.ShouldEnableLiveReload(); err != nil {
			return packit.BuildResult{}, err
			//} else if shouldEnableReload {
			//	nonReloadableProcess, reloadableProcess := reloader.TransformReloadableProcesses(originalProcess, libreload.ReloadableProcessSpec{
			//		WatchPaths: watchPaths,
			//	})
			//	processes = append(processes, nonReloadableProcess, reloadableProcess)
		} else {
			processes = append(processes, originalProcess)
		}

		logger.LaunchProcesses(processes)

		return packit.BuildResult{
			Layers: []packit.Layer{layer},
			Launch: packit.LaunchMetadata{
				Processes: processes,
			},
		}, nil
	}
}
