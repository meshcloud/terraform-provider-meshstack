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

func TestAccBuildingBlockV2DataSource(t *testing.T) {
	bbConfig, bbAddr := testconfig.BuildBBv2WorkspaceConfig(t)

	dataSourceConfig := testconfig.DataSource{Name: "building_block_v2"}.Config(t)
	dataSourceConfig = dataSourceConfig.WithFirstBlock(t,
		testconfig.Traverse(t, "metadata", "uuid")(testconfig.SetRawExpr(bbAddr.Format("%s.metadata.uuid"))))

	config := dataSourceConfig.Join(bbConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.meshstack_building_block_v2.example", tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue("data.meshstack_building_block_v2.example", tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("my-workspace-building-block")),
				},
			},
		},
	})
}
