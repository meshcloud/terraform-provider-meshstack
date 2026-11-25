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

func TestAccWorkspaceUserGroup(t *testing.T) {
	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix, "seeded_username": seededUsername}
	resourceAddress := "meshstack_workspace_user_group.example"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/workspace_user_group", 1, vars),
			},
			{
				Config: hclStep(t, "resources/workspace_user_group", 2, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("my-user-group")),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("ref"), knownvalue.ObjectExact(map[string]knownvalue.Check{
						"kind":               knownvalue.StringExact(client.MeshObjectKind.WorkspaceUserGroup),
						"owned_by_workspace": knownvalue.StringExact("test-ws-" + suffix),
						"name":               knownvalue.StringExact("my-user-group"),
					})),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My User Group")),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("egid"), knownvalue.Null()),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("members"), knownvalue.SetExact([]knownvalue.Check{knownvalue.StringExact(seededUsername)})),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("status").AtMapKey("source_system"), knownvalue.Null()),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("status").AtMapKey("disabled_usernames"), knownvalue.SetExact([]knownvalue.Check{})),
				},
			},
			{
				Config: hclStep(t, "resources/workspace_user_group", 3, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Updated Display Name")),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("egid"), knownvalue.StringExact("egid-"+suffix)),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("members"), knownvalue.SetExact([]knownvalue.Check{})),
				},
			},
			{
				ResourceName:    resourceAddress,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateId:   "test-ws-" + suffix + ".my-user-group",
			},
		},
	})
}
