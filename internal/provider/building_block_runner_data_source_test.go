package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccBuildingBlockRunnerDataSource(t *testing.T) {
	t.Parallel()

	t.Run("hosted", func(t *testing.T) {
		dataSourceAddr := "data.meshstack_building_block_runner.hosted"

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/building_block_runner/hosted", 1, hclStepVars{}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("ref").AtMapKey("uuid"), knownvalue.StringExact(SharedBuildingBlockRunnerUuid)),
					statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("spec").AtMapKey("restriction"), knownvalue.StringExact("PUBLIC")),
				},
			},
		}})
	})

	t.Run("by_uuid", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		runnerAddr := "meshstack_building_block_runner.example_with_wif"
		dataSourceAddr := "data.meshstack_building_block_runner.hosted"

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/building_block_runner/by_uuid", 1, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.CompareValuePairs(dataSourceAddr, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), runnerAddr, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), compare.ValuesSame()),
					statecheck.CompareValuePairs(dataSourceAddr, tfjsonpath.New("spec"), runnerAddr, tfjsonpath.New("spec"), compare.ValuesSame()),
					statecheck.CompareValuePairs(dataSourceAddr, tfjsonpath.New("ref"), runnerAddr, tfjsonpath.New("ref"), compare.ValuesSame()),
				},
			},
		}})
	})

	t.Run("by_display_name", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		runnerAddr := "meshstack_building_block_runner.example"
		dataSourceAddr := "data.meshstack_building_block_runner.example"

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/building_block_runner/by_display_name", 1, vars),
			},
			{
				Config: hclStep(t, "data-sources/building_block_runner/by_display_name", 2, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.CompareValuePairs(dataSourceAddr, tfjsonpath.New("ref"), runnerAddr, tfjsonpath.New("ref"), compare.ValuesSame()),
				},
			},
			{
				Config:      hclStep(t, "data-sources/building_block_runner/by_display_name", 3, vars),
				ExpectError: expectErrorWords("runner", "matches", "spec.display_name", "Such"),
			},
		}})
	})

	t.Run("ambiguous", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config:      hclStep(t, "data-sources/building_block_runner/ambiguous", 1, vars),
				ExpectError: expectErrorWords("ambiguous", "2 building block runners match", "metadata.uuid"),
			},
		}})
	})

	t.Run("cross-workspace read", func(t *testing.T) {
		if IsMockClientTest() {
			t.Skip("cross-workspace test requires real permission boundaries")
		}

		vars := hclStepVars{"suffix": acctest.RandString(8)}
		captureReaderKey, readerKey := otherWorkspaceReader()
		hostedAddr := "data.meshstack_building_block_runner.hosted"

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config:            hclStep(t, "data-sources/building_block_runner/cross_workspace_read", 1, vars),
				ConfigStateChecks: captureReaderKey,
			},
			{
				Config:          hclStep(t, "data-sources/building_block_runner/cross_workspace_read", 2, vars),
				ConfigVariables: readerKey,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(hostedAddr, tfjsonpath.New("ref").AtMapKey("uuid"), knownvalue.StringExact(SharedBuildingBlockRunnerUuid)),
				},
			},
			{
				Config:          hclStep(t, "data-sources/building_block_runner/cross_workspace_read", 3, vars),
				ConfigVariables: readerKey,
				ExpectError:     expectErrorWords("runner", "not found", "metadata.uuid"),
			},
			{
				Config:          hclStep(t, "data-sources/building_block_runner/cross_workspace_read", 4, vars),
				ConfigVariables: readerKey,
				ExpectError:     expectErrorWords("runner", "not found", "spec.display_name"),
			},
		}})
	})
}
