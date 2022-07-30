package integration_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testHttpdReload(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker
		source string
		name   string
	)

	it.Before(func() {
		pack = occam.NewPack().WithVerbose()
		docker = occam.NewDocker()
	})

	context("when BP_LIVE_RELOAD_ENABLED is true", func() {
		var (
			image     occam.Image
			container occam.Container
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())

			source, err = occam.Source(filepath.Join("testdata", "default_app"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("successfully starts a reloadable PHP app with HTTPD and FPM", func() {
			var (
				logs fmt.Stringer
				err  error
			)

			image, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					phpDistBuildpack,
					phpFpmBuildpack,
					httpdBuildpack,
					phpHttpdBuildpack,
					buildpack,
				).
				WithEnv(map[string]string{
					"BP_LOG_LEVEL":  "DEBUG",
					"BP_PHP_SERVER": "httpd",
				}).
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), logs.String)

			Expect(logs).To(ContainLines(
				"  Assigning launch processes:",
				fmt.Sprintf("    web (default): procmgr-binary /layers/%s/php-start/procs.yml", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
			))

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(Serve(ContainSubstring("SUCCESS: date loads.")).OnPort(8080).WithEndpoint("/index.php?date"))

			buffer := bytes.NewBuffer(nil)

			err = pexec.NewExecutable("docker").Execute(pexec.Execution{
				Args: []string{
					"exec",
					container.ID,
					"/bin/bash",
					"-c",
					"sed -i 's/SUCCESS/RELOADED/g' /workspace/htdocs/index.php",
				},
				Stdout: buffer,
				Stderr: buffer,
			})
			Expect(err).NotTo(HaveOccurred(), buffer.String())

			Eventually(container).Should(Serve(ContainSubstring("RELOADED: date loads.")).OnPort(8080).WithEndpoint("/index.php?date"))
		})
	})
}
