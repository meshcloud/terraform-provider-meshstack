package provider

import (
	"testing"

	tfconfig "github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccLandingZonesDataSource(t *testing.T) {
	t.Parallel()

	// plain listing creates a landing zone in a fresh workspace and lists it back by platform_uuid,
	// running identically in mock and acceptance mode.
	t.Run("plain listing", func(t *testing.T) {
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)
		platformConfig, platformAddr, platformTypeAddr := testconfig.CustomPlatform(t, workspaceAddr)
		landingZoneConfig, landingZoneAddr := testconfig.LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)

		dataSourceAddr := "data.meshstack_landingzones.for_platform"
		config := testconfig.DataSource{Name: "landingzones"}.Config(t).WithFirstBlock(
			testconfig.Descend("platform_uuid")(testconfig.SetAddr(platformAddr, "metadata", "uuid")),
			testconfig.Descend("depends_on")(testconfig.SetRawExpr("[%s]", landingZoneAddr)),
		).Join(landingZoneConfig, platformConfig, workspaceConfig)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("landing_zones"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("metadata").AtMapKey("name"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("ref").AtMapKey("kind"), knownvalue.StringExact("meshLandingZone")),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("ref").AtMapKey("name"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("spec").AtMapKey("platform_ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					},
				},
			},
		})
	})

	// cross-workspace listing proves a consumer workspace's restricted key can list the landing zones of
	// a platform published (RESTRICTED) to it. The positive assertion runs in both modes.
	t.Run("cross-workspace listing", func(t *testing.T) {
		operatorWorkspaceConfig, operatorWorkspaceAddr := testconfig.Workspace(t)
		platformConfig, platformAddr, platformTypeAddr := testconfig.CustomPlatform(t, operatorWorkspaceAddr)
		landingZoneConfig, landingZoneAddr := testconfig.LandingZone(t, operatorWorkspaceAddr, platformAddr, platformTypeAddr)

		var consumerWorkspaceAddr testconfig.Traversal
		consumerWorkspaceConfig, _ := testconfig.Workspace(t)
		consumerWorkspaceConfig = consumerWorkspaceConfig.WithFirstBlock(
			testconfig.RenameKey("other"),
			testconfig.ExtractAddress(&consumerWorkspaceAddr),
		)

		platformConfig = platformConfig.WithFirstBlock(
			testconfig.Descend("spec", "availability")(
				testconfig.Descend("restriction")(testconfig.SetString("RESTRICTED")),
				testconfig.Descend("publication_state")(testconfig.SetString("PUBLISHED")),
				testconfig.Descend("restricted_to_workspaces")(testconfig.SetRawExpr("[%s, %s]",
					operatorWorkspaceAddr.Join("metadata", "name"), consumerWorkspaceAddr.Join("metadata", "name"))),
			),
		)

		apiKeyConfig, apiKeyAddr := testconfig.ApiKey(t, consumerWorkspaceAddr)
		apiKeyConfig = apiKeyConfig.WithFirstBlock(
			testconfig.Descend("spec", "permissions")(testconfig.SetRawExpr(`["LANDINGZONE_LIST"]`)),
		)

		supportConfig := landingZoneConfig.Join(operatorWorkspaceConfig, platformConfig, consumerWorkspaceConfig, apiKeyConfig)

		var dataSourceAddress testconfig.Traversal
		example := testconfig.DataSource{Name: "landingzones"}
		config := example.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&dataSourceAddress),
			testconfig.Descend("platform_uuid")(testconfig.SetAddr(platformAddr, "metadata", "uuid")),
			testconfig.Descend("provider")(testconfig.SetRawExpr("meshstack-other")),
		).Join(supportConfig, testconfig.OtherProviderConfig(t))

		var apiKeyClientId, apiKeyClientSecret lazyVariable
		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: supportConfig.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString(func(clientId string) error {
						apiKeyClientId = lazyVariable(clientId)
						return nil
					})),
					statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString(func(clientSecret string) error {
						apiKeyClientSecret = lazyVariable(clientSecret)
						return nil
					})),
					statecheck.ExpectKnownValue(landingZoneAddr.String(), tfjsonpath.New("metadata").AtMapKey("name"), xknownvalue.NotEmptyString()),
				},
			},
			{
				Config: config.String(),
				ConfigVariables: tfconfig.Variables{
					"apikey_client_id":     &apiKeyClientId,
					"apikey_client_secret": &apiKeyClientSecret,
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("landing_zones"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("ref").AtMapKey("kind"), knownvalue.StringExact("meshLandingZone")),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("landing_zones").AtSliceIndex(0).AtMapKey("spec").AtMapKey("platform_ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
				},
			},
		}})
	})
}
