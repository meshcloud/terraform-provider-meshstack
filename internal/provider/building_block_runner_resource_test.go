package provider

import (
	_ "embed"
	"maps"
	"regexp"
	"testing"

	tfconfig "github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	xknownvalue "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// runnerPublicKey is a throwaway RSA public key (no private key) used so the backend can parse
// spec.public_key. The published example uses a truncated placeholder, so tests inject this real key.
//
//go:embed testdata/pubkey.txt
var runnerPublicKey string

// Addresses of the blocks in examples/resources/meshstack_building_block_runner/resource-test-*.tf.
const (
	buildingBlockRunnerAddr    = "meshstack_building_block_runner.example"
	buildingBlockRunnerWifAddr = "meshstack_building_block_runner.example_with_wif"
)

// runnerStepConfig is a runner step with the workspace that owns it.
func runnerStepConfig(t *testing.T, index int) string {
	t.Helper()
	return examples.JoinTestStepConfigs(
		examples.Resource.TestStepConfig(t, "building_block_runner", index, "variables"),
		examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
	)
}

// runnerVariables carries the run suffix plus the real public key the step files read.
func runnerVariables() tfconfig.Variables {
	vars := SuffixVariables(acctest.RandString(8))
	vars["runner_public_key"] = tfconfig.StringVariable(runnerPublicKey)
	return vars
}

func TestAccBuildingBlockRunnerResource(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		vars := runnerVariables()
		var runnerUuid string
		var replacedRunnerUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          runnerStepConfig(t, 1),
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockRunnerAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockRunnerAddr, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(buildingBlockRunnerAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(buildingBlockRunnerAddr, tfjsonpath.New("spec").AtMapKey("implementation_type"), knownvalue.StringExact("TERRAFORM")),
						statecheck.ExpectKnownValue(buildingBlockRunnerAddr, tfjsonpath.New("spec").AtMapKey("restriction"), knownvalue.StringExact("PRIVATE")),
						xknownvalue.Ref(buildingBlockRunnerAddr, "meshBuildingBlockRunner", &runnerUuid),
					},
				},
				{
					Config:          runnerStepConfig(t, 2),
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockRunnerAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockRunnerAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Updated Runner")),
						xknownvalue.Ref(buildingBlockRunnerAddr, "meshBuildingBlockRunner", &runnerUuid),
					},
				},
				{
					// TODO: Change this expectation to ResourceActionUpdate once meshStack supports
					// in-place updates for implementation_type.
					Config:          runnerStepConfig(t, 3),
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockRunnerAddr, plancheck.ResourceActionReplace),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockRunnerAddr, tfjsonpath.New("spec").AtMapKey("implementation_type"), knownvalue.StringExact("GITHUB_WORKFLOW")),
						xknownvalue.Ref(buildingBlockRunnerAddr, "meshBuildingBlockRunner", &replacedRunnerUuid),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ConfigVariables: vars,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return replacedRunnerUuid, nil
					},
					ResourceName: buildingBlockRunnerAddr,
				},
			},
		})
	})

	t.Run("wif", func(t *testing.T) {
		var runnerUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          runnerStepConfig(t, 4),
					ConfigVariables: runnerVariables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockRunnerAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockRunnerAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("GCP WIF Runner")),
						statecheck.ExpectKnownValue(buildingBlockRunnerAddr, tfjsonpath.New("spec").AtMapKey("workload_identity_federation"), xknownvalue.MapExact(map[string]knownvalue.Check{
							"subject_template": knownvalue.StringExact("system:serviceaccount:meshfed:my-runner"),
							"issuer":           knownvalue.StringExact("https://oidc.example.com"),
							"gcp": xknownvalue.MapExact(map[string]knownvalue.Check{
								"audience":   knownvalue.StringExact("//iam.googleapis.com/projects/123456/locations/global/workloadIdentityPools/meshstack/providers/meshfed"),
								"token_path": knownvalue.StringExact("/var/run/secrets/workload-identity/token"),
							}),
							"aws":   knownvalue.Null(),
							"azure": knownvalue.Null(),
						})),
						xknownvalue.Ref(buildingBlockRunnerAddr, "meshBuildingBlockRunner", &runnerUuid),
					},
				},
			},
		})
	})

	t.Run("wif_subject_template", func(t *testing.T) {
		exampleTemplate := "system:serviceaccount:namespace:workspace.{{ workspaceIdentifier }}.buildingblockdefinition.{{ buildingBlockDefinitionUuid }}"
		updatedTemplate := "system:serviceaccount:namespace:bbd.{{ buildingBlockDefinitionUuid }}"
		vars := runnerVariables()
		updatedVars := maps.Clone(vars)
		updatedVars["subject_template"] = tfconfig.StringVariable(updatedTemplate)
		var runnerUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          runnerStepConfig(t, 9),
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockRunnerWifAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockRunnerWifAddr, tfjsonpath.New("spec").AtMapKey("workload_identity_federation"), xknownvalue.MapExact(map[string]knownvalue.Check{
							"subject_template": knownvalue.StringExact(exampleTemplate),
							"issuer":           knownvalue.StringExact("https://oidc.example.com"),
							"gcp": xknownvalue.MapExact(map[string]knownvalue.Check{
								"audience":   knownvalue.StringExact("gcp-workload-identity-provider:namespace"),
								"token_path": knownvalue.StringExact("/var/run/secrets/workload-identity/token"),
							}),
							"aws":   knownvalue.Null(),
							"azure": knownvalue.Null(),
						})),
						xknownvalue.Ref(buildingBlockRunnerWifAddr, "meshBuildingBlockRunner", &runnerUuid),
					},
				},
				{
					Config:          runnerStepConfig(t, 9),
					ConfigVariables: updatedVars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockRunnerWifAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(buildingBlockRunnerWifAddr, tfjsonpath.New("spec").AtMapKey("workload_identity_federation").AtMapKey("subject_template"), knownvalue.StringExact(updatedTemplate)),
						xknownvalue.Ref(buildingBlockRunnerWifAddr, "meshBuildingBlockRunner", &runnerUuid),
					},
				},
			},
		})
	})

	t.Run("wif_validation", func(t *testing.T) {
		vars := runnerVariables()

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          runnerStepConfig(t, 5),
					ConfigVariables: vars,
					ExpectError:     regexp.MustCompile("At least one provider configuration must be set"),
				},
				{
					Config:          runnerStepConfig(t, 6),
					ConfigVariables: vars,
					ExpectError:     regexp.MustCompile(`(?s)attribute "subject_template" is required`),
				},
				{
					Config:          runnerStepConfig(t, 7),
					ConfigVariables: vars,
					ExpectError:     regexp.MustCompile(`(?s)subject_template.*must not\s+be empty\s+or whitespace`),
				},
			},
		})
	})

	t.Run("restriction_replace_mock_only", func(t *testing.T) {
		if !IsMockClientTest() {
			t.Skip("mock-only test: PUBLIC restriction may require admin permissions in real meshStack")
		}

		vars := runnerVariables()
		var runnerUuid string
		var replacedRunnerUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          runnerStepConfig(t, 1),
					ConfigVariables: vars,
					ConfigStateChecks: []statecheck.StateCheck{
						xknownvalue.Ref(buildingBlockRunnerAddr, "meshBuildingBlockRunner", &runnerUuid),
						statecheck.ExpectKnownValue(buildingBlockRunnerAddr, tfjsonpath.New("spec").AtMapKey("restriction"), knownvalue.StringExact("PRIVATE")),
					},
				},
				{
					// TODO: Change this expectation to ResourceActionUpdate once meshStack supports
					// in-place updates for restriction.
					Config:          runnerStepConfig(t, 8),
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(buildingBlockRunnerAddr, plancheck.ResourceActionReplace),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						xknownvalue.Ref(buildingBlockRunnerAddr, "meshBuildingBlockRunner", &replacedRunnerUuid),
						statecheck.ExpectKnownValue(buildingBlockRunnerAddr, tfjsonpath.New("spec").AtMapKey("restriction"), knownvalue.StringExact("PUBLIC")),
					},
				},
			},
		})
	})
}
