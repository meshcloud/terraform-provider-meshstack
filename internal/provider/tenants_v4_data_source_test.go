package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccTenantsV4DataSource(t *testing.T) {
	tenantConfig, tenantAddr := testconfig.TenantV4AndWorkspace(t)

	config := testconfig.DataSource{Name: "tenants"}.Config(t).WithFirstBlock(
		testconfig.Descend("workspace")(testconfig.SetAddr(tenantAddr, "metadata", "owned_by_workspace")),
		testconfig.Descend("project")(testconfig.SetAddr(tenantAddr, "metadata", "owned_by_project"))).
		Join(tenantConfig)

	addr := testconfig.Traversal{"data.meshstack_tenants", "example"}

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("tenants"),
						knownvalue.SetPartial([]knownvalue.Check{
							knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"metadata": knownvalue.ObjectPartial(map[string]knownvalue.Check{
									"owned_by_workspace": xknownvalue.NotEmptyString(),
									"owned_by_project":   xknownvalue.NotEmptyString(),
								}),
							}),
						}),
					),
				},
			},
		},
	})
}
