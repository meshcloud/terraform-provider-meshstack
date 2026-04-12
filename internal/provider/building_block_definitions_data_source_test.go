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

func TestAccBuildingBlockDefinitionsDataSource(t *testing.T) {
	bbdConfig, bbdAddr := testconfig.BuildBBDManualConfig(t)

	dsConfig := testconfig.DataSource{Name: "building_block_definitions"}.Config(t)
	dsConfig = dsConfig.WithFirstBlock(t,
		testconfig.Traverse(t, "workspace_identifier")(testconfig.SetRawExpr(bbdAddr.Join("metadata", "owned_by_workspace").String())),
	)
	config := dsConfig.Join(bbdConfig)

	addr := testconfig.Traversal{"data.meshstack_building_block_definitions", "example"}

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("workspace_identifier"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("building_block_definitions"), knownvalue.ListExact([]knownvalue.Check{
						knownvalue.ObjectPartial(map[string]knownvalue.Check{
							"metadata": knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"uuid":               xknownvalue.NotEmptyString(),
								"owned_by_workspace": xknownvalue.NotEmptyString(),
							}),
							"spec": knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"display_name": xknownvalue.NotEmptyString(),
								"target_type":  xknownvalue.NotEmptyString(),
							}),
							"ref": knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"kind": knownvalue.StringExact("meshBuildingBlockDefinition"),
								"uuid": xknownvalue.NotEmptyString(),
							}),
							"version_latest": knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"uuid":         xknownvalue.NotEmptyString(),
								"number":       knownvalue.Int64Exact(1),
								"state":        knownvalue.StringExact("DRAFT"),
								"content_hash": xknownvalue.NotEmptyString(),
							}),
							"versions": knownvalue.ListExact([]knownvalue.Check{
								knownvalue.ObjectPartial(map[string]knownvalue.Check{
									"uuid":         xknownvalue.NotEmptyString(),
									"number":       knownvalue.Int64Exact(1),
									"state":        knownvalue.StringExact("DRAFT"),
									"content_hash": xknownvalue.NotEmptyString(),
								}),
							}),
							"version_latest_release": knownvalue.Null(),
						}),
					})),
				},
			},
		},
	})
}
