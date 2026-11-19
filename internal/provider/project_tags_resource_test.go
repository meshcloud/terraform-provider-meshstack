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

func TestAccProjectTags(t *testing.T) {
	t.Run("declared_empty_value_list", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		projectTagsAddress := "meshstack_project_tags.test"
		tagKey := "test-key-" + suffix

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/project_tags/declared_empty_value_list", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(projectTagsAddress, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
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
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		projectTagsAddress := "meshstack_project_tags.test"
		firstKey := "test-key-first-" + suffix
		secondKey := "test-key-second-" + suffix

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/project_tags/removing_one_key_of_several", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(projectTagsAddress, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							firstKey:  knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("first")}),
							secondKey: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("second")}),
						})),
					},
				},
				{
					Config: hclStep(t, "resources/project_tags/removing_one_key_of_several", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(projectTagsAddress, plancheck.ResourceActionUpdate),
						},
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(projectTagsAddress, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							firstKey: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("first")}),
						})),
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
					Config:      hclStep(t, "resources/project_tags/project_not_found", 1, vars),
					ExpectError: expectErrorWords("Project", "not", "found"),
				},
			},
		})
	})

	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix}
	projectTagsAddress := "meshstack_project_tags.example"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/project_tags", 1, vars),
			},
			{
				Config: hclStep(t, "resources/project_tags", 2, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(projectTagsAddress, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(projectTagsAddress, tfjsonpath.New("metadata").AtMapKey("workspace_identifier"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(projectTagsAddress, tfjsonpath.New("metadata").AtMapKey("project_identifier"), knownvalue.StringExact("my-project")),
					statecheck.ExpectKnownValue("meshstack_project.test", tfjsonpath.New("spec").AtMapKey("payment_method_identifier"), knownvalue.StringExact("test-pm-"+suffix)),
					statecheck.ExpectKnownValue(projectTagsAddress, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
						"cost-center-" + suffix: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("12345")}),
					})),
				},
			},
			{
				Config: hclStep(t, "resources/project_tags", 3, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(projectTagsAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(projectTagsAddress, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapSizeExact(0)),
				},
			},
			{
				Config: hclStep(t, "resources/project_tags", 4, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(projectTagsAddress, plancheck.ResourceActionUpdate),
					},
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateId:   "test-ws-" + suffix + ".my-project",
				ResourceName:    projectTagsAddress,
			},
			{
				ImportState:   true,
				ResourceName:  projectTagsAddress,
				ImportStateId: "test-ws-" + suffix,
				ExpectError:   expectErrorWords("Unexpected", "Import", "Identifier"),
			},
		},
	})
}
