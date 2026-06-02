package provider

import (
	_ "embed"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	testconfig "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	xknownvalue "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// runnerPublicKey is a throwaway RSA public key (no private key) used so the backend can parse
// spec.public_key. The published example uses a truncated placeholder, so tests inject this real key.
//
//go:embed testdata/pubkey.txt
var runnerPublicKey string

func TestAccBuildingBlockRunnerResource(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		config, runnerAddr, _ := testconfig.BuildingBlockRunnerAndWorkspace(t)
		config = config.WithFirstBlock(testconfig.Descend("spec", "public_key")(testconfig.SetString(runnerPublicKey)))
		var runnerUuid string
		var replacedRunnerUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(runnerAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(runnerAddr.String(), tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(runnerAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(runnerAddr.String(), tfjsonpath.New("spec").AtMapKey("implementation_type"), knownvalue.StringExact("TERRAFORM")),
						statecheck.ExpectKnownValue(runnerAddr.String(), tfjsonpath.New("spec").AtMapKey("restriction"), knownvalue.StringExact("PRIVATE")),
						xknownvalue.Ref(runnerAddr, "meshBuildingBlockRunner", &runnerUuid),
					},
				},
				{
					Config: config.WithFirstBlock(
						testconfig.Descend("spec", "display_name")(testconfig.SetString("Updated Runner")),
					).String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(runnerAddr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(runnerAddr.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Updated Runner")),
						xknownvalue.Ref(runnerAddr, "meshBuildingBlockRunner", &runnerUuid),
					},
				},
				{
					// TODO: Change this expectation to ResourceActionUpdate once meshStack supports
					// in-place updates for implementation_type.
					Config: config.WithFirstBlock(
						testconfig.Descend("spec", "implementation_type")(testconfig.SetString("GITHUB_WORKFLOW")),
					).String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(runnerAddr.String(), plancheck.ResourceActionReplace),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(runnerAddr.String(), tfjsonpath.New("spec").AtMapKey("implementation_type"), knownvalue.StringExact("GITHUB_WORKFLOW")),
						xknownvalue.Ref(runnerAddr, "meshBuildingBlockRunner", &replacedRunnerUuid),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return replacedRunnerUuid, nil
					},
					ResourceName: runnerAddr.String(),
				},
			},
		})
	})

	t.Run("wif", func(t *testing.T) {
		config, runnerAddr, _ := testconfig.BuildingBlockRunnerAndWorkspace(t)
		config = config.WithFirstBlock(testconfig.Descend("spec", "public_key")(testconfig.SetString(runnerPublicKey)))
		var runnerUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.WithFirstBlock(
						testconfig.Descend("spec", "display_name")(testconfig.SetString("GCP WIF Runner")),
						testconfig.Descend("spec", "workload_identity_federation")(testconfig.SetRawExpr(`{
							subject = "system:serviceaccount:meshfed:my-runner"
							issuer = "https://oidc.example.com"
							gcp = {
								audience = "//iam.googleapis.com/projects/123456/locations/global/workloadIdentityPools/meshstack/providers/meshfed"
								token_path = "/var/run/secrets/workload-identity/token"
							}
						}`)),
					).String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(runnerAddr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(runnerAddr.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("GCP WIF Runner")),
						statecheck.ExpectKnownValue(runnerAddr.String(), tfjsonpath.New("spec").AtMapKey("workload_identity_federation"), xknownvalue.MapExact(map[string]knownvalue.Check{
							"subject": knownvalue.StringExact("system:serviceaccount:meshfed:my-runner"),
							"issuer":  knownvalue.StringExact("https://oidc.example.com"),
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

	t.Run("wif_validation", func(t *testing.T) {
		config, _, _ := testconfig.BuildingBlockRunnerAndWorkspace(t)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.WithFirstBlock(
						testconfig.Descend("spec", "workload_identity_federation")(testconfig.SetRawExpr(`{
							subject = "system:serviceaccount:meshfed:my-runner"
							issuer = "https://oidc.example.com"
						}`)),
					).String(),
					ExpectError: regexp.MustCompile("At least one provider configuration must be set"),
				},
				{
					Config: config.WithFirstBlock(
						testconfig.Descend("spec", "workload_identity_federation")(testconfig.SetRawExpr(`{
							issuer = "https://oidc.example.com"
							gcp = {
								audience = "//iam.googleapis.com/projects/123456/locations/global/workloadIdentityPools/meshstack/providers/meshfed"
								token_path = "/var/run/secrets/workload-identity/token"
							}
						}`)),
					).String(),
					ExpectError: regexp.MustCompile(`(?s)(Missing required argument.*\bsubject\b|\bsubject\b.*is required)`),
				},
				{
					Config: config.WithFirstBlock(
						testconfig.Descend("spec", "workload_identity_federation")(testconfig.SetRawExpr(`{
							subject = ""
							issuer = "https://oidc.example.com"
							gcp = {
								audience = "//iam.googleapis.com/projects/123456/locations/global/workloadIdentityPools/meshstack/providers/meshfed"
								token_path = "/var/run/secrets/workload-identity/token"
							}
						}`)),
					).String(),
					ExpectError: regexp.MustCompile(`(?s)subject.*must not be empty\s+or whitespace`),
				},
			},
		})
	})

	t.Run("restriction_replace_mock_only", func(t *testing.T) {
		if !IsMockClientTest() {
			t.Skip("mock-only test: PUBLIC restriction may require admin permissions in real meshStack")
		}

		config, runnerAddr, _ := testconfig.BuildingBlockRunnerAndWorkspace(t)
		var runnerUuid string
		var replacedRunnerUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						xknownvalue.Ref(runnerAddr, "meshBuildingBlockRunner", &runnerUuid),
						statecheck.ExpectKnownValue(runnerAddr.String(), tfjsonpath.New("spec").AtMapKey("restriction"), knownvalue.StringExact("PRIVATE")),
					},
				},
				{
					// TODO: Change this expectation to ResourceActionUpdate once meshStack supports
					// in-place updates for restriction.
					Config: config.WithFirstBlock(
						testconfig.Descend("spec", "restriction")(testconfig.SetString("PUBLIC")),
					).String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(runnerAddr.String(), plancheck.ResourceActionReplace),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						xknownvalue.Ref(runnerAddr, "meshBuildingBlockRunner", &replacedRunnerUuid),
						statecheck.ExpectKnownValue(runnerAddr.String(), tfjsonpath.New("spec").AtMapKey("restriction"), knownvalue.StringExact("PUBLIC")),
					},
				},
			},
		})
	})
}
