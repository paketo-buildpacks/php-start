package phpstart_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/paketo-buildpacks/packit/v2"
	phpstart "github.com/paketo-buildpacks/php-start"
	"github.com/sclevine/spec"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		detect     packit.DetectFunc
	)

	it.Before(func() {
		workingDir = t.TempDir()

		detect = phpstart.Detect()
	})

	context("Detect", func() {
		it("requires either [php, php-fpm, httpd, php-httpd-config] or [php, php-fpm, nginx, php-nginx-config]", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Requires: []packit.BuildPlanRequirement{
					{
						Name: phpstart.Php,
						Metadata: phpstart.BuildPlanMetadata{
							Build: true,
						},
					},
					{
						Name: phpstart.PhpFpm,
						Metadata: phpstart.BuildPlanMetadata{
							Build:  true,
							Launch: true,
						},
					},
					{
						Name: phpstart.Httpd,
						Metadata: phpstart.BuildPlanMetadata{
							Launch: true,
						},
					},
					{
						Name: phpstart.PhpHttpdConfig,
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
								Name: phpstart.Php,
								Metadata: phpstart.BuildPlanMetadata{
									Build: true,
								},
							},
							{
								Name: phpstart.PhpFpm,
								Metadata: phpstart.BuildPlanMetadata{
									Build:  true,
									Launch: true,
								},
							},
							{
								Name: phpstart.Nginx,
								Metadata: phpstart.BuildPlanMetadata{
									Launch: true,
								},
							},
							{
								Name: phpstart.PhpNginxConfig,
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

		context("composer", func() {
			context("with composer.json", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(workingDir, "composer.json"), []byte(""), os.ModePerm)).To(Succeed())
				})

				it("requires composer-packages at launch", func() {
					result, err := detect(packit.DetectContext{
						WorkingDir: workingDir,
					})
					Expect(err).NotTo(HaveOccurred())

					Expect(result.Plan.Requires).To(ContainElements(packit.BuildPlanRequirement{
						Name: "composer-packages",
						Metadata: phpstart.BuildPlanMetadata{
							Launch: true,
						},
					}))
				})
			})

			context("with $COMPOSER", func() {
				it.Before(func() {
					t.Setenv("COMPOSER", "some/other-file.json")
				})

				context("that points to an existing file", func() {
					it.Before(func() {
						Expect(os.Mkdir(filepath.Join(workingDir, "some"), os.ModeDir|os.ModePerm)).To(Succeed())
						Expect(os.WriteFile(filepath.Join(workingDir, "some", "other-file.json"), []byte(""), os.ModePerm)).To(Succeed())
					})

					it("requires composer-packages at launch", func() {
						result, err := detect(packit.DetectContext{
							WorkingDir: workingDir,
						})
						Expect(err).NotTo(HaveOccurred())

						Expect(result.Plan.Requires).To(ContainElements(packit.BuildPlanRequirement{
							Name: "composer-packages",
							Metadata: phpstart.BuildPlanMetadata{
								Launch: true,
							},
						}))
					})
				})

				context("that points to a non existing file", func() {
					it("does not require composer-packages", func() {
						result, err := detect(packit.DetectContext{
							WorkingDir: workingDir,
						})
						Expect(err).NotTo(HaveOccurred())

						Expect(result.Plan.Requires).ToNot(ContainElements(MatchFields(IgnoreExtras, Fields{
							"Name": Equal("composer-packages"),
						})))
					})
				})
			})
		}, spec.Sequential())
	})
}
