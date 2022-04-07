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

func testNginx(t *testing.T, context spec.G, it spec.S) {
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

	context("when the buildpack is run with pack build", func() {
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

		context("Nginx and FPM", func() {
			it("successfully starts a PHP app with Nginx and FPM", func() {
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
					MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, buildpackInfo.Buildpack.Name)),
					"  Getting the layer associated with the server start command",
					fmt.Sprintf("    /layers/%s/php-start", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
				))

				Expect(logs).To(ContainLines(
					"  Determining start commands to include in procs.yml:",
					`    Nginx: nginx -p /workspace -c /workspace/nginx.conf`,
					MatchRegexp(`    FPM: php-fpm -y \/layers\/.*\/.*\/base.conf -c \/layers\/.*\/.*\/etc`),
					fmt.Sprintf("    Writing process file to /layers/%s/php-start/procs.yml", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
				))

				Expect(logs).To(ContainLines(
					fmt.Sprintf("  Copying procmgr-binary into /layers/%s/php-start/bin/procmgr-binary", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
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
			})
		})
	})
}
