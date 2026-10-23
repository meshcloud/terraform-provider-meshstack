package provider

import (
	"context"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccBuildingBlockDefinitionDataSource(t *testing.T) {
	t.Run("simple state check", func(t *testing.T) {
		buildingBlockDefinitionConfig, buildingBlockDefinitionAddr := testconfig.BBDManual(t)

		var dataSourceAddress testconfig.Traversal
		config := testconfig.DataSource{Name: "building_block_definition"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&dataSourceAddress),
			testconfig.Descend("spec")(testconfig.RemoveKey()),
			testconfig.Descend("metadata")(testconfig.SetRawExpr("{ uuid = %s }", buildingBlockDefinitionAddr.Join("metadata", "uuid"))),
		).Join(buildingBlockDefinitionConfig)

		versionRef := knownvalue.ObjectPartial(map[string]knownvalue.Check{
			"uuid":         xknownvalue.NotEmptyString(),
			"number":       knownvalue.Int64Exact(1),
			"state":        knownvalue.StringExact("DRAFT"),
			"content_hash": xknownvalue.NotEmptyString(),
		})
		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("metadata"), knownvalue.ObjectExact(map[string]knownvalue.Check{
						"uuid":               xknownvalue.NotEmptyString(),
						"owned_by_workspace": xknownvalue.NotEmptyString(),
					})),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("spec"), knownvalue.ObjectExact(map[string]knownvalue.Check{
						"display_name": knownvalue.StringExact("Example Building Block"),
						"target_type":  knownvalue.StringExact("WORKSPACE_LEVEL"),
					})),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("version_latest"), versionRef),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("versions"), ascendingVersionsExact(versionRef)),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("version_latest_release"), knownvalue.Null()),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshBuildingBlockDefinition")),
					statecheck.CompareValuePairs(dataSourceAddress.String(), tfjsonpath.New("ref"), buildingBlockDefinitionAddr.String(), tfjsonpath.New("ref"), compare.ValuesSame()),
				},
			},
		}})
	})

	t.Run("filters", func(t *testing.T) {
		definitionConfig, definitionAddr := testconfig.BBDManual(t)
		dataSourceAddr := testconfig.Traversal{"data.meshstack_building_block_definition", "example"}
		example := testconfig.DataSource{Name: "building_block_definition"}.Config(t)

		// Parallel tests create definitions of the same display name, so the workspace keeps the lookup unique.
		byDisplayName := example.WithFirstBlock(
			testconfig.Descend("spec")(testconfig.SetRawExpr("{ display_name = %s }", definitionAddr.Join("spec", "display_name"))),
			testconfig.Descend("metadata")(testconfig.SetRawExpr("{ owned_by_workspace = %s }", definitionAddr.Join("metadata", "owned_by_workspace"))),
		).Join(definitionConfig)
		unknownName := example.WithFirstBlock(
			testconfig.Descend("spec")(testconfig.SetRawExpr(`{ display_name = "No Such Building Block" }`)),
			testconfig.Descend("metadata")(testconfig.SetRawExpr("{ owned_by_workspace = %s }", definitionAddr.Join("metadata", "owned_by_workspace"))),
		).Join(definitionConfig)
		uuidWithOtherTargetType := example.WithFirstBlock(
			testconfig.Descend("spec")(testconfig.SetRawExpr(`{ target_type = "TENANT_LEVEL" }`)),
			testconfig.Descend("metadata")(testconfig.SetRawExpr("{ uuid = %s }", definitionAddr.Join("metadata", "uuid"))),
		).Join(definitionConfig)
		noFilter := example.WithFirstBlock(
			testconfig.Descend("spec")(testconfig.RemoveKey()),
		).Join(definitionConfig)

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config:      noFilter.String(),
				ExpectError: regexp.MustCompile(`(?s)At\s+least\s+one\s+of\s+these\s+attributes\s+must\s+be\s+configured.*metadata.uuid`),
			},
			{
				Config: byDisplayName.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.CompareValuePairs(dataSourceAddr.String(), tfjsonpath.New("ref"), definitionAddr.String(), tfjsonpath.New("ref"), compare.ValuesSame()),
					statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("spec").AtMapKey("target_type"), knownvalue.StringExact("WORKSPACE_LEVEL")),
				},
			},
			{
				Config:      unknownName.String(),
				ExpectError: regexp.MustCompile(`(?s)No building block definition that you can see matches.*spec.display_name\s+=\s+"No\s+Such\s+Building\s+Block"`),
			},
			{
				Config:      uuidWithOtherTargetType.String(),
				ExpectError: regexp.MustCompile(`No building block definition that you can see matches\s+metadata.uuid`),
			},
		}})
	})

	t.Run("cross-workspace read", func(t *testing.T) {
		if IsMockClientTest() {
			t.Skip("cross-workspace test requires real permission boundaries")
		}

		buildingBlockDefinitionConfig, buildingBlockDefinitionAddr := testconfig.BBDManual(t)
		readerConfig, captureReaderKey, readerKey := otherWorkspaceReader(t)

		supportConfig := buildingBlockDefinitionConfig.WithFirstBlock(
			testconfig.Descend("version_spec", "draft")(testconfig.SetValue(cty.False)),
		).Join(readerConfig)

		var dataSourceAddress testconfig.Traversal
		config := testconfig.DataSource{Name: "building_block_definition"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&dataSourceAddress),
			testconfig.Descend("spec")(testconfig.RemoveKey()),
			testconfig.Descend("metadata")(testconfig.SetRawExpr("{ uuid = %s }", buildingBlockDefinitionAddr.Join("metadata", "uuid"))),
			testconfig.Descend("provider")(testconfig.SetRawExpr("meshstack-other")),
		).Join(supportConfig, testconfig.OtherProviderConfig(t))

		versionRef := knownvalue.ObjectPartial(map[string]knownvalue.Check{
			"uuid":         xknownvalue.NotEmptyString(),
			"state":        knownvalue.StringExact("RELEASED"),
			"content_hash": knownvalue.Null(),
		})
		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: supportConfig.String(),
				ConfigStateChecks: append([]statecheck.StateCheck{
					statecheck.ExpectKnownValue(buildingBlockDefinitionAddr.String(), tfjsonpath.New("version_latest_release").AtMapKey("state"), knownvalue.StringExact("RELEASED")),
				}, captureReaderKey...),
			},
			{
				Config:          config.String(),
				ConfigVariables: readerKey,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("version_latest"), versionRef),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("version_latest_release"), versionRef),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("versions"), ascendingVersionsExact(versionRef)),
					statecheck.CompareValuePairs(dataSourceAddress.String(), tfjsonpath.New("ref"), buildingBlockDefinitionAddr.String(), tfjsonpath.New("ref"), compare.ValuesSame()),
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

func (s stubBuildingBlockDefinitionClient) List(context.Context, *string) ([]client.MeshBuildingBlockDefinition, error) {
	return s.listed, nil
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
