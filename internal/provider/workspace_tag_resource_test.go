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

func TestAccWorkspaceTag(t *testing.T) {
	t.Run("two_tags_on_one_workspace", func(t *testing.T) {
		// The resource's primary use case, and the "tags under other keys are read and written back
		// unchanged" claim in its description: each tag is a full read-modify-write of the workspace, so
		// the second write must preserve the first tag and deleting one must leave the other alone. Each
		// resource's Read only looks at its own key, so a clobbered tag surfaces as its resource dropping
		// out of state — hence the empty-plan checks.
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		firstAddress := "meshstack_workspace_tag.first"
		secondAddress := "meshstack_workspace_tag.second"
		firstKey := "test-key-first-" + suffix
		secondKey := "test-key-second-" + suffix

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/workspace_tag/two_tags_on_one_workspace", 1, vars),
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
					// Removing the second tag must not take the first one with it: Delete rewrites the
					// workspace without its own key only.
					Config: hclStep(t, "resources/workspace_tag/two_tags_on_one_workspace", 2, vars),
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

	t.Run("workspace_not_found", func(t *testing.T) {
		// This resource cannot create the workspace it tags, so a missing one must be a clear error
		// rather than a nil-deref or a silent no-op.
		vars := hclStepVars{"suffix": acctest.RandString(8)}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:      hclStep(t, "resources/workspace_tag/workspace_not_found", 1, vars),
					ExpectError: expectErrorWords("Workspace", "not", "found"),
				},
			},
		})
	})

	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix}
	workspaceTagAddress := "meshstack_workspace_tag.example"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/workspace_tag", 1, vars),
			},
			{
				Config: hclStep(t, "resources/workspace_tag", 2, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceTagAddress, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(workspaceTagAddress, tfjsonpath.New("metadata").AtMapKey("workspace_identifier"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(workspaceTagAddress, tfjsonpath.New("metadata").AtMapKey("key"), knownvalue.StringExact("billing-account")),
					statecheck.ExpectKnownValue(workspaceTagAddress, tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("12345")})),
				},
			},
			{
				Config: hclStep(t, "resources/workspace_tag", 3, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceTagAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(workspaceTagAddress, tfjsonpath.New("spec").AtMapKey("values"), knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("12345"), knownvalue.StringExact("67890")})),
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateId:   "test-ws-" + suffix + ".billing-account",
				ResourceName:    workspaceTagAddress,
			},
			{
				ImportState:   true,
				ResourceName:  workspaceTagAddress,
				ImportStateId: "no-dot",
				ExpectError:   expectErrorWords("Unexpected", "Import", "Identifier"),
			},
		},
	})
}
