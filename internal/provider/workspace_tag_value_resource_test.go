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

func TestAccWorkspaceTagValue(t *testing.T) {
	t.Run("tags_in_parallel", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/workspace_tag_value/tags_in_parallel", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("meshstack_workspace_tag_value.team_a", tfjsonpath.New("value"), knownvalue.StringExact("team-a")),
						statecheck.ExpectKnownValue("meshstack_workspace_tag_value.team_b", tfjsonpath.New("value"), knownvalue.StringExact("team-b")),
						statecheck.ExpectKnownValue("meshstack_workspace_tag_value.team_c", tfjsonpath.New("value"), knownvalue.StringExact("team-c")),
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		})
	})

	t.Run("workspace_not_found", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:      hclStep(t, "resources/workspace_tag_value/workspace_not_found", 1, vars),
					ExpectError: expectErrorWords("Workspace", "not", "found"),
				},
			},
		})
	})

	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix}
	teamA := "meshstack_workspace_tag_value.team_a"
	teamB := "meshstack_workspace_tag_value.team_b"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/workspace_tag_value", 1, vars),
			},
			{
				Config: hclStep(t, "resources/workspace_tag_value", 2, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(teamA, plancheck.ResourceActionCreate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(teamA, tfjsonpath.New("workspace"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(teamA, tfjsonpath.New("key"), knownvalue.StringExact("teams-"+suffix)),
					statecheck.ExpectKnownValue(teamA, tfjsonpath.New("value"), knownvalue.StringExact("team-a")),
				},
			},
			{
				Config: hclStep(t, "resources/workspace_tag_value", 3, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(teamA, plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction(teamB, plancheck.ResourceActionCreate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: hclStep(t, "resources/workspace_tag_value", 4, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(teamA, plancheck.ResourceActionDestroy),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(teamB, tfjsonpath.New("value"), knownvalue.StringExact("team-b")),
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateId:   "test-ws-" + suffix + ".teams-" + suffix + "=team-b",
				ResourceName:    teamB,
			},
			{
				ImportState:   true,
				ResourceName:  teamB,
				ImportStateId: "test-ws-" + suffix + ".teams-" + suffix,
				ExpectError:   expectErrorWords("Unexpected", "Import", "Identifier"),
			},
		},
	})
}
