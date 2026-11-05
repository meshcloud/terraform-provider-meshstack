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

func TestAccApiKey(t *testing.T) {
	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix}
	apiKeyAddr := "meshstack_api_key.example"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/api_key", 1, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(apiKeyAddr, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(apiKeyAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(apiKeyAddr, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(apiKeyAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("ci-key")),
					statecheck.ExpectKnownValue(apiKeyAddr, tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(apiKeyAddr, tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString()),
				},
			},
			{
				Config: hclStep(t, "resources/api_key", 2, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(apiKeyAddr, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(apiKeyAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("updated-key")),
					statecheck.ExpectKnownValue(apiKeyAddr, tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString()),
				},
			},
			{
				Config: hclStep(t, "resources/api_key", 3, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(apiKeyAddr, plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(apiKeyAddr, tfjsonpath.New("status").AtMapKey("client_secret")),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(apiKeyAddr, tfjsonpath.New("spec").AtMapKey("expires_at"), knownvalue.StringExact("2099-06-30")),
					statecheck.ExpectKnownValue(apiKeyAddr, tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString()),
				},
			},
		},
	})
}

// An expires_at only known at apply time (e.g. from time_rotating) must still plan a rotation.
func TestAccApiKey_ExpiresAtUnknownUntilApply(t *testing.T) {
	vars := hclStepVars{"suffix": acctest.RandString(8)}
	apiKeyAddr := "meshstack_api_key.test"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "resources/api_key/expires_at_unknown_until_apply", 1, vars),
			},
			{
				Config: hclStep(t, "resources/api_key/expires_at_unknown_until_apply", 2, vars),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(apiKeyAddr, plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(apiKeyAddr, tfjsonpath.New("status").AtMapKey("client_secret")),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(apiKeyAddr, tfjsonpath.New("spec").AtMapKey("expires_at"), knownvalue.StringExact("2099-06-30")),
					statecheck.ExpectKnownValue(apiKeyAddr, tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString()),
				},
			},
		},
	})
}
