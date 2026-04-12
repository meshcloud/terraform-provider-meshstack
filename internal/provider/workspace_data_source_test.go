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

func TestAccWorkspaceDataSource(t *testing.T) {
	workspaceConfig, workspaceAddr := testconfig.BuildWorkspaceConfig(t)

	dataSourceAddress := testconfig.Traversal{"data.meshstack_workspace", "example"}
	dsConfig := testconfig.DataSource{Name: "workspace"}.Config(t)
	dsConfig = dsConfig.WithFirstBlock(t,
		testconfig.Traverse(t, "metadata", "name")(testconfig.SetRawExpr(workspaceAddr.Format("%s.metadata.name"))))
	config := dsConfig.Join(workspaceConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshWorkspace")),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("ref").AtMapKey("identifier"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("metadata").AtMapKey("name"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Workspace's Display Name")),
				},
			},
		},
	})
}
