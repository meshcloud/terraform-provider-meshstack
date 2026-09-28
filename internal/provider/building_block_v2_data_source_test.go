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

const buildingBlockV2DataSourceAddr = "data.meshstack_building_block_v2.example"

func TestAccBuildingBlockV2DataSource(t *testing.T) {
	t.Parallel()

	t.Run("01_workspace", func(t *testing.T) {
		config := examples.JoinTestStepConfigs(
			examples.DataSource.TestStepConfig(t, "building_block_v2", 1),
			examples.Resource.TestStepConfig(t, "building_block_v2", 1, "01_workspace"),
			examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          config,
					ConfigVariables: SuffixVariables(acctest.RandString(8)),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockV2DataSourceAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(buildingBlockV2DataSourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("my-workspace-building-block")),
					},
				},
			},
		})
	})

	t.Run("02_sensitive_input", func(t *testing.T) {
		// The mock client hashes sensitive plaintext on Create, so this test works in both mock
		// and acceptance modes. It verifies that toResourceModelV2Input is used so the data source
		// surfaces the hash in spec.inputs.<key>.value_string instead of returning nil for
		// sensitive inputs.
		config := examples.JoinTestStepConfigs(
			examples.DataSource.TestStepConfig(t, "building_block_v2", 2),
			examples.Resource.TestStepConfig(t, "building_block_v2", 4, "04_sensitive_user_input_bbd", "variables"),
			examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          config,
					ConfigVariables: bbv2Variables(t),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockV2DataSourceAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						// Sensitive user inputs are hashed by the API; toResourceModelV2Input
						// ensures the hash surfaces in spec.inputs instead of being nil.
						statecheck.ExpectKnownValue(buildingBlockV2DataSourceAddr,
							tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("secret_str").AtMapKey("value_string"),
							xknownvalue.NotEmptyString()),
					},
				},
			},
		})
	})
}
