package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

// Addresses of the blocks in examples/{resources,data-sources}/meshstack_payment_method/*-test-*.tf.
const (
	paymentMethodResourceAddr   = "meshstack_payment_method.example"
	paymentMethodDataSourceAddr = "data.meshstack_payment_method.example"
)

// paymentMethodStepConfig is the payment method example's step, with the workspace that owns it.
func paymentMethodStepConfig(t *testing.T, index int, supportNames ...string) string {
	t.Helper()
	return examples.JoinTestStepConfigs(
		examples.Resource.TestStepConfig(t, "payment_method", index, supportNames...),
		examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
	)
}

func TestAccPaymentMethod(t *testing.T) {
	t.Run("superset_tag_reconciliation", func(t *testing.T) {
		// The backend returns a tag superset (an empty-list entry for every defined tag property, even
		// undeclared ones); the mock has no tag-schema logic and can't reproduce it. See the lock-step
		// policy in the acceptance-testing skill.
		if IsMockClientTest() {
			t.Skip("relies on the backend returning an entry for every defined tag property")
		}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          paymentMethodStepConfig(t, 1, "prerequisites", "undeclared-tag"),
					ConfigVariables: SuffixVariables(acctest.RandString(8)),
					ConfigStateChecks: []statecheck.StateCheck{
						// Only the single declared tag remains; the undeclared property's empty-list
						// superset entry was reconciled away.
						statecheck.ExpectKnownValue(paymentMethodResourceAddr, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapSizeExact(1)),
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		})
	})

	suffix := acctest.RandString(8)
	vars := SuffixVariables(suffix)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          paymentMethodStepConfig(t, 1, "prerequisites"),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(paymentMethodResourceAddr, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(paymentMethodResourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("test-pm-"+suffix)),
					statecheck.ExpectKnownValue(paymentMethodResourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Payment Method")),
				},
			},
			{
				Config:          paymentMethodStepConfig(t, 2, "prerequisites"),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(paymentMethodResourceAddr, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(paymentMethodResourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Updated Payment Method")),
				},
			},
			{
				ResourceName:    paymentMethodResourceAddr,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ConfigVariables: vars,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources[paymentMethodResourceAddr]
					if rs == nil {
						return "", fmt.Errorf("resource not found: %s", paymentMethodResourceAddr)
					}
					ws := s.RootModule().Resources[workspaceResourceAddr]
					if ws == nil {
						return "", fmt.Errorf("workspace resource not found: %s", workspaceResourceAddr)
					}
					return ws.Primary.Attributes["metadata.name"] + "." + rs.Primary.Attributes["metadata.name"], nil
				},
			},
		},
	})
}
