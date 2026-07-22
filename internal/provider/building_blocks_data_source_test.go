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

// TestAccBuildingBlocksDataSource creates a real building block with the BBWorkspace builder and
// lists it back through the data source. Like every other data source test in this package, it
// reuses a testconfig builder to actually create the resources under test (rather than
// pre-populating a mock), so it runs identically in mock mode and as a true acceptance test
// (TF_ACC=1) against a local meshStack. Filtering by the freshly-created workspace yields exactly
// the one block, so indexing building_blocks.0 is deterministic; referencing buildingBlockAddr in
// the filter makes Terraform read the data source only after the block exists.
func TestAccBuildingBlocksDataSource(t *testing.T) {
	t.Parallel()

	t.Run("01_workspace", func(t *testing.T) {
		buildingBlockConfig, buildingBlockAddr, _, _ := testconfig.BBWorkspace(t)

		dataSourceAddr := "data.meshstack_building_blocks.all"
		config := testconfig.DataSource{Name: "building_blocks"}.Config(t).WithFirstBlock(
			testconfig.Descend("workspace_identifier")(testconfig.SetAddr(buildingBlockAddr, "metadata", "owned_by_workspace")),
		).Join(buildingBlockConfig)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks").AtSliceIndex(0).AtMapKey("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks").AtSliceIndex(0).AtMapKey("spec").AtMapKey("display_name"), knownvalue.StringExact("my-workspace-building-block")),
						// all_inputs surfaces every backend input read-only (the _01_workspace BBD declares size + environment).
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks").AtSliceIndex(0).AtMapKey("all_inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks").AtSliceIndex(0).AtMapKey("all_inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact(`"dev"`)),
					},
				},
			},
		})
	})

	// 02_version_number_filter exercises the server-side version_number filter (the meshfed change).
	// The block is created from version 1 of its definition, so filtering by the lenient "v1" must
	// return it while "v2" must return nothing. This is acceptance-only: the filter is applied by the
	// backend (the mock store carries only the definition *version uuid*, not the version number), and
	// the "v2 → empty" assertion specifically proves the backend parses + applies the new param — an
	// older backend without it would ignore versionNumber and return the block for both values.
	t.Run("02_version_number_filter", func(t *testing.T) {
		if IsMockClientTest() {
			t.Skip("version_number is filtered server-side; the mock store does not carry the BBD version number")
		}
		buildingBlockConfig, buildingBlockAddr, _, _ := testconfig.BBWorkspace(t)
		dataSourceAddr := "data.meshstack_building_blocks.all"

		base := func(versionNumber string) testconfig.Config {
			return testconfig.DataSource{Name: "building_blocks"}.Config(t).WithFirstBlock(
				testconfig.Descend("workspace_identifier")(testconfig.SetAddr(buildingBlockAddr, "metadata", "owned_by_workspace")),
				testconfig.Descend("version_number")(testconfig.SetRawExpr(`%q`, versionNumber)),
			).Join(buildingBlockConfig)
		}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					// Lenient "v1" matches definition version 1.
					Config: base("v1").String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks").AtSliceIndex(0).AtMapKey("spec").AtMapKey("display_name"), knownvalue.StringExact("my-workspace-building-block")),
					},
				},
				{
					// Version 2 does not exist for this block → empty result (proves the param is applied).
					Config: base("v2").String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("building_blocks"), knownvalue.ListSizeExact(0)),
					},
				},
			},
		})
	})
}
