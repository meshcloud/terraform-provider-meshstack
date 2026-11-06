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

func TestAccWorkspaceTags(t *testing.T) {
	t.Run("declared_empty_value_list", func(t *testing.T) {
		// A declared key with no values must converge. The API returns an entry for every defined tag
		// property — an empty list when unset — and reconcileTags mirrors tracked keys verbatim, so the
		// key stays in state instead of being dropped on every refresh.
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		workspaceTagsAddress := "meshstack_workspace_tags.test"
		tagKey := "test-key-" + suffix

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/workspace_tags/declared_empty_value_list", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(workspaceTagsAddress, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							tagKey: knownvalue.ListSizeExact(0),
						})),
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		})
	})

	t.Run("removing_one_key_of_several", func(t *testing.T) {
		// The map is authoritative per key, not just as a whole: dropping one key of two must remove that
		// tag and leave the other untouched.
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		workspaceTagsAddress := "meshstack_workspace_tags.test"
		firstKey := "test-key-first-" + suffix
		secondKey := "test-key-second-" + suffix

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/workspace_tags/removing_one_key_of_several", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(workspaceTagsAddress, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							firstKey:  knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("first")}),
							secondKey: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("second")}),
						})),
					},
				},
				{
					Config: hclStep(t, "resources/workspace_tags/removing_one_key_of_several", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(workspaceTagsAddress, plancheck.ResourceActionUpdate),
						},
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(workspaceTagsAddress, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							firstKey: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("first")}),
						})),
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
					Config:      hclStep(t, "resources/workspace_tags/workspace_not_found", 1, vars),
					ExpectError: expectErrorWords("Workspace", "not", "found"),
				},
			},
		})
	})

	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix}
	workspaceTagsAddress := "meshstack_workspace_tags.example"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/workspace_tags", 1, vars),
			},
			{
				Config: hclStep(t, "resources/workspace_tags", 2, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceTagsAddress, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(workspaceTagsAddress, tfjsonpath.New("metadata").AtMapKey("workspace_identifier"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(workspaceTagsAddress, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
						"cost-center-" + suffix: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("12345")}),
					})),
				},
			},
			{
				Config: hclStep(t, "resources/workspace_tags", 3, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceTagsAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(workspaceTagsAddress, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapSizeExact(0)),
				},
			},
			{
				// Restore the tag so the import below runs against a workspace that actually has tags.
				Config: hclStep(t, "resources/workspace_tags", 4, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceTagsAddress, plancheck.ResourceActionUpdate),
					},
				},
			},
			{
				// Read must pass the API's tags through when there is no prior state, rather than
				// reconcile against an empty tracked set and import nothing.
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateId:   "test-ws-" + suffix,
				ResourceName:    workspaceTagsAddress,
			},
		},
	})
}
