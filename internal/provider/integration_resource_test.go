package provider

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
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
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// Addresses of the blocks in examples/resources/meshstack_integration/resource-test-*.tf, one per
// integration type. The Entra ID steps are owned by the pre-seeded admin workspace, because
// meshStack permits an Entra ID integration nowhere else; the rest own a freshly created one.
const (
	githubIntegrationAddr      = "meshstack_integration.example_github"
	azureDevopsIntegrationAddr = "meshstack_integration.example_azure_devops"
	gitlabIntegrationAddr      = "meshstack_integration.example_gitlab"
	entraIDIntegrationAddr     = "meshstack_integration.example_entra_id"
)

// The sha256 of "updated-plaintext-secret", which is what non_ephemeral_secret writes to
// secret_version — so rotating either secret to that value plans this hash.
const rotatedSecretVersion = "b889814ec3c1da42df5abf57be4e989de7411b326ba30050fea6366185c0e206"

func nonEphemeralSecretVersion(value string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
}

// integrationStepConfig is an integration step with the workspace that owns it. The Entra ID steps
// (8 through 12) name the admin workspace literally, so they take no workspace step of their own.
func integrationStepConfig(t *testing.T, index int) string {
	t.Helper()
	if index >= 8 && index <= 12 {
		return examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "integration", index),
			examples.Resource.TestSupportConfigs(t, "workspace", "variables"),
		)
	}
	return examples.JoinTestStepConfigs(
		examples.Resource.TestStepConfig(t, "integration", index),
		examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
	)
}

// azureDevopsPatPath is a factory (fresh path per call) to work around the slice copy/clone bug
// in tfjsonpath.Path.AtMapKey. Mirrors aksSecretPath in the platform test.
func azureDevopsPatPath() tfjsonpath.Path {
	return tfjsonpath.New("spec").AtMapKey("config").AtMapKey("azuredevops").AtMapKey("personal_access_token")
}

// A factory for the same reason as azureDevopsPatPath.
func entraIdClientSecretPath() tfjsonpath.Path {
	return tfjsonpath.New("spec").AtMapKey("config").AtMapKey("entraid").AtMapKey("client_secret")
}

func entraIdIdpAliasPath() tfjsonpath.Path {
	return tfjsonpath.New("spec").AtMapKey("config").AtMapKey("entraid").AtMapKey("idp_alias")
}

// integrationCreateUpdateSteps is the create → rename → import run every integration type shares.
// createIndex and updateIndex name the step files; suffix and displayName drive the spec assertions.
func integrationCreateUpdateSteps(t *testing.T, addr, workspace, suffix, displayName string, createIndex, updateIndex int, vars tfconfig.Variables, resourceUuid *string) []resource.TestStep {
	t.Helper()
	return []resource.TestStep{
		{
			Config:          integrationStepConfig(t, createIndex),
			ConfigVariables: vars,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(addr, plancheck.ResourceActionCreate),
				},
			},
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(addr, tfjsonpath.New("metadata"), checkIntegrationMetadata(workspace)),
				statecheck.ExpectKnownValue(addr, tfjsonpath.New("spec"), checkIntegrationSpec(suffix, displayName)),
				statecheck.ExpectKnownValue(addr, tfjsonpath.New("status"), checkIntegrationStatus(knownvalue.Null())),
				xknownvalue.Ref(addr, "meshIntegration", resourceUuid),
			},
		},
		{
			Config:          integrationStepConfig(t, updateIndex),
			ConfigVariables: vars,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
				},
			},
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(addr, tfjsonpath.New("spec"), checkIntegrationSpec(suffix, updatedIntegrationName(displayName))),
				xknownvalue.Ref(addr, "meshIntegration", resourceUuid),
			},
		},
	}
}

// updatedIntegrationName mirrors the rename the -test- step files apply to spec.display_name.
func updatedIntegrationName(displayName string) string {
	return strings.Replace(displayName, "Integration", "Updated Integration", 1)
}

// integrationImportStep imports the integration by the uuid the create step recorded.
func integrationImportStep(addr string, vars tfconfig.Variables, resourceUuid *string) resource.TestStep {
	return resource.TestStep{
		ImportState:     true,
		ImportStateKind: resource.ImportBlockWithID,
		ConfigVariables: vars,
		ImportStateIdFunc: func(state *terraform.State) (string, error) {
			return *resourceUuid, nil
		},
		ResourceName: addr,
	}
}

