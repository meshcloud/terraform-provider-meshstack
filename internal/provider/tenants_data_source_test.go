package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccTenantsDataSource(t *testing.T) {
	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix}

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/tenants", 1, vars),
			},
			{
				Config: hclStep(t, "data-sources/tenants", 2, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.meshstack_tenants.example", tfjsonpath.New("tenants"),
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
