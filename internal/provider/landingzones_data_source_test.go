package provider

import (
	"testing"

	tfconfig "github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccLandingZonesDataSource(t *testing.T) {
	t.Parallel()

	// plain listing creates a landing zone in a fresh workspace and lists it back by platform_uuid,
	// running identically in mock and acceptance mode.
	t.Run("plain listing", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := platformTypeStepVars(suffix)
		dataSourceAddr := "data.meshstack_landingzones.for_platform"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "data-sources/landingzones/plain_listing", 1, vars),
				},
				{
					Config: hclStep(t, "data-sources/landingzones/plain_listing", 2, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("landing_zones"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("metadata").AtMapKey("name"), knownvalue.StringExact("test-lz-"+suffix)),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("ref").AtMapKey("kind"), knownvalue.StringExact("meshLandingZone")),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("ref").AtMapKey("name"), knownvalue.StringExact("test-lz-"+suffix)),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("spec").AtMapKey("platform_ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					},
				},
			},
		})
	})

	// cross-workspace listing proves a consumer workspace's restricted key can list the landing zones of
	// a platform published (RESTRICTED) to it. The positive assertion runs in both modes.
	t.Run("cross-workspace listing", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := platformTypeStepVars(suffix)
		landingZoneAddr := "meshstack_landingzone.test"
		apiKeyAddr := "meshstack_api_key.test"
		dataSourceAddress := "data.meshstack_landingzones.for_platform"

		var apiKeyClientId, apiKeyClientSecret lazyVariable
		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/landingzones/cross_workspace_listing", 1, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(apiKeyAddr, tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString(func(clientId string) error {
						apiKeyClientId = lazyVariable(clientId)
						return nil
					})),
					statecheck.ExpectKnownValue(apiKeyAddr, tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString(func(clientSecret string) error {
						apiKeyClientSecret = lazyVariable(clientSecret)
						return nil
					})),
					statecheck.ExpectKnownValue(landingZoneAddr, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("test-lz-"+suffix)),
				},
			},
			{
				Config: hclStep(t, "data-sources/landingzones/cross_workspace_listing", 2, vars),
				ConfigVariables: tfconfig.Variables{
					"apikey_client_id":     &apiKeyClientId,
					"apikey_client_secret": &apiKeyClientSecret,
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("landing_zones"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("ref").AtMapKey("kind"), knownvalue.StringExact("meshLandingZone")),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("spec").AtMapKey("platform_ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
				},
			},
		}})
	})
}
