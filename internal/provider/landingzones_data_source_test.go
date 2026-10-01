package provider

import (
	"testing"

	tfconfig "github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

const landingZonesDataSourceAddr = "data.meshstack_landingzones.for_platform"

func TestAccLandingZonesDataSource(t *testing.T) {
	t.Parallel()

	// plain listing creates a landing zone in a fresh workspace and lists it back by platform_uuid,
	// running identically in mock and acceptance mode.
	t.Run("plain listing", func(t *testing.T) {
		suffix := acctest.RandString(8)

		config := examples.JoinTestStepConfigs(
			examples.DataSource.TestStepConfig(t, "landingzones", 1),
			landingZoneStepConfig(t, 1),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          config,
					ConfigVariables: SuffixVariables(suffix),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(landingZonesDataSourceAddr, tfjsonpath.New("landing_zones"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(landingZonesDataSourceAddr, tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("metadata").AtMapKey("name"), knownvalue.StringExact("test-lz-"+suffix)),
						statecheck.ExpectKnownValue(landingZonesDataSourceAddr, tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("ref").AtMapKey("kind"), knownvalue.StringExact("meshLandingZone")),
						statecheck.ExpectKnownValue(landingZonesDataSourceAddr, tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("ref").AtMapKey("name"), knownvalue.StringExact("test-lz-"+suffix)),
						statecheck.ExpectKnownValue(landingZonesDataSourceAddr, tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("spec").AtMapKey("platform_ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					},
				},
			},
		})
	})

	// cross-workspace listing proves a consumer workspace's restricted key can list the landing zones of
	// a platform published (RESTRICTED) to it. The positive assertion runs in both modes.
	t.Run("cross-workspace listing", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := SuffixVariables(suffix)

		// Platform variant 9 is the RESTRICTED one, published to the consumer workspace as well.
		supportConfig := examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "landingzone", 1, "bbd"),
			examples.Resource.TestStepConfig(t, "platform", 9),
			examples.Resource.TestStepConfig(t, "platform_type", 1),
			examples.Resource.TestStepConfig(t, "api_key", 5),
			examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites", "consumer-workspace"),
		)

		listConfig := examples.JoinTestStepConfigs(
			examples.DataSource.TestStepConfig(t, "landingzones", 2),
			supportConfig,
			examples.Resource.TestSupportConfigs(t, "api_key", "other_provider"),
		)

		var apiKeyClientId, apiKeyClientSecret lazyVariable
		listVars := tfconfig.Variables{
			"apikey_client_id":     &apiKeyClientId,
			"apikey_client_secret": &apiKeyClientSecret,
		}
		for name, value := range vars {
			listVars[name] = value
		}

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config:          supportConfig,
				ConfigVariables: vars,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString(func(clientId string) error {
						apiKeyClientId = lazyVariable(clientId)
						return nil
					})),
					statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString(func(clientSecret string) error {
						apiKeyClientSecret = lazyVariable(clientSecret)
						return nil
					})),
					statecheck.ExpectKnownValue(landingZoneResourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("test-lz-"+suffix)),
				},
			},
			{
				Config:          listConfig,
				ConfigVariables: listVars,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(landingZonesDataSourceAddr, tfjsonpath.New("landing_zones"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(landingZonesDataSourceAddr, tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("ref").AtMapKey("kind"), knownvalue.StringExact("meshLandingZone")),
					statecheck.ExpectKnownValue(landingZonesDataSourceAddr, tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("spec").AtMapKey("platform_ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
				},
			},
		}})
	})
}
