package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccTenantDataSource(t *testing.T) {
	// The singular meshstack_tenant data source resolves the tenant via the list endpoint by the real
	// platform identifier, which the mock (storing platform by ref uuid) cannot reproduce.
	if IsMockClientTest() {
		t.Skip("meshstack_tenant data source composite lookup requires a real meshStack")
	}

	workspaceConfig, workspaceAddr := testconfig.Workspace(t)
	projectConfig, projectAddr := testconfig.Project(t, workspaceAddr)
	platformConfig, platformAddr, platformTypeAddr := testconfig.CustomPlatform(t, workspaceAddr)
	landingZoneConfig, landingZoneAddr := testconfig.LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)
	tenantConfig, tenantAddr := testconfig.Tenant(t, projectAddr, platformAddr, landingZoneAddr)

	dsAddress := testconfig.Traversal{"data.meshstack_tenant", "name"}
	config := testconfig.DataSource{Name: "tenant"}.Config(t).WithFirstBlock(
		testconfig.Descend("metadata")(
			testconfig.Descend("owned_by_workspace")(testconfig.SetAddr(tenantAddr, "metadata", "owned_by_workspace")),
			testconfig.Descend("owned_by_project")(testconfig.SetAddr(tenantAddr, "metadata", "owned_by_project")),
			testconfig.Descend("platform_identifier")(testconfig.SetAddr(platformAddr, "identifier")),
		),
	).Join(workspaceConfig, projectConfig, platformConfig, landingZoneConfig, tenantConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dsAddress.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dsAddress.String(), tfjsonpath.New("spec").AtMapKey("platform_ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dsAddress.String(), tfjsonpath.New("status").AtMapKey("tenant_name"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dsAddress.String(), tfjsonpath.New("status").AtMapKey("platform_type_identifier"), xknownvalue.NotEmptyString()),
				},
			},
		},
	})
}
