package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testNginxReload(t *testing.T, context spec.G, it spec.S) {
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

	context("when no additional configuration specified", func() {
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

		it("automatically reloads served files", func() {
			var (
				logs fmt.Stringer
				err  error
			)

			image, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					phpDistBuildpack,
					phpFpmBuildpack,
					nginxBuildpack,
					phpNginxBuildpack,
					buildpack,
				).
				WithEnv(map[string]string{
					"BP_LOG_LEVEL":  "DEBUG",
					"BP_PHP_SERVER": "nginx",
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

			err = docker.Container.Exec.ExecuteBash(container.ID, "sed -i 's/SUCCESS/RELOADED/g' /workspace/htdocs/index.php")
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(Serve(ContainSubstring("RELOADED: date loads.")).OnPort(8080).WithEndpoint("/index.php?date"))
		})
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

			source, err = occam.Source(filepath.Join("testdata", "nginx_conf_reload"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			//Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			//Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			t.Logf("container.ID: %s", container.ID)
			t.Logf("image.ID: %s", image.ID)
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("will reload nginx config files", func() {
			var (
				logs fmt.Stringer
				err  error
			)

			image, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					phpDistBuildpack,
					phpFpmBuildpack,
					nginxBuildpack,
					phpNginxBuildpack,
					watchexecBuildpack,
					buildpack,
				).
				WithEnv(map[string]string{
					"BP_LOG_LEVEL":           "DEBUG",
					"BP_PHP_SERVER":          "nginx",
					"BP_LIVE_RELOAD_ENABLED": "true",
				}).
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), logs.String)

			Expect(logs).To(ContainLines(
				"  Determining start commands to include in procs.yml:",
				MatchRegexp(`    Nginx: watchexec --watch /workspace/\.nginx\.conf\.d --on-busy-update signal --signal SIGHUP -- nginx -p /workspace -c /workspace/nginx\.conf`),
				MatchRegexp(`    FPM: watchexec --watch /workspace/\.php\.fpm\.d --on-busy-update signal --signal SIGHUP -- php-fpm -y /layers/.*/php-fpm-config/base.conf -c /layers/.*/php/etc`),
			))

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

			Eventually(container).Should(Serve(WithHeader("X-Paketo-Testing", "original-nginx-value")).
				OnPort(8080).
				WithEndpoint("/index.php?date"))

			err = docker.Container.Exec.ExecuteBash(container.ID, "sed -i 's/original-nginx-value/reloaded-nginx-value/g' /workspace/.nginx.conf.d/header-server.conf")
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(Serve(WithHeader("X-Paketo-Testing", "reloaded-nginx-value")).
				OnPort(8080).
				WithEndpoint("/index.php?date"))
		})
	})
}
