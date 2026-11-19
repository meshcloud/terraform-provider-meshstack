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

func TestAccProjectTag(t *testing.T) {
	t.Run("two_tags_on_one_project", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		firstAddress := "meshstack_project_tag.first"
		secondAddress := "meshstack_project_tag.second"
		firstKey := "test-key-first-" + suffix
		secondKey := "test-key-second-" + suffix

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/project_tag/two_tags_on_one_project", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(firstAddress, tfjsonpath.New("metadata").AtMapKey("key"), knownvalue.StringExact(firstKey)),
						statecheck.ExpectKnownValue(firstAddress, tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("12345")})),
						statecheck.ExpectKnownValue(secondAddress, tfjsonpath.New("metadata").AtMapKey("key"), knownvalue.StringExact(secondKey)),
						statecheck.ExpectKnownValue(secondAddress, tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("second")})),
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
				{
					Config: hclStep(t, "resources/project_tag/two_tags_on_one_project", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(secondAddress, plancheck.ResourceActionDestroy),
						},
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(firstAddress, tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("12345")})),
					},
				},
			},
		})
	})

	t.Run("tags_in_parallel", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/project_tag/tags_in_parallel", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("meshstack_project_tag.a", tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("a")})),
						statecheck.ExpectKnownValue("meshstack_project_tag.b", tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("b")})),
						statecheck.ExpectKnownValue("meshstack_project_tag.c", tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("c")})),
						statecheck.ExpectKnownValue("meshstack_project_tag.d", tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("d")})),
						statecheck.ExpectKnownValue("meshstack_project_tag.e", tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("e")})),
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
					Config:      hclStep(t, "resources/project_tag/project_not_found", 1, vars),
					ExpectError: expectErrorWords("Project", "not", "found"),
				},
			},
		})
	})

	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix}
	projectTagAddress := "meshstack_project_tag.example"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/project_tag", 1, vars),
			},
			{
				Config: hclStep(t, "resources/project_tag", 2, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(projectTagAddress, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(projectTagAddress, tfjsonpath.New("metadata").AtMapKey("workspace_identifier"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(projectTagAddress, tfjsonpath.New("metadata").AtMapKey("project_identifier"), knownvalue.StringExact("my-project")),
					statecheck.ExpectKnownValue("meshstack_project.test", tfjsonpath.New("spec").AtMapKey("payment_method_identifier"), knownvalue.StringExact("test-pm-"+suffix)),
					statecheck.ExpectKnownValue(projectTagAddress, tfjsonpath.New("metadata").AtMapKey("key"), knownvalue.StringExact("cost-center-"+suffix)),
					statecheck.ExpectKnownValue(projectTagAddress, tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("12345")})),
				},
			},
			{
				Config: hclStep(t, "resources/project_tag", 3, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(projectTagAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(projectTagAddress, tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("12345"), knownvalue.StringExact("67890")})),
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateId:   "test-ws-" + suffix + ".my-project.cost-center-" + suffix,
				ResourceName:    projectTagAddress,
			},
			{
				ImportState:   true,
				ResourceName:  projectTagAddress,
				ImportStateId: "test-ws-" + suffix + ".my-project",
				ExpectError:   expectErrorWords("Unexpected", "Import", "Identifier"),
			},
		},
	})
}
