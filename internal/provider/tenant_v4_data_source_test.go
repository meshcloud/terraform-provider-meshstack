package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccTenantV4DataSource(t *testing.T) {
	tenantConfig, tenantAddr := testconfig.TenantV4AndWorkspace(t)

	config := testconfig.DataSource{Name: "tenant_v4"}.Config(t).WithFirstBlock(
		testconfig.Descend("metadata", "uuid")(testconfig.SetAddr(tenantAddr, "metadata", "uuid"))).
		Join(tenantConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.meshstack_tenant_v4.example", tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
				},
			},
		},
	})
}
