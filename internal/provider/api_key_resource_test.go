package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccApiKey(t *testing.T) {
	workspaceConfig, workspaceAddr := testconfig.Workspace(t)
	apiKeyConfig, apiKeyAddr := testconfig.ApiKey(t, workspaceAddr)

	config := apiKeyConfig.Join(workspaceConfig)

	updateConfig := config.WithFirstBlock(
		testconfig.Descend("spec", "display_name")(testconfig.SetString("updated-key")))

	rotateConfig := config.WithFirstBlock(
		testconfig.Descend("spec", "expires_at")(testconfig.SetString("2099-06-30")))

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(apiKeyAddr.String(), plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("ci-key")),
					statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString()),
				},
			},
			{
				Config: updateConfig.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(apiKeyAddr.String(), plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("updated-key")),
					statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString()),
				},
			},
			{
				Config: rotateConfig.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(apiKeyAddr.String(), plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_secret")),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("spec").AtMapKey("expires_at"), knownvalue.StringExact("2099-06-30")),
					statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString()),
				},
			},
		},
	})
}