func TestAccIntegrationResource(t *testing.T) {
	t.Parallel()

	t.Run("01_github", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := NewVariablesWithSuffix(suffix)
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: append(
				integrationCreateUpdateSteps(t, githubIntegrationAddr, "test-ws-"+suffix, "01_github", "GitHub Integration", 1, 2, vars, &resourceUuid),
				integrationImportStep(githubIntegrationAddr, vars, &resourceUuid),
			),
		})
	})

	t.Run("02_azure_devops", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := NewVariablesWithSuffix(suffix)
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: append(
				integrationCreateUpdateSteps(t, azureDevopsIntegrationAddr, "test-ws-"+suffix, "02_azure_devops", "Azure DevOps Integration", 3, 4, vars, &resourceUuid),
				// A different value gives a different secret_version hash, which rotates secret_value.
				resource.TestStep{
					Config:          integrationStepConfig(t, 5),
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(azureDevopsIntegrationAddr, plancheck.ResourceActionUpdate),
							plancheck.ExpectKnownValue(azureDevopsIntegrationAddr, azureDevopsPatPath().AtMapKey("secret_version"), knownvalue.StringExact(rotatedSecretVersion)),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(azureDevopsIntegrationAddr, tfjsonpath.New("metadata"), checkIntegrationMetadata("test-ws-"+suffix)),
						statecheck.ExpectKnownValue(azureDevopsIntegrationAddr, tfjsonpath.New("spec"), xknownvalue.MapExact(map[string]knownvalue.Check{
							"display_name": knownvalue.StringExact("Azure DevOps Integration"),
							"config":       checkAzureDevopsIntegrationConfig(rotatedSecretVersion),
						})),
						statecheck.ExpectKnownValue(azureDevopsIntegrationAddr, tfjsonpath.New("status"), checkIntegrationStatus(knownvalue.Null())),
						xknownvalue.Ref(azureDevopsIntegrationAddr, "meshIntegration", &resourceUuid),
					},
				},
				// On import the config wants secret_version as the value's sha256, but the backend
				// returns its own hash, so the two differ and the first plan sends the secret again.
				// That plan is expected, not drift, and matches the platform AKS example.
				resource.TestStep{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ConfigVariables: vars,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName:       azureDevopsIntegrationAddr,
					ExpectNonEmptyPlan: true,
					ImportPlanChecks: resource.ImportPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(azureDevopsIntegrationAddr, plancheck.ResourceActionUpdate),
							plancheck.ExpectUnknownValue(azureDevopsIntegrationAddr, azureDevopsPatPath().AtMapKey("secret_hash")),
							plancheck.ExpectKnownValue(azureDevopsIntegrationAddr, azureDevopsPatPath().AtMapKey("secret_version"), knownvalue.StringExact(rotatedSecretVersion)),
						},
					},
				},
			),
		})
	})

	t.Run("03_gitlab", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := NewVariablesWithSuffix(suffix)
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: append(
				integrationCreateUpdateSteps(t, gitlabIntegrationAddr, "test-ws-"+suffix, "03_gitlab", "GitLab Integration", 6, 7, vars, &resourceUuid),
				integrationImportStep(gitlabIntegrationAddr, vars, &resourceUuid),
			),
		})
	})

	t.Run("04_entra_id", func(t *testing.T) {
		vars := NewVariablesWithSuffix(acctest.RandString(8))
		var resourceUuid string

		steps := integrationCreateUpdateSteps(t, entraIDIntegrationAddr, AdminWorkspaceIdentifier, "04_entra_id", "Entra ID Integration", 8, 9, vars, &resourceUuid)
		// Entra ID reports a redirect_url in status, unlike the other types.
		steps[0].ConfigStateChecks[2] = statecheck.ExpectKnownValue(entraIDIntegrationAddr, tfjsonpath.New("status"), checkIntegrationStatus(xknownvalue.MapExact(map[string]knownvalue.Check{
			"redirect_url": xknownvalue.NotEmptyString(),
		})))

		ApplyAndTest(t, resource.TestCase{
			Steps: append(steps,
				// Applying this would delete the identity provider the integration points at, so the plan
				// has to fail rather than destroy and recreate.
				resource.TestStep{
					Config:          integrationStepConfig(t, 10),
					ConfigVariables: vars,
					ExpectError:     regexp.MustCompile(`identity provider alias of an Entra ID integration cannot be changed`),
				},
				// A different value gives a different secret_version hash, which rotates secret_value.
				resource.TestStep{
					Config:          integrationStepConfig(t, 11),
					ConfigVariables: vars,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(entraIDIntegrationAddr, plancheck.ResourceActionUpdate),
							plancheck.ExpectKnownValue(entraIDIntegrationAddr, entraIdClientSecretPath().AtMapKey("secret_version"), knownvalue.StringExact(rotatedSecretVersion)),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(entraIDIntegrationAddr, tfjsonpath.New("spec"), checkIntegrationSpec("04_entra_id", "Entra ID Integration")),
						xknownvalue.Ref(entraIDIntegrationAddr, "meshIntegration", &resourceUuid),
					},
				},
				// On import the config wants secret_version as the value's sha256, but the backend returns
				// its own hash, so the first plan sends the secret again. Same as 02_azure_devops.
				resource.TestStep{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ConfigVariables: vars,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName:       entraIDIntegrationAddr,
					ExpectNonEmptyPlan: true,
					ImportPlanChecks: resource.ImportPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(entraIDIntegrationAddr, plancheck.ResourceActionUpdate),
							plancheck.ExpectUnknownValue(entraIDIntegrationAddr, entraIdClientSecretPath().AtMapKey("secret_hash")),
						},
					},
				},
			),
		})
	})

	// Mock-only: adopting a fixed alias twice would collide on the real backend, which rejects an alias
	// another Entra ID integration already uses.
	t.Run("05_entra_id_adopt_mock_only", func(t *testing.T) {
		if !IsMockClientTest() {
			t.Skip("mock-only test: a fixed idp_alias cannot be re-adopted across runs on a real meshStack")
		}

		vars := NewVariablesWithSuffix(acctest.RandString(8))

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Creating with an alias adopts it rather than having meshStack generate one.
				{
					Config:          integrationStepConfig(t, 12),
					ConfigVariables: vars,
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(entraIDIntegrationAddr, entraIdIdpAliasPath(), knownvalue.StringExact("pre-existing-idp")),
					},
				},
				// Dropping it from the configuration keeps the adopted alias, so the plan is empty.
				{
					Config:          integrationStepConfig(t, 8),
					ConfigVariables: vars,
					PlanOnly:        true,
				},
			},
		})
	})
}

