package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

const tenantsDataSourceAddr = "data.meshstack_tenants.example"

func TestAccTenantsDataSource(t *testing.T) {
	config := examples.JoinTestStepConfigs(
		examples.DataSource.TestStepConfig(t, "tenants", 1),
		tenantStepConfig(t, 1, 8, 1),
	)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: SuffixVariables(acctest.RandString(8)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(tenantsDataSourceAddr, tfjsonpath.New("tenants"),
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
