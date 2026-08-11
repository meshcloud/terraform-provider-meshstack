package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccBuildingBlockDataSource(t *testing.T) {
	t.Parallel()

	dataSourceAddr := testconfig.Traversal{"data.meshstack_building_block", "example"}

	t.Run("01_workspace", func(t *testing.T) {
		buildingBlockConfig, buildingBlockAddr, _, _ := testconfig.BBWorkspace(t)

		config := testconfig.DataSource{Name: "building_block"}.Config(t).WithFirstBlock(
			testconfig.Descend("metadata", "uuid")(testconfig.SetAddr(buildingBlockAddr, "metadata", "uuid")),
		).Join(buildingBlockConfig)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("my-workspace-building-block")),
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("status").AtMapKey("status"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("all_inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("all_inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact(`"dev"`)),
						xknownvalue.Ref(dataSourceAddr, client.MeshObjectKind.BuildingBlock, nil),
						statecheck.CompareValuePairs(
							buildingBlockAddr.String(), tfjsonpath.New("ref"),
							dataSourceAddr.String(), tfjsonpath.New("ref"),
							compare.ValuesSame(),
						),
					},
				},
			},
		})
	})

	t.Run("02_parent_child", func(t *testing.T) {
		buildingBlockConfig, parentAddr, childAddr := testconfig.BBWorkspaceParentChild(t)

		config := testconfig.DataSource{Name: "building_block"}.Config(t).WithFirstBlock(
			testconfig.Descend("metadata", "uuid")(testconfig.SetAddr(childAddr, "metadata", "uuid")),
		).Join(buildingBlockConfig)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("my-child-building-block")),
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("spec").AtMapKey("parent_building_block_refs"), knownvalue.SetSizeExact(1)),
						statecheck.CompareValuePairs(
							parentAddr.String(), tfjsonpath.New("ref"),
							dataSourceAddr.String(), tfjsonpath.New("spec").AtMapKey("parent_building_block_refs").AtSliceIndex(0),
							compare.ValuesSame(),
						),
					},
				},
			},
		})
	})
}
