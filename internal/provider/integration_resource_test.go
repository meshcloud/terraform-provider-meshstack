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
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// azureDevopsPatPath is a factory (fresh path per call) to work around the slice copy/clone bug
// in tfjsonpath.Path.AtMapKey. Mirrors aksSecretPath in the platform test.
func azureDevopsPatPath() tfjsonpath.Path {
	return tfjsonpath.New("spec").AtMapKey("config").AtMapKey("azuredevops").AtMapKey("personal_access_token")
}

// A factory for the same reason as azureDevopsPatPath.
func entraIdClientSecretPath() tfjsonpath.Path {
	return tfjsonpath.New("spec").AtMapKey("config").AtMapKey("entraid").AtMapKey("client_secret")
}

func seedAdminWorkspace(mock clientmock.Client) {
	mock.Workspace.Store.Set(AdminWorkspaceIdentifier, &client.MeshWorkspace{
		Metadata: client.MeshWorkspaceMetadata{Name: AdminWorkspaceIdentifier, Tags: map[string][]string{}},
		Spec:     client.MeshWorkspaceSpec{DisplayName: "Admin Workspace"},
	})
}

func TestAccIntegrationResource(t *testing.T) {
	t.Parallel()

	t.Run("01_github", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		resourceAddress := "meshstack_integration.example_github"
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/integration/01_github", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata"), checkIntegrationMetadata("test-ws-"+suffix)),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec"), checkIntegrationSpec("01_github", "GitHub Integration")),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("status"), checkIntegrationStatus(knownvalue.Null())),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				{
					Config: hclStep(t, "resources/integration/01_github", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec"), checkIntegrationSpec("01_github", "GitHub Updated Integration")),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: resourceAddress,
				},
			},
		})
	})

	t.Run("02_azure_devops", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		resourceAddress := "meshstack_integration.example_azure_devops"
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/integration/02_azure_devops", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata"), checkIntegrationMetadata("test-ws-"+suffix)),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec"), checkIntegrationSpec("02_azure_devops", "Azure DevOps Integration")),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("status"), checkIntegrationStatus(knownvalue.Null())),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				{
					Config: hclStep(t, "resources/integration/02_azure_devops", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec"), checkIntegrationSpec("02_azure_devops", "Azure DevOps Updated Integration")),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				// A different value gives a different secret_version hash, which rotates secret_value.
				{
					Config: hclStep(t, "resources/integration/02_azure_devops", 3, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
							plancheck.ExpectKnownValue(resourceAddress, azureDevopsPatPath().AtMapKey("secret_version"), knownvalue.StringExact("b889814ec3c1da42df5abf57be4e989de7411b326ba30050fea6366185c0e206")),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata"), checkIntegrationMetadata("test-ws-"+suffix)),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec"), checkIntegrationSpec("02_azure_devops", "Azure DevOps Integration")),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("status"), checkIntegrationStatus(knownvalue.Null())),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				// On import the config wants secret_version as the value's sha256, but the backend
				// returns its own hash, so the two differ and the first plan sends the secret again.
				// That plan is expected, not drift, and matches the platform AKS example.
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName:       resourceAddress,
					ExpectNonEmptyPlan: true,
					ImportPlanChecks: resource.ImportPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
							plancheck.ExpectUnknownValue(resourceAddress, azureDevopsPatPath().AtMapKey("secret_hash")),
							plancheck.ExpectKnownValue(resourceAddress, azureDevopsPatPath().AtMapKey("secret_version"), knownvalue.StringExact("b889814ec3c1da42df5abf57be4e989de7411b326ba30050fea6366185c0e206")),
						},
					},
				},
			},
		})
	})

	t.Run("03_gitlab", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		resourceAddress := "meshstack_integration.example_gitlab"
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/integration/03_gitlab", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata"), checkIntegrationMetadata("test-ws-"+suffix)),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec"), checkIntegrationSpec("03_gitlab", "GitLab Integration")),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("status"), checkIntegrationStatus(knownvalue.Null())),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				{
					Config: hclStep(t, "resources/integration/03_gitlab", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec"), checkIntegrationSpec("03_gitlab", "GitLab Updated Integration")),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: resourceAddress,
				},
			},
		})
	})

	t.Run("04_entra_id", func(t *testing.T) {
		resourceAddress := "meshstack_integration.example_entra_id"
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/integration/04_entra_id", 1, nil),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata"), checkIntegrationMetadata(AdminWorkspaceIdentifier)),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec"), checkIntegrationSpec("04_entra_id", "Entra ID Integration")),
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("status"), checkIntegrationStatus(xknownvalue.MapExact(map[string]knownvalue.Check{
							"redirect_url": xknownvalue.NotEmptyString(),
						}))),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				{
					Config: hclStep(t, "resources/integration/04_entra_id", 2, nil),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec"), checkIntegrationSpec("04_entra_id", "Entra ID Updated Integration")),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				// Applying this would delete the identity provider the integration points at, so the plan
				// has to fail rather than destroy and recreate.
				{
					Config:      hclStep(t, "resources/integration/04_entra_id", 3, nil),
					ExpectError: expectErrorWords("alias", "Entra", "changed"),
				},
				// A different value gives a different secret_version hash, which rotates secret_value.
				{
					Config: hclStep(t, "resources/integration/04_entra_id", 4, nil),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
							plancheck.ExpectKnownValue(resourceAddress, entraIdClientSecretPath().AtMapKey("secret_version"), knownvalue.StringExact("b889814ec3c1da42df5abf57be4e989de7411b326ba30050fea6366185c0e206")),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec"), checkIntegrationSpec("04_entra_id", "Entra ID Integration")),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				// On import the config wants secret_version as the value's sha256, but the backend returns
				// its own hash, so the first plan sends the secret again. Same as 02_azure_devops.
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName:       resourceAddress,
					ExpectNonEmptyPlan: true,
					ImportPlanChecks: resource.ImportPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
							plancheck.ExpectUnknownValue(resourceAddress, entraIdClientSecretPath().AtMapKey("secret_hash")),
						},
					},
				},
			},
		}, SeedingMock(seedAdminWorkspace))
	})

	// Mock-only: adopting a fixed alias twice would collide on the real backend, which rejects an alias
	// another Entra ID integration already uses.
	t.Run("05_entra_id_adopt_mock_only", func(t *testing.T) {
		if !IsMockClientTest() {
			t.Skip("mock-only test: a fixed idp_alias cannot be re-adopted across runs on a real meshStack")
		}

		resourceAddress := "meshstack_integration.test"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Creating with an alias adopts it rather than having meshStack generate one.
				{
					Config: hclStep(t, "resources/integration/05_entra_id_adopt_mock_only", 1, nil),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, entraIdIdpAliasPath(), knownvalue.StringExact("pre-existing-idp")),
					},
				},
				// Dropping it from the configuration keeps the adopted alias, so the plan is empty.
				{
					Config:   hclStep(t, "resources/integration/05_entra_id_adopt_mock_only", 2, nil),
					PlanOnly: true,
				},
			},
		})
	})
}

