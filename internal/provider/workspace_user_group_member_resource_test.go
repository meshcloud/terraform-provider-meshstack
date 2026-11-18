package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
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
						plancheck.ExpectResourceAction("meshstack_workspace_user_group.test", plancheck.ResourceActionNoop),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("workspace_identifier"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("group_identifier"), knownvalue.StringExact("my-user-group")),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("username"), knownvalue.StringExact(seededUsername)),
					statecheck.ExpectKnownValue("meshstack_workspace_user_group.test", tfjsonpath.New("spec").AtMapKey("members"), knownvalue.Null()),
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
