package phpstart

import (
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/fs"
)

// BuildPlanMetadata is the buildpack specific data included in build plan
// requirements.
type BuildPlanMetadata struct {

	// Launch flag requests the given requirement be made available during the
	// launch phase of the buildpack lifecycle.
	Launch bool `toml:"launch"`

	// Build flag requests the given requirement be made available during the
	// build phase of the buildpack lifecycle.
	Build bool `toml:"build"`
}

// Detect will return a packit.DetectFunc that will be invoked during the
// detect phase of the buildpack lifecycle.
//
// This buildpack has two requirement groups:
// One for HTTPD in which the following are required at launch time:
// - "php"
// - "php-fpm"
// - "httpd"
// - "httpd-config"
// Another for HTTPD in which the following are required at launch time:
// - "php"
// - "php-fpm"
// - "nginx"
// - "nginx-config"
//
// Additionally, this buildpack will require 'composer-packages' when a composer.json is found.
//
// This buildpack will always detect.
func Detect() packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		baseRequirements := []packit.BuildPlanRequirement{
			{
				Name: Php,
				Metadata: BuildPlanMetadata{
					Build: true,
				},
			},
			{
				Name: PhpFpm,
				Metadata: BuildPlanMetadata{
					Build:  true,
					Launch: true,
				},
			},
		}

		composerJsonPath := filepath.Join(context.WorkingDir, "composer.json")

		if value, found := os.LookupEnv("COMPOSER"); found {
			composerJsonPath = filepath.Join(context.WorkingDir, value)
		}

		if exists, err := fs.Exists(composerJsonPath); err != nil {
			return packit.DetectResult{}, err
		} else if exists {
			baseRequirements = append(baseRequirements, packit.BuildPlanRequirement{
				Name: "composer-packages",
				Metadata: BuildPlanMetadata{
					Launch: true,
				},
			})
		}

		httpdFpmPlan := packit.BuildPlan{
			Requires: []packit.BuildPlanRequirement{
				{
					Name: Httpd,
					Metadata: BuildPlanMetadata{
						Launch: true,
					},
				},
				{
					Name: PhpHttpdConfig,
					Metadata: BuildPlanMetadata{
						Build:  true,
						Launch: true,
					},
				},
			},
		}

		nginxFpmPlan := packit.BuildPlan{
			Requires: []packit.BuildPlanRequirement{
				{
					Name: Nginx,
					Metadata: BuildPlanMetadata{
						Launch: true,
					},
				},
				{
					Name: PhpNginxConfig,
					Metadata: BuildPlanMetadata{
						Build:  true,
						Launch: true,
					},
				},
			},
		}

		httpdFpmPlan.Requires = append(baseRequirements, httpdFpmPlan.Requires...)
		nginxFpmPlan.Requires = append(baseRequirements, nginxFpmPlan.Requires...)
		httpdFpmPlan.Or = []packit.BuildPlan{nginxFpmPlan}

		return packit.DetectResult{
			Plan: httpdFpmPlan,
		}, nil
	}
}