func checkIntegrationMetadata(workspace string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"uuid":               xknownvalue.NotEmptyString(),
		"owned_by_workspace": knownvalue.StringExact(workspace),
	})
}

func checkIntegrationSpec(exampleSuffix string, displayName string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"display_name": knownvalue.StringExact(displayName),
		"config":       checkIntegrationConfig(exampleSuffix),
	})
}

func checkIntegrationConfig(exampleSuffix string) knownvalue.Check {
	switch exampleSuffix {
	case "01_github":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
			"github": xknownvalue.MapExact(map[string]knownvalue.Check{
				"owner":    knownvalue.StringExact("my-org"),
				"base_url": knownvalue.StringExact("https://github.com"),
				"app_id":   knownvalue.StringExact("123456"),
				"app_private_key": xknownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    xknownvalue.NotEmptyString(),
					"secret_version": xknownvalue.NotEmptyString(),
				}),
				"runner_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
					"uuid": knownvalue.StringExact(SharedBuildingBlockRunnerUuid),
					"kind": knownvalue.StringExact("meshBuildingBlockRunner"),
				}),
			}),
			"azuredevops": knownvalue.Null(),
			"gitlab":      knownvalue.Null(),
			"entraid":     knownvalue.Null(),
		})
	case "02_azure_devops":
		return checkAzureDevopsIntegrationConfig(nonEphemeralSecretVersion("mock-pat-token-12345"))
	case "03_gitlab":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
			"github":      knownvalue.Null(),
			"azuredevops": knownvalue.Null(),
			"gitlab": xknownvalue.MapExact(map[string]knownvalue.Check{
				"base_url": knownvalue.StringExact("https://gitlab.com"),
				"runner_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
					"uuid": knownvalue.StringExact(SharedBuildingBlockRunnerUuid),
					"kind": knownvalue.StringExact("meshBuildingBlockRunner"),
				}),
			}),
			"entraid": knownvalue.Null(),
		})
	case "04_entra_id":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
			"github":      knownvalue.Null(),
			"azuredevops": knownvalue.Null(),
			"gitlab":      knownvalue.Null(),
			"entraid": xknownvalue.MapExact(map[string]knownvalue.Check{
				"tenant_id": knownvalue.StringExact("xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"),
				"client_id": knownvalue.StringExact("yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy"),
				"client_secret": xknownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    xknownvalue.NotEmptyString(),
					"secret_version": xknownvalue.NotEmptyString(),
				}),
				"idp_alias": xknownvalue.NotEmptyString(),
			}),
		})
	default:
		panic("unknown example suffix: " + exampleSuffix)
	}
}

func checkAzureDevopsIntegrationConfig(patSecretVersion string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"github": knownvalue.Null(),
		"azuredevops": xknownvalue.MapExact(map[string]knownvalue.Check{
			"base_url":     knownvalue.StringExact("https://dev.azure.com"),
			"organization": knownvalue.StringExact("my-organization"),
			"personal_access_token": xknownvalue.MapExact(map[string]knownvalue.Check{
				"secret_value":   knownvalue.Null(),
				"secret_hash":    xknownvalue.NotEmptyString(),
				"secret_version": knownvalue.StringExact(patSecretVersion),
			}),
			"runner_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
				"uuid": knownvalue.StringExact(SharedBuildingBlockRunnerUuid),
				"kind": knownvalue.StringExact("meshBuildingBlockRunner"),
			}),
		}),
		"gitlab":  knownvalue.Null(),
		"entraid": knownvalue.Null(),
	})
}

func checkIntegrationStatus(entraId knownvalue.Check) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"entraid":                      entraId,
		"is_built_in":                  knownvalue.Bool(false),
		"workload_identity_federation": knownvalue.Null(),
	})
}

// TestAccIntegrationResourceEmptyConfig covers a spec.config without any variant, which used to crash the
// provider at apply time. The schema rejects it at plan time, so no backend is involved.
func TestAccIntegrationResourceEmptyConfig(t *testing.T) {
	ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{{
		Config:          integrationStepConfig(t, 14),
		ConfigVariables: NewVariablesWithSuffix(acctest.RandString(8)),
		ExpectError:     regexp.MustCompile(`exactly one is\s+required`),
	}}})
}
