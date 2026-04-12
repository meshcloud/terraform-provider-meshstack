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
	// The v3 tenant DS reads from a different mock store than the v4 tenant resource writes to,
	// so this test only works against a real meshStack (acceptance mode).
	if IsMockClientTest() {
		t.Skip("v3 tenant DS mock incompatible with v4 tenant resource mock")
	}

	tenantConfig, tenantAddr := testconfig.BuildTenantConfig(t)

	dsAddress := testconfig.Traversal{"data.meshstack_tenant", "name"}
	dsConfig := testconfig.DataSource{Name: "tenant"}.Config(t)
	dsConfig = dsConfig.WithFirstBlock(t,
		testconfig.Traverse(t, "metadata")(
			testconfig.Traverse(t, "owned_by_workspace")(testconfig.SetRawExpr(tenantAddr.Format("%s.metadata.owned_by_workspace"))),
			testconfig.Traverse(t, "owned_by_project")(testconfig.SetRawExpr(tenantAddr.Format("%s.metadata.owned_by_project"))),
			testconfig.Traverse(t, "platform_identifier")(testconfig.SetRawExpr(tenantAddr.Format("%s.spec.platform_identifier"))),
		),
	)
	config := dsConfig.Join(tenantConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dsAddress.String(), tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dsAddress.String(), tfjsonpath.New("metadata").AtMapKey("owned_by_project"), xknownvalue.NotEmptyString()),
				},
			},
		},
	})
}
