package provider

import (
	"context"
	"iter"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccBuildingBlockDefinitionDataSource(t *testing.T) {
	t.Run("simple state check", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		buildingBlockDefinitionAddr := "meshstack_building_block_definition.example_03_manual"
		dataSourceAddress := "data.meshstack_building_block_definition.example"

		versionRef := knownvalue.ObjectPartial(map[string]knownvalue.Check{
			"uuid":         xknownvalue.NotEmptyString(),
			"number":       knownvalue.Int64Exact(1),
			"state":        knownvalue.StringExact("DRAFT"),
			"content_hash": xknownvalue.NotEmptyString(),
		})
		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/building_block_definition/simple_state_check", 1, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("metadata"), knownvalue.ObjectExact(map[string]knownvalue.Check{
						"uuid":               xknownvalue.NotEmptyString(),
						"owned_by_workspace": knownvalue.StringExact("test-ws-" + suffix),
					})),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("spec"), knownvalue.ObjectExact(map[string]knownvalue.Check{
						"display_name": knownvalue.StringExact("Example Building Block"),
						"target_type":  knownvalue.StringExact("WORKSPACE_LEVEL"),
					})),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("version_latest"), versionRef),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("versions"), ascendingVersionsExact(versionRef)),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("version_latest_release"), knownvalue.Null()),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshBuildingBlockDefinition")),
					statecheck.CompareValuePairs(dataSourceAddress, tfjsonpath.New("ref"), buildingBlockDefinitionAddr, tfjsonpath.New("ref"), compare.ValuesSame()),
				},
			},
		}})
	})

	t.Run("filters", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		definitionAddr := "meshstack_building_block_definition.example_03_manual"
		dataSourceAddr := "data.meshstack_building_block_definition.example"

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/building_block_definition/filters", 1, vars),
			},
			{
				Config:      hclStep(t, "data-sources/building_block_definition/filters", 2, vars),
				ExpectError: expectErrorWords("least", "attributes", "configured", "metadata.uuid"),
			},
			{
				// Parallel tests create definitions of the same display name, so the workspace keeps the lookup unique.
				Config: hclStep(t, "data-sources/building_block_definition/filters", 3, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.CompareValuePairs(dataSourceAddr, tfjsonpath.New("ref"), definitionAddr, tfjsonpath.New("ref"), compare.ValuesSame()),
					statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("spec").AtMapKey("target_type"), knownvalue.StringExact("WORKSPACE_LEVEL")),
				},
			},
			{
				Config:      hclStep(t, "data-sources/building_block_definition/filters", 4, vars),
				ExpectError: expectErrorWords("definition", "matches", "spec.display_name", "Such"),
			},
			{
				Config:      hclStep(t, "data-sources/building_block_definition/filters", 5, vars),
				ExpectError: expectErrorWords("definition", "matches", "metadata.uuid"),
			},
		}})
	})

	t.Run("cross-workspace read", func(t *testing.T) {
		if IsMockClientTest() {
			t.Skip("cross-workspace test requires real permission boundaries")
		}

		vars := hclStepVars{"suffix": acctest.RandString(8)}
		buildingBlockDefinitionAddr := "meshstack_building_block_definition.example_03_manual"
		dataSourceAddress := "data.meshstack_building_block_definition.example"
		captureReaderKey, readerKey := otherWorkspaceReader()

		versionRef := knownvalue.ObjectPartial(map[string]knownvalue.Check{
			"uuid":         xknownvalue.NotEmptyString(),
			"state":        knownvalue.StringExact("RELEASED"),
			"content_hash": knownvalue.Null(),
		})
		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/building_block_definition/cross_workspace_read", 1, vars),
				ConfigStateChecks: append([]statecheck.StateCheck{
					statecheck.ExpectKnownValue(buildingBlockDefinitionAddr, tfjsonpath.New("version_latest_release").AtMapKey("state"), knownvalue.StringExact("RELEASED")),
				}, captureReaderKey...),
			},
			{
				Config:          hclStep(t, "data-sources/building_block_definition/cross_workspace_read", 2, vars),
				ConfigVariables: readerKey,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("version_latest"), versionRef),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("version_latest_release"), versionRef),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("versions"), ascendingVersionsExact(versionRef)),
					statecheck.CompareValuePairs(dataSourceAddress, tfjsonpath.New("ref"), buildingBlockDefinitionAddr, tfjsonpath.New("ref"), compare.ValuesSame()),
				},
			},
		}})
	})
}

type stubBuildingBlockDefinitionClient struct {
	client.MeshBuildingBlockDefinitionClient
	listed  []client.MeshBuildingBlockDefinition
	readErr error
}

func (s stubBuildingBlockDefinitionClient) List(context.Context, client.MeshBuildingBlockDefinitionFilter, ...client.ListOption) iter.Seq2[client.MeshBuildingBlockDefinition, error] {
	return func(yield func(client.MeshBuildingBlockDefinition, error) bool) {
		for _, definition := range s.listed {
			if !yield(definition, nil) {
				return
			}
		}
	}
}

func (s stubBuildingBlockDefinitionClient) Read(context.Context, string) (*client.MeshBuildingBlockDefinition, error) {
	return nil, s.readErr
}

func Test_buildingBlockDefinitionDataSourceCandidates(t *testing.T) {
	t.Run("reports a uuid the caller may not read as not found", func(t *testing.T) {
		dataSource := buildingBlockDefinitionDataSource{
			meshBuildingBlockDefinitionClient: stubBuildingBlockDefinitionClient{readErr: client.HttpError{StatusCode: http.StatusForbidden}},
		}
		filter := buildingBlockDefinitionFilter{Uuid: new("unpublished-uuid")}

		var diags diag.Diagnostics
		filter.only(dataSource.candidates(t.Context(), filter, &diags), &diags)

		require.Len(t, diags, 1)
		assert.Equal(t, "Building block definition not found", diags[0].Summary())
	})

	t.Run("reports a listed definition without uuid", func(t *testing.T) {
		dataSource := buildingBlockDefinitionDataSource{
			meshBuildingBlockDefinitionClient: stubBuildingBlockDefinitionClient{listed: []client.MeshBuildingBlockDefinition{{}}},
		}

		var diags diag.Diagnostics
		candidates := dataSource.candidates(t.Context(), buildingBlockDefinitionFilter{DisplayName: new("")}, &diags)

		assert.Nil(t, candidates)
		require.Len(t, diags, 1)
		assert.Equal(t, "Building block definition UUID missing", diags[0].Summary())
	})
}
