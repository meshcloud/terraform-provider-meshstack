package provider

import (
	"testing"

	tfconfig "github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

const buildingBlocksDataSourceAddr = "data.meshstack_building_blocks.all"

// TestAccBuildingBlocksDataSource creates a real building block and lists it back through the data
// source. Like every other data source test in this package, it creates the resources under test
// rather than pre-populating a mock, so it runs identically in mock mode and as a true acceptance
// test (TF_ACC=1) against a local meshStack.
func TestAccBuildingBlocksDataSource(t *testing.T) {
	t.Parallel()

	t.Run("01_workspace", func(t *testing.T) {
		config := examples.JoinTestStepConfigs(
			examples.DataSource.TestStepConfig(t, "building_blocks", 1),
			buildingBlockWorkspaceStepConfig(t, 1),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          config,
					ConfigVariables: SuffixVariables(acctest.RandString(8)),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlocksDataSourceAddr, tfjsonpath.New("building_blocks"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(buildingBlocksDataSourceAddr, tfjsonpath.New("building_blocks").AtSliceIndex(0).AtMapKey("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(buildingBlocksDataSourceAddr, tfjsonpath.New("building_blocks").AtSliceIndex(0).AtMapKey("spec").AtMapKey("display_name"), knownvalue.StringExact("my-workspace-building-block")),
						// all_inputs surfaces every backend input read-only (the 01_workspace BBD declares size + environment).
						statecheck.ExpectKnownValue(buildingBlocksDataSourceAddr, tfjsonpath.New("building_blocks").AtSliceIndex(0).AtMapKey("all_inputs").AtMapKey("size").AtMapKey("value"), knownvalue.StringExact("16")),
						statecheck.ExpectKnownValue(buildingBlocksDataSourceAddr, tfjsonpath.New("building_blocks").AtSliceIndex(0).AtMapKey("all_inputs").AtMapKey("environment").AtMapKey("value"), knownvalue.StringExact(`"dev"`)),
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

		suffix := acctest.RandString(8)
		config := examples.JoinTestStepConfigs(
			examples.DataSource.TestStepConfig(t, "building_blocks", 2),
			buildingBlockWorkspaceStepConfig(t, 1),
		)

		withVersion := func(versionNumber string) tfconfig.Variables {
			vars := SuffixVariables(suffix)
			vars["version_number"] = tfconfig.StringVariable(versionNumber)
			return vars
		}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					// Lenient "v1" matches definition version 1.
					Config:          config,
					ConfigVariables: withVersion("v1"),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlocksDataSourceAddr, tfjsonpath.New("building_blocks"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(buildingBlocksDataSourceAddr, tfjsonpath.New("building_blocks").AtSliceIndex(0).AtMapKey("spec").AtMapKey("display_name"), knownvalue.StringExact("my-workspace-building-block")),
					},
				},
				{
					// Version 2 does not exist for this block → empty result (proves the param is applied).
					Config:          config,
					ConfigVariables: withVersion("v2"),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlocksDataSourceAddr, tfjsonpath.New("building_blocks"), knownvalue.ListSizeExact(0)),
					},
				},
			},
		})
	})
}
