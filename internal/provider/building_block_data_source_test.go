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

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccBuildingBlockDataSource(t *testing.T) {
	t.Parallel()

	dataSourceAddr := "data.meshstack_building_block.example"

	t.Run("01_workspace", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		buildingBlockAddr := "meshstack_building_block.example_workspace"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "data-sources/building_block/01_workspace", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), knownvalue.StringExact("test-ws-"+suffix)),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("my-workspace-building-block")),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("status").AtMapKey("health_status"), knownvalue.StringExact("SUCCEEDED")),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("status").AtMapKey("action"), knownvalue.StringExact("NO_ACTION")),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("status").AtMapKey("succeeded"), knownvalue.Bool(true)),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("all_inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("all_inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact(`"dev"`)),
						xknownvalue.Ref(dataSourceAddr, client.MeshObjectKind.BuildingBlock, nil),
						statecheck.CompareValuePairs(
							buildingBlockAddr, tfjsonpath.New("ref"),
							dataSourceAddr, tfjsonpath.New("ref"),
							compare.ValuesSame(),
						),
					},
				},
			},
		})
	})

	t.Run("02_parent_child", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		parentAddr := "meshstack_building_block.parent"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "data-sources/building_block/02_parent_child", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("my-child-building-block")),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("spec").AtMapKey("parent_building_block_refs"), knownvalue.SetSizeExact(1)),
						statecheck.CompareValuePairs(
							parentAddr, tfjsonpath.New("ref"),
							dataSourceAddr, tfjsonpath.New("spec").AtMapKey("parent_building_block_refs").AtSliceIndex(0),
							compare.ValuesSame(),
						),
					},
				},
			},
		})
	})
}
