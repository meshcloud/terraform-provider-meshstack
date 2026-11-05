package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"
)

func TestAccPaymentMethod(t *testing.T) {
	t.Run("superset_tag_reconciliation", func(t *testing.T) {
		// The backend returns a tag superset (an empty-list entry for every defined tag property, even
		// undeclared ones); the mock has no tag-schema logic and can't reproduce it. See the lock-step
		// policy in the acceptance-testing skill.
		if IsMockClientTest() {
			t.Skip("relies on the backend returning an entry for every defined tag property")
		}

		vars := hclStepVars{"suffix": acctest.RandString(8)}
		pmAddr := "meshstack_payment_method.test"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/payment_method/superset_tag_reconciliation", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						// Only the single declared tag remains; the undeclared property's empty-list
						// superset entry was reconciled away.
						statecheck.ExpectKnownValue(pmAddr, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapSizeExact(1)),
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
	resourceAddress := "meshstack_payment_method.example"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/payment_method", 1, vars),
			},
			{
				Config: hclStep(t, "resources/payment_method", 2, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("test-pm-"+suffix)),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Payment Method")),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact(client.MeshObjectKind.PaymentMethod)),
					statecheck.CompareValuePairs(resourceAddress, tfjsonpath.New("ref").AtMapKey("name"), resourceAddress, tfjsonpath.New("metadata").AtMapKey("name"), compare.ValuesSame()),
				},
			},
			{
				Config: hclStep(t, "resources/payment_method", 3, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Updated Payment Method")),
				},
			},
			{
				ResourceName:    resourceAddress,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateId:   "test-ws-" + suffix + ".test-pm-" + suffix,
			},
		},
	})
}
