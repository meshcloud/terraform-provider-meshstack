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

func TestAccProjectGroupBindingDataSource(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("Skipping: requires user group 'my-user-group' in local meshStack")
	}

	projectConfig, projectAddr, workspaceAddr := testconfig.ProjectAndWorkspace(t)

	var resourceAddress testconfig.Traversal
	bindingConfig := testconfig.Resource{Name: "project_group_binding"}.Config(t).WithFirstBlock(
		testconfig.ExtractAddress(&resourceAddress),
		testconfig.Descend("target_ref")(
			testconfig.Descend("owned_by_workspace")(testconfig.SetAddr(workspaceAddr, "metadata", "name")),
			testconfig.Descend("name")(testconfig.SetAddr(projectAddr, "metadata", "name")),
		),
	)

	config := testconfig.DataSource{Name: "project_group_binding"}.Config(t).WithFirstBlock(
		testconfig.Descend("metadata", "name")(testconfig.SetAddr(resourceAddress, "metadata", "name")),
	).Join(bindingConfig, projectConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.meshstack_project_group_binding.example", tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("this-is-an-example")),
					statecheck.ExpectKnownValue("data.meshstack_project_group_binding.example", tfjsonpath.New("role_ref").AtMapKey("name"), xknownvalue.NotEmptyString()),
				},
			},
		},
	})
}
