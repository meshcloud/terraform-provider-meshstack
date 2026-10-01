package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// Addresses of the blocks in examples/{resources,data-sources}/meshstack_buildingblock/*-test-*.tf.
const (
	buildingBlockV1ResourceAddr   = "meshstack_buildingblock.my_buildingblock"
	buildingBlockV1DataSourceAddr = "data.meshstack_buildingblock.example"
)

// buildingBlockV1StepConfig is the v1 building block's step with its full dependency chain: the
// tenant it targets (and everything under that) plus the definition it instantiates.
func buildingBlockV1StepConfig(t *testing.T) string {
	t.Helper()
	return examples.JoinTestStepConfigs(
		examples.Resource.TestStepConfig(t, "buildingblock", 1),
		examples.Resource.TestSupportConfigs(t, "building_block_v2", "02_tenant"),
		tenantStepConfig(t, 1, 8, 1),
	)
}

func TestAccBuildingblock(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("Skipping: BB v1 resource has no wait_for_completion, BB run stays PENDING and blocks destroy")
	}

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          buildingBlockV1StepConfig(t),
				ConfigVariables: SuffixVariables(acctest.RandString(8)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(buildingBlockV1ResourceAddr, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(buildingBlockV1ResourceAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(buildingBlockV1ResourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("my-buildingblock")),
					statecheck.ExpectKnownValue(buildingBlockV1ResourceAddr, tfjsonpath.New("status").AtMapKey("status"), knownvalue.StringExact("SUCCEEDED")),
				},
			},
		},
	})
}
