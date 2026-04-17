package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccWorkspaceGroupBinding(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("Skipping: requires user group 'my-user-group' in local meshStack")
	}

	workspaceConfig, workspaceAddr := testconfig.Workspace(t)

	var resourceAddress testconfig.Traversal
	config := testconfig.Resource{Name: "workspace_group_binding"}.Config(t).WithFirstBlock(
		testconfig.ExtractAddress(&resourceAddress),
		testconfig.Descend("target_ref", "name")(testconfig.SetAddr(workspaceAddr, "metadata", "name"))).
		Join(workspaceConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("this-is-an-example")),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("role_ref").AtMapKey("name"), knownvalue.StringExact("Workspace Member")),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("target_ref").AtMapKey("name"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("subject").AtMapKey("name"), knownvalue.StringExact("my-user-group")),
				},
			},
			{
				ResourceName:    resourceAddress.String(),
				ImportState:     true,
				ImportStateId:   "this-is-an-example",
				ImportStateKind: resource.ImportBlockWithID,
			},
		},
	})
}
