package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
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
			testconfig.Descend("metadata", "uuid")(testconfig.SetAddr(buildingBlockDefinitionAddr, "metadata", "uuid")),
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
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{versionRef})),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("version_latest_release"), knownvalue.Null()),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshBuildingBlockDefinition")),
					statecheck.CompareValuePairs(dataSourceAddress.String(), tfjsonpath.New("ref"), buildingBlockDefinitionAddr.String(), tfjsonpath.New("ref"), compare.ValuesSame()),
				},
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
			testconfig.Descend("metadata", "uuid")(testconfig.SetAddr(buildingBlockDefinitionAddr, "metadata", "uuid")),
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
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{versionRef})),
					statecheck.CompareValuePairs(dataSourceAddress.String(), tfjsonpath.New("ref"), buildingBlockDefinitionAddr.String(), tfjsonpath.New("ref"), compare.ValuesSame()),
				},
			},
		}})
	})
}
