package phpstart_test

import (
	"os"
	"testing"

	"github.com/paketo-buildpacks/packit/v2"
	phpstart "github.com/paketo-buildpacks/php-start"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		detect     packit.DetectFunc
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		detect = phpstart.Detect()
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("Detect", func() {
		it("requires php, php-fpm, httpd, httpd-conf, and httpd-start and provides httpd-start", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Requires: []packit.BuildPlanRequirement{
					{
						Name: "php",
						Metadata: phpstart.BuildPlanMetadata{
							Build: true,
						},
					},
					{
						Name: "php-fpm",
						Metadata: phpstart.BuildPlanMetadata{
							Build:  true,
							Launch: true,
						},
					},
					{
						Name: "httpd",
						Metadata: phpstart.BuildPlanMetadata{
							Launch: true,
						},
					},
					{
						Name: "httpd-config",
						Metadata: phpstart.BuildPlanMetadata{
							Launch: true,
							Build:  true,
						},
					},
				},
				Or: []packit.BuildPlan{
					{
						Requires: []packit.BuildPlanRequirement{
							{
								Name: "php",
								Metadata: phpstart.BuildPlanMetadata{
									Build: true,
								},
							},
							{
								Name: "php-fpm",
								Metadata: phpstart.BuildPlanMetadata{
									Build:  true,
									Launch: true,
								},
							},
							{
								Name: "nginx",
								Metadata: phpstart.BuildPlanMetadata{
									Launch: true,
								},
							},
							{
								Name: "nginx-config",
								Metadata: phpstart.BuildPlanMetadata{
									Launch: true,
									Build:  true,
								},
							},
						},
					},
				},
			}))
		})
	})
}
