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

func TestAccProject(t *testing.T) {
	t.Run("restricted_default_tag", func(t *testing.T) {
		// Backend-materialized default: the mock has no tag-restriction business logic, so it can't
		// reproduce TagService.determineTags injecting a restricted tag's default on create. See the
		// lock-step policy in the acceptance-testing skill.
		if IsMockClientTest() {
			t.Skip("relies on the backend injecting a restricted tag's default value on create")
		}

		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		resourceAddress := "meshstack_project.test"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/project/restricted_default_tag", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							"test-key-" + suffix: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("blue")}),
						})),
					},
					// Refresh reads back the injected superset; reconcileTrackedTags must reconcile it
					// away so no drift remains.
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		}, TouchesExclusively(client.MeshObjectKind.Project))
	})

	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix}
	resourceAddress := "meshstack_project.example"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/project", 1, vars),
			},
			{
				Config: hclStep(t, "resources/project", 2, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("my-project")),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Project's Display Name")),
				},
			},
			{
				Config: hclStep(t, "resources/project", 3, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Updated Display Name")),
				},
			},
			{
				ResourceName:    resourceAddress,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateId:   "test-ws-" + suffix + ".my-project",
			},
		},
	})
}
