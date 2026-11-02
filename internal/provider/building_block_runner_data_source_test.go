package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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

}
