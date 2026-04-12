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

func TestAccProjectUserBindingDataSource(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("Skipping: requires user 'user@meshcloud.io' in local meshStack")
	}

	projectConfig, projectAddr, workspaceAddr := testconfig.BuildProjectAndWorkspaceConfig(t)

	bindingConfig := testconfig.Resource{Name: "project_user_binding"}.Config(t)
	var resourceAddress testconfig.Traversal
	bindingConfig = bindingConfig.WithFirstBlock(t,
		testconfig.ExtractIdentifier(&resourceAddress),
		testconfig.Traverse(t, "target_ref")(
			testconfig.Traverse(t, "owned_by_workspace")(testconfig.SetRawExpr(workspaceAddr.Format("%s.metadata.name"))),
			testconfig.Traverse(t, "name")(testconfig.SetRawExpr(projectAddr.Format("%s.metadata.name"))),
		),
	)

	dataSourceConfig := testconfig.DataSource{Name: "project_user_binding"}.Config(t)
	dataSourceConfig = dataSourceConfig.WithFirstBlock(t,
		testconfig.Traverse(t, "metadata", "name")(testconfig.SetRawExpr(resourceAddress.Format("%s.metadata.name"))),
	)

	config := dataSourceConfig.Join(bindingConfig, projectConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.meshstack_project_user_binding.example", tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("this-is-an-example")),
					statecheck.ExpectKnownValue("data.meshstack_project_user_binding.example", tfjsonpath.New("role_ref").AtMapKey("name"), xknownvalue.NotEmptyString()),
				},
			},
		},
	})
}
