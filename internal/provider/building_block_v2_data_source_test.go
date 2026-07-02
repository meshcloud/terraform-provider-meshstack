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

func TestAccBuildingBlockV2DataSource(t *testing.T) {
	t.Parallel()

	t.Run("01_workspace", func(t *testing.T) {
		buildingBlockConfig, buildingBlockAddr := testconfig.BBv2Workspace(t)

		config := testconfig.DataSource{Name: "building_block_v2"}.Config(t).WithFirstBlock(
			testconfig.Descend("metadata", "uuid")(testconfig.SetAddr(buildingBlockAddr, "metadata", "uuid"))).
			Join(buildingBlockConfig)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("data.meshstack_building_block_v2.example", tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue("data.meshstack_building_block_v2.example", tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("my-workspace-building-block")),
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
		workspaceConfig, workspaceAddr := testconfig.Workspace(t)
		exampleResource := testconfig.Resource{Name: "building_block_v2", Suffix: "_04_sensitive_user_input"}

		var buildingBlockDefinitionAddr testconfig.Traversal
		buildingBlockDefinitionConfig := exampleResource.TestSupportConfig(t, "_bbd").WithFirstBlock(
			testconfig.ExtractAddress(&buildingBlockDefinitionAddr),
			testconfig.OwnedByWorkspace(workspaceAddr),
			// Point at the committed bare repo served over loopback so the run completes and the block
			// reaches a final state under the default wait_for_completion (unused in mock mode).
			testconfig.Descend("version_spec", "implementation", "terraform", "repository_url")(
				testconfig.SetRawExpr("%q", terraformTestdataRepoURL(t)),
			),
		)

		var buildingBlockAddr testconfig.Traversal
		buildingBlockConfig := exampleResource.TestSupportConfig(t, "").WithFirstBlock(
			testconfig.ExtractAddress(&buildingBlockAddr),
			testconfig.Descend("spec", "building_block_definition_version_ref")(testconfig.SetAddr(buildingBlockDefinitionAddr, "version_latest")),
			testconfig.Descend("spec", "target_ref")(testconfig.SetAddr(workspaceAddr, "ref")),
		).Join(workspaceConfig, buildingBlockDefinitionConfig)

		dataSourceConfig := testconfig.DataSource{Name: "building_block_v2"}.Config(t).WithFirstBlock(
			testconfig.Descend("metadata", "uuid")(testconfig.SetAddr(buildingBlockAddr, "metadata", "uuid")),
		).Join(buildingBlockConfig)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: dataSourceConfig.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("data.meshstack_building_block_v2.example", tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						// Sensitive user inputs are hashed by the API; toResourceModelV2Input
						// ensures the hash surfaces in spec.inputs instead of being nil.
						statecheck.ExpectKnownValue("data.meshstack_building_block_v2.example",
							tfjsonpath.New("spec").AtMapKey("inputs").AtMapKey("secret_str").AtMapKey("value_string"),
							xknownvalue.NotEmptyString()),
					},
				},
			},
		})
	})
}
