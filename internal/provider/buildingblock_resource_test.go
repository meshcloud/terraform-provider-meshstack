package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccBuildingBlock(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("Skipping: BB v1 resource has no wait_for_completion, BB run stays PENDING and blocks destroy")
	}

	// Build a full tenant chain: workspace + project + platform + landing zone + tenant
	workspaceConfig, workspaceAddr := testconfig.BuildWorkspaceConfig(t)
	projectConfig, projectAddr := testconfig.BuildProjectConfig(t, workspaceAddr)
	platformConfig, platformAddr, platformTypeAddr := testconfig.BuildCustomPlatformConfig(t, workspaceAddr)
	landingZoneConfig, landingZoneAddr := testconfig.BuildLandingZoneConfig(t, workspaceAddr, platformAddr, platformTypeAddr)

	tenantConfig := testconfig.Resource{Name: "tenant_v4"}.Config(t)
	var tenantAddr testconfig.Traversal
	tenantConfig = tenantConfig.WithFirstBlock(t,
		testconfig.ExtractIdentifier(&tenantAddr),
		testconfig.Traverse(t, "metadata")(
			testconfig.Traverse(t, "owned_by_workspace")(testconfig.SetRawExpr(projectAddr.Format("%s.metadata.owned_by_workspace"))),
			testconfig.Traverse(t, "owned_by_project")(testconfig.SetRawExpr(projectAddr.Format("%s.metadata.name"))),
		),
		testconfig.Traverse(t, "spec")(
			testconfig.Traverse(t, "platform_identifier")(testconfig.SetRawExpr(platformAddr.Format(`"${%s.metadata.name}.${%s.spec.location_ref.name}"`, platformAddr.String()))),
			testconfig.Traverse(t, "landing_zone_identifier")(testconfig.SetRawExpr(landingZoneAddr.Format("%s.metadata.name"))),
		),
	)

	// Create a tenant-level BBD using the BB v2 tenant test-support file
	bbdConfig := testconfig.Resource{Name: "building_block_v2", Suffix: "_02_tenant"}.TestSupportConfig(t, "")
	bbdConfig = bbdConfig.WithFirstBlock(t, testconfig.OwnedByWorkspace(t, workspaceAddr))
	bbdConfig = bbdConfig.WithFirstBlock(t,
		testconfig.Traverse(t, "spec", "supported_platforms")(testconfig.SetRawExpr(platformTypeAddr.Format("[{ name = %s.metadata.name }]"))))

	// Build BB v1 config, replacing hardcoded values with resource references
	bbConfig := testconfig.Resource{Name: "buildingblock"}.Config(t)
	var resourceAddress testconfig.Traversal
	bbConfig = bbConfig.WithFirstBlock(t,
		testconfig.ExtractIdentifier(&resourceAddress),
		testconfig.Traverse(t, "metadata")(
			testconfig.Traverse(t, "definition_uuid")(testconfig.SetRawExpr(`meshstack_building_block_definition.example_tenant.ref.uuid`)),
			testconfig.Traverse(t, "definition_version")(testconfig.SetRawExpr(`meshstack_building_block_definition.example_tenant.version_latest.number`)),
			testconfig.Traverse(t, "tenant_identifier")(testconfig.SetRawExpr(
				tenantAddr.Format(`"${%s.metadata.owned_by_workspace}.${%s.metadata.owned_by_project}.${%s.spec.platform_identifier}"`,
					tenantAddr.String(), tenantAddr.String()))),
		),
		testconfig.Traverse(t, "spec", "inputs", "environment")(testconfig.SetRawExpr(`{ value_single_select = "dev" }`)),
	)

	config := bbConfig.Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig, tenantConfig, bbdConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("my-buildingblock")),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("status").AtMapKey("status"), xknownvalue.NotEmptyString()),
				},
			},
		},
	})
}
