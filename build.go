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
func Build(procs ProcMgr, logger scribe.Emitter) packit.BuildFunc {
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

		logger.Process("Determining start commands to include in procs.yml:")
		// HTTPD Case
		httpdConfPath := os.Getenv("PHP_HTTPD_PATH")
		if httpdConfPath != "" {
			serverProc := NewProc("httpd", []string{"-f", httpdConfPath, "-k", "start", "-DFOREGROUND"})
			procs.Add("httpd", serverProc)
			logger.Subprocess("HTTPD: %s %v", serverProc.Command, strings.Join(serverProc.Args, " "))
		}

		// FPM Case
		fpmConfPath := os.Getenv("PHP_FPM_PATH")
		if fpmConfPath != "" {
			phprcPath, ok := os.LookupEnv("PHPRC")
			if !ok {
				return packit.BuildResult{}, errors.New("failed to lookup $PHPRC path for FPM")
			}
			fpmProc := NewProc("php-fpm", []string{"-y", fpmConfPath, "-c", phprcPath})
			procs.Add("fpm", fpmProc)
			logger.Subprocess("FPM: %s %v", fpmProc.Command, strings.Join(fpmProc.Args, " "))
		}

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

		processes := []packit.Process{
			{
				Type:    "web",
				Command: "procmgr-binary",
				Args:    []string{filepath.Join(layer.Path, "procs.yml")},
				Default: true,
				Direct:  true,
			},
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
