package phpstart

import (
	"github.com/paketo-buildpacks/packit/v2"
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
// This buildpack has a less-common provides/requires structure. There are two
// main requirement groups:
// One for HTTPD in which "php", "php-fpm", "httpd"
// and "httpd-config" are required at launch-time.
// The second one for for Nginx in which "php", "php-fpm", "nginx"
// and "nginx-config" are required at launch-time.

// This buildpack will always detect, and in the case of HTTPD, the buildpack
// will provide and require `httpd-start`. In the case of Nginx, the buildpack
// will provide and require `nginx-start`. This is unusual, but will allow the
// buildpack Build function access to which web server start command is needed,
// since the requirements are not easily checked otherwise.
func Detect() packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		baseRequirements := []packit.BuildPlanRequirement{
			{
				Name: "php",
				Metadata: BuildPlanMetadata{
					Build: true,
				},
			},
			{
				Name: "php-fpm",
				Metadata: BuildPlanMetadata{
					Build:  true,
					Launch: true,
				},
			},
		}

		httpdFpmPlan := packit.BuildPlan{
			Requires: []packit.BuildPlanRequirement{
				{
					Name: "httpd",
					Metadata: BuildPlanMetadata{
						Launch: true,
					},
				},
				{
					Name: "httpd-config",
					Metadata: BuildPlanMetadata{
						Build:  true,
						Launch: true,
					},
				},
			},
		}

		httpdFpmPlan.Requires = append(baseRequirements, httpdFpmPlan.Requires...)

		// Eventually we can create a slice of plans that are Or'd together for
		// HTTPD, Nginx, and the FPM included cases.
		// See python-start detect phase as an example
		return packit.DetectResult{
			Plan: httpdFpmPlan,
		}, nil
	}
}
