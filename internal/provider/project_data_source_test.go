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

func TestAccProjectDataSource(t *testing.T) {
	projectConfig, projectAddr, workspaceAddr := testconfig.BuildProjectAndWorkspaceConfig(t)

	dataSourceAddress := testconfig.Traversal{"data.meshstack_project", "example"}
	dataSourceConfig := testconfig.DataSource{Name: "project"}.Config(t)
	dataSourceConfig = dataSourceConfig.WithFirstBlock(t,
		testconfig.Traverse(t, "metadata")(
			testconfig.Traverse(t, "name")(testconfig.SetRawExpr(projectAddr.Format("%s.metadata.name"))),
			testconfig.Traverse(t, "owned_by_workspace")(testconfig.SetRawExpr(workspaceAddr.Format("%s.metadata.name"))),
		))

	config := dataSourceConfig.Join(projectConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("metadata").AtMapKey("name"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Project's Display Name")),
				},
			},
		},
	})
}
