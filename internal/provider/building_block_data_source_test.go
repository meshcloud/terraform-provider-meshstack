package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// Addresses of the blocks in examples/{resources,data-sources}/meshstack_building_block/*-test-*.tf.
const (
	buildingBlockWorkspaceAddr  = "meshstack_building_block.example_workspace"
	buildingBlockParentAddr     = "meshstack_building_block.parent"
	buildingBlockChildAddr      = "meshstack_building_block.child"
	buildingBlockTenantAddr     = "meshstack_building_block.example_tenant"
	buildingBlockDataSourceAddr = "data.meshstack_building_block.example"
)

// buildingBlockWorkspaceStepConfig joins the named building block step files with the definition
// they instantiate and the workspace they target.
func buildingBlockWorkspaceStepConfig(t *testing.T, indexes ...int) string {
	t.Helper()
	parts := make([]string, 0, len(indexes)+2)
	for _, index := range indexes {
		parts = append(parts, examples.Resource.TestStepConfig(t, "building_block", index))
	}
	parts = append(parts,
		examples.Resource.TestSupportConfigs(t, "building_block", "lifecycle-variables", "01_workspace"),
		examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
	)
	return examples.JoinTestStepConfigs(parts...)
}

func TestAccBuildingBlockDataSource(t *testing.T) {
	t.Parallel()

	t.Run("01_workspace", func(t *testing.T) {
		suffix := acctest.RandString(8)

		config := examples.JoinTestStepConfigs(
			examples.DataSource.TestStepConfig(t, "building_block", 1),
			buildingBlockWorkspaceStepConfig(t, 1),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          config,
					ConfigVariables: NewVariablesWithSuffix(suffix),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockDataSourceAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(buildingBlockDataSourceAddr, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), knownvalue.StringExact("test-ws-"+suffix)),
						statecheck.ExpectKnownValue(buildingBlockDataSourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("my-workspace-building-block")),
						statecheck.ExpectKnownValue(buildingBlockDataSourceAddr, tfjsonpath.New("status").AtMapKey("status"), knownvalue.StringExact("SUCCEEDED")),
						statecheck.ExpectKnownValue(buildingBlockDataSourceAddr, tfjsonpath.New("all_inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
						statecheck.ExpectKnownValue(buildingBlockDataSourceAddr, tfjsonpath.New("all_inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact(`"dev"`)),
						xknownvalue.Ref(buildingBlockDataSourceAddr, client.MeshObjectKind.BuildingBlock, nil),
						statecheck.CompareValuePairs(
							buildingBlockWorkspaceAddr, tfjsonpath.New("ref"),
							buildingBlockDataSourceAddr, tfjsonpath.New("ref"),
							compare.ValuesSame(),
						),
					},
				},
			},
		})
	})

	t.Run("02_parent_child", func(t *testing.T) {
		// Both blocks come from the same definition, so each one is its own step file.
		config := examples.JoinTestStepConfigs(
			examples.DataSource.TestStepConfig(t, "building_block", 2),
			buildingBlockWorkspaceStepConfig(t, 2, 3),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          config,
					ConfigVariables: NewVariablesWithSuffix(acctest.RandString(8)),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockDataSourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("my-child-building-block")),
						statecheck.ExpectKnownValue(buildingBlockDataSourceAddr, tfjsonpath.New("spec").AtMapKey("parent_building_block_refs"), knownvalue.SetSizeExact(1)),
						statecheck.CompareValuePairs(
							buildingBlockParentAddr, tfjsonpath.New("ref"),
							buildingBlockDataSourceAddr, tfjsonpath.New("spec").AtMapKey("parent_building_block_refs").AtSliceIndex(0),
							compare.ValuesSame(),
						),
					},
				},
			},
		})
	})
}
