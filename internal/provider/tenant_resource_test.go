package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccTenant(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("Skipping: tenant v3 resource has no wait_for_completion on delete, workspace cleanup fails")
	}

	workspaceConfig, workspaceAddr := testconfig.BuildWorkspaceConfig(t)
	projectConfig, projectAddr := testconfig.BuildProjectConfig(t, workspaceAddr)
	platformConfig, platformAddr, _ := testconfig.BuildCustomPlatformConfig(t, workspaceAddr)
	landingZoneConfig, landingZoneAddr := testconfig.BuildSimpleLandingZoneConfig(t, workspaceAddr, platformAddr)

	tenantConfig := testconfig.Resource{Name: "tenant"}.Config(t)
	var resourceAddress testconfig.Traversal
	tenantConfig = tenantConfig.WithFirstBlock(t,
		testconfig.ExtractIdentifier(&resourceAddress),
		testconfig.Traverse(t, "metadata")(
			testconfig.Traverse(t, "owned_by_workspace")(testconfig.SetRawExpr(projectAddr.Format("%s.metadata.owned_by_workspace"))),
			testconfig.Traverse(t, "owned_by_project")(testconfig.SetRawExpr(projectAddr.Format("%s.metadata.name"))),
			testconfig.Traverse(t, "platform_identifier")(testconfig.SetRawExpr(platformAddr.Format(`"${%s.metadata.name}.${%s.spec.location_ref.name}"`, platformAddr.String()))),
		),
		testconfig.Traverse(t, "spec")(
			testconfig.Traverse(t, "landing_zone_identifier")(testconfig.SetRawExpr(landingZoneAddr.Format("%s.metadata.name"))),
		),
	)

	config := tenantConfig.Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig)

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
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata").AtMapKey("owned_by_project"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec").AtMapKey("landing_zone_identifier"), xknownvalue.NotEmptyString()),
				},
			},
		},
	})
}
