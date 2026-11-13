package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	xknownvalue "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccBuildingBlockRunnerResource(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		runnerAddr := "meshstack_building_block_runner.example"
		var runnerUuid string
		var replacedRunnerUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/building_block_runner/basic", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(runnerAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(runnerAddr, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), knownvalue.StringExact("test-ws-"+suffix)),
						statecheck.ExpectKnownValue(runnerAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(runnerAddr, tfjsonpath.New("spec").AtMapKey("implementation_type"), knownvalue.StringExact("TERRAFORM")),
						statecheck.ExpectKnownValue(runnerAddr, tfjsonpath.New("spec").AtMapKey("restriction"), knownvalue.StringExact("PRIVATE")),
						statecheck.ExpectKnownValue(runnerAddr, tfjsonpath.New("status"), xknownvalue.MapExact(map[string]knownvalue.Check{
							"last_seen":          knownvalue.Null(),
							"last_known_version": knownvalue.Null(),
							"is_default":         knownvalue.Bool(false),
						})),
						xknownvalue.Ref(runnerAddr, "meshBuildingBlockRunner", &runnerUuid),
					},
				},
				{
					Config: hclStep(t, "resources/building_block_runner/basic", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(runnerAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(runnerAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Updated Runner")),
						xknownvalue.Ref(runnerAddr, "meshBuildingBlockRunner", &runnerUuid),
					},
				},
				{
					// TODO: Change this expectation to ResourceActionUpdate once meshStack supports
					// in-place updates for implementation_type.
					Config: hclStep(t, "resources/building_block_runner/basic", 3, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(runnerAddr, plancheck.ResourceActionReplace),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(runnerAddr, tfjsonpath.New("spec").AtMapKey("implementation_type"), knownvalue.StringExact("GITHUB_WORKFLOW")),
						xknownvalue.Ref(runnerAddr, "meshBuildingBlockRunner", &replacedRunnerUuid),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return replacedRunnerUuid, nil
					},
					ResourceName: runnerAddr,
				},
			},
		})
	})

	t.Run("wif", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		runnerAddr := "meshstack_building_block_runner.example"
		var runnerUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/building_block_runner/wif", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(runnerAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(runnerAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("GCP WIF Runner")),
						statecheck.ExpectKnownValue(runnerAddr, tfjsonpath.New("spec").AtMapKey("workload_identity_federation"), xknownvalue.MapExact(map[string]knownvalue.Check{
							"subject_template": knownvalue.StringExact("system:serviceaccount:meshfed:my-runner"),
							"issuer":           knownvalue.StringExact("https://oidc.example.com"),
							"gcp": xknownvalue.MapExact(map[string]knownvalue.Check{
								"audience":   knownvalue.StringExact("//iam.googleapis.com/projects/123456/locations/global/workloadIdentityPools/meshstack/providers/meshfed"),
								"token_path": knownvalue.StringExact("/var/run/secrets/workload-identity/token"),
							}),
							"aws":   knownvalue.Null(),
							"azure": knownvalue.Null(),
						})),
						xknownvalue.Ref(runnerAddr, "meshBuildingBlockRunner", &runnerUuid),
					},
				},
			},
		})
	})

	t.Run("wif_subject_template", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		runnerAddr := "meshstack_building_block_runner.example_with_wif"
		exampleTemplate := "system:serviceaccount:namespace:workspace.{{ workspaceIdentifier }}.buildingblockdefinition.{{ buildingBlockDefinitionUuid }}"
		updatedTemplate := "system:serviceaccount:namespace:bbd.{{ buildingBlockDefinitionUuid }}"
		var runnerUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/building_block_runner/wif_subject_template", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(runnerAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(runnerAddr, tfjsonpath.New("spec").AtMapKey("workload_identity_federation"), xknownvalue.MapExact(map[string]knownvalue.Check{
							"subject_template": knownvalue.StringExact(exampleTemplate),
							"issuer":           knownvalue.StringExact("https://oidc.example.com"),
							"gcp": xknownvalue.MapExact(map[string]knownvalue.Check{
								"audience":   knownvalue.StringExact("gcp-workload-identity-provider:namespace"),
								"token_path": knownvalue.StringExact("/var/run/secrets/workload-identity/token"),
							}),
							"aws":   knownvalue.Null(),
							"azure": knownvalue.Null(),
						})),
						xknownvalue.Ref(runnerAddr, "meshBuildingBlockRunner", &runnerUuid),
					},
				},
				{
					Config: hclStep(t, "resources/building_block_runner/wif_subject_template", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(runnerAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(runnerAddr, tfjsonpath.New("spec").AtMapKey("workload_identity_federation").AtMapKey("subject_template"), knownvalue.StringExact(updatedTemplate)),
						xknownvalue.Ref(runnerAddr, "meshBuildingBlockRunner", &runnerUuid),
					},
				},
			},
		})
	})

	t.Run("wif_validation", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:      hclStep(t, "resources/building_block_runner/wif_validation", 1, vars),
					ExpectError: expectErrorWords("provider", "configuration", "set"),
				},
				{
					Config:      hclStep(t, "resources/building_block_runner/wif_validation", 2, vars),
					ExpectError: expectErrorWords("subject_template", "required"),
				},
				{
					Config:      hclStep(t, "resources/building_block_runner/wif_validation", 3, vars),
					ExpectError: expectErrorWords("subject_template", "empty", "whitespace"),
				},
			},
		})
	})

	t.Run("restriction_replace_mock_only", func(t *testing.T) {
		if !IsMockClientTest() {
			t.Skip("mock-only test: PUBLIC restriction may require admin permissions in real meshStack")
		}

		vars := hclStepVars{"suffix": acctest.RandString(8)}
		runnerAddr := "meshstack_building_block_runner.example"
		var runnerUuid string
		var replacedRunnerUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/building_block_runner/restriction_replace_mock_only", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						xknownvalue.Ref(runnerAddr, "meshBuildingBlockRunner", &runnerUuid),
						statecheck.ExpectKnownValue(runnerAddr, tfjsonpath.New("spec").AtMapKey("restriction"), knownvalue.StringExact("PRIVATE")),
					},
				},
				{
					// TODO: Change this expectation to ResourceActionUpdate once meshStack supports
					// in-place updates for restriction.
					Config: hclStep(t, "resources/building_block_runner/restriction_replace_mock_only", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(runnerAddr, plancheck.ResourceActionReplace),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						xknownvalue.Ref(runnerAddr, "meshBuildingBlockRunner", &replacedRunnerUuid),
						statecheck.ExpectKnownValue(runnerAddr, tfjsonpath.New("spec").AtMapKey("restriction"), knownvalue.StringExact("PUBLIC")),
					},
				},
			},
		})
	})
}
