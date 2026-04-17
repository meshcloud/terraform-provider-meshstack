package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
)

func TestAccPlatformTypesDataSource(t *testing.T) {
	// Create a platform type first so the mock store is non-empty, with depends_on to ensure ordering.
	workspaceConfig, workspaceAddr := testconfig.Workspace(t)
	platformTypeConfig, platformTypeAddr := testconfig.PlatformType(t, workspaceAddr)
	config := testconfig.DataSource{Name: "platform_types"}.Config(t).WithFirstBlock(
		testconfig.Descend("depends_on")(testconfig.SetRawExpr("[%s]", platformTypeAddr))).
		Join(workspaceConfig, platformTypeConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.meshstack_platform_types.all", tfjsonpath.New("platform_types"), knownvalue.NotNull()),
				},
			},
		},
	})
}
