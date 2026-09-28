package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// Address of the block in examples/resources/meshstack_api_key/resource-test-*.tf.
const apiKeyResourceAddr = "meshstack_api_key.example"

func TestAccApiKey(t *testing.T) {
	vars := SuffixVariables(acctest.RandString(8))

	stepConfig := func(index int) string {
		return examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "api_key", index),
			examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
		)
	}

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          stepConfig(1),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(apiKeyResourceAddr, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("ci-key")),
					statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString()),
				},
			},
			{
				Config:          stepConfig(2),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(apiKeyResourceAddr, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("updated-key")),
					statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString()),
				},
			},
			{
				Config:          stepConfig(3),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(apiKeyResourceAddr, plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(apiKeyResourceAddr, tfjsonpath.New("status").AtMapKey("client_secret")),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("spec").AtMapKey("expires_at"), knownvalue.StringExact("2099-06-30")),
					statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString()),
				},
			},
		},
	})
}
