package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	testconfig "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
)

func TestAccBuildingBlockRunnerDataSource(t *testing.T) {
	t.Parallel()

	t.Run("by_uuid", func(t *testing.T) {
		runnerConfig, runnerAddr, _ := testconfig.BuildingBlockRunnerWifAndWorkspace(t)
		runnerConfig = runnerConfig.WithFirstBlock(testconfig.Descend("spec", "public_key")(testconfig.SetString(runnerPublicKey)))

		var dataSourceAddr testconfig.Traversal
		config := testconfig.DataSource{Name: "building_block_runner"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&dataSourceAddr),
			testconfig.Descend("spec")(testconfig.RemoveKey()),
			testconfig.Descend("metadata")(testconfig.SetRawExpr("{ uuid = %s }", runnerAddr.Join("metadata", "uuid"))),
		).Join(runnerConfig)

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.CompareValuePairs(dataSourceAddr.String(), tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), runnerAddr.String(), tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), compare.ValuesSame()),
					statecheck.CompareValuePairs(dataSourceAddr.String(), tfjsonpath.New("spec"), runnerAddr.String(), tfjsonpath.New("spec"), compare.ValuesSame()),
					statecheck.CompareValuePairs(dataSourceAddr.String(), tfjsonpath.New("ref"), runnerAddr.String(), tfjsonpath.New("ref"), compare.ValuesSame()),
				},
			},
		}})
	})

	t.Run("filters", func(t *testing.T) {
		runnerConfig, runnerAddr, workspaceAddr := testconfig.BuildingBlockRunnerWifAndWorkspace(t)
		runnerConfig = runnerConfig.WithFirstBlock(testconfig.Descend("spec", "public_key")(testconfig.SetString(runnerPublicKey)))
		otherRunnerConfig, _ := testconfig.BuildingBlockRunner(t, workspaceAddr)
		otherRunnerConfig = otherRunnerConfig.WithFirstBlock(
			testconfig.Descend("spec", "public_key")(testconfig.SetString(runnerPublicKey)),
			testconfig.Descend("spec", "display_name")(testconfig.SetString("Other Runner")),
		)
		supportConfig := runnerConfig.Join(otherRunnerConfig)

		dataSourceAddr := testconfig.Traversal{"data.meshstack_building_block_runner", "example"}
		example := testconfig.DataSource{Name: "building_block_runner"}.Config(t)
		// Parallel tests create runners of the same display name, so the workspace keeps the lookup unique.
		byDisplayName := func(displayName string) testconfig.Config {
			return example.WithFirstBlock(
				testconfig.Descend("spec")(testconfig.SetRawExpr("{ display_name = %s }", displayName)),
				testconfig.Descend("metadata")(testconfig.SetRawExpr("{ owned_by_workspace = %s }", runnerAddr.Join("metadata", "owned_by_workspace"))),
			).Join(supportConfig)
		}
		noFilter := example.WithFirstBlock(testconfig.Descend("spec")(testconfig.RemoveKey())).Join(supportConfig)

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: noFilter.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("ref").AtMapKey("uuid"), knownvalue.StringExact(SharedBuildingBlockRunnerUuid)),
					statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("spec").AtMapKey("restriction"), knownvalue.StringExact("PUBLIC")),
				},
			},
			{
				Config: byDisplayName(runnerAddr.Join("spec", "display_name").String()).String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.CompareValuePairs(dataSourceAddr.String(), tfjsonpath.New("ref"), runnerAddr.String(), tfjsonpath.New("ref"), compare.ValuesSame()),
				},
			},
			{
				Config:      byDisplayName(`"No Such Runner"`).String(),
				ExpectError: expectErrorWords("runner", "matches", "spec.display_name", "Such"),
			},
		}})
	})

	t.Run("ambiguous", func(t *testing.T) {
		runnerConfig, runnerAddr, workspaceAddr := testconfig.BuildingBlockRunnerWifAndWorkspace(t)
		runnerConfig = runnerConfig.WithFirstBlock(testconfig.Descend("spec", "public_key")(testconfig.SetString(runnerPublicKey)))
		twinConfig, _ := testconfig.BuildingBlockRunner(t, workspaceAddr)
		twinConfig = twinConfig.WithFirstBlock(
			testconfig.Descend("spec", "public_key")(testconfig.SetString(runnerPublicKey)),
			testconfig.Descend("spec", "display_name")(testconfig.SetRawExpr("%s", runnerAddr.Join("spec", "display_name"))),
		)

		config := testconfig.DataSource{Name: "building_block_runner"}.Config(t).WithFirstBlock(
			testconfig.Descend("spec")(testconfig.SetRawExpr("{ display_name = %s }", runnerAddr.Join("spec", "display_name"))),
			testconfig.Descend("metadata")(testconfig.SetRawExpr("{ owned_by_workspace = %s }", runnerAddr.Join("metadata", "owned_by_workspace"))),
		).Join(runnerConfig, twinConfig)

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config:      config.String(),
				ExpectError: expectErrorWords("ambiguous", "2 building block runners match", "metadata.uuid"),
			},
		}})
	})

	t.Run("cross-workspace read", func(t *testing.T) {
		if IsMockClientTest() {
			t.Skip("cross-workspace test requires real permission boundaries")
		}

		runnerConfig, runnerAddr, _ := testconfig.BuildingBlockRunnerWifAndWorkspace(t)
		runnerConfig = runnerConfig.WithFirstBlock(testconfig.Descend("spec", "public_key")(testconfig.SetString(runnerPublicKey)))
		readerConfig, captureReaderKey, readerKey := otherWorkspaceReader(t, "BUILDINGBLOCKRUNNER_LIST")
		supportConfig := runnerConfig.Join(readerConfig)

		dataSourceAddr := testconfig.Traversal{"data.meshstack_building_block_runner", "example"}
		asReader := func(modifiers ...testconfig.ExpressionConsumer) testconfig.Config {
			return testconfig.DataSource{Name: "building_block_runner"}.Config(t).WithFirstBlock(
				append(modifiers, testconfig.Descend("provider")(testconfig.SetRawExpr("meshstack-other")))...,
			).Join(supportConfig, testconfig.OtherProviderConfig(t))
		}
		hosted := asReader(testconfig.Descend("spec")(testconfig.RemoveKey()))
		privateByUuid := asReader(
			testconfig.Descend("spec")(testconfig.RemoveKey()),
			testconfig.Descend("metadata")(testconfig.SetRawExpr("{ uuid = %s }", runnerAddr.Join("metadata", "uuid"))),
		)
		privateByName := asReader(
			testconfig.Descend("spec")(testconfig.SetRawExpr("{ display_name = %s }", runnerAddr.Join("spec", "display_name"))),
			testconfig.Descend("metadata")(testconfig.SetRawExpr("{ owned_by_workspace = %s }", runnerAddr.Join("metadata", "owned_by_workspace"))),
		)

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config:            supportConfig.String(),
				ConfigStateChecks: captureReaderKey,
			},
			{
				Config:          hosted.String(),
				ConfigVariables: readerKey,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("ref").AtMapKey("uuid"), knownvalue.StringExact(SharedBuildingBlockRunnerUuid)),
				},
			},
			{
				Config:          privateByUuid.String(),
				ConfigVariables: readerKey,
				ExpectError:     expectErrorWords("runner", "not found", "metadata.uuid"),
			},
			{
				Config:          privateByName.String(),
				ConfigVariables: readerKey,
				ExpectError:     expectErrorWords("runner", "not found", "spec.display_name"),
			},
		}})
	})
}
