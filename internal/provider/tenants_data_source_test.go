package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

const tenantsDataSourceAddr = "data.meshstack_tenants.example"

func TestAccTenantsDataSource(t *testing.T) {
	config := examples.JoinTestStepConfigs(
		examples.DataSource.TestStepConfig(t, "tenants", 1),
		tenantStepConfig(t, 1, 8, 1),
	)

	suffix := acctest.RandString(8)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: SuffixVariables(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(tenantsDataSourceAddr, tfjsonpath.New("tenants"),
						knownvalue.SetPartial([]knownvalue.Check{
							knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"metadata": knownvalue.ObjectPartial(map[string]knownvalue.Check{
									"owned_by_workspace": knownvalue.StringExact("test-ws-" + suffix),
									"owned_by_project":   knownvalue.StringExact("test-proj-" + suffix),
								}),
							}),
						}),
					),
				},
			},
		},
	})
}
