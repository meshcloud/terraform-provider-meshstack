package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"
)

func TestAccWorkspaceUserGroupMember(t *testing.T) {
	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix, "seeded_username": seededUsername}
	resourceAddress := "meshstack_workspace_user_group_member.example"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/workspace_user_group_member", 1, vars),
			},
			{
				Config: hclStep(t, "resources/workspace_user_group_member", 2, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
						plancheck.ExpectResourceAction("meshstack_workspace_user_group.example", plancheck.ResourceActionNoop),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("group_ref"), knownvalue.ObjectExact(map[string]knownvalue.Check{
						"kind":               knownvalue.StringExact(client.MeshObjectKind.WorkspaceUserGroup),
						"owned_by_workspace": knownvalue.StringExact("test-ws-" + suffix),
						"name":               knownvalue.StringExact("my-user-group"),
					})),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("user_ref"), knownvalue.ObjectExact(map[string]knownvalue.Check{
						"kind": knownvalue.StringExact(client.MeshObjectKind.User),
						"name": knownvalue.StringExact(seededUsername),
					})),
					statecheck.ExpectKnownValue("meshstack_workspace_user_group.example", tfjsonpath.New("spec").AtMapKey("members"), knownvalue.Null()),
				},
			},
			{
				ResourceName:    resourceAddress,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateId:   "test-ws-" + suffix + ".my-user-group." + seededUsername,
			},
		},
	})
}
