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

func TestAccProjectTagValue(t *testing.T) {
	t.Run("tags_in_parallel", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/project_tag_value/tags_in_parallel", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("meshstack_project_tag_value.team_a", tfjsonpath.New("spec").AtMapKey("value"), knownvalue.StringExact("team-a")),
						statecheck.ExpectKnownValue("meshstack_project_tag_value.team_b", tfjsonpath.New("spec").AtMapKey("value"), knownvalue.StringExact("team-b")),
						statecheck.ExpectKnownValue("meshstack_project_tag_value.team_c", tfjsonpath.New("spec").AtMapKey("value"), knownvalue.StringExact("team-c")),
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		})
	})

	t.Run("project_not_found", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:      hclStep(t, "resources/project_tag_value/project_not_found", 1, vars),
					ExpectError: expectErrorWords("Project", "not", "found"),
				},
			},
		})
	})

	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix}
	teamA := "meshstack_project_tag_value.team_a"
	teamB := "meshstack_project_tag_value.team_b"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/project_tag_value", 1, vars),
			},
			{
				Config: hclStep(t, "resources/project_tag_value", 2, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(teamA, plancheck.ResourceActionCreate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(teamA, tfjsonpath.New("metadata").AtMapKey("workspace_identifier"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(teamA, tfjsonpath.New("metadata").AtMapKey("project_identifier"), knownvalue.StringExact("my-project")),
					statecheck.ExpectKnownValue(teamA, tfjsonpath.New("metadata").AtMapKey("key"), knownvalue.StringExact("teams-"+suffix)),
					statecheck.ExpectKnownValue(teamA, tfjsonpath.New("spec").AtMapKey("value"), knownvalue.StringExact("team-a")),
				},
			},
			{
				Config: hclStep(t, "resources/project_tag_value", 3, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(teamA, plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction(teamB, plancheck.ResourceActionCreate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: hclStep(t, "resources/project_tag_value", 4, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(teamA, plancheck.ResourceActionDestroy),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(teamB, tfjsonpath.New("spec").AtMapKey("value"), knownvalue.StringExact("team-b")),
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateId:   "test-ws-" + suffix + ".my-project.teams-" + suffix + "=team-b",
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
