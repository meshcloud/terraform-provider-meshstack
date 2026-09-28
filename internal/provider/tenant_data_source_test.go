package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

const tenantByNameDataSourceAddr = "data.meshstack_tenant.name"

func TestAccTenantDataSource(t *testing.T) {
	// The singular meshstack_tenant data source resolves the tenant via the list endpoint by the real
	// platform identifier, which the mock (storing platform by ref uuid) cannot reproduce.
	if IsMockClientTest() {
		t.Skip("meshstack_tenant data source composite lookup requires a real meshStack")
	}

	config := examples.JoinTestStepConfigs(
		examples.DataSource.TestStepConfig(t, "tenant", 1),
		tenantStepConfig(t, 1, 8, 1),
	)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: SuffixVariables(acctest.RandString(8)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(tenantByNameDataSourceAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(tenantByNameDataSourceAddr, tfjsonpath.New("spec").AtMapKey("platform_ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(tenantByNameDataSourceAddr, tfjsonpath.New("status").AtMapKey("tenant_name"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(tenantByNameDataSourceAddr, tfjsonpath.New("status").AtMapKey("platform_type_identifier"), xknownvalue.NotEmptyString()),
				},
			},
		},
	})
}
