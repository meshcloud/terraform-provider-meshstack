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

	byName := testconfig.DataSource{Name: "building_block_runner"}
	hosted := testconfig.DataSource{Name: "building_block_runner", Suffix: "_hosted"}
	withPublicKey := testconfig.Descend("spec", "public_key")(testconfig.SetString(runnerPublicKey))
	inWorkspaceOf := func(runnerAddr testconfig.Traversal) testconfig.ExpressionConsumer {
		return testconfig.Descend("metadata")(testconfig.SetRawExpr("{ owned_by_workspace = %s }", runnerAddr.Join("metadata", "owned_by_workspace")))
	}
	withUuidOf := func(runnerAddr testconfig.Traversal) testconfig.ExpressionConsumer {
		return testconfig.Descend("metadata")(testconfig.SetRawExpr("{ uuid = %s }", runnerAddr.Join("metadata", "uuid")))
	}

	t.Run("hosted", func(t *testing.T) {
		var dataSourceAddr testconfig.Traversal
		config := hosted.Config(t).WithFirstBlock(testconfig.ExtractAddress(&dataSourceAddr))

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("ref").AtMapKey("uuid"), knownvalue.StringExact(SharedBuildingBlockRunnerUuid)),
					statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("spec").AtMapKey("restriction"), knownvalue.StringExact("PUBLIC")),
				},
			},
		}})
	})

	t.Run("by_uuid", func(t *testing.T) {
		runnerConfig, runnerAddr, _ := testconfig.BuildingBlockRunnerWifAndWorkspace(t)
		runnerConfig = runnerConfig.WithFirstBlock(withPublicKey)

		var dataSourceAddr testconfig.Traversal
		config := hosted.Config(t).WithFirstBlock(testconfig.ExtractAddress(&dataSourceAddr), withUuidOf(runnerAddr)).Join(runnerConfig)

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

	t.Run("by_display_name", func(t *testing.T) {
		runnerConfig, runnerAddr, workspaceAddr := testconfig.BuildingBlockRunnerAndWorkspace(t)
		runnerConfig = runnerConfig.WithFirstBlock(withPublicKey)
		wifRunnerConfig, _ := testconfig.BuildingBlockRunnerWif(t, workspaceAddr)
		supportConfig := runnerConfig.Join(wifRunnerConfig.WithFirstBlock(withPublicKey))

		var dataSourceAddr testconfig.Traversal
		// Parallel tests create runners of the same display name, so the workspace keeps the lookup unique.
		config := byName.Config(t).WithFirstBlock(testconfig.ExtractAddress(&dataSourceAddr), inWorkspaceOf(runnerAddr)).Join(supportConfig)
		unknownName := byName.Config(t).WithFirstBlock(
			inWorkspaceOf(runnerAddr),
			testconfig.Descend("spec", "display_name")(testconfig.SetString("No Such Runner")),
		).Join(supportConfig)

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.CompareValuePairs(dataSourceAddr.String(), tfjsonpath.New("ref"), runnerAddr.String(), tfjsonpath.New("ref"), compare.ValuesSame()),
				},
			},
			{
				Config:      unknownName.String(),
				ExpectError: expectErrorWords("runner", "matches", "spec.display_name", "Such"),
			},
		}})
	})

	t.Run("ambiguous", func(t *testing.T) {
		runnerConfig, runnerAddr, workspaceAddr := testconfig.BuildingBlockRunnerAndWorkspace(t)
		twinConfig, _ := testconfig.BuildingBlockRunner(t, workspaceAddr)
		supportConfig := runnerConfig.WithFirstBlock(withPublicKey).Join(twinConfig.WithFirstBlock(testconfig.RenameKey("twin"), withPublicKey))

		config := byName.Config(t).WithFirstBlock(inWorkspaceOf(runnerAddr)).Join(supportConfig)

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

		runnerConfig, runnerAddr, _ := testconfig.BuildingBlockRunnerAndWorkspace(t)
		readerConfig, captureReaderKey, readerKey := otherWorkspaceReader(t, "BUILDINGBLOCKRUNNER_LIST")
		supportConfig := runnerConfig.WithFirstBlock(withPublicKey).Join(readerConfig)

		asReader := func(example testconfig.DataSource, modifiers ...testconfig.ExpressionConsumer) testconfig.Config {
			return example.Config(t).WithFirstBlock(
				append(modifiers, testconfig.Descend("provider")(testconfig.SetRawExpr("meshstack-other")))...,
			).Join(supportConfig, testconfig.OtherProviderConfig(t))
		}
		var hostedAddr testconfig.Traversal
		hostedAsReader := asReader(hosted, testconfig.ExtractAddress(&hostedAddr))
		privateByUuid := asReader(hosted, withUuidOf(runnerAddr))
		privateByName := asReader(byName, inWorkspaceOf(runnerAddr))

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config:            supportConfig.String(),
				ConfigStateChecks: captureReaderKey,
			},
			{
				Config:          hostedAsReader.String(),
				ConfigVariables: readerKey,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(hostedAddr.String(), tfjsonpath.New("ref").AtMapKey("uuid"), knownvalue.StringExact(SharedBuildingBlockRunnerUuid)),
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
