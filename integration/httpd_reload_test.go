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

			source, err = occam.Source(filepath.Join("testdata", "httpd_conf_reload"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("will reload httpd config files", func() {
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
					watchexecBuildpack,
					buildpack,
				).
				WithEnv(map[string]string{
					"BP_LOG_LEVEL":           "DEBUG",
					"BP_PHP_SERVER":          "httpd",
					"BP_LIVE_RELOAD_ENABLED": "true",
				}).
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), logs.String)

			Expect(logs).To(ContainLines(
				"  Determining start commands to include in procs.yml:",
				MatchRegexp(`    HTTPD: watchexec --watch /workspace/\.httpd\.conf\.d --on-busy-update signal --signal SIGHUP --shell none -- httpd -f /layers/.*/php-httpd-config/httpd\.conf -k start -DFOREGROUND`),
				MatchRegexp(`    FPM: watchexec --watch /workspace/\.php\.fpm\.d --on-busy-update signal --signal SIGUSR2 --shell none -- php-fpm -y /layers/.*/php-fpm-config/base.conf -c /layers/.*/php/etc`),
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

			Eventually(container).Should(Serve(WithHeader("X-Paketo-Testing", "original-httpd-value")).
				OnPort(8080).
				WithEndpoint("/index.php?date"))

			err = docker.Container.Exec.ExecuteBash(container.ID, "sed -i 's/original-httpd-value/reloaded-httpd-value/g' /workspace/.httpd.conf.d/header-server.conf")
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(Serve(WithHeader("X-Paketo-Testing", "reloaded-httpd-value")).
				OnPort(8080).
				WithEndpoint("/index.php?date"))
		})

		it("will reload FPM config files", func() {
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
					watchexecBuildpack,
					buildpack,
				).
				WithEnv(map[string]string{
					"BP_LOG_LEVEL":           "DEBUG",
					"BP_PHP_SERVER":          "httpd",
					"BP_LIVE_RELOAD_ENABLED": "true",
				}).
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), logs.String)

			Expect(logs).To(ContainLines(
				"  Determining start commands to include in procs.yml:",
				MatchRegexp(`    HTTPD: watchexec --watch /workspace/\.httpd\.conf\.d --on-busy-update signal --signal SIGHUP --shell none -- httpd -f /layers/.*/php-httpd-config/httpd\.conf -k start -DFOREGROUND`),
				MatchRegexp(`    FPM: watchexec --watch /workspace/\.php\.fpm\.d --on-busy-update signal --signal SIGUSR2 --shell none -- php-fpm -y /layers/.*/php-fpm-config/base.conf -c /layers/.*/php/etc`),
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

			Eventually(container).Should(Serve(ContainSubstring("SUCCESS: date loads.")).OnPort(8080).WithEndpoint("/index.php?date"))

			err = docker.Container.Exec.ExecuteBash(container.ID, "sed -i 's/pm.max_children = 5/pm.max_children = 6/g' /workspace/.php.fpm.d/user.conf")
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				cLogs, err := docker.Container.Logs.Execute(container.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}).Should(And(
				ContainSubstring(`Reloading in progress ...`),
				ContainSubstring(`reloading: execvp("php-fpm", {"php-fpm", "-y"`),
			))
			Eventually(container).Should(Serve(ContainSubstring("SUCCESS: date loads.")).OnPort(8080).WithEndpoint("/index.php?date"))
		})
	})
}
