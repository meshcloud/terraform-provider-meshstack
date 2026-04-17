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

	workspaceConfig, workspaceAddr := testconfig.Workspace(t)
	projectConfig, projectAddr := testconfig.Project(t, workspaceAddr)
	platformConfig, platformAddr, _ := testconfig.CustomPlatform(t, workspaceAddr)
	landingZoneConfig, landingZoneAddr := testconfig.SimpleLandingZone(t, workspaceAddr, platformAddr)
	tenantConfig, tenantAddr := testconfig.TenantV3(t, projectAddr, platformAddr, landingZoneAddr)

	config := tenantConfig.Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(tenantAddr.String(), plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("metadata").AtMapKey("owned_by_project"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(tenantAddr.String(), tfjsonpath.New("spec").AtMapKey("landing_zone_identifier"), xknownvalue.NotEmptyString()),
				},
			},
		},
	})
}