func entraIdIdpAliasPath() tfjsonpath.Path {
	return tfjsonpath.New("spec").AtMapKey("config").AtMapKey("entraid").AtMapKey("idp_alias")
}

func checkIntegrationMetadata(ownedByWorkspace string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"uuid":               xknownvalue.NotEmptyString(),
		"owned_by_workspace": knownvalue.StringExact(ownedByWorkspace),
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
		return xknownvalue.MapExact(map[string]knownvalue.Check{
			"github": knownvalue.Null(),
			"azuredevops": xknownvalue.MapExact(map[string]knownvalue.Check{
				"base_url":     knownvalue.StringExact("https://dev.azure.com"),
				"organization": knownvalue.StringExact("my-organization"),
				"personal_access_token": xknownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    xknownvalue.NotEmptyString(),
					"secret_version": xknownvalue.NotEmptyString(),
				}),
				"runner_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
					"uuid": knownvalue.StringExact(SharedBuildingBlockRunnerUuid),
					"kind": knownvalue.StringExact("meshBuildingBlockRunner"),
				}),
			}),
			"gitlab":  knownvalue.Null(),
			"entraid": knownvalue.Null(),
		})
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
	vars := hclStepVars{"suffix": acctest.RandString(8)}

	ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{{
		Config:      hclStep(t, "resources/integration/empty_config", 1, vars),
		ExpectError: expectErrorWords("exactly", "required"),
	}}})
}
