package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccWorkspace(t *testing.T) {
	t.Run("superset_tag_reconciliation", func(t *testing.T) {
		// The backend returns a tag superset (an empty-list entry for every defined tag property, even
		// undeclared ones); the mock has no tag-schema logic and can't reproduce it. See the lock-step
		// policy in the acceptance-testing skill.
		if IsMockClientTest() {
			t.Skip("relies on the backend returning an entry for every defined tag property")
		}

		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		workspaceAddress := "meshstack_workspace.test"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/workspace/superset_tag_reconciliation", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						// Only the single declared tag remains; the undeclared property's empty-list
						// superset entry was reconciled away.
						statecheck.ExpectKnownValue(workspaceAddress, tfjsonpath.New("metadata").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							"test-key-" + suffix: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("12345")}),
						})),
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		})
	})

	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix}
	resourceAddress := "meshstack_workspace.example"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/workspace", 1, vars),
			},
			{
				Config: hclStep(t, "resources/workspace", 2, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
						// `kind` is the single constant value, so it is known already at plan time;
						// only the identifier is computed on create.
						plancheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshWorkspace")),
						plancheck.ExpectUnknownValue(resourceAddress, tfjsonpath.New("ref").AtMapKey("name")),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					// Metadata
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("created_on"), xknownvalue.NotEmptyString()),

					// Spec
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Workspace's Display Name")),

					// Ref
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshWorkspace")),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("ref").AtMapKey("name"), knownvalue.StringExact("test-ws-"+suffix)),
				},
			},
			{
				Config: hclStep(t, "resources/workspace", 3, vars),
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
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateId:   "test-ws-" + suffix,
				ResourceName:    resourceAddress,
			},
		},
	})
}
